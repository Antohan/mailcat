package ui

// Тесты чистой логики экрана календаря (PLAN.md этап 4): месячная сетка
// (Пн-первый, пометки дней с событиями, выбранный день), повестка дня,
// детали встречи с русским статусом ответа.

import (
	"strings"
	"testing"
	"time"

	"mailcat/internal/mail"
)

func nov(t *testing.T, day, h, m int) time.Time {
	t.Helper()
	// Календарь показывает локальное время — тестовые события тоже локальные.
	return time.Date(2026, 11, day, h, m, 0, 0, time.Local)
}

func TestMonthGridStructure(t *testing.T) {
	// Ноябрь 2026: 1-е число — воскресенье; при Пн-первом первая строка
	// заканчивается на 1, последняя (30-е) начинается с понедельника.
	grid := MonthGrid(2026, time.November, []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30)},
		{Start: nov(t, 20, 11, 0), End: nov(t, 20, 11, 45)},
	})
	if len(grid) != 6 || len(grid[0]) != 7 {
		t.Fatalf("размер сетки = %d×%d, хочу 6×7", len(grid), len(grid[0]))
	}
	if grid[0][6].Day != 1 {
		t.Errorf("первое число не в вс (7-я колонка): %+v", grid[0])
	}
	for _, row := range grid[:5] {
		for _, cell := range row {
			if cell.Day == 0 && (cell.HasEvents || cell.Selected) {
				t.Errorf("пустая ячейка с флагами: %+v", cell)
			}
		}
	}
	// День с событиями помечен, соседний без — нет.
	nov17 := findCell(grid, 17)
	if nov17 == nil || !nov17.HasEvents {
		t.Errorf("17 ноября не помечен: %+v", nov17)
	}
	nov16 := findCell(grid, 16)
	if nov16 == nil || nov16.HasEvents {
		t.Errorf("16 ноября ошибочно помечен: %+v", nov16)
	}
	// Многодневное событие помечает все свои дни.
	grid = MonthGrid(2026, time.November, []mail.Event{
		{Start: nov(t, 10, 9, 0), End: nov(t, 12, 18, 0)},
	})
	for _, d := range []int{10, 11, 12} {
		if c := findCell(grid, d); c == nil || !c.HasEvents {
			t.Errorf("день %d многодневной встречи не помечен", d)
		}
	}
}

func findCell(grid [][]DayCell, day int) *DayCell {
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c].Day == day {
				return &grid[r][c]
			}
		}
	}
	return nil
}

func TestDayEvents(t *testing.T) {
	events := []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк"},
		{Start: nov(t, 18, 14, 0), End: nov(t, 18, 15, 0), Subject: "Планирование"},
		{Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 30), Subject: "1:1"},
		// многодневная, захватывающая 17-е
		{Start: nov(t, 16, 9, 0), End: nov(t, 18, 10, 0), Subject: "Конференция"},
	}
	day := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	got := DayEvents(events, day)
	if len(got) != 3 {
		t.Fatalf("событий дня = %d, хочу 3 (включая многодневную): %+v", len(got), got)
	}
}

// TestResponseMark: маркеры статуса ответа для строки повестки (UI-4);
// Decline (✗) в плане не назван — добавлен по симметрии с ярлыками
// ResponseTypeLabel; Unknown — пробел, чтобы тема стояла в ту же колонку.
func TestResponseMark(t *testing.T) {
	cases := map[string]string{
		"Accept":             "✓",
		"Tentative":          "~",
		"NoResponseReceived": "?",
		"Organizer":          "★",
		"Decline":            "✗",
		"Unknown":            " ",
		"":                   " ",
	}
	for in, want := range cases {
		if got := responseMark(in); got != want {
			t.Errorf("responseMark(%q) = %q, хочу %q", in, got, want)
		}
	}
}

func TestFormatAgenda(t *testing.T) {
	events := []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк команды",
			Location: "https://zoom.example/j/987654321?pwd=abcdef", MyResponseType: "Accept"},
		{Start: nov(t, 17, 14, 30), End: nov(t, 17, 15, 0), Subject: "Планирование",
			Location: "Переговорная 3", MyResponseType: "Tentative"},
		{Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 45), Subject: "1:1", MyResponseType: "NoResponseReceived"},
		{Start: nov(t, 17, 18, 0), End: nov(t, 17, 18, 30), Subject: "Демо заказчику", MyResponseType: "Organizer"},
	}
	reg := mail.NewLinkRegistry()
	lines := FormatAgenda(events, reg)
	if len(lines) != 4 {
		t.Fatalf("строк повестки = %d, хочу 4", len(lines))
	}
	// URL из «Места» уходит в реестр: в строке — маркер [N] и хост, не полный URL.
	// Маркер проверяем в экранированной для tview.List форме («[1[]» → на экране
	// «[1]»), как в TestFormatMessageRowEscaped: экранирование обязательно.
	for _, want := range []string{"10:00–10:30", "✓", "Синк команды", "[1[] zoom.example"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("в строке повестки нет %q: %s", want, lines[0])
		}
	}
	if strings.Contains(lines[0], "987654321") {
		t.Errorf("полный URL не убран из строки: %s", lines[0])
	}
	if len(reg.List()) != 1 {
		t.Fatalf("в реестре %d ссылок, хочу 1", len(reg.List()))
	}
	// Обычное место остаётся текстом; статусы — ~, ?, ★.
	for _, c := range []struct{ line, want string }{
		{lines[1], "14:30–15:00"},
		{lines[1], "~"},
		{lines[1], "Планирование"},
		{lines[1], "Переговорная 3"},
		{lines[2], "?"},
		{lines[3], "★"},
	} {
		if !strings.Contains(c.line, c.want) {
			t.Errorf("в строке повестки нет %q: %s", c.want, c.line)
		}
	}
}

// TestAgendaTitle: заголовок панели «Повестка: 18.11 — N встреч» с русскими
// формами множественного числа (1 встреча, 2–4 встречи, 5+ встреч, 11–14 встреч).
func TestAgendaTitle(t *testing.T) {
	day := time.Date(2026, 11, 18, 0, 0, 0, 0, time.Local)
	cases := []struct {
		n    int
		want string
	}{
		{0, "Повестка: 18.11 — 0 встреч"},
		{1, "Повестка: 18.11 — 1 встреча"},
		{3, "Повестка: 18.11 — 3 встречи"},
		{5, "Повестка: 18.11 — 5 встреч"},
		{11, "Повестка: 18.11 — 11 встреч"},
		{21, "Повестка: 18.11 — 21 встреча"},
		{22, "Повестка: 18.11 — 22 встречи"},
	}
	for _, c := range cases {
		if got := AgendaTitle(day, c.n); got != c.want {
			t.Errorf("AgendaTitle(%d) = %q, хочу %q", c.n, got, c.want)
		}
	}
}

// TestOnDaySelectedAgenda: повестка дня собирается через FormatAgenda —
// маркеры и реестр ссылок видны в элементах списка, dayLinks наполняется
// для модала g.
func TestOnDaySelectedAgenda(t *testing.T) {
	a := newTestApp(t)
	s := a.calendar
	s.year, s.month = 2026, time.November
	s.events = []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк",
			Location: "https://meet.example/x", MyResponseType: "Accept"},
	}
	grid := MonthGrid(2026, time.November, s.events)
	row, col := cellOf(grid, 17)
	s.onDaySelected(row+1, col)
	if got := s.agenda.GetItemCount(); got != 1 {
		t.Fatalf("элементов повестки = %d, хочу 1", got)
	}
	line, _ := s.agenda.GetItemText(0)
	if !strings.Contains(line, "✓") || !strings.Contains(line, "[1[]") {
		t.Errorf("строка повестки без маркеров: %q", line)
	}
	if len(s.dayLinks) != 1 {
		t.Errorf("dayLinks = %d ссылок, хочу 1 (для модала g)", len(s.dayLinks))
	}
	if title := s.agenda.GetTitle(); !strings.Contains(title, "1 встреча") {
		t.Errorf("заголовок панели: %q", title)
	}
}

func TestFormatEventDetails(t *testing.T) {
	ev := mail.Event{
		Subject:           "Синк команды",
		Start:             nov(t, 17, 10, 0),
		End:               nov(t, 17, 10, 30),
		Location:          "Zoom",
		Organizer:         mail.Address{Name: "Иванов И.", Email: "ivanov@company.example"},
		MyResponseType:    "Accept",
		RequiredAttendees: []mail.Address{{Email: "apetrova@company.example"}},
		OptionalAttendees: []mail.Address{{Email: "dkuznetsov@company.example"}},
	}
	got := FormatEventDetails(ev)
	for _, want := range []string{
		"Синк команды", "10:00", "10:30", "Zoom",
		"Иванов И. <ivanov@company.example>",
		"Обязательные:", "apetrova@company.example",
		"Необязательные:", "dkuznetsov@company.example",
		"Ваш статус", "принято",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("в деталях нет %q:\n%s", want, got)
		}
	}
}

func TestMonthBounds(t *testing.T) {
	start, end := MonthBounds(2026, time.November)
	if start.Day() != 1 || start.Month() != time.November {
		t.Errorf("start = %v", start)
	}
	if end.Day() != 30 || end.Month() != time.November {
		t.Errorf("end = %v", end)
	}
}
