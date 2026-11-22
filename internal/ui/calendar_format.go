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
	Day      int // 0 = пустая ячейка (соседний месяц)
	InMonth  bool
	Events   int // число событий дня (многодневная встреча считает каждый свой день)
	Selected bool
}

// MonthBounds возвращает [первое число месяца 00:00 локально, последнее
// 23:59:59] — диапазон для FindEvents.
func MonthBounds(year int, month time.Month) (time.Time, time.Time) {
	loc := time.Local
	start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	end := time.Date(year, month+1, 1, 0, 0, 0, 0, loc).Add(-time.Second)
	return start, end
}

// MonthGrid строит сетку 6×7 (строки-недели, Пн-первая колонка) с числом
// событий по дням (многодневные считают каждый свой день).
func MonthGrid(year int, month time.Month, events []mail.Event) [][]DayCell {
	// Дни месяца с числом событий.
	count := map[int]int{}
	for _, ev := range events {
		for d := ev.Start.Local(); !d.After(ev.End.Local()); d = d.AddDate(0, 0, 1) {
			if d.Month() == month && d.Year() == year {
				count[d.Day()]++
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
				grid[row][col] = DayCell{Day: day, InMonth: true, Events: count[day]}
			}
			day++
		}
	}
	return grid
}

// russianMonths — именительный падеж для заголовка панели («Ноябрь 2026»);
// time.Month из time.Date всегда 1–12, выход за границы исключён.
var russianMonths = [...]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// MonthTitle — заголовок панели месяца: «Ноябрь 2026».
func MonthTitle(year int, month time.Month) string {
	return fmt.Sprintf("%s %d", russianMonths[month-1], year)
}

// FormatDayCell — текст ячейки сетки месяца: число дня, при событиях — их
// количество («17·2»).
// WHY: маркер «сегодня» и выбранный день живут в стиле ячейки, а не в тексте
// (V-33): скобки «[N]» путались с маркерами ссылок.
func FormatDayCell(cell DayCell) string {
	if cell.Day == 0 {
		return ""
	}
	text := fmt.Sprintf("%2d", cell.Day)
	if cell.Events > 0 {
		text += fmt.Sprintf("·%d", cell.Events)
	}
	return text
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

// responseMark — маркер статуса ответа в строке повестки (UI-4):
// ✓ принят · ~ под вопросом · ? нет ответа · ★ организатор.
// ✗ (отклонено) в плане не назван — добавлен по симметрии с ярлыками
// ResponseTypeLabel; Unknown → пробел, чтобы тема стояла в ту же колонку.
func responseMark(responseType string) string {
	switch strings.TrimSpace(responseType) {
	case "Accept":
		return "✓"
	case "Tentative":
		return "~"
	case "Decline":
		return "✗"
	case "NoResponseReceived":
		return "?"
	case "Organizer":
		return "★"
	default:
		return " "
	}
}

// FormatAgenda — строки повестки выбранного дня: «14:30–15:00 ✓ Тема [1] host».
// Длинные URL из «Места» уходят в реестр reg (модал g), вместо них — маркер
// [N] host; обычное место остаётся текстом в скобках. Строки идут в
// tview.List — Sanitize + tview.Escape (escapeForList) обязательны.
func FormatAgenda(events []mail.Event, reg *mail.LinkRegistry) []string {
	if len(events) == 0 {
		return nil
	}
	lines := make([]string, 0, len(events))
	for _, ev := range events {
		line := fmt.Sprintf("%s–%s %s %s",
			ev.Start.Local().Format("15:04"), ev.End.Local().Format("15:04"),
			responseMark(ev.MyResponseType), escapeForList(ev.Subject))
		if loc := reg.RenderBody(ev.Location); loc != "" {
			line += " (" + escapeForList(loc) + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

// AgendaTitle — левая часть заголовка повестки: «Повестка · 18.11».
// V-31: без капса и без счётчика — он уходит в правую часть (meetingsCount).
func AgendaTitle(day time.Time) string {
	return "Повестка · " + day.Format("02.01")
}

// meetingsCount — правая часть шапки панели: «3 встречи» (месяц/повестка).
// WHY: формы множественного числа нужны и сетке месяца, и повестке; без
// капса — счётчик стоит рядом с данными, а не UI-подписью.
func meetingsCount(n int) string {
	return fmt.Sprintf("%d %s", n, meetingsWord(n))
}

// meetingsWord — русская форма множественного числа: 1 встреча, 2–4 встречи,
// 5+ встреч (включая 11–14).
func meetingsWord(n int) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "встреч"
	}
	switch n % 10 {
	case 1:
		return "встреча"
	case 2, 3, 4:
		return "встречи"
	default:
		return "встреч"
	}
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
