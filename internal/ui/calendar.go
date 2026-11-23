package ui

// Экран календаря (PLAN.md этап 4): Flex[ месяц-сетка 55% | повестка 45% ].
// Повторяющиеся встречи уже раскрыты сервером (CalendarView); переключение
// Почта↔Календарь переиспользует один Client (DoD). Загруженный месяц
// кэшируется до смены месяца.

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/mail"
)

type calendarScreen struct {
	app    *App
	root   *tview.Flex
	grid   *monthGrid
	agenda *tview.List
	// V-34: полоса прокрутки повестки — колонка за рамкой панели (как на почте).
	agendaBar *scrollbar

	// Строки-заголовки панелей (RD-2): месяц и повестка дня.
	gridHeader   *panelHeader
	agendaHeader *panelHeader

	year      int
	month     time.Month
	events    []mail.Event // кэш текущего месяца
	loadedKey string       // "2026-11" — загружен ли месяц

	dayEvents []mail.Event    // события выбранного дня (для повестки)
	dayLinks  []mail.LinkInfo // ссылки повестки дня (модал g)

	// V-36: описания встреч по ItemId — GetItem один раз на встречу.
	eventBodies  map[string]mail.EventBody
	eventLoading map[string]bool
}

func newCalendarScreen(a *App) *calendarScreen {
	s := &calendarScreen{app: a}
	now := time.Now()
	s.year, s.month = now.Year(), now.Month()

	s.grid = newMonthGrid()
	s.grid.SetBorder(true)
	// V-34: повестка — список в стиле писем: заливка на всю строку, тема
	// primary, выбранная строка — акцент (цвета строк — из FormatAgendaLine).
	s.agenda = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetMainTextColor(colorTextPrimary).
		SetSelectedBackgroundColor(colorSelectedBg).
		SetSelectedTextColor(colorSelectedFg)
	s.agenda.SetBorder(true)

	// RD-2: заголовки — отдельные строки над панелями (см. panel.go).
	s.gridHeader = newPanelHeader("Календарь")
	s.agendaHeader = newPanelHeader("Повестка дня")
	// V-30: баннера месяца нет — месяц показывает шапка панели (RD-2), сетка
	// начинается сразу под ней.
	// WHY (V-31): разделитель между колонками — как на почте (V-3): правый
	// счётчик шапки сетки иначе упирается в заголовок «Повестка» соседней
	// панели. Не делать: отступ пустой колонкой — линия есть у почты.
	// V-34: полоса прокрутки повестки — своя колонка справа, трек — строки
	// списка (offset/total знает List, геометрию — свежий rect панели).
	s.agendaBar = newScrollbar()
	s.agendaBar.geom = func() (int, int, int) {
		x, _, _, _ := s.agendaBar.GetRect()
		_, y, _, h := s.agenda.GetInnerRect()
		return x, y, h
	}
	s.agendaBar.track = func() (int, int) {
		off, _ := s.agenda.GetOffset()
		return off, s.agenda.GetItemCount()
	}
	agendaCol := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(panelColumn(s.agendaHeader, s.agenda), 0, 1, false).
		AddItem(s.agendaBar, 1, 0, false)

	s.root = tview.NewFlex().
		AddItem(panelColumn(s.gridHeader, s.grid), 0, 55, true).
		AddItem(newDivider(), 1, 0, false).
		AddItem(agendaCol, 0, 45, false)

	s.grid.SetSelectFunc(func(row, col int) {
		s.onDaySelected(row, col)
	})
	s.agenda.SetSelectedFunc(func(idx int, _ string, _ string, _ rune) {
		s.openEventDetails(idx)
	})
	// V-34: при смене курсора строки пересобираются (время/место выбранной —
	// onAccent), как applyActiveFolderStyle у папок. Индекс — из аргумента:
	// changed приходит ДО записи currentItem в List.
	s.agenda.SetChangedFunc(func(idx int, _ string, _ string, _ rune) {
		s.restyleAgenda(idx)
	})

	// Смена месяца: Ctrl-←/→ или n/p (PLAN этап 4); g — модал ссылок
	// повестки дня (UI-4); 1/2/q — глобально (app).
	monthHook := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			if event.Modifiers()&tcell.ModCtrl != 0 {
				s.shiftMonth(-1)
				return nil
			}
		case tcell.KeyRight:
			if event.Modifiers()&tcell.ModCtrl != 0 {
				s.shiftMonth(1)
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'n', 'N':
				s.shiftMonth(1)
				return nil
			case 'p', 'P':
				s.shiftMonth(-1)
				return nil
			case 'g', 'G':
				// WHY: фокус запоминаем до модала: Esc из модала ссылок
				// должен вернуть фокус той панели, с которой нажали g.
				prev := s.app.tapp.GetFocus()
				s.app.showLinks(s.dayLinks, func() {
					s.app.pages.SwitchToPage(pageCalendar)
					s.app.tapp.SetFocus(prev)
				})
				return nil
			}
		}
		return event
	}
	// V-32: стрелки на сетке двигают выбор на день/неделю с зажимом в месяце;
	// на повестке те же клавиши остаются списку (hook туда не вешаем).
	// V-35: Tab/Shift-Tab — сетка↔повестка, Enter — фокус в повестку.
	gridHook := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			s.cycleFocus()
			return nil
		case tcell.KeyEnter:
			s.app.tapp.SetFocus(s.agenda)
			s.app.showHint()
			return nil
		}
		if event.Modifiers() == tcell.ModNone {
			switch event.Key() {
			case tcell.KeyLeft:
				s.grid.Move(-1)
				return nil
			case tcell.KeyRight:
				s.grid.Move(1)
				return nil
			case tcell.KeyUp:
				s.grid.Move(-7)
				return nil
			case tcell.KeyDown:
				s.grid.Move(7)
				return nil
			}
		}
		return monthHook(event)
	}
	s.grid.SetInputCapture(gridHook)
	// V-35: Esc и ← (без Ctrl) с повестки возвращают фокус сетке; Enter на
	// повестке — детали встречи (SetSelectedFunc у List).
	s.agenda.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			s.cycleFocus()
			return nil
		case tcell.KeyEsc:
			s.focusGrid()
			return nil
		case tcell.KeyLeft:
			if event.Modifiers()&tcell.ModCtrl == 0 {
				s.focusGrid()
				return nil
			}
		}
		return monthHook(event)
	})
	return s
}

// cycleFocus перебирает панели календаря (Tab). Панелей две, поэтому
// Shift-Tab (как и Tab) ведёт в соседнюю: сетка ↔ повестка.
func (s *calendarScreen) cycleFocus() {
	var next tview.Primitive = s.grid
	if s.app.tapp.GetFocus() == s.grid {
		next = s.agenda
	}
	s.app.tapp.SetFocus(next)
	// V-23: набор подсказок зависит от фокуса — его пересобирает showHint.
	s.app.showHint()
}

// focusGrid возвращает фокус сетке (Esc/← на повестке).
func (s *calendarScreen) focusGrid() {
	s.app.tapp.SetFocus(s.grid)
	s.app.showHint()
}

// ensureFocus ставит фокус на сетку, если он остался в скрытой панели почты:
// переключение 1/2 фокус не переносит, иначе стрелки уходили бы в список
// писем. Фокус, уже стоящий в календаре (повестка), не трогаем.
func (s *calendarScreen) ensureFocus() {
	if p := s.app.tapp.GetFocus(); p == s.grid || p == s.agenda {
		return
	}
	s.app.tapp.SetFocus(s.grid)
}

// ensureLoaded подгружает месяц при входе на экран (кэш — до смены месяца).
func (s *calendarScreen) ensureLoaded() {
	if s.loadedKey == s.monthKey() {
		return
	}
	s.loadMonth()
}

func (s *calendarScreen) monthKey() string {
	return fmt.Sprintf("%04d-%02d", s.year, int(s.month))
}

func (s *calendarScreen) shiftMonth(delta int) {
	t := time.Date(s.year, s.month+time.Month(delta), 1, 0, 0, 0, 0, time.Local)
	s.year, s.month = t.Year(), t.Month()
	// V-41: верхний бар календаря не зависит от месяца — не пересобираем.
	s.loadMonth()
}

// monthStale сообщает, устарел ли ответ запроса месяца keyAtRequest
// относительно текущего месяца keyNow (ревью этапа 4: приземление ответа
// после смены месяца не должно портить кэш и события).
func monthStale(keyAtRequest, keyNow string) bool { return keyAtRequest != keyNow }

// loadMonth запрашивает встречи за месяц (CalendarView) и перерисовывает экран.
// Ответ месяца, сменившегося за время запроса, отбрасывается (monthStale).
func (s *calendarScreen) loadMonth() {
	key := s.monthKey()
	start, end := MonthBounds(s.year, s.month)
	id := s.app.beginLoad(fmt.Sprintf("Загрузка календаря: %s…", MonthTitle(s.year, s.month)))
	go func() {
		raw, err := s.app.client.FindEvents(s.app.ctx, start, end)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() {
				s.app.endLoad(id)
				s.app.showError(err)
			})
			return
		}
		events := mail.Events(raw)
		s.app.tapp.QueueUpdateDraw(func() {
			s.app.endLoad(id)
			if monthStale(key, s.monthKey()) {
				return // месяц уже сменился — ответ неактуален
			}
			s.events = events
			s.loadedKey = key
			s.render()
		})
	}()
}

// render перерисовывает сетку и выбирает текущий день (или 1-е число).
// V-32: tview.Table заменён monthGrid — ячейки заполняют панель по высоте.
func (s *calendarScreen) render() {
	s.gridHeader.SetListTitle(MonthTitle(s.year, s.month), meetingsCount(len(s.events)))
	grid := MonthGrid(s.year, s.month, s.events)
	now := time.Now()
	inMonth := now.Year() == s.year && now.Month() == s.month
	today := 0
	if inMonth {
		today = now.Day()
	}
	s.grid.SetCells(grid, today)
	// Выбор: сегодняшний день месяца, иначе 1-е — onSelect сам обновит повестку.
	pick := 1
	if inMonth {
		pick = now.Day()
	}
	s.grid.SelectDay(pick)
}

// onDaySelected обновляет повестку при движении по сетке (UI-4): строки
// собирает FormatAgendaLine (маркеры статуса + реестр ссылок из «Места»),
// пустой день — один muted-пункт без действия по Enter.
func (s *calendarScreen) onDaySelected(row, col int) {
	if row < 0 || row >= len(s.grid.cells) || col < 0 || col >= 7 {
		return
	}
	if s.grid.cells[row][col].Day == 0 {
		return
	}
	day := time.Date(s.year, s.month, s.grid.cells[row][col].Day, 0, 0, 0, 0, time.Local)
	s.dayEvents = DayEvents(s.events, day)
	s.agenda.Clear()
	if len(s.dayEvents) == 0 {
		s.dayLinks = nil
		s.agenda.AddItem(mutedText("Встреч нет"), "", 0, nil)
	} else {
		for range s.dayEvents {
			s.agenda.AddItem("", "", 0, nil)
		}
		s.restyleAgenda(0) // строки заполняет общая формула (курсор — первый)
	}
	s.agendaHeader.SetListTitle(AgendaTitle(day), meetingsCount(len(s.dayEvents)))
}

// restyleAgenda пересобирает строки повестки под курсор sel (V-34): только
// SetItemText — курсор и прокрутка остаются на месте (как applyActiveFolderStyle,
// V-26). Реестр ссылок новый, но порядок тот же — маркеры [N] не меняются.
func (s *calendarScreen) restyleAgenda(sel int) {
	// WHY: AddItem/InsertItem тоже дергает changed — при пересборке списка
	// элементов ещё нет, SetItemText по ним паникует. Не убирать зажим по
	// GetItemCount: он дешёвый и страхует весь цикл сборки (финал — вызов
	// после AddItem в onDaySelected).
	n := len(s.dayEvents)
	if c := s.agenda.GetItemCount(); n > c {
		n = c
	}
	if sel < 0 || sel >= n {
		return
	}
	reg := mail.NewLinkRegistry()
	for i, ev := range s.dayEvents[:n] {
		s.agenda.SetItemText(i, FormatAgendaLine(ev, reg, i == sel), "")
	}
	s.dayLinks = reg.List()
}

// openEventDetails показывает детали выбранной встречи: описание догружается
// GetItem (V-36) один раз на ItemId — повторный Enter берёт кэш; пока идёт
// запрос, второй Enter ничего не запускает.
func (s *calendarScreen) openEventDetails(idx int) {
	if idx < 0 || idx >= len(s.dayEvents) {
		return
	}
	ev := s.dayEvents[idx]
	if body, ok := s.eventBodies[ev.ItemID]; ok {
		s.app.showEventDetails(ev, body)
		return
	}
	if s.eventLoading[ev.ItemID] {
		return
	}
	if s.eventLoading == nil {
		s.eventLoading = make(map[string]bool)
	}
	s.eventLoading[ev.ItemID] = true
	id := s.app.beginLoad("Загрузка встречи…")
	go func() {
		raw, err := s.app.client.GetEventBody(s.app.ctx, ev.ItemID, ev.ChangeKey)
		s.app.tapp.QueueUpdateDraw(func() {
			s.app.endLoad(id)
			delete(s.eventLoading, ev.ItemID)
			if err != nil {
				s.app.showError(err)
				return
			}
			if s.eventBodies == nil {
				s.eventBodies = make(map[string]mail.EventBody)
			}
			body := mail.FullEventBody(raw)
			s.eventBodies[ev.ItemID] = body
			s.app.showEventDetails(ev, body)
		})
	}()
}

func cellOf(grid [][]DayCell, day int) (int, int) {
	for row := 0; row < len(grid); row++ {
		for col := 0; col < 7; col++ {
			if grid[row][col].Day == day {
				return row, col
			}
		}
	}
	return -1, -1
}
