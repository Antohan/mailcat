package ui

// Модал вложений (клавиша v на экране деталей): s — сохранить в
// ~/Downloads/mailcat, o — открыть временно (TMPDIR/mailcat + системное
// приложение). Лимит размера проверяется ДО скачивания (ews.GetAttachment);
// опасные расширения — предупреждение перед открытием (SECURITY.md §5).

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/platform"
	"mailcat/internal/security"
)

// showAttachments открывает модал вложений текущего письма.
func (a *App) showAttachments(attachments []mail.Attachment) {
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
			a.pages.SwitchToPage(pageDetails)
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
}

// confirmDangerous просит явного подтверждения для опасного расширения
// (PLAN.md этап 3: предупреждение перед открытием).
func (a *App) confirmDangerous(name string, onYes func()) {
	a.pages.RemovePage(pageConfirm)
	modal := tview.NewModal().
		SetText(dangerousConfirmText(name)).
		AddButtons([]string{"Открыть (риск)", "Отмена"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage(pageConfirm)
			if buttonIndex == 0 {
				onYes()
			} else {
				a.pages.SwitchToPage(pageAttachments)
			}
		})
	a.pages.AddPage(pageConfirm, modal, true, true)
}

// saveAttachment скачивает вложение (с проверкой лимита) и кладёт в каталог
// загрузок; при open=true — во временный каталог с открытием системным
// приложением (SECURITY.md §4.4: вложение открывается только с диска).
func (a *App) saveAttachment(att mail.Attachment, open bool) {
	a.setStatus("Скачивание " + security.Sanitize(att.Name) + "…")
	go func() {
		data, err := a.client.GetAttachment(a.ctx, ewsAttachment(att), int64(a.cfg.MaxAttachmentMB)*1024*1024)
		if err != nil {
			a.tapp.QueueUpdateDraw(func() { a.showError(err) })
			return
		}
		var dir string
		if open {
			dir, err = security.TempDir()
		} else {
			dir, err = security.DownloadDir()
		}
		if err != nil {
			a.tapp.QueueUpdateDraw(func() { a.showError(err) })
			return
		}
		path := security.UniquePath(dir, att.Name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
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
