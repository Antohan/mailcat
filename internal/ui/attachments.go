package ui

// Модал вложений (клавиша v на экране деталей): s — сохранить в
// ~/Downloads/mailcat, o — открыть временно (TMPDIR/mailcat + системное
// приложение). Лимит размера проверяется ДО скачивания (ews.GetAttachment);
// опасные расширения — предупреждение перед открытием (SECURITY.md §5).

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/platform"
	"mailcat/internal/security"
)

// showAttachments открывает модал вложений текущего письма. back — возврат
// на исходный экран (V-15): модал открывается и из панели чтения, и со
// страницы деталей, Esc должен возвращать туда, откуда пришли.
func (a *App) showAttachments(attachments []mail.Attachment, back func()) {
	a.pages.RemovePage(pageAttachments)
	list := tview.NewList().ShowSecondaryText(true)
	list.SetBorder(true).SetTitle(" Вложения (s — сохранить, o — открыть, Esc — назад) ")

	for _, att := range attachments {
		a := att
		main, secondary := attachmentListEntry(a)
		list.AddItem(main, secondary, 0, func() {})
	}
	if len(attachments) == 0 {
		list.AddItem("(в письме нет вложений)", "", 0, nil)
	}

	selected := func() (mail.Attachment, bool) {
		idx := list.GetCurrentItem()
		if idx < len(attachments) {
			return attachments[idx], true
		}
		return mail.Attachment{}, false
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			back()
			a.showHint()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 's', 'S':
				if att, ok := selected(); ok {
					a.saveAttachment(att, false)
				}
				return nil
			case 'o', 'O':
				if att, ok := selected(); ok {
					if security.IsDangerousExtension(att.Name) {
						a.confirmDangerous(att.Name, func() { a.saveAttachment(att, true) })
						return nil
					}
					a.saveAttachment(att, true)
				}
				return nil
			}
		}
		return event
	})

	a.pages.AddPage(pageAttachments, list, true, true)
	a.tapp.SetFocus(list)
	a.showHint()
}

// confirmDangerous просит явного подтверждения для опасного расширения
// (PLAN.md этап 3: предупреждение перед открытием).
func (a *App) confirmDangerous(name string, onYes func()) {
	a.pages.RemovePage(pageConfirm)
	modal := tview.NewModal().
		SetText(dangerousConfirmText(name)).
		AddButtons([]string{"Открыть (риск)", "Отмена"}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			a.pages.RemovePage(pageConfirm)
			if buttonIndex == 0 {
				onYes()
			} else {
				a.pages.SwitchToPage(pageAttachments)
				a.showHint()
			}
		})
	a.pages.AddPage(pageConfirm, modal, true, true)
	a.showHint()
}

// writeAttachment пишет данные во вложение и закрывает файл. При ошибке
// записи или закрытия недописанный файл удаляется: в загрузках не остаётся
// обрезанного вложения, неотличимого от целого.
// WHY: ошибки Close и Remove объединяются с первичной — иначе на сбойном
// диске хвост файла остаётся молча; errors.Join сохраняет все причины.
// Not to do: удалять до Close — файл ещё открыт, Remove вернёт ошибку.
func writeAttachment(path string, w io.WriteCloser, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return errors.Join(err, w.Close(), removePartial(path, w))
	}
	if err := w.Close(); err != nil {
		return errors.Join(err, removePartial(path, w))
	}
	return nil
}

// removePartial удаляет файл, который закрыл w. Если писатель сообщает другое
// имя, удаление пропускается: чужой файл не сносится из-за ошибки в пути.
func removePartial(path string, w io.WriteCloser) error {
	if n, ok := w.(interface{ Name() string }); ok && n.Name() != path {
		return fmt.Errorf("недописанный файл %s не удалён: писатель пишет в %s", path, n.Name())
	}
	return os.Remove(path)
}

// saveAttachment скачивает вложение (с проверкой лимита) и кладёт в каталог
// загрузок; при open=true — во временный каталог с открытием системным
// приложением (SECURITY.md §4.4: вложение открывается только с диска).
// Каталоги созданы при старте (prepareAttachmentDirs, T-1): сбой доступа
// виден до скачивания, в горутине ничего не создаётся; имя файла занимается
// атомарно (security.CreateUnique, O_EXCL).
func (a *App) saveAttachment(att mail.Attachment, open bool) {
	if a.attDirErr != nil {
		a.showError(fmt.Errorf("каталоги вложений недоступны: %w", a.attDirErr))
		return
	}
	dir := a.attDownloadDir
	if open {
		dir = a.attTempDir
	}
	a.setStatus("Скачивание " + security.Sanitize(att.Name) + "…")
	go func() {
		data, err := a.client.GetAttachment(a.ctx, ewsAttachment(att), int64(a.cfg.MaxAttachmentMB)*1024*1024)
		if err != nil {
			a.tapp.QueueUpdateDraw(func() { a.showError(err) })
			return
		}
		f, path, err := security.CreateUnique(dir, att.Name)
		if err != nil {
			a.tapp.QueueUpdateDraw(func() { a.showError(err) })
			return
		}
		if err := writeAttachment(path, f, data); err != nil {
			a.tapp.QueueUpdateDraw(func() {
				a.showError(fmt.Errorf("запись вложения %s: %w", filepath.Base(path), err))
			})
			return
		}
		if open {
			if err := platform.Open(path); err != nil {
				a.tapp.QueueUpdateDraw(func() { a.showError(err) })
				return
			}
			a.tapp.QueueUpdateDraw(func() { a.setStatus("Открыто: " + security.Sanitize(filepath.Base(path))) })
			return
		}
		a.tapp.QueueUpdateDraw(func() { a.setStatus("Сохранено: " + security.Sanitize(path)) })
	}()
}
