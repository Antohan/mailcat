package ui

// Каркас TUI-приложения: страницы Почта/Календарь, статусная строка,
// глобальные клавиши (1 — почта, 2 — календарь, q/Ctrl-C — выход).
// Календарь на этапе 2 — заглушка (экран появится на этапе 4).

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/mail"
	"mailcat/internal/security"
)

const (
	pageMail        = "mail"
	pageCalendar    = "calendar"
	pageDetails     = "details"
	pageLinks       = "links"
	pageAttachments = "attachments"
	pageConfirm     = "confirm"
	statusHelpFmt   = "u обновить · Enter письмо · Tab панель · 1 почта · 2 календарь · q выход"
)

// App — корневое TUI-приложение mailcat.
type App struct {
	tapp   *tview.Application
	pages  *tview.Pages
	status *tview.TextView
	detail *tview.TextView // экран деталей письма (страница pageDetails)

	cfg    *config.Config
	client *ews.Client

	// Текущее открытое письмо (экран деталей): для модалов g (ссылки)
	// и v (вложения).
	currentLinks       []mail.LinkInfo
	currentAttachments []mail.Attachment

	ctx    context.Context
	cancel context.CancelFunc

	mail *mailScreen
}

// New собирает приложение. Пароль уже получен вызывающим (Keychain), клиент
// EWS готов (один на сессию — ARCHITECTURE.md §3).
func New(cfg *config.Config, client *ews.Client) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		cfg:    cfg,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
	a.tapp = tview.NewApplication()
	a.pages = tview.NewPages()
	a.status = tview.NewTextView().SetText(statusHelpFmt)

	a.mail = newMailScreen(a)
	a.pages.AddPage(pageMail, a.mail.root, true, true)

	calendarPlaceholder := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Календарь будет реализован на этапе 4 разработки.\n\nНажмите 1, чтобы вернуться к почте.")
	a.pages.AddPage(pageCalendar, calendarPlaceholder, true, false)

	details := tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(false)
	details.SetBorder(true).SetTitle(" Письмо ")
	details.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.pages.SwitchToPage(pageMail)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g', 'G':
				a.showLinks(a.currentLinks)
				return nil
			case 'v', 'V':
				a.showAttachments(a.currentAttachments)
				return nil
			}
		}
		return event
	})
	a.detail = details
	a.pages.AddPage(pageDetails, details, true, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false)

	a.tapp.SetRoot(root, true).EnableMouse(true)
	a.tapp.SetInputCapture(a.globalKeys)
	return a
}

// Run запускает цикл событий; блокируется до выхода (q/Ctrl-C).
func (a *App) Run() error {
	a.mail.loadFolders()
	err := a.tapp.Run()
	a.cancel() // контекст сессии гасится вместе с TUI
	return err
}

// globalKeys обрабатывает клавиши, действующие на любом экране.
func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		a.tapp.Stop()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case '1':
			a.pages.SwitchToPage(pageMail)
			return nil
		case '2':
			a.pages.SwitchToPage(pageCalendar)
			return nil
		case 'q':
			a.tapp.Stop()
			return nil
		}
	}
	return event
}

// showDetails выводит полное письмо на экран деталей (страница поверх почты).
// Тело проходит через реестр ссылок ([N] host, ⚠, [IDN]) и санитайзер;
// клавиши g/v открывают модалы ссылок и вложений.
func (a *App) showDetails(m mail.Message) {
	reg := mail.NewLinkRegistry()
	var body string
	if m.BodyHTML != "" {
		body = reg.RenderBodyHTML(m.BodyHTML)
	} else {
		body = reg.RenderBody(m.BodyText)
	}
	a.currentLinks = reg.List()
	a.currentAttachments = m.Attachments

	text := FormatMessageHeader(m) + "\n" + security.Sanitize(body)
	if m.InternetMessageID != "" {
		text += "\nMessage-ID: " + security.Sanitize(m.InternetMessageID) + "\n"
	}
	if atts := FormatAttachments(m.Attachments); atts != "" {
		text += "\n" + atts + "\n\n(g — ссылки, v — вложения)"
	} else if len(a.currentLinks) > 0 {
		text += "\n\n(g — ссылки)"
	}
	a.detail.SetText(text)
	a.detail.ScrollToBeginning()
	a.pages.SwitchToPage(pageDetails)
	a.tapp.SetFocus(a.detail)
	a.setStatus("Esc — к списку · g ссылки · v вложения · q выход")
}

// setStatus выводит подсказку/статус в строку состояния (текст наш, не из EWS).
func (a *App) setStatus(text string) {
	a.status.SetText(text)
}

// showError показывает ошибку в статусной строке; не падает и не модалит:
// сетевые проблемы (в т.ч. VPN) пользователь видит и может повторить (u).
// Текст ошибок EWS — серверный: Sanitize обязателен (§3, ревью этапа 2);
// усечение — по рунам, чтобы не рвать кириллицу.
func (a *App) showError(err error) {
	msg := security.Sanitize(err.Error())
	if runes := []rune(msg); len(runes) > 200 {
		msg = string(runes[:200]) + "…"
	}
	a.status.SetText("Ошибка: " + msg + "  ·  u — повторить")
}
