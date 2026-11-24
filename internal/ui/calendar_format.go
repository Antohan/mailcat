package ui

// Чистая логика экрана календаря (PLAN.md этап 4): месячная сетка 6×7
// (Пн-первый), повестка дня, детали встречи. Сюда же — границы месяца
// для FindEvents.

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/security"
)

// DayCell — ячейка месячной сетки.
type DayCell struct {
	Day     int  // число дня; у соседнего месяца — своё число (V-46)
	InMonth bool // false — день соседнего месяца: только отображение
	Events  int  // число событий дня (многодневная встреча считает каждый свой день)
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
// событий по дням (многодневные считают каждый свой день). Ячейки соседних
// месяцев заполнены их числами (V-46) — сетка без пустых квадратов, но такие
// дни не выбираются и счётчиков не получают.
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
	prevDays := first.AddDate(0, 0, -1).Day()

	// Сдвиг до первого понедельника: вс=0 → 6, пн=1 → 0, ...
	lead := int(first.Weekday()+6) % 7

	grid := make([][]DayCell, 6)
	day := 1 - lead
	for row := 0; row < 6; row++ {
		grid[row] = make([]DayCell, 7)
		for col := 0; col < 7; col++ {
			switch {
			case day < 1:
				grid[row][col] = DayCell{Day: prevDays + day}
			case day <= daysInMonth:
				grid[row][col] = DayCell{Day: day, InMonth: true, Events: count[day]}
			default:
				grid[row][col] = DayCell{Day: day - daysInMonth}
			}
			day++
		}
	}
	return grid
}

// MonthWeeks — ISO-номера недель (пн-первая, time.ISOWeek) строк сетки месяца:
// weeks[row] — номер недели понедельника этой строки. Считается по дате, а не
// по числу дня, поэтому строка соседнего месяца получает свой номер.
func MonthWeeks(year int, month time.Month) []int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	monday := first.AddDate(0, 0, -((int(first.Weekday()) + 6) % 7))
	weeks := make([]int, 6)
	for row := range weeks {
		_, weeks[row] = monday.AddDate(0, 0, 7*row).ISOWeek()
	}
	return weeks
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
	text := fmt.Sprintf("%d", cell.Day)
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

// responseStatus — маркер статуса ответа и его роль в повестке (UI-4):
// ✓ принят (connected) · ~ под вопросом (weekend) · ? нет ответа (secondary)
// · ★ организатор (accent). ✗ (отклонено) в плане не назван — добавлен по
// симметрии с ярлыками ResponseTypeLabel (error); Unknown → пробел, чтобы тема
// стояла в ту же колонку.
func responseStatus(responseType string) (mark string, color tcell.Color) {
	switch strings.TrimSpace(responseType) {
	case "Accept":
		return "✓", colorConnected
	case "Tentative":
		return "~", colorWeekend
	case "Decline":
		return "✗", colorError
	case "NoResponseReceived":
		return "?", colorTextSecondary
	case "Organizer":
		return "★", colorAccent
	default:
		return " ", colorTextSecondary
	}
}

// AgendaSegments — блок повестки выбранного дня в стиле списка писем
// (V-34/V-51): две строки — «14:30–15:00 ✓ Тема» (время и тема primary,
// статус — роль ответа) и «(Место)» muted. Прошедшая встреча (End < now) — всё
// muted; выбранная — onAccent: на акцентной заливке primary тёмный, а ★ совпал
// бы с фоном. Вторая строка пустая, если места нет (строка остаётся — ритм
// двухстрочных событий V-51).
// Длинные URL из «Места» уходят в реестр reg (модал g), вместо них — маркер
// [N] host; обычное место остаётся текстом в скобках.
// WHY (V-42): сегменты {текст, стиль}, а не строка с тегами tview — повестка
// рисуется собственным Draw через tcell, теги там не разбираются. Поэтому
// tview.Escape не нужен, обязателен только Sanitize (SECURITY.md §3).
// Не делать: возврат к tview.List и тегам — перенос строк тогда невозможен.
func AgendaSegments(ev mail.Event, reg *mail.LinkRegistry, selected bool, now time.Time) [][]agendaSeg {
	subject, muted := colorTextPrimary, colorTextSecondary
	mark, status := responseStatus(ev.MyResponseType)
	switch {
	case selected:
		subject, muted, status = colorOnAccent, colorOnAccent, colorOnAccent
	case ev.End.Before(now): // прошедшая — только по локальному времени (V-51)
		subject, muted, status = colorTextSecondary, colorTextSecondary, colorTextSecondary
	}
	head := []agendaSeg{
		{fmt.Sprintf("%s–%s ", ev.Start.Local().Format("15:04"), ev.End.Local().Format("15:04")),
			tcell.StyleDefault.Foreground(subject)},
		{mark + " ", tcell.StyleDefault.Foreground(status)},
		{security.Sanitize(ev.Subject), tcell.StyleDefault.Foreground(subject)},
	}
	var place []agendaSeg
	if loc := reg.RenderBody(ev.Location); loc != "" {
		place = []agendaSeg{{"(" + security.Sanitize(loc) + ")", tcell.StyleDefault.Foreground(muted)}}
	}
	return [][]agendaSeg{head, place}
}

// AgendaTitle — левая часть заголовка повестки: «Повестка · 18.11».
// V-31: без капса и без счётчика — он уходит в правую часть (meetingsCount).
func AgendaTitle(day time.Time) string {
	return "Повестка · " + day.Format("02.01")
}

// NearestEvent — ближайшее будущее событие месяца (V-53): Start строго после
// now, самое раннее из подходящих. Прошедшие (Start ≤ now) не берём.
func NearestEvent(events []mail.Event, now time.Time) (mail.Event, bool) {
	var best mail.Event
	found := false
	for _, ev := range events {
		if !ev.Start.After(now) {
			continue
		}
		if !found || ev.Start.Before(best.Start) {
			best, found = ev, true
		}
	}
	return best, found
}

// NearestLine — строка «Ближайшее: 14:30–15:00 Тема» под сеткой (V-53):
// подпись muted, время и тема primary; если будущих событий нет — muted
// «Больше событий нет». Тема — через Sanitize: сегменты рисуются tcell,
// tview.Escape не нужен (как у AgendaSegments).
func NearestLine(events []mail.Event, now time.Time) []agendaSeg {
	muted := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	ev, ok := NearestEvent(events, now)
	if !ok {
		return []agendaSeg{{"Больше событий нет", muted}}
	}
	primary := tcell.StyleDefault.Foreground(colorTextPrimary).Background(colorPanelBg)
	return []agendaSeg{
		{"Ближайшее: ", muted},
		{fmt.Sprintf("%s–%s ", ev.Start.Local().Format("15:04"), ev.End.Local().Format("15:04")), primary},
		{security.Sanitize(ev.Subject), primary},
	}
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

// FormatEventHeader — шапка деталей встречи в типографике письма (V-36):
// подписи как в FormatMessageHeader, тема жирным, линия-разделитель; тело
// печатает showEventDetails. «Место» проходит через реестр ссылок: URL
// заменяется маркером [N] host (модал g), как в повестке. Все данные EWS —
// Sanitize + tview.Escape (строка попадает в TextView с dynamic colors).
func FormatEventHeader(ev mail.Event, reg *mail.LinkRegistry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[::b]Тема:  %s[-::-]\n", tview.Escape(security.Sanitize(ev.Subject)))
	if org := ev.Organizer.String(); org != "" {
		b.WriteString("От:    " + tview.Escape(security.Sanitize(org)) + "\n")
	}
	if len(ev.RequiredAttendees) > 0 {
		b.WriteString("Кому:  " + tview.Escape(security.Sanitize(joinAddresses(ev.RequiredAttendees))) + "\n")
	}
	if len(ev.OptionalAttendees) > 0 {
		b.WriteString("Копия: " + tview.Escape(security.Sanitize(joinAddresses(ev.OptionalAttendees))) + "\n")
	}
	fmt.Fprintf(&b, "Дата:  %s–%s\n",
		ev.Start.Local().Format("02.01.2006 15:04"), ev.End.Local().Format("15:04"))
	if ev.Location != "" {
		loc := reg.RenderBody(ev.Location)
		b.WriteString("Место: " + tview.Escape(security.Sanitize(loc)) + "\n")
	}
	fmt.Fprintf(&b, "Статус: %s\n", mail.ResponseTypeLabel(ev.MyResponseType))
	b.WriteString(mutedText(strings.Repeat("─", headerRuleWidth)) + "\n")
	return b.String()
}
