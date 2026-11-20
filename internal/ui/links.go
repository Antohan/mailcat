package ui

// Модал ссылок (клавиша g на экране деталей): полный список URL письма
// с пометками ⚠/[IDN]; открытие — только явным Enter, только http/https,
// через platform.Open (SECURITY.md §4.2: полный URL виден до открытия).

import (
	"fmt"
	"net/url"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/platform"
	"mailcat/internal/security"
)

// escapeForList — Sanitize + tview.Escape в одном шаге: обязательная
// обработка строк из EWS для tview.List/tview.Modal (они интерпретируют
// style-теги безусловно; SECURITY.md §3, ревью этапов 2–3).
func escapeForList(s string) string {
	return tview.Escape(security.Sanitize(s))
}

// linkListEntry — строки пункта модала ссылок (главная/вторичная).
func linkListEntry(l mail.LinkInfo) (main, secondary string) {
	main = fmt.Sprintf("[%d] %s", l.N, escapeForList(l.Host))
	secondary = escapeForList(l.URL)
	if l.IDN {
		secondary += "  [IDN: punycode-домен — проверьте адрес]"
	}
	if l.Mismatch {
		secondary += "  ⚠ текст ссылки ведёт на другой адрес"
	}
	return main, secondary
}

// attachmentListEntry — строки пункта модала вложений.
func attachmentListEntry(a mail.Attachment) (main, secondary string) {
	main = escapeForList(a.Name) + " (" + a.SizeLabel() + ")"
	if security.IsDangerousExtension(a.Name) {
		secondary = "⚠ ОПАСНОЕ РАСШИРЕНИЕ — перед открытием убедитесь, что отправителю можно доверять"
	} else {
		secondary = "приложение по типу содержимого"
	}
	return main, secondary
}

// dangerousConfirmText — текст модала подтверждения для опасного расширения.
func dangerousConfirmText(name string) string {
	return "«" + escapeForList(name) + "» — опасное расширение файла.\n" +
		"Макросы и скрипты из почты могут скомпрометировать компьютер.\n\nОткрыть всё равно?"
}

// showLinks открывает модал ссылок. back — возврат на исходный экран
// (страница + фокус): модал открывается и из письма, и из встречи, и из
// повестки, Esc должен возвращать туда, откуда пришли (UI-4).
func (a *App) showLinks(links []mail.LinkInfo, back func()) {
	a.pages.RemovePage(pageLinks)
	list := tview.NewList().
		ShowSecondaryText(true).
		SetMainTextStyle(tcell.StyleDefault)
	list.SetBorder(true).SetTitle(" Ссылки (Enter — открыть, Esc — назад) ")

	for _, l := range links {
		main, secondary := linkListEntry(l)
		link := l
		list.AddItem(main, secondary, 0, func() {
			a.openLink(link.URL)
		})
	}
	if len(links) == 0 {
		list.AddItem("(ссылок нет)", "", 0, nil)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			back()
			return nil
		}
		return event
	})

	a.pages.AddPage(pageLinks, list, true, true)
	a.tapp.SetFocus(list)
}

// openLink открывает ссылку системным браузером — только по явному действию
// и только http/https (SECURITY.md §4.2, §7).
func (a *App) openLink(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		a.setStatus("Отказ: открытие возможно только для http/https-ссылок")
		return
	}
	go func() {
		if err := platform.Open(security.Sanitize(rawURL)); err != nil {
			a.tapp.QueueUpdateDraw(func() { a.showError(err) })
			return
		}
		a.tapp.QueueUpdateDraw(func() { a.setStatus("Открыто: " + security.Sanitize(u.Host)) })
	}()
}
