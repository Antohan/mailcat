package ui

// Каркас TUI-приложения: страницы Почта/Календарь, две строки статуса
// (UI-9: сообщения — «Загрузка…»/ошибки/результаты, ниже — подсказки клавиш,
// видны всегда), глобальные клавиши (1 — почта, 2 — календарь, q/Ctrl-C —
// выход). Календарь на этапе 2 — заглушка (экран появится на этапе 4).

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
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

	// errStatusTTL — сколько висит ошибка в статус-баре до автоочистки
	// (UI-7): хватает прочитать текст, статус-бар не «залипает».
	errStatusTTL = 8 * time.Second
)

// hintSeg — «клавиша — описание» подсказки экрана (RD-6): пустое desc —
// клавиша без пояснения. opt — сегмент необязательный: formatHintFit убирает
// его на узком экране, чтобы подсказка не обрезалась (V-17).
type hintSeg struct {
	key, desc string
	opt       bool
}

// Подсказки статус-бара по экрану (UI-7): у почты и календаря свои наборы
// клавиш, модалы — свои (дублируют заголовки панелей). RD-6: клавиши и
// описания разведены по полям, красит их formatHint.
// V-7: у КАЖДОЙ клавиши есть подпись (П.3) — бейдж без пояснения читается
// как ребус. «1/2» подписано «экран», чтобы строка влезала в 100 колонок.
var (
	// V-17: «←/→ панели» — стрелки переключают панели; сегмент opt: на 100
	// колонках подсказка иначе не влезает и обрезала бы «q».
	hintMail = []hintSeg{{"↑↓", "выбор", false}, {"Enter", "читать", false}, {"u", "обновить", false}, {"v", "вложения", false}, {"g", "ссылки", false}, {"←/→", "панели", true}, {"1/2", "экран", false}, {"q", "выход", false}}
	// V-24: у панели папок свой набор — из папок можно только → в письма,
	// v/g там не работают. Все сегменты обязательные: набор укладывается в
	// 100 колонок без opt (проверяет тест), иначе терялась бы «q».
	hintFolders = []hintSeg{{"↑↓", "выбор", false}, {"Enter", "открыть", false}, {"→", "панели", false}, {"u", "обновить", false}, {"1/2", "экран", false}, {"q", "выход", false}}
	// V-35: у календаря два набора — по фокусу (сетка/повестка), как у почты.
	// V-47: «t сегодня» — opt: на 100 колонках влезает, уже — снимает formatHintFit.
	hintCalendarGrid   = []hintSeg{{"←→↑↓", "день", false}, {"n/p", "месяц", false}, {"t", "сегодня", true}, {"Enter", "повестка", false}, {"1/2", "экран", false}, {"q", "выход", false}}
	hintCalendarAgenda = []hintSeg{{"↑↓", "встреча", false}, {"Enter", "детали", false}, {"Esc", "к сетке", false}, {"g", "ссылки", false}, {"t", "сегодня", true}, {"1/2", "экран", false}, {"q", "выход", false}}
	hintDetails        = []hintSeg{{"Esc", "к списку", false}, {"g", "ссылки", false}, {"v", "вложения", false}, {"1/2", "экран", false}, {"q", "выход", false}}
	// V-15: панель чтения — стрелки листают тело, v/g работают без ухода
	// на отдельную страницу.
	hintReading     = []hintSeg{{"↑↓", "прокрутка", false}, {"Esc", "к списку", false}, {"g", "ссылки", false}, {"v", "вложения", false}, {"1/2", "экран", false}, {"q", "выход", false}}
	hintEvent       = []hintSeg{{"Esc", "к календарю", false}, {"g", "ссылки", false}, {"1/2", "экран", false}, {"q", "выход", false}}
	hintLinks       = []hintSeg{{"Enter", "открыть", false}, {"Esc", "назад", false}}
	hintAttachments = []hintSeg{{"s", "сохранить", false}, {"o", "открыть", false}, {"Esc", "назад", false}}
	hintConfirm     = []hintSeg{{"←→", "выбор", false}, {"Enter", "подтвердить", false}}
)

// bordered — панель почты/календаря с рамкой (UI-6): у Box нет геттера цвета
// рамки, поэтому цвет задаётся заново перед каждой отрисовкой.
type bordered interface {
	tview.Primitive
	HasFocus() bool
	SetBorder(bool) *tview.Box
	SetBorderColor(tcell.Color) *tview.Box
}

// focusPanel — панель и её строка-заголовок (RD-2). Рамку и заголовок
// переключает paintFocusFrames: у активной панели рамка акцентная, у
// остальных рамка скрыта.
type focusPanel struct {
	body   bordered
	header *panelHeader
}

// App — корневое TUI-приложение mailcat.
type App struct {
	tapp   *tview.Application
	pages  *tview.Pages
	status *tview.TextView // строка сообщений (UI-9): «Загрузка…», ошибки, результаты
	hint   *tview.TextView // строка подсказок (UI-9): видна всегда, ниже строки сообщений
	// hintSegs — набор подсказок текущей страницы для DrawFunc: в отрисовке
	// нельзя звать pageHintSegs, она читает GetFocus под блокировкой приложения
	// (V-17: deadlock при открытом письме). Обновляет только showHint.
	hintSegs []hintSeg
	// Верхний бар (V-6, V-11): одна строка — только «mailcat · <папка>» слева;
	// дата/время и точка связи убраны (статус связи — футер сайдбара, V-4).
	topLeft *tview.TextView
	detail  *tview.TextView // экран деталей письма (страница pageDetails)
	event   *tview.TextView // экран деталей встречи (страница pageEvent)

	// Колонка чтения письма (V-5, П.1): спейсеры по бокам a.detail; ширина
	// readWidth по центру или вся ширина минус поля на узком терминале.
	detailPage  *tview.Flex
	detailsWide bool         // текущий режим колонки (пересчёт в layoutDetails)
	detailsW    int          // ширина, под которую собрана карточка шапки
	detailsMsg  mail.Message // письмо страницы деталей: пересборка шапки на resize
	detailsTail string       // тело+вложения+подсказки: кэш, чтобы resize не перегенерировал реестр ссылок

	cfg    *config.Config
	client *ews.Client
	log    *slog.Logger

	// Текущее открытое письмо (экран деталей): для модалов g (ссылки)
	// и v (вложения).
	currentLinks       []mail.LinkInfo
	currentAttachments []mail.Attachment

	ctx    context.Context
	cancel context.CancelFunc

	// Поколение строки сообщений (UI-7): showError запускает таймер
	// автоочистки; каждое новое сообщение меняет поколение, и устаревший
	// таймер не затирает свежий статус.
	statusGen int

	// RD-7: braille-спиннер идёт, пока есть in-flight загрузки; errorShown —
	// в строке ошибка (showError): спиннер стопается, чтобы не затирать её.
	spinnerOn  bool
	errorShown bool

	// In-flight загрузки (UI-8): id → надпись индикатора. Пока карта не
	// пуста, строка сообщений держит индикатор; строка подсказок от
	// загрузок не зависит (UI-9). Доступ только из главного потока TUI
	// (ввод и QueueUpdateDraw) — без мьютекса.
	loads   map[int]string
	loadSeq int

	mail     *mailScreen
	calendar *calendarScreen

	// Каталоги вложений, созданные при старте (T-1): сбой доступа к диску
	// виден при первом сохранении — до скачивания файла, а не после.
	attDownloadDir string
	attTempDir     string
	attDirErr      error

	// Панели, рамка активной из которых красится акцентом (UI-6, RD-2).
	focusPanels []focusPanel
}

// New собирает приложение. Пароль уже получен вызывающим (Keychain), клиент
// EWS готов (один на сессию — ARCHITECTURE.md §3). Логгер (nil — discard)
// нужен для §9: ошибки osascript логируются, но не прерывают TUI.
func New(cfg *config.Config, client *ews.Client, log *slog.Logger) *App {
	applyTheme(cfg.UI.Theme) // палитра — до создания виджетов (theme.go)
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
	a.status = tview.NewTextView()
	a.hint = tview.NewTextView().
		SetDynamicColors(true). // RD-6: красит клавиши; текст наш, не из EWS — escape-инвариант не задет
		SetText(formatHint(hintMail))
	// V-17: подсказка подстраивается под ширину — необязательные сегменты
	// (opt) снимаются на узком экране, иначе строка обрезалась бы по краю.
	a.hintSegs = hintMail
	a.hint.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		a.hint.SetText(formatHintFit(a.hintSegs, w))
		return x, y, w, h
	})

	a.mail = newMailScreen(a)
	// V-21: старт — список писем: подсказка стартового экрана — набор клавиш
	// списка (hintMail), акцентная рамка должна стоять там же.
	a.tapp.SetFocus(a.mail.messages)
	a.pages.AddPage(pageMail, a.mail.root, true, true)

	a.calendar = newCalendarScreen(a)
	a.pages.AddPage(pageCalendar, a.calendar.root, true, false)

	// UI-6: рамка активной панели — акцентный цвет. Обе панели экрана
	// (сетка и повестка) и все три панели почты; карточки и модалы —
	// полноэкранные, остаются с обычной серой рамкой.
	a.focusPanels = []focusPanel{
		{a.mail.folders, nil}, // V-16: у папок и просмотра заголовка нет
		{a.mail.messages, a.mail.messagesHeader},
		{a.mail.preview, nil},
		{a.calendar.grid, a.calendar.gridHeader},
		{a.calendar.agenda, a.calendar.agendaHeader},
	}
	a.tapp.SetBeforeDrawFunc(a.paintFocusFrames)

	details := tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(true) // RD-5: жирная тема/линия; сторонний текст — только через tview.Escape в showDetails
	details.SetBorder(true).SetTitle(" Письмо ")
	details.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.pages.SwitchToPage(pageMail)
			a.showHint()
			return nil
		case tcell.KeyRight:
			// V-27: перенос тела делает горизонтальный скролл → бессмысленным
			// (tview режет начало строк) — клавишу поглощаем, как и h/l.
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h', 'l':
				return nil // V-27: горизонтальный скролл при SetWrap(true)
			case 'g', 'G':
				a.showLinks(a.currentLinks, func() {
					a.pages.SwitchToPage(pageDetails)
					a.tapp.SetFocus(a.detail)
				})
				return nil
			case 'v', 'V':
				a.showAttachments(a.currentAttachments, func() {
					a.pages.SwitchToPage(pageDetails)
					a.tapp.SetFocus(a.detail)
				})
				return nil
			}
		}
		return event
	})
	// Колонка чтения (V-5, П.1): спейсеры по бокам a.detail; раскладку при
	// resize пересчитывает layoutDetails (до отрисовки, WHY у неё).
	a.detailPage = tview.NewFlex()
	a.detailsWide = true // стартовая раскладка широкая; layoutDetails поправит
	a.detailPage.AddItem(nil, 0, 1, false).
		AddItem(details, readWidth, 0, true).
		AddItem(nil, 0, 1, false)
	// WHY: дата в шапке-карточке паддингом под конкретную ширину — карточку
	// собираем заново в DrawFunc, когда ширина колонки изменилась (гард по
	// detailsW — без цикла перерисовки); SetText отрисовывается тем же кадром,
	// потому что TextView печатает текст после Box.Draw. Тело — из detailsTail:
	// resize не перегенерирует реестр ссылок.
	details.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		if inner := w - 2; inner > 2 && inner != a.detailsW {
			a.setDetailsText(inner)
		}
		return x + 1, y + 1, w - 2, h - 2
	})
	a.detail = details
	a.pages.AddPage(pageDetails, a.detailPage, true, false)

	event := tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(true)
	event.SetBorder(true).SetTitle(" Встреча ")
	event.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.pages.SwitchToPage(pageCalendar)
			a.showHint()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g', 'G':
				a.showLinks(a.currentLinks, func() {
					a.pages.SwitchToPage(pageEvent)
					a.tapp.SetFocus(a.event)
				})
				return nil
			}
		}
		return event
	})
	a.event = event
	a.pages.AddPage(pageEvent, event, true, false)

	// Верхний бар (V-11): одна строка — «mailcat · <папка>» на всю ширину.
	a.topLeft = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	a.updateTopBar()

	// WHY (UI-9): две строки статуса вместо одной — подсказки клавиш видны
	// ВСЕГДА: раньше ошибки и «Загрузка…» вытесняли их из единственной строки.
	// Не делать: вернуть совмещённую строку «сообщение поверх подсказки» —
	// при ошибке пользователь снова теряет клавиши экрана.
	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.topLeft, 1, 0, false). // верхний бар — одна строка (V-6, V-11)
		AddItem(a.pages, 0, 1, true).
		AddItem(a.status, 1, 0, false). // строка сообщений — над подсказками
		AddItem(a.hint, 1, 0, false)    // строка подсказок — самая нижняя

	a.tapp.SetRoot(root, true).EnableMouse(true)
	a.tapp.SetInputCapture(a.globalKeys)
	a.tapp.SetMouseCapture(a.foldersMouse)
	return a
}

// foldersMouse переводит клик по разделителю групп панели папок на следующую
// папку: курсор туда, фокус — панели (V-29).
// WHY: событие гасим, а не сдвигаем координату на строку ниже — List по
// Selected у папки вызвал бы selectFolder, и клик по линии открывал бы папку.
// Not to do: отдавать клик List с подменённой позицией.
func (a *App) foldersMouse(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
	if action != tview.MouseLeftClick {
		return event, action
	}
	if name, _ := a.pages.GetFrontPage(); name != pageMail {
		return event, action // модал поверх: клик не наш
	}
	x, y, w, h := a.mail.folders.GetInnerRect()
	mx, my := event.Position()
	if mx < x || mx >= x+w || my < y || my >= y+h {
		return event, action
	}
	off, _ := a.mail.folders.GetOffset()
	row := my - y + off
	if row < 0 || row >= a.mail.folders.GetItemCount() || a.mail.isFolderRow(row) {
		return event, action // папка/«Ещё» и пустота ниже — штатный обработчик
	}
	a.tapp.SetFocus(a.mail.folders)
	a.mail.folders.SetCurrentItem(a.mail.folderRowNear(row, 1))
	// SetCurrentItem зовёт changed до сдвига курсора (WHY в moveFolderCursor).
	a.mail.applyActiveFolderStyle()
	return nil, action
}

// Run запускает цикл событий; блокируется до выхода (q/Ctrl-C).
func (a *App) Run() error {
	a.prepareAttachmentDirs() // T-1: каталоги вложений — при старте, не при первом сохранении
	a.mail.loadFolders()
	if a.cfg.Notify.Enabled { // enabled=false — watcher не запускается вовсе (DoD этапа 6)
		a.startWatcher()
	}
	// V-11: часы верхнего бара убраны вместе с датой/временем — тикер не нужен.
	err := a.tapp.Run()
	a.cancel() // контекст сессии гасится вместе с TUI
	return err
}

// updateTopBar пересобирает верхний бар (V-6, V-11): «mailcat · <папка>» на
// почте и «mailcat · календарь» на календаре (V-41) — по фронт-странице.
// Имя папки из EWS: Sanitize + Escape (инвариант escape). Дата/время и точка
// связи убраны: статус связи живёт в футере сайдбара (V-4, V-12).
// Зовётся при старте, смене папки (selectFolder) и 1/2 (globalKeys).
// WHY: на календаре — подпись страницы, а не месяц.
// Why: месяц уже в шапке сетки; на смену месяца бар не перерисовывается.
// Not to do: возвращать MonthTitle — это дубль строки заголовка.
func (a *App) updateTopBar() {
	left := "[::b]mailcat[-::-]"
	name := a.mail.folderName
	if page, _ := a.pages.GetFrontPage(); page == pageCalendar {
		name = "календарь"
	}
	if name != "" {
		left += mutedText(" · ") + tview.Escape(security.Sanitize(name))
	}
	a.topLeft.SetText(left)
}

// prepareAttachmentDirs создаёт каталоги вложений при старте (T-1): ошибка
// доступа к ~/Downloads всплывает при первом же сохранении с ясной
// диагностикой, а не после скачивания вложения; сами пути запоминаются —
// saveAttachment ничего не создаёт в горутине.
func (a *App) prepareAttachmentDirs() {
	a.attDownloadDir, a.attDirErr = security.DownloadDir()
	if a.attDirErr != nil {
		return
	}
	a.attTempDir, a.attDirErr = security.TempDir()
}

// startWatcher поднимает фоновый опрос «Входящих» (этап 6). Ошибки сети —
// тихий backoff внутри watcher.Run; уведомления — только по правилам.
func (a *App) startWatcher() {
	w := &notify.Watcher{
		Rule:       a.cfg.Notify,
		OrgDomains: a.cfg.InternalDomains(),
		PageSize:   a.cfg.PageSize,
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
			a.closeModals(a.mail.messages)
			a.pages.SwitchToPage(pageMail)
			a.updateTopBar() // V-31: на почте — папка
			a.showHint()
			return nil
		case '2':
			// Сначала страница и подсказка, потом ensureLoaded: без кэша
			// он выведет «Загрузка…» в строку сообщений — подсказки
			// останутся на месте (UI-9).
			a.closeModals(a.calendar.grid)
			a.pages.SwitchToPage(pageCalendar)
			// V-35: 1/2 фокус не переносит — иначе стрелки уходят в скрытый
			// список писем. Фокус повестки календаря сохраняется.
			a.calendar.ensureFocus()
			a.updateTopBar() // V-41: на календаре — «календарь»
			a.showHint()
			a.calendar.ensureLoaded()
			return nil
		case 'q':
			a.tapp.Stop()
			return nil
		}
	}
	return event
}

// closeModals убирает модальные страницы (links/attachments/confirm) при
// переключении 1/2 (T-1, «Esc-источник модала»): SwitchToPage скрывает
// модал, но фокус остаётся у его скрытого списка — Esc затем уводил на
// страницу-источник модала, давно покинутую пользователем, а остальные
// клавиши шли в невидимый список. Если сверху не модал — фокус не трогаем.
func (a *App) closeModals(focus tview.Primitive) {
	front, _ := a.pages.GetFrontPage()
	if front != pageLinks && front != pageAttachments && front != pageConfirm {
		return
	}
	a.pages.RemovePage(pageLinks)
	a.pages.RemovePage(pageAttachments)
	a.pages.RemovePage(pageConfirm)
	a.tapp.SetFocus(focus)
}

// showDetails выводит полное письмо на экран деталей (страница поверх почты).
// Тело проходит через реестр ссылок ([N] host, ⚠, [IDN]) и санитайзер;
// клавиши g/v открывают модалы ссылок и вложений.
func (a *App) showDetails(m mail.Message) {
	tail, links := messageView(m)
	a.currentLinks = links
	a.currentAttachments = m.Attachments

	// Тело кэшируется: DrawFunc на resize пересобирает только карточку шапки.
	a.detailsMsg = m
	a.detailsTail = tail
	a.setDetailsText(readWidth - 2) // широкая колонка; DrawFunc поправит узкую
	a.detail.ScrollToBeginning()
	a.pages.SwitchToPage(pageDetails)
	a.tapp.SetFocus(a.detail)
	a.showHint()
}

// setDetailsText собирает текст страницы деталей: карточка шапки (V-5) под
// текущую ширину колонки + закэшированный хвост (тело, вложения). Зовётся из
// showDetails и из DrawFunc при resize — зачем так, см. WHY у DrawFunc.
func (a *App) setDetailsText(inner int) {
	a.detailsW = inner
	a.detail.SetText(FormatMessageCard(a.detailsMsg, inner) + "\n" + a.detailsTail)
}

// Параметры колонки чтения письма (V-5, П.1): readWidth — из format.go.
const (
	detailsWideMin   = 100 // терминал уже — колонка на всю ширину минус поля
	detailsNarrowPad = 2
)

// layoutDetails — ширина чтения письма (П.1, V-5): ≥ detailsWideMin — колонка
// readWidth по центру, уже — вся ширина минус узкие поля. Спейсеры 1:1
// центрируют колонку сами при любом resize внутри одного режима. Вызов из
// paintFocusFrames (before-draw): кадр рисуется сразу в новой раскладке.
// WHY: не в DrawFunc страницы — раскладку после начала отрисовки не
// перестроить до следующего кадра.
func (a *App) layoutDetails(w int) {
	wide := w >= detailsWideMin
	if wide == a.detailsWide {
		return
	}
	a.detailsWide = wide
	a.detailPage.Clear()
	if wide {
		a.detailPage.AddItem(nil, 0, 1, false).
			AddItem(a.detail, readWidth, 0, true).
			AddItem(nil, 0, 1, false)
	} else {
		a.detailPage.AddItem(nil, detailsNarrowPad, 0, false).
			AddItem(a.detail, 0, 1, true).
			AddItem(nil, detailsNarrowPad, 0, false)
	}
}

// showEventDetails выводит детали встречи: шапка в типографике письма (V-36),
// описание — тем же конвейером (реестр ссылок + formatBody/Sanitize+Escape),
// пустое — заглушкой. Ссылки «Места» и тела попадают в модал g.
func (a *App) showEventDetails(ev mail.Event, body mail.EventBody) {
	reg := mail.NewLinkRegistry()
	header := FormatEventHeader(ev, reg)
	var text string
	if body.BodyHTML != "" {
		text = reg.RenderBodyHTML(body.BodyHTML)
	} else {
		text = reg.RenderBody(body.BodyText)
	}
	if formatted := formatBody(text); formatted == "" {
		text = mutedText("(Встреча без описания)")
	} else {
		text = formatted
	}
	a.currentLinks = reg.List()
	a.currentAttachments = nil
	a.event.SetText(header + text)
	a.event.ScrollToBeginning()
	a.pages.SwitchToPage(pageEvent)
	a.tapp.SetFocus(a.event)
	a.showHint()
}

// setStatus выводит сообщение в строку сообщений (UI-9) — «Загрузка…»,
// «Скачивание …», «Открыто: …» (текст наш, не из EWS). Цвет возвращается
// к обычному: красным бывает только ошибка (showError). Смена поколения
// (statusGen) гасит висящий таймер автоочистки ошибки.
func (a *App) setStatus(text string) {
	a.statusGen++
	a.errorShown = false
	a.status.SetTextColor(tview.Styles.PrimaryTextColor)
	a.status.SetText(text)
	a.startSpinner() // есть загрузка в полёте — вернётся анимация
}

// showHint ставит подсказку текущего экрана (UI-7) в нижнюю строку (UI-9):
// у каждого экрана свой набор клавиш, и подсказки видны ВСЕГДА — сообщения,
// ошибки и загрузки живут строкой выше и эту строку не трогают.
func (a *App) showHint() {
	a.hintSegs = a.pageHintSegs()
	a.hint.SetText(formatHint(a.hintSegs))
}

// beginLoad показывает индикатор загрузки на время GetItem/FindItem (UI-8)
// и возвращает id для endLoad. Каждая ветка завершения горутины (успех,
// ошибка, устаревший ответ) обязана ровно один раз вызвать endLoad(id)
// внутри QueueUpdateDraw — иначе индикатор не погаснет.
func (a *App) beginLoad(label string) int {
	a.loadSeq++
	if a.loads == nil {
		a.loads = make(map[int]string)
	}
	a.loads[a.loadSeq] = label
	a.setStatus(label)
	return a.loadSeq
}

// endLoad снимает завершившуюся загрузку: полётов не осталось — строка
// сообщений гаснет (подсказки и так видны всегда), остался — надпись ещё
// идущей (загрузки могут перекрываться).
func (a *App) endLoad(id int) {
	delete(a.loads, id)
	if label, ok := a.loadLabel(); ok {
		a.setStatus(label)
		return
	}
	a.setStatus("")
}

// loadLabel — надпись любой незавершённой загрузки; одновременно их
// практически не бывает, поэтому «первый из карты» достаточен.
func (a *App) loadLabel() (string, bool) {
	for _, label := range a.loads {
		return label, true
	}
	return "", false
}

// RD-7: braille-спиннер вместо статичной надписи загрузки. Текст пишется
// напрямую (не setStatus): смена поколения statusGen здесь не нужна и
// погасила бы таймер автоочистки ошибки, если он висит.
const (
	spinnerFrames   = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	spinnerInterval = 100 * time.Millisecond
)

// startSpinner включает анимацию; пока она идёт или загрузок нет — no-op.
func (a *App) startSpinner() {
	if a.spinnerOn || len(a.loads) == 0 {
		return
	}
	a.spinnerOn = true
	a.spinnerStep(0)
}

// spinnerStep рисует очередной кадр и планирует следующий; тик останавливает
// анимацию, когда загрузки кончились или в строке показана ошибка.
func (a *App) spinnerStep(pos int) {
	if !a.spinnerOn || a.errorShown || len(a.loads) == 0 {
		a.spinnerOn = false
		return
	}
	frames := []rune(spinnerFrames)
	label, _ := a.loadLabel()
	a.status.SetText(string(frames[pos%len(frames)]) + " " + label)
	time.AfterFunc(spinnerInterval, func() {
		a.tapp.QueueUpdateDraw(func() { a.spinnerStep(pos + 1) })
	})
}

// formatHint собирает текст нижней строки (V-7, RD-6): клавиша — бейджем
// `[клавиша]` акцентом, подпись — второстепенным цветом, разделитель « · ».
// WHY: tview не рисует скруглённые бейджи в одной строке TextView — план
// разрешает вариант `[клавиша]` акцентом; литерал «[» экранируем, иначе
// dynamic colors примет бейдж за тег цвета (tview.Escape). Не делать:
// возвращать голые клавиши — пропадает визуальная граница клавиши/подписи.
// Цвета — hex ролей темы: роль — переменная applyTheme, у тегов tview имён
// ролей нет. Текст подсказок — наш, не из EWS: теги безопасны.
func formatHint(segs []hintSeg) string {
	key := fmt.Sprintf("[#%06x::b]", colorAccent.Hex())
	desc := fmt.Sprintf("[#%06x]", colorTextSecondary.Hex())
	var b strings.Builder
	for i, s := range segs {
		if i > 0 {
			b.WriteString(" " + desc + "· ")
		}
		b.WriteString(key + keycap(s.key))
		if s.desc != "" {
			b.WriteString(" " + desc + s.desc)
		}
	}
	return b.String()
}

// formatHintFit — подсказка под ширину max: необязательные сегменты (opt)
// снимаются с конца, пока текст не влезет; max <= 0 — ширина неизвестна,
// отдаём полный набор. Если влезть не удалось — возвращаем что есть (край
// обрежется, но обязательные клавиши останутся).
func formatHintFit(segs []hintSeg, max int) string {
	if max <= 0 {
		return formatHint(segs)
	}
	for {
		text := formatHint(segs)
		if tview.TaggedStringWidth(text) <= max {
			return text
		}
		drop := -1
		for i := len(segs) - 1; i >= 0; i-- {
			if segs[i].opt {
				drop = i
				break
			}
		}
		if drop < 0 {
			return text
		}
		next := make([]hintSeg, 0, len(segs)-1)
		next = append(next, segs[:drop]...)
		next = append(next, segs[drop+1:]...)
		segs = next
	}
}

// keycap — клавиша в бейдже `[клавиша]`. WHY: tview.Escape получит ВСЮ
// скобку целиком — одиночный «[» паттерн escape не ловит, и `[Enter]`/`[u]`
// съедаются парсером как цвет-теги (проверено скриншотом V-7).
func keycap(k string) string {
	return tview.Escape("[" + k + "]")
}

// pageHint — подсказка для верхней видимой страницы (модалы закрывают экран).
func (a *App) pageHint() string { return formatHint(a.pageHintSegs()) }

// pageHintSegs — набор подсказок видимой страницы (без форматирования: его
// под ширину подбирает formatHintFit в DrawFunc строки подсказок).
func (a *App) pageHintSegs() []hintSeg {
	switch name, _ := a.pages.GetFrontPage(); name {
	case pageCalendar:
		// V-35: у календаря два набора — по фокусу панели: на повестке Enter
		// открывает детали, Esc/← возвращают к сетке.
		if a.tapp.GetFocus() == a.calendar.agenda {
			return hintCalendarAgenda
		}
		return hintCalendarGrid
	case pageDetails:
		return hintDetails
	case pageEvent:
		return hintEvent
	case pageLinks:
		return hintLinks
	case pageAttachments:
		return hintAttachments
	case pageConfirm:
		return hintConfirm
	default:
		// V-15: фокус в панели чтения после Enter — набор клавиш панели,
		// а не списка (там Enter «читать», здесь Esc «к списку»).
		if a.mail.reading && a.tapp.GetFocus() == a.mail.preview {
			return hintReading
		}
		// V-24: фокус на папках — свой набор клавиш (Enter «открыть», → в письма).
		if a.tapp.GetFocus() == a.mail.folders {
			return hintFolders
		}
		return hintMail
	}
}

// showError показывает ошибку в строке сообщений красным (UI-7); не падает и
// не модалит: сетевые проблемы (в т.ч. VPN) пользователь видит и может
// повторить (u). Через errStatusTTL строка сама гаснет (подсказки — в своей
// строке ниже, их ошибка не трогает, UI-9); таймер живёт по поколению
// statusGen и не трогает более новый статус. Текст ошибок EWS — серверный:
// Sanitize обязателен (§3, ревью этапа 2); усечение — по рунам, чтобы не
// рвать кириллицу.
func (a *App) showError(err error) {
	msg := security.Sanitize(err.Error())
	if runes := []rune(msg); len(runes) > 200 {
		msg = string(runes[:200]) + "…"
	}
	a.statusGen++
	a.errorShown = true // спиннер не должен затирать ошибку своим кадром
	if a.mail != nil {
		a.mail.setConn(false) // красная точка в сайдбаре (V-4)
	}
	gen := a.statusGen
	a.status.SetTextColor(colorError)
	a.status.SetText("Ошибка: " + msg + "  ·  u — повторить")
	time.AfterFunc(errStatusTTL, func() {
		a.tapp.QueueUpdateDraw(func() { a.clearError(gen) })
	})
}

// clearError гасит сообщение после таймаута ошибки (UI-7): идёт загрузка —
// в строке сообщений возвращается её индикатор, иначе строка пуста.
// Поколение не совпало — статус уже сменился, ничего не трогаем.
func (a *App) clearError(gen int) {
	if a.statusGen != gen {
		return
	}
	if label, ok := a.loadLabel(); ok {
		a.setStatus(label)
		return
	}
	a.setStatus("")
}

// paintFocusFrames красит рамку панели с фокусом акцентным цветом, у
// остальных прячет её (UI-6, RD-2). Хук before-draw — единственная точка,
// которая видит все смены фокуса: Tab/←/→ (mail.go), клик мыши, возврат из
// модалов (app.go, links.go), старт экрана.
// WHY: не по событию SetFocus — вызовов много, один забытый и рамка врёт;
// и не GetFocus() внутри хука — draw() уже держит мьютекс Application, а
// GetFocus берёт его же (deadlock). HasFocus() у Box обновляется при
// SetFocus/Blur и блокировок не берёт.
// Неактивные — tview.Styles.BorderColor, а не константа: цвет «прочих» рамок
// живёт в теме. Возврат false — отрисовка продолжается.
func (a *App) paintFocusFrames(screen tcell.Screen) bool {
	// V-5: пересборка колонки чтения (П.1) — до отрисовки кадра. screen nil —
	// в юнит-тестах хук зовут без экрана (TestPaintFocusFramesWired).
	if screen != nil {
		if w, _ := screen.Size(); w > 0 {
			a.layoutDetails(w)
		}
	}
	for _, p := range a.focusPanels {
		active := p.body.HasFocus()
		if active {
			p.body.SetBorderColor(colorFocusBorder)
		} else {
			// WHY (RD-2): рамка неактивной панели не выключается через
			// SetBorder(false), а красится цветом поверхности — иначе при
			// каждой смене фокуса содержимое сдвигалось бы на 2 строки и 2
			// столбца, и раскладка «прыгала» бы на Tab.
			// Не делать: убирать рамку у неактивных — нестабильная сетка.
			p.body.SetBorderColor(colorPanelBg)
		}
		if p.header != nil {
			p.header.setActive(active)
		}
	}
	return false
}
