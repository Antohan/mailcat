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

// showLinks открывает модал ссылок текущего письма.
func (a *App) showLinks(links []mail.LinkInfo) {
	a.pages.RemovePage(pageLinks)
	list := tview.NewList().
		ShowSecondaryText(true).
		SetMainTextStyle(tcell.StyleDefault)
	list.SetBorder(true).SetTitle(" Ссылки (Enter — открыть, Esc — назад) ")

	for _, l := range links {
		main := fmt.Sprintf("[%d] %s", l.N, security.Sanitize(l.Host))
		secondary := security.Sanitize(l.URL)
		if l.IDN {
			secondary += "  [IDN: punycode-домен — проверьте адрес]"
		}
		if l.Mismatch {
			secondary += "  ⚠ текст ссылки ведёт на другой адрес"
		}
		link := l
		list.AddItem(main, secondary, 0, func() {
			a.openLink(link.URL)
		})
	}
	if len(links) == 0 {
		list.AddItem("(в письме нет ссылок)", "", 0, nil)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.SwitchToPage(pageDetails)
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
