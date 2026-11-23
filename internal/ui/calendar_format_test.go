package ui

// Тесты чистой логики экрана календаря (PLAN.md этап 4): месячная сетка
// (Пн-первый, пометки дней с событиями, выбранный день), повестка дня,
// детали встречи с русским статусом ответа.

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

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
			if cell.Day == 0 && (cell.Events > 0 || cell.Selected) {
				t.Errorf("пустая ячейка с флагами: %+v", cell)
			}
		}
	}
	// День с событиями помечен, соседний без — нет.
	nov17 := findCell(grid, 17)
	if nov17 == nil || nov17.Events != 1 {
		t.Errorf("17 ноября: %+v, хочу 1 событие", nov17)
	}
	nov16 := findCell(grid, 16)
	if nov16 == nil || nov16.Events != 0 {
		t.Errorf("16 ноября: %+v, хочу 0 событий", nov16)
	}
	// Многодневное событие помечает все свои дни.
	grid = MonthGrid(2026, time.November, []mail.Event{
		{Start: nov(t, 10, 9, 0), End: nov(t, 12, 18, 0)},
	})
	for _, d := range []int{10, 11, 12} {
		if c := findCell(grid, d); c == nil || c.Events != 1 {
			t.Errorf("день %d многодневной встречи: %+v, хочу 1 событие", d, c)
		}
	}
	// Несколько встреч в день складываются (интенсивность 1–2 vs 3+).
	grid = MonthGrid(2026, time.November, []mail.Event{
		{Start: nov(t, 18, 9, 0), End: nov(t, 18, 9, 30)},
		{Start: nov(t, 18, 11, 0), End: nov(t, 18, 11, 30)},
		{Start: nov(t, 18, 14, 0), End: nov(t, 18, 14, 30)},
	})
	if c := findCell(grid, 18); c == nil || c.Events != 3 {
		t.Errorf("18 ноября: %+v, хочу 3 события", c)
	}
}

// TestMonthTitle: заголовок панели месяца — русское имя и год (UI-5).
func TestMonthTitle(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  string
	}{
		{2026, time.November, "Ноябрь 2026"},
		{2026, time.December, "Декабрь 2026"},
		{2027, time.January, "Январь 2027"},
	}
	for _, c := range cases {
		if got := MonthTitle(c.year, c.month); got != c.want {
			t.Errorf("MonthTitle(%d, %v) = %q, хочу %q", c.year, c.month, got, c.want)
		}
	}
}

// TestFormatDayCell: маркеры ячейки (V-33, V-39) — количество событий «17·2»;
// без %2d-паддинга (число выровнено влево); скобок нет: «сегодня» и выбранный
// день показывает стиль ячейки.
func TestFormatDayCell(t *testing.T) {
	cases := []struct {
		name string
		cell DayCell
		want string
	}{
		{"без событий", DayCell{Day: 5}, "5"},
		{"двузначный день", DayCell{Day: 17}, "17"},
		{"одно событие", DayCell{Day: 17, Events: 1}, "17·1"},
		{"три события", DayCell{Day: 5, Events: 3}, "5·3"},
		{"пустая ячейка", DayCell{}, ""},
	}
	for _, c := range cases {
		if got := FormatDayCell(c.cell); got != c.want {
			t.Errorf("%s: FormatDayCell = %q, хочу %q", c.name, got, c.want)
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

// TestResponseStatus: маркеры статуса ответа и их роли в повестке (UI-4,
// V-34); Decline (✗) в плане не назван — добавлен по симметрии с ярлыками
// ResponseTypeLabel; Unknown — пробел, чтобы тема стояла в ту же колонку.
func TestResponseStatus(t *testing.T) {
	cases := map[string]struct {
		mark string
		role tcell.Color
	}{
		"Accept":             {"✓", colorConnected},
		"Tentative":          {"~", colorWeekend},
		"NoResponseReceived": {"?", colorTextSecondary},
		"Organizer":          {"★", colorAccent},
		"Decline":            {"✗", colorError},
		"Unknown":            {" ", colorTextSecondary},
		"":                   {" ", colorTextSecondary},
	}
	for in, want := range cases {
		mark, role := responseStatus(in)
		if mark != want.mark || role != want.role {
			t.Errorf("responseStatus(%q) = %q/%v, хочу %q/%v",
				in, mark, role, want.mark, want.role)
		}
	}
}

// TestFormatAgendaLine: строка повестки в стиле списка писем (V-34) — время,
// статус и место цветными ролями, тема primary; в выбранной строке всё
// onAccent. URL из «Места» уходит в реестр, маркер [N] экранирован для List.
func TestFormatAgendaLine(t *testing.T) {
	ev := mail.Event{
		Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк команды",
		Location: "https://zoom.example/j/987654321?pwd=abcdef", MyResponseType: "Accept",
	}
	reg := mail.NewLinkRegistry()
	line := FormatAgendaLine(ev, reg, false)
	for _, want := range []string{
		"[" + hexTag(colorTextSecondary) + "]10:00–10:30[-]",
		"[" + hexTag(colorConnected) + "]✓[-]",
		"[" + hexTag(colorTextPrimary) + "]Синк команды[-]",
		"[1[] zoom.example", // маркер реестра, экранированный для tview.List
	} {
		if !strings.Contains(line, want) {
			t.Errorf("в строке повестки нет %q: %s", want, line)
		}
	}
	if strings.Contains(line, "987654321") {
		t.Errorf("полный URL не убран из строки: %s", line)
	}
	if len(reg.List()) != 1 {
		t.Fatalf("в реестре %d ссылок, хочу 1", len(reg.List()))
	}

	// Обычное место остаётся текстом в скобках, muted.
	reg = mail.NewLinkRegistry()
	plain := FormatAgendaLine(mail.Event{
		Start: nov(t, 17, 14, 30), End: nov(t, 17, 15, 0), Subject: "Планирование",
		Location: "Переговорная 3", MyResponseType: "Tentative",
	}, reg, false)
	if !strings.Contains(plain, "["+hexTag(colorWeekend)+"]~[-]") {
		t.Errorf("статус «~» не в weekend-роли: %s", plain)
	}
	if !strings.Contains(plain, "["+hexTag(colorTextSecondary)+"](Переговорная 3)[-]") {
		t.Errorf("обычное место не muted в скобках: %s", plain)
	}
	if len(reg.List()) != 0 {
		t.Errorf("обычное место попало в реестр ссылок: %d", len(reg.List()))
	}

	// Выбранная строка: время/статус/тема — onAccent (на акцентной заливке
	// primary тёмный, ★ совпал бы с фоном).
	sel := FormatAgendaLine(ev, mail.NewLinkRegistry(), true)
	for _, want := range []string{
		"[" + hexTag(colorOnAccent) + "]10:00–10:30[-]",
		"[" + hexTag(colorOnAccent) + "]✓[-]",
		"[" + hexTag(colorOnAccent) + "]Синк команды[-]",
	} {
		if !strings.Contains(sel, want) {
			t.Errorf("в выбранной строке нет %q: %s", want, sel)
		}
	}
	for _, unwanted := range []string{hexTag(colorTextPrimary), hexTag(colorConnected)} {
		if strings.Contains(sel, unwanted) {
			t.Errorf("выбранная строка не на акценте (%s): %s", unwanted, sel)
		}
	}
}

// TestAgendaTitle: левая часть шапки повестки без капса и без счётчика (V-31),
// счётчик — в правой части meetingsCount, с русскими формами множественного
// числа (1 встреча, 2–4 встречи, 5+ встреч, 11–14 встреч).
func TestAgendaTitle(t *testing.T) {
	day := time.Date(2026, 11, 18, 0, 0, 0, 0, time.Local)
	if got, want := AgendaTitle(day), "Повестка · 18.11"; got != want {
		t.Errorf("AgendaTitle = %q, хочу %q", got, want)
	}
	cases := []struct {
		n    int
		want string
	}{
		{0, "0 встреч"},
		{1, "1 встреча"},
		{3, "3 встречи"},
		{5, "5 встреч"},
		{11, "11 встреч"},
		{21, "21 встреча"},
		{22, "22 встречи"},
	}
	for _, c := range cases {
		if got := meetingsCount(c.n); got != c.want {
			t.Errorf("meetingsCount(%d) = %q, хочу %q", c.n, got, c.want)
		}
	}
}

// TestOnDaySelectedAgenda: повестка дня собирается через FormatAgendaLine —
// строка под курсором onAccent, остальные primary, маркеры и реестр ссылок
// видны в элементах списка, dayLinks наполняется для модала g; пустой день —
// один muted-пункт, Enter без действия.
func TestOnDaySelectedAgenda(t *testing.T) {
	a := newTestApp(t)
	s := a.calendar
	s.year, s.month = 2026, time.November
	s.events = []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк",
			Location: "https://meet.example/x", MyResponseType: "Accept"},
		{Start: nov(t, 17, 14, 0), End: nov(t, 17, 14, 30), Subject: "Разбор",
			MyResponseType: "Accept"},
	}
	grid := MonthGrid(2026, time.November, s.events)
	s.grid.SetCells(grid, 0) // V-32: onDaySelected читает выбранную ячейку из сетки
	row, col := cellOf(grid, 17)
	s.onDaySelected(row, col)
	if got := s.agenda.GetItemCount(); got != 2 {
		t.Fatalf("элементов повестки = %d, хочу 2", got)
	}
	first, _ := s.agenda.GetItemText(0)
	if !strings.Contains(first, "✓") || !strings.Contains(first, "[1[]") ||
		!strings.Contains(first, "["+hexTag(colorOnAccent)+"]Синк[-]") {
		t.Errorf("строка под курсором не onAccent или без маркеров: %q", first)
	}
	second, _ := s.agenda.GetItemText(1)
	if !strings.Contains(second, "["+hexTag(colorTextPrimary)+"]Разбор[-]") {
		t.Errorf("невыбранная строка не primary: %q", second)
	}
	if len(s.dayLinks) != 1 {
		t.Errorf("dayLinks = %d ссылок, хочу 1 (для модала g)", len(s.dayLinks))
	}
	// RD-2: заголовок панели живёт в строке-заголовке, не в рамке (V-31:
	// слева — без капса, справа — счётчик встреч).
	if title := s.agendaHeader.title; title != "Повестка · 17.11" {
		t.Errorf("левый заголовок панели: %q", title)
	}
	if right := s.agendaHeader.right; right != "2 встречи" {
		t.Errorf("счётчик повестки: %q", right)
	}

	// Пустой день — одна muted-строка, ссылок нет, Enter ничего не открывает.
	row, col = cellOf(grid, 18)
	s.onDaySelected(row, col)
	if got := s.agenda.GetItemCount(); got != 1 {
		t.Fatalf("элементов пустой повестки = %d, хочу 1", got)
	}
	if line, _ := s.agenda.GetItemText(0); line != mutedText("Встреч нет") {
		t.Errorf("пустой день: %q, хочу muted «Встреч нет»", line)
	}
	if s.dayLinks != nil {
		t.Errorf("пустой день: dayLinks = %v, хочу nil", s.dayLinks)
	}
	s.openEventDetails(0) // Enter на заглушке — без действия
}

// TestAgendaListStyles: sim — повестка в стиле писем (V-34): выбранная строка
// залита на всю ширину акцентом, время второй строки muted, тема primary,
// полоса прокрутки — колонка справа за рамкой панели (30 встреч > видимых).
func TestAgendaListStyles(t *testing.T) {
	a := newTestApp(t)
	s := a.calendar
	s.year, s.month = 2026, time.November
	day := time.Date(2026, time.November, 22, 0, 0, 0, 0, time.Local)
	events := make([]mail.Event, 30)
	for i := range events {
		start := day.Add(time.Duration(8+i/2) * time.Hour).Add(time.Duration(i%2) * 30 * time.Minute)
		events[i] = mail.Event{Start: start, End: start.Add(30 * time.Minute),
			Subject: fmt.Sprintf("Встреча %d", i+1), MyResponseType: "Accept"}
	}
	s.events = events
	s.render()
	s.grid.SelectDay(22)
	a.pages.SwitchToPage(pageCalendar)

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	const w, h = 100, 30
	a.tapp.SetScreen(sim)
	sim.SetSize(w, h)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = a.tapp.Run()
	}()
	a.tapp.QueueUpdateDraw(func() {})
	styleAt := func(x, y int) tcell.Style {
		_, _, st, _ := sim.GetContent(x, y)
		return st
	}
	// tview-тег «[#ffffff]» разворачивается в RGB-цвет, а colorOnAccent —
	// палитровый ColorWhite; сравниваем по RGB, иначе ложное расхождение.
	same := func(a, b tcell.Color) bool {
		ar, ag, ab := a.RGB()
		br, bg, bb := b.RGB()
		return ar == br && ag == bg && ab == bb
	}

	ix, iy, iw, ih := s.agenda.GetInnerRect()
	// Выбранная первая встреча: текст onAccent, заливка на всю ширину строки.
	if fg, bg, _ := styleAt(ix, iy).Decompose(); !same(fg, colorSelectedFg) || !same(bg, colorSelectedBg) {
		t.Errorf("выбранная строка: fg/bg=%v/%v, хочу %v/%v", fg, bg, colorSelectedFg, colorSelectedBg)
	}
	if _, bg, _ := styleAt(ix+iw-1, iy).Decompose(); !same(bg, colorSelectedBg) {
		t.Errorf("выбранная строка залита не целиком: правый край bg=%v", bg)
	}
	// Вторая строка: время muted, тема primary.
	if fg, bg, _ := styleAt(ix, iy+1).Decompose(); !same(fg, colorTextSecondary) || same(bg, colorSelectedBg) {
		t.Errorf("время невыбранной строки: fg/bg=%v/%v, хочу %v", fg, bg, colorTextSecondary)
	}
	found := false
	for x := ix; x < ix+iw; x++ {
		if r, _, st, _ := sim.GetContent(x, iy+1); r == 'В' {
			if fg, _, _ := st.Decompose(); !same(fg, colorTextPrimary) {
				t.Errorf("тема невыбранной строки: fg=%v, хочу primary", fg)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("темы второй встречи нет в повестке")
	}
	// Полоса прокрутки: колонка за рамкой панели, бегунок залит secondary.
	bx, _, _, _ := s.agendaBar.GetRect()
	thumb := false
	for y := iy; y < iy+ih; y++ {
		if _, bg, _ := styleAt(bx, y).Decompose(); same(bg, colorTextSecondary) {
			thumb = true
			break
		}
	}
	if !thumb {
		t.Errorf("полосы прокрутки повестки нет в колонке %d", bx)
	}

	a.tapp.Stop()
	<-done
}

func TestFormatEventHeader(t *testing.T) {
	ev := mail.Event{
		Subject:           "Синк команды",
		Start:             nov(t, 17, 10, 0),
		End:               nov(t, 17, 10, 30),
		Location:          "https://meet.example/x",
		Organizer:         mail.Address{Name: "Иванов И.", Email: "ivanov@company.example"},
		MyResponseType:    "Accept",
		RequiredAttendees: []mail.Address{{Email: "apetrova@company.example"}},
		OptionalAttendees: []mail.Address{{Email: "dkuznetsov@company.example"}},
	}
	reg := mail.NewLinkRegistry()
	got := FormatEventHeader(ev, reg)
	for _, want := range []string{
		"[::b]Тема:  Синк команды[-::-]",
		"От:    Иванов И. <ivanov@company.example>",
		"Кому:  apetrova@company.example",
		"Копия: dkuznetsov@company.example",
		"10:00–10:30",
		"Место: [1[] meet.example", // URL в «Месте» — маркер реестра, не сырой адрес
		"Статус: принято",
		mutedText(strings.Repeat("─", headerRuleWidth)),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("в шапке встречи нет %q:\n%s", want, got)
		}
	}
	if len(reg.List()) != 1 {
		t.Errorf("ссылок «Места» в реестре = %d, хочу 1", len(reg.List()))
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

// TestShowEventDetailsBody (V-36): страница встречи — шапка в типографике
// письма, тело тем же конвейером (HTML → текст, ссылки, Escape); пустое
// описание — заглушка. Повторный Enter берёт кэш (без клиента — иначе паника).
func TestShowEventDetailsBody(t *testing.T) {
	a := newTestApp(t)
	ev := mail.Event{
		ItemID: "AAMkCAL", Subject: "Синк",
		Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30),
		Location: "https://meet.example/x", MyResponseType: "Accept",
	}

	a.showEventDetails(ev, mail.EventBody{})
	text := a.event.GetText(false)
	for _, want := range []string{"Тема:  Синк", "Место: [1[] meet.example", "Статус: принято", "(Встреча без описания)"} {
		if !strings.Contains(text, want) {
			t.Errorf("пустое описание: нет %q:\n%s", want, text)
		}
	}
	if len(a.currentLinks) != 1 {
		t.Errorf("ссылок «Места» = %d, хочу 1", len(a.currentLinks))
	}

	a.showEventDetails(ev, mail.EventBody{
		BodyHTML: `<p>Повестка: релиз</p><p>Созвон https://call.example/room</p>`,
		BodyText: "текст",
	})
	text = a.event.GetText(false)
	for _, want := range []string{"Повестка", "Созвон [2[] call.example"} {
		if !strings.Contains(text, want) {
			t.Errorf("HTML-описание: нет %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "(Встреча без описания)") {
		t.Error("при непустом описании осталась заглушка")
	}
	if len(a.currentLinks) != 2 {
		t.Errorf("ссылок = %d, хочу 2 (место + тело)", len(a.currentLinks))
	}

	// Кэш: описание уже загружено — второй Enter не идёт в сеть (client nil).
	s := a.calendar
	s.dayEvents = []mail.Event{ev}
	s.eventBodies = map[string]mail.EventBody{ev.ItemID: {BodyText: "из кэша"}}
	s.openEventDetails(0)
	s.openEventDetails(0)
	if got := a.event.GetText(false); !strings.Contains(got, "из кэша") {
		t.Errorf("кэш не использован:\n%s", got)
	}
	if len(s.eventLoading) != 0 {
		t.Errorf("кэш-путь оставил загрузку: %v", s.eventLoading)
	}
}

// TestEventDetailsScreen (V-36): sim страницы — шапка и описание видны в
// терминале, тело переносится по ширине.
func TestEventDetailsScreen(t *testing.T) {
	a := newTestApp(t)
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(100, 30)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	ev := mail.Event{
		Subject: "Синк команды", Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30),
		Location: "Zoom", MyResponseType: "Accept",
	}
	a.tapp.QueueUpdateDraw(func() {
		a.showEventDetails(ev, mail.EventBody{BodyText: "Повестка: релиз\n\nОбсуждение сроков."})
	})
	joined := ""
	for y := 0; y < 30; y++ {
		joined += string(simRow(sim, y))
	}
	for _, want := range []string{"Встреча", "Тема:  Синк команды", "10:00–10:30", "Место: Zoom", "Обсуждение сроков"} {
		if !strings.Contains(joined, want) {
			t.Errorf("на экране нет %q", want)
		}
	}
	a.tapp.Stop()
	<-done
}
