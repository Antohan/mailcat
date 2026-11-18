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
	grid   *tview.Table
	agenda *tview.List

	year      int
	month     time.Month
	events    []mail.Event // кэш текущего месяца
	loadedKey string       // "2026-11" — загружен ли месяц

	dayEvents []mail.Event // события выбранного дня (для повестки)
}

func newCalendarScreen(a *App) *calendarScreen {
	s := &calendarScreen{app: a}
	now := time.Now()
	s.year, s.month = now.Year(), now.Month()

	s.grid = tview.NewTable().SetSelectable(true, true).SetSeparator(' ')
	s.grid.SetBorder(true)
	s.agenda = tview.NewList().ShowSecondaryText(false)
	s.agenda.SetBorder(true).SetTitle(" Повестка дня ")

	s.root = tview.NewFlex().
		AddItem(s.grid, 0, 55, true).
		AddItem(s.agenda, 0, 45, false)

	s.grid.SetSelectionChangedFunc(func(row, col int) {
		s.onDaySelected(row, col)
	})
	s.agenda.SetSelectedFunc(func(idx int, _ string, _ string, _ rune) {
		s.openEventDetails(idx)
	})

	// Смена месяца: Ctrl-←/→ или n/p (PLAN этап 4); 1/2/q — глобально (app).
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
			}
		}
		return event
	}
	s.grid.SetInputCapture(monthHook)
	s.agenda.SetInputCapture(monthHook)
	return s
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
	s.loadMonth()
}

// loadMonth запрашивает встречи за месяц (CalendarView) и перерисовывает экран.
func (s *calendarScreen) loadMonth() {
	start, end := MonthBounds(s.year, s.month)
	s.app.setStatus(fmt.Sprintf("Загрузка календаря: %s…", s.month.String()))
	go func() {
		raw, err := s.app.client.FindEvents(s.app.ctx, start, end)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() { s.app.showError(err) })
			return
		}
		events := mail.Events(raw)
		s.app.tapp.QueueUpdateDraw(func() {
			s.events = events
			s.loadedKey = s.monthKey()
			s.render()
			s.app.setStatus("←/→/↑/↓ день · n/p месяц · Enter детали · 1 почта · q выход")
		})
	}()
}

// render перерисовывает сетку и выбирает текущий день (или 1-е число).
func (s *calendarScreen) render() {
	s.grid.Clear()
	weekdays := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
	for col, name := range weekdays {
		s.grid.SetCell(0, col, tview.NewTableCell(name).
			SetTextColor(tcell.ColorGray).SetSelectable(false))
	}
	grid := MonthGrid(s.year, s.month, s.events)
	now := time.Now()
	for row := 0; row < 6; row++ {
		for col := 0; col < 7; col++ {
			cell := grid[row][col]
			if cell.Day == 0 {
				s.grid.SetCell(row+1, col, tview.NewTableCell("").SetSelectable(false))
				continue
			}
			text := fmt.Sprintf("%2d", cell.Day)
			if cell.HasEvents {
				text = "•" + text // день с событиями
			}
			tc := tview.NewTableCell(text)
			if col >= 5 {
				tc.SetTextColor(tcell.ColorYellow) // Сб/Вс
			}
			if !cell.InMonth {
				tc.SetTextColor(tcell.ColorGray)
			}
			s.grid.SetCell(row+1, col, tc)
		}
	}
	// Выбор: сегодняшний дню месяца, иначе 1-е.
	pick := 1
	if now.Year() == s.year && now.Month() == s.month {
		pick = now.Day()
	}
	if row, col := cellOf(grid, pick); row >= 0 {
		s.grid.Select(row+1, col)
	}
}

// onDaySelected обновляет повестку при движении по сетке.
func (s *calendarScreen) onDaySelected(row, col int) {
	if row < 1 {
		return
	}
	grid := MonthGrid(s.year, s.month, s.events)
	if row-1 >= len(grid) || col >= 7 || grid[row-1][col].Day == 0 {
		return
	}
	day := time.Date(s.year, s.month, grid[row-1][col].Day, 0, 0, 0, 0, time.Local)
	s.dayEvents = DayEvents(s.events, day)
	s.agenda.Clear()
	if len(s.dayEvents) == 0 {
		s.agenda.AddItem("Встреч нет", "", 0, nil)
		return
	}
	for _, ev := range s.dayEvents {
		line := fmt.Sprintf("%s–%s  %s",
			ev.Start.Local().Format("15:04"), ev.End.Local().Format("15:04"),
			escapeForList(ev.Subject))
		if ev.Location != "" {
			line += " (" + escapeForList(ev.Location) + ")"
		}
		s.agenda.AddItem(line, "", 0, nil)
	}
	s.agenda.SetTitle(fmt.Sprintf(" Повестка: %s ", day.Format("02.01.2006")))
}

// openEventDetails показывает детали выбранной встречи; ссылки на созвон
// (в поле «Место») — через реестр ссылок, как в письмах (PLAN этап 4).
func (s *calendarScreen) openEventDetails(idx int) {
	if idx < 0 || idx >= len(s.dayEvents) {
		return
	}
	ev := s.dayEvents[idx]
	s.app.showEventDetails(ev)
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
