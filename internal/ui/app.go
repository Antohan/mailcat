package ui

// Каркас TUI-приложения: страницы Почта/Календарь, статусная строка,
// глобальные клавиши (1 — почта, 2 — календарь, q/Ctrl-C — выход).
// Календарь на этапе 2 — заглушка (экран появится на этапе 4).

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/mail"
	"mailcat/internal/notify"
	"mailcat/internal/platform"
	"mailcat/internal/security"
)

const (
	pageMail        = "mail"
	pageCalendar    = "calendar"
	pageDetails     = "details"
	pageEvent       = "event"
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
	event  *tview.TextView // экран деталей встречи (страница pageEvent)

	cfg    *config.Config
	client *ews.Client
	log    *slog.Logger

	// Текущее открытое письмо (экран деталей): для модалов g (ссылки)
	// и v (вложения).
	currentLinks       []mail.LinkInfo
	currentAttachments []mail.Attachment

	ctx    context.Context
	cancel context.CancelFunc

	mail     *mailScreen
	calendar *calendarScreen
}

// New собирает приложение. Пароль уже получен вызывающим (Keychain), клиент
// EWS готов (один на сессию — ARCHITECTURE.md §3). Логгер (nil — discard)
// нужен для §9: ошибки osascript логируются, но не прерывают TUI.
func New(cfg *config.Config, client *ews.Client, log *slog.Logger) *App {
	applyLightTheme() // светлый фон — до создания виджетов (theme.go)
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		cfg:    cfg,
		client: client,
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}
	a.tapp = tview.NewApplication()
	a.pages = tview.NewPages()
	a.status = tview.NewTextView().SetText(statusHelpFmt)

	a.mail = newMailScreen(a)
	a.pages.AddPage(pageMail, a.mail.root, true, true)

	a.calendar = newCalendarScreen(a)
	a.pages.AddPage(pageCalendar, a.calendar.root, true, false)

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

	event := tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(false)
	event.SetBorder(true).SetTitle(" Встреча ")
	event.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.pages.SwitchToPage(pageCalendar)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g', 'G':
				a.showLinks(a.currentLinks)
				return nil
			}
		}
		return event
	})
	a.event = event
	a.pages.AddPage(pageEvent, event, true, false)

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
	if a.cfg.Notify.Enabled { // enabled=false — watcher не запускается вовсе (DoD этапа 6)
		a.startWatcher()
	}
	err := a.tapp.Run()
	a.cancel() // контекст сессии гасится вместе с TUI
	return err
}

// startWatcher поднимает фоновый опрос «Входящих» (этап 6). Ошибки сети —
// тихий backoff внутри watcher.Run; уведомления — только по правилам.
func (a *App) startWatcher() {
	w := &notify.Watcher{
		Rule:      a.cfg.Notify,
		OrgDomain: a.cfg.OrgDomain,
		PageSize:  a.cfg.PageSize,
		Notify: func(title, body string) {
			// SECURITY.md §9: текст писем — через Sanitize до osascript;
			// ошибки выполнения логируются, но не прерывают TUI.
			if err := platform.Notify(security.Sanitize(title), security.Sanitize(body), a.cfg.Notify.Sound); err != nil {
				a.log.Warn("уведомление macOS не показано", "err", err.Error())
			}
		},
		OnUnread: func(n int) {
			a.tapp.QueueUpdateDraw(func() { a.mail.updateInboxUnread(n) })
		},
	}
	ps := a.cfg.PageSize
	go w.Run(a.ctx, time.Duration(a.cfg.Notify.IntervalSec)*time.Second,
		func() (notify.Page, error) {
			p, err := a.client.FindDistinguishedMessages(a.ctx, "inbox", ps, 0)
			return notify.Page{Items: mail.MessageSummaries(p.Items)}, err
		},
		func() int {
			folders, err := a.client.FindFolders(a.ctx)
			if err != nil {
				return -1
			}
			for _, f := range folders {
				if f.DisplayName == "Входящие" {
					return f.UnreadCount
				}
			}
			return -1
		})
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
			a.calendar.ensureLoaded()
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

// showEventDetails выводит детали встречи; ссылки на созвон из «Места» —
// через реестр (модал g, как в письмах; PLAN.md этап 4).
func (a *App) showEventDetails(ev mail.Event) {
	reg := mail.NewLinkRegistry()
	text := FormatEventDetails(ev)
	if ev.Location != "" {
		rendered := reg.RenderBody(ev.Location)
		if len(reg.List()) > 0 {
			text += "\nСсылки: " + security.Sanitize(rendered) + "\n(g — все ссылки)"
		}
	}
	a.currentLinks = reg.List()
	a.currentAttachments = nil
	a.event.SetText(security.Sanitize(text))
	a.event.ScrollToBeginning()
	a.pages.SwitchToPage(pageEvent)
	a.tapp.SetFocus(a.event)
	a.setStatus("Esc — к календарю · g ссылки · 1 почта · q выход")
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
