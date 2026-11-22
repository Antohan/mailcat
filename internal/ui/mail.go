package ui

// Экран почты: Flex[ папки 20% | список 40% | просмотр 40% ] (PLAN.md этап 2).
// Изменения состояния — только из главного потока tview (обработчики и
// QueueUpdateDraw), поэтому мьютекс не нужен (ARCHITECTURE.md §3).

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/ews"
	"mailcat/internal/mail"
	"mailcat/internal/security"
)

const pageSizeCap = 100 // защита от нулевого page_size в конфиге

// Пропорции панелей почты в процентах ширины (RD-2, PLAN-ITER3): папки 20 /
// список 42 / просмотр 38. tview.Flex распределяет их по весам, сумма = 100.
const (
	pctFolders  = 20
	pctMessages = 42
	pctPreview  = 38
)

// Маркеры сворачиваемой секции техпапок (UI-1). Не проходят FormatFolder —
// статический текст, не из EWS.
const (
	moreFoldersClosed = "Ещё ▸"
	moreFoldersOpen   = "Ещё ▾"
)

type mailScreen struct {
	app  *App
	root *tview.Flex

	folders  *tview.List
	messages *tview.Table
	preview  *tview.TextView

	// Заголовок списка писем (RD-2, V-2): цвет/капс переключает
	// paintFocusFrames. У панелей ПАПКИ и ПРОСМОТР заголовка нет (V-16).
	messagesHeader *panelHeader

	// Состояние экрана.
	folder     ews.FolderID // выбранная папка (пустая = distinguished inbox)
	folderName string
	items      []mail.MessageSummary
	// itemRows — строка таблицы для каждого элемента items (О-1): строки-
	// разделители дней сдвигают нумерацию, сырой row колбэков в items писать
	// нельзя. Заполняется только в appendRows.
	itemRows []int
	// selRow — строка таблицы с выделенным пунктом (строка 1, V-1): строка
	// темы и подушка ниже невыбираемы, их выделение красит paintItemTail вручную.
	// -1 = выделения нет.
	selRow      int
	offset      int  // Offset следующей страницы FindItem
	lastPage    bool // IncludesLastItemInRange последней страницы
	totalInView int  // TotalItemsInView последней страницы (счётчик «всего», V-2)
	loading     bool // запрос страниц в полёте (ревью этапа 2: защита от дублей)

	// RD-7: пересборка списка после перечитывания/смены папки. Старые строки
	// остаются на экране до прихода свежих (без мерцания); выделение
	// восстанавливается по ItemId, скролл — по сохранённой верхней строке.
	rebuildPending bool
	restoreID      string
	restoreRow     int
	restoreScroll  int

	// Папки панели (UI-1): основной список и техпапки секции «Ещё ▸».
	mainFolders []mail.Folder
	techFolders []mail.Folder
	techOpen    bool
	// rowByItem — папка на каждой строке списка: индекс пункта tview.List →
	// папка. Разделители групп и «Ещё ▸/▾» в карту не входят. По ней активная
	// папка красится и восстанавливается при движении курсора.
	// WHY (V-4): не срез с индексами 1:1 — разделители групп сдвигают индексы.
	rowByItem map[int]mail.Folder
	// moreIdx — индекс пункта «Ещё ▸/▾»; len(mainFolders) больше не равен ему:
	// разделители групп стоят в списке тоже. Ставится в renderFolders.
	moreIdx int
	// foldersW — внутренняя ширина панели папок, известная только при
	// отрисовке (V-26): от неё зависит длина строки со счётчиком. 0 — ещё
	// не рисовали, строки идут по прежней колонке folderNameCol.
	foldersW int
	// conn — футер сайдбара (V-12, по референсу): две строки — «● Connected»
	// и username под ним. Текст собирает connText, ставит setConn или DrawFunc
	// (ширина панели известна только при отрисовке — обрезка «…» по ней).
	conn *tview.TextView
	// connLabel/connColor — состояние связи для connText: пишет setConn.
	connLabel string
	connColor tcell.Color
	// WHY: поколение выбора папки — ответ, пришедший после переключения,
	// относится к прежней папке и не должен стирать уже загруженный список.
	// Не делать: сравнивать имя папки вместо поколения — при возврате A→B→A
	// старый ответ A снова станет «свежим».
	gen int

	// V-15/V-17: письмо прочитано в панели просмотра (Enter) — g/v работают;
	// шапка подробная (Тема/От/Кому/Дата) собирается один раз в showReading,
	// от ширины она не зависит. reading=false — в панели черновик строки.
	reading bool

	// V-17: полосы прокрутки списка и просмотра (tview своей полосы не имеет):
	// offset/total знают Table и TextView, геометрию — их свежий rect.
	msgBar     *scrollbar
	previewBar *scrollbar
	// Кэш числа строк превью с переносом: GetWrappedLineCount парсит весь
	// текст, поэтому пересчёт — только после SetText (dirty) или смены ширины.
	previewLines      int
	previewLinesW     int
	previewLinesDirty bool

	pageSize int
}

func newMailScreen(a *App) *mailScreen {
	s := &mailScreen{app: a, pageSize: a.cfg.PageSize, selRow: -1, previewLinesDirty: true}
	if s.pageSize < 1 || s.pageSize > pageSizeCap {
		s.pageSize = 50
	}

	s.folders = tview.NewList().
		ShowSecondaryText(false).
		// V-17: заливка — на всю строку панели, как в референсе (было — только
		// по ширине текста, «чип» с рваным правым краем).
		SetHighlightFullLine(true).
		SetMainTextColor(colorTextPrimary).
		SetSelectedBackgroundColor(colorSelectedBg).
		SetSelectedTextColor(colorSelectedFg)
	s.folders.SetBorder(true)
	// V-26: счётчик непрочитанных прижат к правому краю панели, а длина
	// строки зависит от её ширины — та известна только при отрисовке.
	// WHY: SetItemText в DrawFunc безопасен (не будит SetChangedFunc и не
	// двигает курсор); Clear потерял бы позицию, GetFocus сломал бы порядок
	// отрисовки tview. Возвращаем внутренний прямоугольник рамки.
	s.folders.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		s.setFoldersWidth(w - 2)
		return x + 1, y + 1, max(w-2, 0), max(h-2, 0)
	})

	// V-17: разделитель колонок — пробел, а не «│»: вертикальная линия резала
	// заливку выбранного пункта пополам (в референсе между текстом и временем
	// её нет).
	s.messages = tview.NewTable().
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.Background(colorSelectedBg).Foreground(colorSelectedFg))
	s.messages.SetBorder(true)
	// WHY (V-1): кап ширины колонки 0 обязателен. tview выбрасывает колонки,
	// не влезающие целиком, — без капа длинная тема выталкивает колонку
	// времени (наблюдалось на симуляции). Кап пересчитывается на каждом Draw,
	// до раскладки колонок, — переживает resize терминала; усечение с «…» —
	// truncateEllipsis там же (встроенное многоточие tview не работает для
	// кириллицы).
	s.messages.SetDrawFunc(s.clampMessageColumns)

	// WHY (RD-5): dynamic colors включены — тема жирная, линия и заглушки
	// muted. Весь текст из EWS попадает сюда через FormatMessageHeader
	// (Sanitize + tview.Escape внутри); SetText руками — только литеральные
	// строки интерфейса ниже. Не упрощать: «просто SetText» с данными EWS
	// без Escape даст тег-инъекцию в отрисованную панель.
	s.preview = tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(true)
	s.preview.SetBorder(true)
	s.preview.SetText(mutedText(previewEmptyHint))

	// Полосы прокрутки: рисуются в своей колонке после панели, поэтому geom
	// берёт её свежий rect (колонка — сразу за рамкой панели).
	s.msgBar = newScrollbar()
	s.msgBar.geom = func() (int, int, int) {
		// Колонка полосы — своя; строки — внутренняя область таблицы (без
		// рамки), поэтому трек совпадает с текстом.
		x, _, _, _ := s.msgBar.GetRect()
		_, y, _, h := s.messages.GetInnerRect()
		return x, y, h
	}
	s.msgBar.track = func() (int, int) {
		off, _ := s.messages.GetOffset()
		return off, s.messages.GetRowCount()
	}
	s.previewBar = newScrollbar()
	s.previewBar.geom = func() (int, int, int) {
		x, _, _, _ := s.previewBar.GetRect()
		_, y, _, h := s.preview.GetInnerRect()
		return x, y, h
	}
	s.previewBar.track = func() (int, int) {
		off, _ := s.preview.GetScrollOffset()
		return off, s.wrappedLines()
	}

	// WHY (RD-2): рамки у неактивных панелей нет — у папок и просмотра
	// заголовка тоже нет (V-16): назначение очевидно, фокус показывает рамка.
	// Не делать: возвращать заголовки в рамку (Box.SetTitle) — у неактивной
	// панели рамки нет, и заголовок был бы не виден.
	s.messagesHeader = newPanelHeader("Письма")
	// WHY (V-12): футер сайдбара — две строки: «● Connected» и username под
	// ним; длинный username обрезается «…» по ширине панели. Обрезка — в
	// DrawFunc, а не при setConn: ширина панели меняется при resize, а
	// SetWrap(false) перенос не сделает — вторая строка просто не влезла бы.
	s.connLabel, s.connColor = "подключение…", colorTextSecondary
	s.conn = tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	s.conn.SetDrawFunc(func(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
		s.conn.SetText(s.connText(w))
		return x, y, w, h
	})
	foldersCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.folders, 0, 1, true).
		AddItem(s.conn, 2, 0, false)
	// WHY (V-3, П.4б): разделители между колонками — по одной тонкой линии,
	// той же толщины и цвета, что линия заголовка ПИСЬМА. Рамки панелей при
	// этом остаются невидимыми у неактивных (RD-2) — двойных линий между
	// колонками нет.
	d1, d2 := newDivider(), newDivider()
	// Полоса — отдельная колонка справа от панели (1 клетка), чтобы она не
	// наезжала на текст и рамку; заголовок ПИСЬМА тоже ужимается на клетку.
	messagesCol := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(panelColumn(s.messagesHeader, s.messages), 0, 1, true).
		AddItem(s.msgBar, 1, 0, false)
	previewCol := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(s.preview, 0, 1, true).
		AddItem(s.previewBar, 1, 0, false)
	s.root = tview.NewFlex().
		AddItem(foldersCol, 0, pctFolders, false).
		AddItem(d1, 1, 0, false).
		AddItem(messagesCol, 0, pctMessages, true).
		AddItem(d2, 1, 0, false).
		AddItem(previewCol, 0, pctPreview, false)

	s.folders.SetChangedFunc(func(_ int, _, _ string, _ rune) {
		// V-20: курсор меняет только вид строк — открытую папку (folderName и
		// accent) задаёт selectFolder. Пересчитываются все строки: курсор мог
		// и въехать, и съехать.
		// WHY: запись folderName здесь переименовывала открытую папку вслед за
		// курсором — accent и заголовок списка переезжали на неоткрытую строку.
		// Не делать: выводить имя открытой папки из курсора — «Входящие»
		// (distinguished inbox с пустым ID) тогда не отличить от выбора.
		s.applyActiveFolderStyle()
	})
	s.folders.SetSelectedFunc(func(_ int, _ string, _ string, _ rune) {
		s.app.tapp.SetFocus(s.messages)
		s.app.showHint()
	})
	s.messages.SetSelectionChangedFunc(func(row, _ int) {
		if row != s.selRow {
			if s.selRow >= 0 {
				s.paintItemTail(s.selRow+1, false)
			}
			s.paintItemTail(row+1, true)
			s.selRow = row
		}
		s.showPreview(row)
		s.scrollItemIntoView(row)
		s.maybeLoadNextPage(row)
	})
	s.messages.SetSelectedFunc(func(row, _ int) {
		s.readMessage(row)
	})

	// Tab/BackTab — циклический фокус панелей; ←/→ — соседняя панель (PLAN
	// этап 2 DoD: «←/→ или Tab — между панелями»); u — обновить папку.
	tabHook := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			s.cycleFocus()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'u' || event.Rune() == 'U' {
				s.refreshFolder()
				return nil
			}
		}
		return event
	}
	// V-17: ←/→ — панели. Из папок → уводит в письма; левее панели нет —
	// стрелку не отдаём List (его ←/→ — горизонтальный скролл).
	s.folders.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRight:
			s.app.tapp.SetFocus(s.messages)
			s.app.showHint()
			return nil
		case tcell.KeyLeft:
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
			// V-29: разделители групп курсором не выбираются — двигаем сами
			// и перескакиваем их (правила tview.List — в moveFolderCursor).
			s.moveFolderCursor(event.Key())
			return nil
		}
		return tabHook(event)
	})
	s.messages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			s.app.tapp.SetFocus(s.folders)
			s.app.showHint()
			return nil
		case tcell.KeyRight:
			s.app.tapp.SetFocus(s.preview)
			s.app.showHint()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'g' || event.Rune() == 'G' {
				row, _ := s.messages.GetSelection()
				s.openLinks(row)
				return nil
			}
		}
		return tabHook(event)
	})
	s.preview.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc, tcell.KeyLeft:
			// V-15: Esc из панели чтения — назад к списку (стрелки листают
			// тело, поэтому выход на←/Esc, а не на стрелку).
			s.app.tapp.SetFocus(s.messages)
			s.app.showHint()
			return nil
		case tcell.KeyRight:
			// WHY (V-27): SetWrap(true) не отключает горизонтальный скролл
			// tview — → двигает columnOffset и режет начало строк. Переносить
			// уже нечего, клавишу поглощаем (как и h/l ниже).
			// Не делать: SetScrollable(false) — пропадёт вертикальная прокрутка
			// тела и полоса V-17.
			return nil
		case tcell.KeyRune:
			// V-27: h/l — тот же горизонтальный скролл TextView, см. → выше.
			if r := event.Rune(); r == 'h' || r == 'l' {
				return nil
			}
			if !s.reading {
				break
			}
			switch event.Rune() {
			case 'g', 'G':
				s.app.showLinks(s.app.currentLinks, s.backToReading)
				return nil
			case 'v', 'V':
				s.app.showAttachments(s.app.currentAttachments, s.backToReading)
				return nil
			}
		}
		return tabHook(event)
	})
	return s
}

// backToReading возвращает фокус в панель чтения после модала (V-15):
// подсказки экрана — снова набор панели просмотра.
func (s *mailScreen) backToReading() {
	s.app.pages.SwitchToPage(pageMail)
	s.app.tapp.SetFocus(s.preview)
	s.app.showHint()
}

// readMessage — Enter на списке (V-15): письмо догружается GetItem и
// рендерится в правой панели. Полноэкранная страница pageDetails больше не
// открывается по Enter (сама страница остаётся в pages, но не переключается).
func (s *mailScreen) readMessage(row int) {
	s.loadMessage(s.itemIndex(row), s.showReading)
}

// showReading выводит полное письмо в панели просмотра (V-15, V-17): подробная
// шапка (Тема/От/Кому/Дата) + тело + вложения; фокус уходит в панель — сразу
// работают g/v и прокрутка. Шапка от ширины не зависит (в отличие от
// карточки-аватара V-5), поэтому текст собирается здесь один раз.
func (s *mailScreen) showReading(m mail.Message) {
	tail, links := messageView(m)
	s.reading = true
	s.preview.SetText(FormatMessageHeader(m) + "\n" + tail)
	s.previewLinesDirty = true
	s.app.currentLinks = links
	s.app.currentAttachments = m.Attachments
	s.preview.ScrollToBeginning()
	s.app.tapp.SetFocus(s.preview)
	s.app.showHint()
}

// loadMessage догружает письмо GetItem (тело, вложения) и отдаёт его на
// UI-потоке в done. From — из строки списка: GetItem его не присылает.
func (s *mailScreen) loadMessage(idx int, done func(mail.Message)) {
	if idx < 0 || idx >= len(s.items) {
		return
	}
	ref := s.items[idx]
	id := s.app.beginLoad("Загрузка письма…")
	go func() {
		detail, err := s.app.client.GetMessage(s.app.ctx, ref.ItemID, ref.ChangeKey)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() {
				s.app.endLoad(id)
				s.app.showError(err)
			})
			return
		}
		full := mail.FullMessage(detail)
		full.From = ref.From // From не приходит в GetItem — берём из строки списка
		s.app.tapp.QueueUpdateDraw(func() {
			s.app.endLoad(id)
			s.setConn(true)
			done(full)
		})
	}()
}

// openLinks — g на списке: реестр ссылок выбранного письма. В сводке FindItem
// тела нет, поэтому догружаем GetItem; ссылок нет — сообщение в статус-строке
// вместо пустого модала.
func (s *mailScreen) openLinks(row int) {
	s.loadMessage(s.itemIndex(row), func(m mail.Message) {
		links := messageLinks(m)
		if len(links) == 0 {
			s.app.setStatus("Ссылок нет")
			return
		}
		s.app.showLinks(links, func() {
			s.app.pages.SwitchToPage(pageMail)
			s.app.tapp.SetFocus(s.messages)
		})
	})
}

// messageLinks собирает ссылки тела тем же реестром, что и экран деталей.
func messageLinks(m mail.Message) []mail.LinkInfo {
	reg := mail.NewLinkRegistry()
	if m.BodyHTML != "" {
		reg.RenderBodyHTML(m.BodyHTML)
	} else {
		reg.RenderBody(m.BodyText)
	}
	return reg.List()
}

// loadFolders заполняет панель папок (FindFolder от msgfolderroot): основной
// список + техпапки в секции «Ещё ▸» (UI-1) — и открывает «Входящие»
// (distinguished inbox — первая страница писем).
func (s *mailScreen) loadFolders() {
	id := s.app.beginLoad("Загрузка папок…")
	go func() {
		list, err := s.app.client.FindFolders(s.app.ctx)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() {
				s.app.endLoad(id)
				s.app.showError(err)
			})
			return
		}
		folders := mail.Folders(list)
		s.app.tapp.QueueUpdateDraw(func() {
			s.mainFolders, s.techFolders = SplitFolders(folders, s.app.cfg.UI.HiddenFolders)
			s.renderFolders()
			s.setConn(true)
			s.app.endLoad(id) // погасит индикатор; selectFolder сменит на «Загрузка писем…»
			// Стартуем с входящих (distinguished) — отдельным запросом.
			s.selectFolder(ews.FolderID{}, "Входящие")
		})
	}()
}

// renderFolders перестраивает панель целиком: блок стандартных папок, линия,
// блок пользовательских, линия, пункт «Ещё ▸/▾» и, когда секция развернута,
// техпапки (V-4, UI-1). Целиком — потому что сворачивание/разворачивание
// меняет хвост списка одним махом.
func (s *mailScreen) renderFolders() {
	prev := s.folders.GetCurrentItem()
	wasFolder, isFolder := s.rowByItem[prev]
	onMore := false
	if !isFolder && prev < s.folders.GetItemCount() {
		if text, _ := s.folders.GetItemText(prev); text == moreFoldersClosed || text == moreFoldersOpen {
			onMore = true
		}
	}
	s.folders.Clear()
	s.rowByItem = make(map[int]mail.Folder)
	add := func(f mail.Folder) {
		idx := s.folders.GetItemCount()
		s.folders.AddItem(FormatFolder(f, s.isActiveFolder(f), false, s.foldersW), "", 0, func() {
			s.selectFolder(ews.FolderID{ID: f.ID, ChangeKey: f.ChangeKey}, f.DisplayName)
		})
		s.rowByItem[idx] = f
	}
	// Разделитель — пункт списка: Enter перепрыгивает его, а не уводит фокус
	// (глобальный SetSelectedFunc перескочил бы в список писем).
	addSeparator := func() {
		idx := s.folders.GetItemCount()
		s.folders.AddItem(fmt.Sprintf("[%s]%s[-]", hexTag(colorTextSecondary), strings.Repeat("─", 13)), "", 0, func() {
			if idx+1 < s.folders.GetItemCount() {
				s.folders.SetCurrentItem(idx + 1)
			}
		})
	}
	isStandard := func(f mail.Folder) bool {
		for _, name := range standardFolders {
			if f.DisplayName == name {
				return true
			}
		}
		return false
	}
	seenUser := false
	for _, f := range s.mainFolders {
		if !isStandard(f) && !seenUser {
			seenUser = true
			if s.folders.GetItemCount() > 0 {
				addSeparator()
			}
		}
		add(f)
	}
	if len(s.techFolders) == 0 {
		s.applyActiveFolderStyle()
		s.restoreFolderCursor(wasFolder, onMore)
		return
	}
	if s.folders.GetItemCount() > 0 {
		addSeparator()
	}
	mark := moreFoldersClosed
	if s.techOpen {
		mark = moreFoldersOpen
	}
	s.moreIdx = s.folders.GetItemCount()
	s.folders.AddItem(mark, "", 0, s.toggleTech)
	if s.techOpen {
		for _, f := range s.techFolders {
			add(f)
		}
	}
	s.applyActiveFolderStyle()
	s.updateListTitle()
	s.restoreFolderCursor(wasFolder, onMore)
}

// restoreFolderCursor возвращает курсор на прежнюю папку после пересборки
// (V-19): Clear сбрасывает выделение на первую строку. onMore — курсор стоял
// на «Ещё ▸/▾». Папка ищется по ID, при пустом ID — по имени (distinguished
// inbox до первого переключения). Не нашли (старт, нет папок, разделитель) —
// первая строка.
func (s *mailScreen) restoreFolderCursor(prev mail.Folder, onMore bool) {
	if onMore && s.moreIdx < s.folders.GetItemCount() {
		s.folders.SetCurrentItem(s.moreIdx)
		return
	}
	for i, f := range s.rowByItem {
		if prev.ID != "" {
			if f.ID != prev.ID {
				continue
			}
		} else if prev.DisplayName == "" || f.DisplayName != prev.DisplayName {
			continue
		}
		s.folders.SetCurrentItem(i)
		return
	}
	s.folders.SetCurrentItem(0)
}

// isActiveFolder — строка соответствует открытой папке. ID пуст только у
// distinguished inbox на старте — тогда сравниваем по имени.
// WHY: имя, а не ID, ровно до первого переключения папки; сравнение по ID
// после загрузки — надёжно и не зависит от переименований.
func (s *mailScreen) isActiveFolder(f mail.Folder) bool {
	if s.folder.ID != "" {
		return f.ID == s.folder.ID
	}
	return f.DisplayName == s.folderName
}

// moveFolderCursor двигает курсор панели папок по правилам tview.List
// (list.go InputHandler: ↑/↓ — ±1, PgUp/PgDn — ±высота, края без обёртки),
// но с разделителя групп съезжает на ближайшую папку в сторону движения
// (V-29). Клавиша поглощается в capture — иначе List поставил бы курсор на
// линию.
func (s *mailScreen) moveFolderCursor(key tcell.Key) {
	n := s.folders.GetItemCount()
	if n == 0 {
		return
	}
	cur := s.folders.GetCurrentItem()
	to, dir := cur, 1
	switch key {
	case tcell.KeyUp:
		to, dir = cur-1, -1
	case tcell.KeyDown:
		to = cur + 1
	case tcell.KeyHome:
		to = 0
	case tcell.KeyEnd:
		to, dir = n-1, -1
	case tcell.KeyPgUp:
		_, _, _, h := s.folders.GetInnerRect()
		to, dir = cur-h, -1
	case tcell.KeyPgDn:
		_, _, _, h := s.folders.GetInnerRect()
		to = cur + h
	}
	if to < 0 {
		to = 0
	} else if to >= n {
		to = n - 1
	}
	s.folders.SetCurrentItem(s.folderRowNear(to, dir))
	// WHY: SetCurrentItem зовёт SetChangedFunc ДО смены currentItem — вид
	// строк пересчитался бы по старой позиции, и выделение уехало бы со
	// строки. Повтор после сдвига ставит выделение на новую.
	s.applyActiveFolderStyle()
}

// folderRowNear — ближайшая в направлении dir выбираемая строка (папка или
// «Ещё»); разделители групп пропускаются (V-29). В направлении упёрлись —
// курсор остаётся на месте.
func (s *mailScreen) folderRowNear(row, dir int) int {
	for ; row >= 0 && row < s.folders.GetItemCount(); row += dir {
		if s.isFolderRow(row) {
			return row
		}
	}
	return s.folders.GetCurrentItem()
}

// isFolderRow — строку можно выбрать: папка из rowByItem или пункт «Ещё».
// WHY: «Ещё» ищем по тексту, а не по moreIdx: при пустых техпапках
// renderFolders выходит раньше и moreIdx остаётся от прежней сборки.
func (s *mailScreen) isFolderRow(i int) bool {
	if _, ok := s.rowByItem[i]; ok {
		return true
	}
	text, _ := s.folders.GetItemText(i)
	return text == moreFoldersClosed || text == moreFoldersOpen
}

// applyActiveFolderStyle — вид строк: открытая папка accent + жирный, но
// БЕЗ тегов, когда курсор стоит на ней самой; у строки под курсором счётчик
// перекрашен в colorSelectedFg (П.2б), иначе серый tag не читается на фоне
// выделения. Пересчитываются ВСЕ строки: курсор мог и въехать, и съехать.
// WHY: выделенная строка красится инверсией (фон accent, инвариант RD-1),
// и accent-текст на accent-фоне становится невидимым; у выделенной строки
// остаётся штатный вид инверсии. SetItemText не будит SetChangedFunc —
// рекурсии нет.
func (s *mailScreen) applyActiveFolderStyle() {
	for i, f := range s.rowByItem {
		s.folders.SetItemText(i, s.folderRowText(i, f), "")
	}
}

// folderRowText — вид строки панели: активность и выделение по курсору,
// ширина — последняя известная (V-26; 0 до первой отрисовки). Одна формула
// на все пути записи строки: мусорные кэши разъехались бы на resize.
func (s *mailScreen) folderRowText(i int, f mail.Folder) string {
	cursorOnRow := i == s.folders.GetCurrentItem()
	return FormatFolder(f, s.isActiveFolder(f) && !cursorOnRow, cursorOnRow, s.foldersW)
}

// setFoldersWidth пересобирает строки панели под новую внутреннюю ширину
// (V-26): счётчик прижат к правому краю, длинное имя усекается «…».
// Только SetItemText — курсор и прокрутка остаются на месте (WHY в DrawFunc).
func (s *mailScreen) setFoldersWidth(width int) {
	if width < 1 || width == s.foldersW {
		return
	}
	s.foldersW = width
	s.applyActiveFolderStyle()
}

// toggleTech сворачивает/разворачивает секцию техпапок. Курсор возвращается
// на пункт «Ещё», чтобы повторное Enter срабатывало как сворачивание.
func (s *mailScreen) toggleTech() {
	s.techOpen = !s.techOpen
	s.renderFolders()
	// WHY: индекс «Ещё» берём из moreIdx, а не len(mainFolders) —
	// разделители групп сдвинули пункты списка (см. поле).
	s.folders.SetCurrentItem(s.moreIdx)
}

// setConn — футер сайдбара (V-12): точка зелёная, если последняя EWS-операция
// прошла, красная при ошибке. Текст на красной — «Нет связи»: «Connected» с
// красной точкой врал бы.
func (s *mailScreen) setConn(ok bool) {
	s.connColor, s.connLabel = colorConnected, "Connected"
	if !ok {
		s.connColor, s.connLabel = colorError, "Нет связи"
	}
	s.conn.SetText(s.connText(s.connWidth()))
}

// wrappedLines — число строк превью с учётом переноса (для бегунка).
// WHY (V-27): считаем сами через tview.WordWrap, а НЕ TextView.GetWrappedLineCount.
// Тот парсит с t.width (у нас не задан = 0 = «без переноса») и дописывает в общий
// индекс строк неразвёрнутые строки: дальше Draw режет их по краю панели посреди
// слова и теряет хвост (наблюдалось на живом письме). Индекс строк TextView
// трогает только его собственный Draw.
// Кэш — до смены текста (previewLinesDirty) или ширины панели.
func (s *mailScreen) wrappedLines() int {
	_, _, w, _ := s.preview.GetInnerRect()
	if w <= 0 {
		return 0 // раскладки ещё нет — переносить не по чему
	}
	if s.previewLinesDirty || w != s.previewLinesW {
		s.previewLines = len(tview.WordWrap(s.preview.GetText(false), w))
		s.previewLinesW = w
		s.previewLinesDirty = false
	}
	return s.previewLines
}

// connWidth — текущая ширина футера; 0 до первой раскладки.
func (s *mailScreen) connWidth() int {
	_, _, w, _ := s.conn.GetRect()
	return w
}

// connText — текст футера: «● <состояние>» и username под ним, обрезанный
// «…» по ширине панели (w ≤ 0 — ширина ещё неизвестна, не режем).
// Обрезаем ДО tview.Escape: усечение escaped-строки разрезало бы тег.
func (s *mailScreen) connText(w int) string {
	// V-17: выравнивание как у папок — точка в колонке левого поля (1 клетка,
	// как рамка списка), «Connected» и адрес — в колонке имён (рамка 1 + отступ
	// FormatFolder 2), иначе футер выбивался из общей колонки.
	const padText = 3
	user := security.Sanitize(s.app.cfg.Username)
	if w > padText {
		user = truncateEllipsis(user, w-padText)
	} else {
		user = ""
	}
	return fmt.Sprintf(" [%s]●[-][%s] %s[-]\n%s[%s]%s[-]",
		hexTag(s.connColor), hexTag(colorTextSecondary), s.connLabel,
		strings.Repeat(" ", padText), hexTag(colorTextSecondary), tview.Escape(user))
}

// selectFolder переключает текущую папку и перечитывает список. Старые строки
// остаются на экране до прихода свежих (RD-7, без мерцания): Clear делает
// только rebuildRows по свежему ответу — устаревший отсечёт поколение gen.
func (s *mailScreen) selectFolder(id ews.FolderID, name string) {
	// V-17: смена папки — выделение на первое письмо; прежнее место (RD-7)
	// возвращаем только при перечитывании той же папки («u»): иначе курсор
	// оставался на позиции из прошлой папки.
	switched := id.ID != s.folder.ID || name != s.folderName
	s.folder = id
	s.folderName = name
	s.app.updateTopBar() // «mailcat · <папка>» верхнего бара (V-6)
	s.offset = 0
	s.lastPage = false
	s.totalInView = 0 // счётчики прежней папки не показываем — ждём ответ (WHY в updateListTitle)
	if switched {
		s.restoreID, s.restoreRow, s.restoreScroll = "", 0, 0
	} else {
		s.keepSelection()
	}
	s.rebuildPending = true
	s.reading = false // V-15: смена папки — панель снова пустая, не старое письмо
	s.preview.SetText(mutedText(previewEmptyHint))
	s.previewLinesDirty = true
	s.loadMessages()
}

// refreshFolder перечитывает текущую папку (клавиша u).
func (s *mailScreen) refreshFolder() {
	// WHY: если FindFolders упал при старте (VPN не поднят), сайдбар пуст;
	// прежний «u» перегружал только письма через selectFolder с пустым
	// folderName — письма появлялись, заголовок списка и сайдбар оставались
	// сломанными до перезапуска (живой кейс 21.11.2026). Пустой сайдбар —
	// грузим всё: loadFolders сам завершается selectFolder («Входящие»).
	if len(s.mainFolders) == 0 && len(s.techFolders) == 0 {
		s.loadFolders()
		return
	}
	s.selectFolder(s.folder, s.folderName)
}

// loadMessages догружает страницу писем (FindItem, offset — курсор пагинации).
// Ответ устаревшего запроса (папка/offset сменились) отбрасывается.
func (s *mailScreen) loadMessages() {
	if s.loading {
		return
	}
	s.loading = true
	s.gen++
	gen := s.gen
	id := s.app.beginLoad("Загрузка писем…")
	offset := s.offset
	folder := s.folder
	go func() {
		var page ews.Page
		var err error
		if folder.ID == "" {
			page, err = s.app.client.FindDistinguishedMessages(s.app.ctx, "inbox", s.pageSize, offset)
		} else {
			page, err = s.app.client.FindMessages(s.app.ctx, folder, s.pageSize, offset)
		}
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() {
				s.loading = false
				s.app.endLoad(id)
				s.app.showError(err)
			})
			return
		}
		added := mail.MessageSummaries(page.Items)
		s.app.tapp.QueueUpdateDraw(func() {
			s.loading = false
			s.app.endLoad(id)
			s.setConn(true)
			if s.gen != gen {
				return // папка уже сменилась — ответ устарел
			}
			s.offset = page.NextOffset
			s.lastPage = page.IncludesLast
			s.totalInView = page.TotalInView
			s.updateListTitle()
			if s.rebuildPending {
				s.rebuildPending = false
				s.rebuildRows(added)
				return
			}
			s.items = append(s.items, added...)
			s.appendRows(added)
			if len(s.items) == 0 {
				// RD-5: пустая папка — muted-заглушка (colorTextSecondary).
				s.messages.SetCell(0, 0, tview.NewTableCell("(писем нет)").SetTextColor(colorTextSecondary))
			}
		})
	}()
}

// keepSelection запоминает выделенное письмо (ItemId) и верхнюю видимую
// строку списка, чтобы rebuildRows вернул пользователя на то же место (RD-7).
func (s *mailScreen) keepSelection() {
	s.restoreRow, _ = s.messages.GetSelection()
	s.restoreScroll, _ = s.messages.GetOffset()
	s.restoreRow = s.itemIndex(s.restoreRow) // restoreRow — индекс в items (О-1)
	if s.restoreRow >= 0 && s.restoreRow < len(s.items) {
		s.restoreID = s.items[s.restoreRow].ItemID
	} else {
		s.restoreID = ""
	}
}

// rebuildRows пересобирает список по свежей странице (RD-7): Clear один раз
// по готовым данным — мерцания нет; выделение — по ItemId (перечитывание той
// же папки) либо по прежней позиции (новая папка), скролл — как был.
func (s *mailScreen) rebuildRows(added []mail.MessageSummary) {
	s.items = added
	s.itemRows = nil
	s.messages.Clear()
	s.selRow = -1 // ячейки пересобраны: без сброса Select на ту же строку пропустил бы paintItemTail (WHY у paintItemTail)
	s.appendRows(added)
	if len(s.items) == 0 {
		s.messages.SetCell(0, 0, tview.NewTableCell("(писем нет)").SetTextColor(colorTextSecondary))
		s.messages.Select(0, 0)
		return
	}
	row := s.itemRow(rowAfterReload(s.items, s.restoreID, s.restoreRow))
	s.messages.Select(row, 0)
	if s.restoreScroll < row {
		s.messages.SetOffset(s.restoreScroll, 0)
	} // иначе Select уже показал выделенную строку — offset не откатываем
}

// itemRow / itemIndex — перевод «индекс в s.items ↔ строка таблицы» (О-1):
// строки-разделители дней сдвигают нумерацию строк, поэтому сырой row из
// колбэков tview в s.items[...] подставлять нельзя. -1 — строка не письмо
// (tview разделители не выделяет: NotSelectable); защита на будущее.
func (s *mailScreen) itemRow(i int) int { return s.itemRows[i] }

func (s *mailScreen) itemIndex(row int) int {
	for i, r := range s.itemRows {
		if r == row {
			return i
		}
	}
	return -1
}

// rowAfterReload решает, где встанет курсор после пересборки списка (RD-7):
// письмо ищется по ItemId, при промахе курсор остаётся на прежней позиции,
// стянутой в пределы нового списка.
func rowAfterReload(items []mail.MessageSummary, id string, prev int) int {
	if prev < 0 || prev >= len(items) {
		prev = 0
	}
	if id == "" {
		return prev
	}
	for i := range items {
		if items[i].ItemID == id {
			return i
		}
	}
	return prev
}

// clampMessageColumns ограничивает ширину колонки 0 (строки 1 и 2 пункта,
// V-1) внутренней шириной панели минус колонка времени и режет текст с
// «…» под эту ширину (truncateEllipsis). Полный текст берётся из s.items
// при каждом Draw, а не хранится в ячейке: иначе усечение не пережило бы
// расширение панели (resize). WHY: встроенное многоточие tview не рисуется
// для кириллицы — байты против клеток (WHY у truncateEllipsis). Возврат —
// стандартный внутренний прямоугольник Box с рамкой.
func (s *mailScreen) clampMessageColumns(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
	const (
		// V-17: время + поле справа — ширина колонки времени в клетках.
		timeW = 5 + listPadRight
		// Отступ заливки от рамки панели: блок не примыкает к краям.
		padX = 1
	)
	if inner := w - 2 - 2*padX; inner > timeW+1+listPadLeft {
		// Меньше maxW — на левое поле ячейки: ширина колонки = maxW+listPadLeft.
		maxW := inner - timeW - 1 - listPadLeft
		now := time.Now()
		for row := 0; row < s.messages.GetRowCount(); row++ {
			if c := s.messages.GetCell(row, 0); c != nil {
				c.SetMaxWidth(maxW + listPadLeft)
			}
		}
		for i, row := range s.itemRows {
			if i >= len(s.items) {
				break
			}
			// V-25: серия отправителя считается здесь заново — текст ячейки
			// хранить нельзя, при resize он пересобирается (WHY выше).
			repeat := i > 0 && sameSenderRun(s.items[i-1], s.items[i], now)
			parts := FormatMessageListItem(s.items[i], s.app.cfg.InternalDomains(), now, repeat)
			if c := s.messages.GetCell(row, 0); c != nil {
				c.SetText(strings.Repeat(" ", listPadLeft) +
					dotLine(truncateEllipsis(parts[0], maxW), !s.items[i].IsRead))
			}
			if c := s.messages.GetCell(row+1, 0); c != nil {
				c.SetText(strings.Repeat(" ", listPadLeft) + truncateEllipsis(parts[2], maxW))
			}
		}
	}
	return x + 1 + padX, y + 1, w - 2 - 2*padX, h - 2
}

// scrollOffsetForItem — offset списка, при котором пункт виден целиком:
// tview держит выбранную строку у нижнего края, и вторая строка пункта
// (тема) уезжала за границу. Возвращает текущий offset, если пункт уже
// виден или выделение ушло выше видимой области (там tview сам ставит
// строку к верхнему краю).
func scrollOffsetForItem(off, h, row int) int {
	if h <= 0 || row < off || row+1 < off+h {
		return off
	}
	return row + 2 - h
}

// scrollItemIntoView двигает offset так, чтобы тема выделенного пункта не
// обрезалась нижней границей списка (вызывается из SelectionChanged).
func (s *mailScreen) scrollItemIntoView(row int) {
	off, _ := s.messages.GetOffset()
	_, _, _, h := s.messages.GetInnerRect()
	if next := scrollOffsetForItem(off, h, row); next != off {
		s.messages.SetOffset(next, 0)
	}
}

// appendRows дописывает строки в таблицу (RD-4, V-1): пункт письма — две
// строки таблицы. Строка 1 (точка + [ВНЕШН] + отправитель) — невычитанные
// жирный accent (colorUnread), прочитанные — textPrimary; время — правая
// колонка colorRead (приглушено в обеих группах). [ВНЕШН] отдельного цвета
// не получает (SECURITY §4). Строка 2 (тема) — colorTextSecondary и
// невыбираема: курсор прыгает пункт→пункт, а её выделение красит paintItemTail
// вручную (tview красит только выбираемые строки). О-1: перед первой строкой
// каждой группы дня — разделитель «Сегодня / Вчера / Ранее» (не выделяемая
// строка colorTextSecondary).
func (s *mailScreen) appendRows(added []mail.MessageSummary) {
	base := len(s.items) - len(added)
	now := time.Now()
	prevBucket := -1
	// prev — индекс предыдущего пункта в s.items: серия отправителя (V-25)
	// продолжается и через границу страницы (base-1).
	prev := base - 1
	if base > 0 {
		prevBucket = dayBucket(now, s.items[base-1].Received)
	}
	for _, m := range added {
		if b := dayBucket(now, m.Received); b != prevBucket {
			prevBucket = b
			s.messages.SetCell(s.messages.GetRowCount(), 0,
				tview.NewTableCell(dayBuckets[b]).
					SetTextColor(colorTextSecondary).
					SetSelectable(false)) // без Expansion: расширил бы колонку 0
		}
		repeat := prev >= 0 && sameSenderRun(s.items[prev], m, now)
		parts := FormatMessageListItem(m, s.app.cfg.InternalDomains(), now, repeat)
		row := s.messages.GetRowCount()
		s.itemRows = append(s.itemRows, row)
		// dotLine: точка своим цветом тегом (muted у прочитанных, П.1);
		// базовый цвет ячейки — отправитель (WHY у dotLine).
		line1 := tview.NewTableCell(strings.Repeat(" ", listPadLeft) +
			dotLine(parts[0], !m.IsRead)).SetExpansion(1)
		if !m.IsRead {
			line1.SetTextColor(colorUnread).SetAttributes(tcell.AttrBold)
		} else {
			line1.SetTextColor(colorTextPrimary)
		}
		s.messages.SetCell(row, 0, line1)
		s.messages.SetCell(row, 1, tview.NewTableCell(parts[1]+strings.Repeat(" ", listPadRight)).
			SetTextColor(colorRead).SetAlign(tview.AlignRight))
		// V-17: тема невыбираема (курсор идёт пункт→пункт), её выделение
		// красит paintItemTail. Пустая ячейка времени во второй строке держит
		// правый край заливки ровным: блок выбранного письма — два столбца на
		// обеих строках, дата не «ступенькой».
		s.messages.SetCell(row+1, 0, tview.NewTableCell(strings.Repeat(" ", listPadLeft)+parts[2]).
			SetExpansion(1).SetTextColor(colorTextSecondary).SetSelectable(false))
		s.messages.SetCell(row+1, 1, tview.NewTableCell("").SetSelectable(false))
		prev++
	}
}

// paintItemTail красит невыбираемую строку темы (row) — тему и пустую
// ячейку времени, оба столбца: tview сам её не красит. После пересборки
// списка ячеек уже нет — невыставленные GetCell не хранит, краска просто
// никуда не попадёт.
func (s *mailScreen) paintItemTail(row int, selected bool) {
	for col := 0; col <= 1; col++ {
		c := s.messages.GetCell(row, col)
		if selected {
			c.SetBackgroundColor(colorSelectedBg)
		} else {
			// WHY (V-13): SetBackgroundColor снимает прозрачность ячейки,
			// поэтому ColorDefault оставлял на каждой посещённой строке
			// тему полосу терминального фона. SetTransparency(true)
			// возвращает поверхность панели — фон чистый.
			// Не делать: ColorDefault/любой цвет вместо прозрачности.
			c.SetTransparency(true)
		}
	}
	c := s.messages.GetCell(row, 0)
	if selected {
		c.SetTextColor(colorSelectedFg)
	} else {
		c.SetTextColor(colorTextSecondary)
	}
}

// maybeLoadNextPage подгружает следующую страницу при достижении конца списка;
// повторный запуск при загрузке в полёте игнорируется.
func (s *mailScreen) maybeLoadNextPage(row int) {
	idx := s.itemIndex(row)
	if s.loading || s.lastPage || idx < 0 || idx < len(s.items)-2 {
		return
	}
	if _, more := ews.NextOffset(ews.Page{NextOffset: s.offset, IncludesLast: s.lastPage}); more {
		s.loadMessages()
	}
}

// ewsAttachment конвертирует доменное вложение в метаданные для GetAttachment.
func ewsAttachment(a mail.Attachment) ews.AttachmentMeta {
	return ews.AttachmentMeta{
		AttachmentID:  a.AttachmentID,
		Name:          a.Name,
		ContentType:   a.ContentType,
		ContentLength: a.ContentLength,
	}
}

// Пустые состояния (RD-5): литеральные строки интерфейса (не из EWS),
// красит mutedText — Sanitize/Escape не нужны.
const (
	previewEmptyHint  = "Выберите письмо — Enter"
	noBodyPlaceholder = "(Письмо без текста)"
)

// showPreview показывает черновик письма в панели просмотра (без загрузки).
func (s *mailScreen) showPreview(row int) {
	// WHY (V-15): курсор ушёл с прочитанного письма — панель снова черновик,
	// reading=false обязателен: иначе DrawFunc при resize перерисует в неё
	// старое прочитанное письмо поверх черновика.
	s.reading = false
	idx := s.itemIndex(row)
	if idx < 0 || idx >= len(s.items) {
		return
	}
	m := s.items[idx]
	// RD-5: превью — тот же блок заголовков, что в деталях; в FindItem-сводке
	// Кому/вложений нет — FormatMessageHeader на пустом Message их опускает.
	text := FormatMessageHeader(mail.Message{MessageSummary: m}) + "\n" +
		mutedText("Enter — прочитать письмо…")
	s.preview.SetText(text)
	s.previewLinesDirty = true
	s.preview.ScrollToBeginning()
}

// updateInboxUnread обновляет счётчик непрочитанных «Входящих» из watcher'а
// (этап 6: QueueUpdateDraw). n<0 — счётчик неизвестен, пункт не трогаем.
// Источник истины — mainFolders, а не текст строки: текст теперь содержит
// иконку и цветовые теги (RD-3), по нему матчиться нельзя.
func (s *mailScreen) updateInboxUnread(n int) {
	if n < 0 {
		return
	}
	for i := range s.mainFolders {
		if s.mainFolders[i].DisplayName == "Входящие" {
			s.mainFolders[i].UnreadCount = n
			s.refreshFolderRow(s.mainFolders[i])
			s.updateListTitle()
			return
		}
	}
}

// refreshFolderRow перекрашивает одну строку панели без пересборки списка:
// тик watcher'а не должен сбрасывать курсор и прокрутку пользователя.
// WHY: SetItemText не будит SetChangedFunc, а renderFolders делает Clear и
// теряет позицию; вид строки — folderRowText, как в applyActiveFolderStyle.
func (s *mailScreen) refreshFolderRow(f mail.Folder) {
	for i, row := range s.rowByItem {
		if f.ID != "" && row.ID != f.ID {
			continue
		}
		if f.ID == "" && row.DisplayName != f.DisplayName {
			continue
		}
		// WHY: вид строки — из f (в нём новый UnreadCount), а rowByItem хранит
		// копию папки на момент renderFolders и счётчик бы отстал.
		s.folders.SetItemText(i, s.folderRowText(i, f), "")
		return
	}
}

// updateListTitle — заголовок панели писем (V-2): слева «Папка · N непрочитанных»,
// справа «всего T · стр. p/P». Непрочитанные — из кэша FindFolder; T —
// TotalInView, p = offset/pageSize (IndexedPagingOffset выдаёт NextOffset
// кратным pageSize, finditem_test). Правая часть скрывается, пока в этой папке
// не загружена ни одна страница (после selectFolder счётчики прежней папки не
// показываем — «всего 0» во время загрузки выглядело бы как пустая папка).
func (s *mailScreen) updateListTitle() {
	left := tview.Escape(security.Sanitize(s.folderName))
	if u, ok := s.currentFolderUnread(); ok && u > 0 {
		left += " · " + pluralUnread(u)
	}
	right := ""
	if s.offset > 0 || s.lastPage {
		// Оба — ceil: последняя страница неполная, offset=1234 при
		// pageSize=50 — это страница 25, а не 24.
		pages := (s.totalInView + s.pageSize - 1) / s.pageSize
		page := (s.offset + s.pageSize - 1) / s.pageSize
		if pages < 1 {
			pages = 1
		}
		if page < 1 {
			page = 1
		}
		if page > pages {
			page = pages
		}
		right = fmt.Sprintf("всего %d · стр. %d/%d", s.totalInView, page, pages)
	}
	s.messagesHeader.SetListTitle(left, right)
}

// currentFolderUnread — непрочитанные открытой папки из кэша списка папок.
// false — папки в FindFolder-ответе нет, счётчик неизвестен (пустое имя тоже).
func (s *mailScreen) currentFolderUnread() (int, bool) {
	if s.folderName == "" {
		return 0, false
	}
	for _, fs := range [][]mail.Folder{s.mainFolders, s.techFolders} {
		for _, f := range fs {
			if s.isActiveFolder(f) {
				return f.UnreadCount, true
			}
		}
	}
	return 0, false
}

// cycleFocus перебирает панели почты по кругу (Tab).
func (s *mailScreen) cycleFocus() {
	focus := []tview.Primitive{s.folders, s.messages, s.preview}
	next := tview.Primitive(s.messages)
	for i, p := range focus {
		if s.app.tapp.GetFocus() == p {
			next = focus[(i+1)%len(focus)]
			break
		}
	}
	s.app.tapp.SetFocus(next)
	// V-23: набор подсказок зависит от фокуса (V-17) — после SetFocus его
	// пересобирает только showHint; DrawFunc кэш a.hintSegs сам не обновляет.
	s.app.showHint()
}
