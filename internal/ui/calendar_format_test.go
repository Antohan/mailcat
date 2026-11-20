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

func TestFormatAgenda(t *testing.T) {
	events := []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк команды", Location: "Zoom"},
		{Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 45), Subject: "1:1 с руководителем", Location: ""},
	}
	got := FormatAgenda(events)
	for _, want := range []string{"10:00–10:30", "Синк команды", "Zoom", "16:00–16:45", "1:1 с руководителем"} {
		if !strings.Contains(got, want) {
			t.Errorf("в повестке нет %q:\n%s", want, got)
		}
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
