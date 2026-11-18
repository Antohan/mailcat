package ui

// Чистая логика экрана календаря (PLAN.md этап 4): месячная сетка 6×7
// (Пн-первый), повестка дня, детали встречи. Сюда же — границы месяца
// для FindEvents.

import (
	"fmt"
	"strings"
	"time"

	"mailcat/internal/mail"
	"mailcat/internal/security"
)

// DayCell — ячейка месячной сетки.
type DayCell struct {
	Day       int // 0 = пустая ячейка (соседний месяц)
	InMonth   bool
	HasEvents bool
	Selected  bool
}

// MonthBounds возвращает [первое число месяца 00:00 локально, последнее
// 23:59:59] — диапазон для FindEvents.
func MonthBounds(year int, month time.Month) (time.Time, time.Time) {
	loc := time.Local
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	end := time.Date(year, month+1, 1, 0, 0, 0, 0, loc).Add(-time.Second)
	return start, end
}

// MonthGrid строит сетку 6×7 (строки-недели, Пн-первая колонка) с пометками
// дней, занятых событиями (многодневные помечают каждый свой день).
func MonthGrid(year int, month time.Month, events []mail.Event) [][]DayCell {
	// Дни месяца с событиями.
	busy := map[int]bool{}
	for _, ev := range events {
		for d := ev.Start.Local(); !d.After(ev.End.Local()); d = d.AddDate(0, 0, 1) {
			if d.Month() == month && d.Year() == year {
				busy[d.Day()] = true
			}
			if ev.End.Sub(ev.Start) < 24*time.Hour && ev.Start.Local().Day() != ev.End.Local().Day() {
				break // встреча в один календарный день
			}
		}
	}

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, -1).Day()

	// Сдвиг до первого понедельника: вс=0 → 6, пн=1 → 0, ...
	lead := int(first.Weekday()+6) % 7

	grid := make([][]DayCell, 6)
	day := 1 - lead
	for row := 0; row < 6; row++ {
		grid[row] = make([]DayCell, 7)
		for col := 0; col < 7; col++ {
			if day >= 1 && day <= daysInMonth {
				grid[row][col] = DayCell{Day: day, InMonth: true, HasEvents: busy[day]}
			}
			day++
		}
	}
	return grid
}

// DayEvents выбирает события, попадающие в календарный день (многодневные
// включаются в каждый свой день), отсортированные по началу (сервер уже
// отдаёт Ascending, но пере-сортируем для устойчивости).
func DayEvents(events []mail.Event, day time.Time) []mail.Event {
	var out []mail.Event
	for _, ev := range events {
		for d := ev.Start.Local(); !d.After(ev.End.Local()); d = d.AddDate(0, 0, 1) {
			sameDay := d.Year() == day.Year() && d.Month() == day.Month() && d.Day() == day.Day()
			if sameDay {
				out = append(out, ev)
				break
			}
			if ev.End.Sub(ev.Start) < 24*time.Hour {
				break
			}
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Start.Before(out[j-1].Start); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// FormatAgenda — повестка выбранного дня: время, тема, место.
func FormatAgenda(events []mail.Event) string {
	if len(events) == 0 {
		return "На этот день встреч нет."
	}
	lines := make([]string, 0, len(events))
	for _, ev := range events {
		line := fmt.Sprintf("%s–%s  %s",
			ev.Start.Local().Format("15:04"), ev.End.Local().Format("15:04"),
			security.Sanitize(ev.Subject))
		if ev.Location != "" {
			line += " (" + security.Sanitize(ev.Location) + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// FormatEventDetails — детали встречи: организатор, участники, ваш статус.
func FormatEventDetails(ev mail.Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Тема:    %s\n", security.Sanitize(ev.Subject))
	fmt.Fprintf(&b, "Начало:  %s\n", ev.Start.Local().Format("02.01.2006 15:04"))
	fmt.Fprintf(&b, "Конец:   %s\n", ev.End.Local().Format("02.01.2006 15:04"))
	if ev.Location != "" {
		fmt.Fprintf(&b, "Место:   %s\n", security.Sanitize(ev.Location))
	}
	fmt.Fprintf(&b, "Организатор: %s\n", security.Sanitize(ev.Organizer.String()))
	fmt.Fprintf(&b, "Ваш статус:  %s\n", mail.ResponseTypeLabel(ev.MyResponseType))
	if len(ev.RequiredAttendees) > 0 {
		b.WriteString("Обязательные: ")
		for i, a := range ev.RequiredAttendees {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(security.Sanitize(a.String()))
		}
		b.WriteString("\n")
	}
	if len(ev.OptionalAttendees) > 0 {
		b.WriteString("Необязательные: ")
		for i, a := range ev.OptionalAttendees {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(security.Sanitize(a.String()))
		}
		b.WriteString("\n")
	}
	return b.String()
}
