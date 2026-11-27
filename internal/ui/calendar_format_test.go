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
	"github.com/rivo/tview"

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
	// V-46: сетка заполнена 42 датами; соседние месяцы — свои числа,
	// InMonth=false и без счётчиков.
	inMonth := 0
	for _, row := range grid {
		for _, cell := range row {
			if cell.Day < 1 || cell.Day > 31 {
				t.Errorf("ячейка без числа дня: %+v", cell)
			}
			if cell.InMonth {
				inMonth++
			} else if cell.Events > 0 {
				t.Errorf("счётчик у дня соседнего месяца: %+v", cell)
			}
		}
	}
	if inMonth != 30 {
		t.Errorf("дней ноября в сетке = %d, хочу 30", inMonth)
	}
	if c := grid[0][0]; c.InMonth || c.Day != 26 {
		t.Errorf("первый понедельник до 1-го: %+v, хочу 26 октября", c)
	}
	if c := grid[5][6]; c.InMonth || c.Day != 6 {
		t.Errorf("последняя ячейка: %+v, хочу 6 декабря", c)
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

// TestMonthWeeks (V-52): ISO-номера недель строк сетки — пн-первая, номер
// недели понедельника, включая переход через год (2026-W53 → 2027-W01).
func TestMonthWeeks(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  []int
	}{
		{2026, time.November, []int{44, 45, 46, 47, 48, 49}},
		{2027, time.January, []int{53, 1, 2, 3, 4, 5}},
	}
	for _, c := range cases {
		got := MonthWeeks(c.year, c.month)
		if len(got) != len(c.want) {
			t.Fatalf("%d-%02d: недель %d, ждали %d", c.year, c.month, len(got), len(c.want))
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("%d-%02d: строка %d — неделя %d, ждали %d",
					c.year, c.month, i, got[i], c.want[i])
			}
		}
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
			if grid[r][c].InMonth && grid[r][c].Day == day {
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

// segText — текст сегментов повестки одной строкой (для проверок).
func segText(segs []agendaSeg) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.text)
	}
	return b.String()
}

// agendaSegs — все сегменты блока события (обе строки) одним срезом.
func agendaSegs(lines [][]agendaSeg) []agendaSeg {
	var out []agendaSeg
	for _, line := range lines {
		out = append(out, line...)
	}
	return out
}

// agendaText — текст блока события (обе строки) одной строкой.
func agendaText(lines [][]agendaSeg) string {
	return segText(agendaSegs(lines))
}

// segFg — цвет первого сегмента, содержащего part.
func segFg(t *testing.T, segs []agendaSeg, part string) tcell.Color {
	t.Helper()
	for _, s := range segs {
		if strings.Contains(s.text, part) {
			fg, _, _ := s.style.Decompose()
			return fg
		}
	}
	t.Fatalf("в сегментах нет %q: %q", part, segText(segs))
	return 0
}

// TestAgendaSegments: блок повестки в стиле списка писем (V-34/V-51) — две
// строки: время primary, статус и тема цветными ролями; место muted отдельной
// строкой. Прошедшая встреча (End < now) — всё muted, выбранная — onAccent.
// URL из «Места» уходит в реестр, маркер [N] — обычным текстом.
func TestAgendaSegments(t *testing.T) {
	ev := mail.Event{
		Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк команды",
		Location: "https://zoom.example/j/987654321?pwd=abcdef", MyResponseType: "Accept",
	}
	reg := mail.NewLinkRegistry()
	blocks := agendaSegments(ev, reg, false, nov(t, 17, 9, 0))
	if len(blocks) != 2 {
		t.Fatalf("строк в блоке = %d, хочу 2", len(blocks))
	}
	segs, place := blocks[0], blocks[1]
	line := segText(segs)
	for _, want := range []string{"10:00–10:30", "✓", "Синк команды"} {
		if !strings.Contains(line, want) {
			t.Errorf("в первой строке повестки нет %q: %s", want, line)
		}
	}
	if got := segText(place); got != "([1] zoom.example)" {
		t.Errorf("вторая строка (место) = %q, хочу «([1] zoom.example)»", got)
	}
	if strings.Contains(agendaText(blocks), "987654321") {
		t.Errorf("полный URL не убран из блока: %s", agendaText(blocks))
	}
	for part, want := range map[string]tcell.Color{
		"10:00": colorTextPrimary, "✓": colorConnected, "Синк": colorTextPrimary, "zoom": colorTextSecondary,
	} {
		if got := segFg(t, agendaSegs(blocks), part); got != want {
			t.Errorf("цвет %q = %v, хочу %v", part, got, want)
		}
	}
	if len(reg.List()) != 1 {
		t.Fatalf("в реестре %d ссылок, хочу 1", len(reg.List()))
	}

	// Обычное место остаётся текстом в скобках, muted, во второй строке.
	reg = mail.NewLinkRegistry()
	plain := agendaSegments(mail.Event{
		Start: nov(t, 17, 14, 30), End: nov(t, 17, 15, 0), Subject: "Планирование",
		Location: "Переговорная 3", MyResponseType: "Tentative",
	}, reg, false, nov(t, 17, 9, 0))
	if got := segFg(t, plain[0], "~"); got != colorWeekend {
		t.Errorf("статус «~» не в weekend-роли: %v", got)
	}
	if got := segFg(t, plain[1], "Переговорная"); got != colorTextSecondary {
		t.Errorf("обычное место не muted: %v", got)
	}
	if text := segText(plain[1]); !strings.Contains(text, "(Переговорная 3)") {
		t.Errorf("обычное место не в скобках: %q", text)
	}
	if len(reg.List()) != 0 {
		t.Errorf("обычное место попало в реестр ссылок: %d", len(reg.List()))
	}

	// Событие без места: вторая строка пустая, но остаётся (двухстрочный ритм).
	noPlace := agendaSegments(mail.Event{
		Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 30), Subject: "Без места",
	}, mail.NewLinkRegistry(), false, nov(t, 17, 9, 0))
	if len(noPlace) != 2 || len(noPlace[1]) != 0 {
		t.Errorf("блок без места = %q, хочу две строки, вторая пустая", agendaText(noPlace))
	}

	// Прошедшая встреча: всё muted (конец раньше now).
	past := agendaSegments(ev, mail.NewLinkRegistry(), false, nov(t, 17, 11, 0))
	for _, seg := range agendaSegs(past) {
		if got, _, _ := seg.style.Decompose(); got != colorTextSecondary {
			t.Errorf("прошедший сегмент %q не muted: %v", seg.text, got)
		}
	}

	// Выбранное событие: всё onAccent (на акцентной заливке primary тёмный,
	// ★ совпал бы с фоном).
	sel := agendaSegments(ev, mail.NewLinkRegistry(), true, nov(t, 17, 11, 0))
	for _, seg := range agendaSegs(sel) {
		if got, _, _ := seg.style.Decompose(); got != colorOnAccent {
			t.Errorf("сегмент %q не onAccent: %v", seg.text, got)
		}
	}

	// Текст из EWS — враждебный: управляющие символы вычищает Sanitize ещё до
	// раскладки/отрисовки (SECURITY.md §3); tview.Escape не нужен — tcell.
	dirty := agendaSegments(mail.Event{
		Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 30),
		Subject: "Синк\x1b[31m", Location: "Зал\x07",
	}, mail.NewLinkRegistry(), false, nov(t, 17, 9, 0))
	if text := agendaText(dirty); strings.ContainsAny(text, "\x1b\x07") {
		t.Errorf("управляющие символы дошли до повестки: %q", text)
	}
}

// TestAgendaWrap: блок события переносится по словам под ширину панели
// (V-42), длинное слово режется принудительно, текст не теряется.
func TestAgendaWrap(t *testing.T) {
	blocks := agendaSegments(mail.Event{
		Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30),
		Subject:  "Синк по релизу с командой платформы и поддержкой",
		Location: "Переговорная 3", MyResponseType: "Accept",
	}, mail.NewLinkRegistry(), false, nov(t, 17, 9, 0))
	full := strings.Join(strings.Fields(agendaText(blocks)), "")
	for _, width := range []int{20, 40, 80, 100, 220} {
		var got strings.Builder
		for _, segs := range blocks {
			for _, line := range wrapSegments(segs, width) {
				if n := len([]rune(segText(line))); n > width {
					t.Errorf("ширина %d: строка %d рун", width, n)
				}
				got.WriteString(segText(line))
				got.WriteByte(' ')
			}
		}
		if got := strings.Join(strings.Fields(got.String()), ""); got != full {
			t.Errorf("ширина %d: текст потерян:\n%q\nхочу %q", width, got, full)
		}
	}
	// Одно слово длиннее строки — режется, ни одна строка не шире лимита.
	long := []agendaSeg{{text: strings.Repeat("я", 45)}}
	for _, line := range wrapSegments(long, 20) {
		if n := len([]rune(segText(line))); n > 20 {
			t.Errorf("длинное слово не порезано: %d рун", n)
		}
	}
}

// fixAgendaNow закрепляет часы повестки (V-51): иначе стили прошедших встреч
// зависят от даты прогона тестов.
func fixAgendaNow(t *testing.T, now time.Time) {
	t.Helper()
	prev := agendaNow
	agendaNow = func() time.Time { return now }
	t.Cleanup(func() { agendaNow = prev })
}

// TestAgendaClick: клик выбирает событие под курсором и передаёт фокус;
// клик по зазору между событиями только фокусирует панель (V-51).
func TestAgendaClick(t *testing.T) {
	l := newAgendaList()
	l.SetRect(0, 0, 40, 10)
	items := [][][]agendaSeg{
		{{{text: "10:00–10:30 Синк"}}},
		{{{text: "12:00–13:00 Разбор"}}},
	}
	l.SetItems(items, 0)
	focused := false
	handler := l.MouseHandler()
	click := func(y int) {
		t.Helper()
		handled, _ := handler(tview.MouseLeftClick,
			tcell.NewEventMouse(1, y, tcell.Button1, tcell.ModNone),
			func(tview.Primitive) { focused = true })
		if !handled || !focused {
			t.Fatalf("клик не обработан или фокус не передан: %v/%v", handled, focused)
		}
	}
	// У событий теста по одной строке: событие 0 — строка 0, зазор — 1,
	// событие 1 — строка 2.
	click(1) // зазор
	if got := l.Selected(); got != 0 {
		t.Errorf("клик по зазору выбрал %d, хочу 0", got)
	}
	click(2)
	if got := l.Selected(); got != 1 {
		t.Errorf("после клика выбрано %d, хочу 1", got)
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

// TestNearestEvent: самое раннее событие с Start > now; прошедшие и Start ==
// now не берутся; пустой список — false (V-53).
func TestNearestEvent(t *testing.T) {
	at := nov(t, 17, 12, 0)
	events := []mail.Event{
		{Start: nov(t, 17, 9, 0), End: nov(t, 17, 9, 30), Subject: "Прошло"},
		{Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 30), Subject: "Позже"},
		{Start: at, End: at.Add(30 * time.Minute), Subject: "Сейчас"},
		{Start: nov(t, 17, 14, 0), End: nov(t, 17, 14, 30), Subject: "Скоро"},
	}
	ev, ok := NearestEvent(events, at)
	if !ok || ev.Subject != "Скоро" {
		t.Errorf("NearestEvent = %q/%v, хочу «Скоро»", ev.Subject, ok)
	}
	if _, ok := NearestEvent(events[:1], at); ok {
		t.Error("NearestEvent взял прошедшее событие")
	}
	if _, ok := NearestEvent(nil, at); ok {
		t.Error("NearestEvent пустого списка вернул событие")
	}
}

// TestNearestLine: «Ближайшее: 15:00–15:30 Тема» — подпись muted, время и тема
// primary; будущих нет — muted «Больше событий нет»; тема через Sanitize.
func TestNearestLine(t *testing.T) {
	now := nov(t, 17, 14, 30)
	segs := nearestLine([]mail.Event{
		{Start: nov(t, 17, 15, 0), End: nov(t, 17, 15, 30), Subject: "Digital\x1b[31m Corp. Daily"},
		{Start: nov(t, 17, 16, 0), End: nov(t, 17, 16, 30), Subject: "Позже"},
	}, now)
	text := segText(segs)
	if want := "Ближайшее: 15:00–15:30 Digital Corp. Daily"; text != want {
		t.Errorf("текст футера = %q, хочу %q", text, want)
	}
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("в футер прошёл ESC: %q", text)
	}
	for _, c := range []struct {
		part string
		want tcell.Color
	}{
		{"Ближайшее", colorTextSecondary},
		{"15:00", colorTextPrimary},
		{"Digital", colorTextPrimary},
	} {
		if got := segFg(t, segs, c.part); got != c.want {
			t.Errorf("%q: fg=%v, хочу %v", c.part, got, c.want)
		}
	}

	empty := nearestLine(nil, now)
	if got := segText(empty); got != "Больше событий нет" {
		t.Errorf("пустой футер = %q", got)
	}
	if got := segFg(t, empty, "Больше"); got != colorTextSecondary {
		t.Errorf("пустой футер: fg=%v, хочу muted", got)
	}
}

// TestCalendarNearestFooter (V-53): sim — под сеткой одна строка «Ближайшее:
// …», сетка отдаёт ей нижнюю строку; подпись muted, время primary; месяц без
// будущих событий — «Больше событий нет».
func TestCalendarNearestFooter(t *testing.T) {
	prev := agendaNow
	at := nov(t, 18, 14, 30)
	agendaNow = func() time.Time { return at }
	t.Cleanup(func() { agendaNow = prev })

	a := newTestApp(t)
	s := a.calendar
	s.year, s.month = 2026, time.November
	s.events = []mail.Event{
		{Start: nov(t, 18, 15, 0), End: nov(t, 18, 15, 30), Subject: "Digital Corp. Daily"},
	}
	s.render()
	a.pages.SwitchToPage(pageCalendar)

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	const w, h = 100, 30
	a.tapp.SetScreen(sim)
	sim.SetSize(w, h)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	a.tapp.QueueUpdateDraw(func() {})
	defer func() { a.tapp.Stop(); <-done }()

	x, y, _, _ := s.gridFooter.GetRect()
	if w <= 0 || y <= 0 {
		t.Fatalf("у футера нет места: x=%d y=%d", x, y)
	}
	// Сетка отдала футеру нижнюю строку: её рамка кончается строкой выше.
	if r, _, _, _ := sim.GetContent(x, y-1); r != '└' {
		t.Errorf("над футером не низ сетки: %q", r)
	}
	row := simRow(sim, y)
	if runeIndex(row, "Ближайшее: 15:00–15:30 Digital Corp. Daily") < 0 {
		t.Errorf("строки «Ближайшее» нет: %q", string(row))
	}
	same := func(c1, c2 tcell.Color) bool {
		r1, g1, b1 := c1.RGB()
		r2, g2, b2 := c2.RGB()
		return r1 == r2 && g1 == g2 && b1 == b2
	}
	styleAt := func(ix int) tcell.Style {
		_, _, st, _ := sim.GetContent(ix, y)
		return st
	}
	if fg, _, _ := styleAt(x + 1).Decompose(); !same(fg, colorTextSecondary) {
		t.Errorf("подпись футера: fg=%v, хочу muted", fg)
	}
	if fg, _, _ := styleAt(x + 1 + len([]rune("Ближайшее: "))).Decompose(); !same(fg, colorTextPrimary) {
		t.Errorf("время футера: fg=%v, хочу primary", fg)
	}

	// Месяц без будущих событий — muted-заглушка (пересчёт в render).
	past := mail.Event{Start: nov(t, 18, 8, 0), End: nov(t, 18, 8, 30), Subject: "Прошло"}
	a.tapp.QueueUpdateDraw(func() {
		s.events = []mail.Event{past}
		s.render()
	})
	row = simRow(sim, y)
	if runeIndex(row, "Больше событий нет") < 0 {
		t.Errorf("нет заглушки пустого футера: %q", string(row))
	}
	if fg, _, _ := styleAt(x + 1).Decompose(); !same(fg, colorTextSecondary) {
		t.Errorf("заглушка футера: fg=%v, хочу muted", fg)
	}
}

// TestOnDaySelectedAgenda: повестка дня собирается через agendaSegments —
// выбранный блок onAccent, остальные primary, маркеры и реестр ссылок видны
// в блоках, dayLinks наполняется для модала g; пустой день — без выбора,
// Enter без действия.
func TestOnDaySelectedAgenda(t *testing.T) {
	a := newTestApp(t)
	fixAgendaNow(t, nov(t, 17, 9, 0))
	s := a.calendar
	s.year, s.month = 2026, time.November
	s.events = []mail.Event{
		{Start: nov(t, 17, 10, 0), End: nov(t, 17, 10, 30), Subject: "Синк",
			Location: "https://meet.example/x", MyResponseType: "Accept"},
		{Start: nov(t, 17, 14, 0), End: nov(t, 17, 14, 30), Subject: "Разбор",
			MyResponseType: "Accept"},
	}
	grid := MonthGrid(2026, time.November, s.events)
	s.grid.SetCells(grid, MonthWeeks(2026, time.November), 0) // V-32: onDaySelected читает выбранную ячейку из сетки
	row, col := cellOf(grid, 17)
	s.onDaySelected(row, col)
	if got := len(s.agenda.items); got != 2 {
		t.Fatalf("событий повестки = %d, хочу 2", got)
	}
	if s.agenda.Selected() != 0 {
		t.Fatalf("выбрано %d, хочу первое событие", s.agenda.Selected())
	}
	first := agendaText(s.agenda.items[0])
	if !strings.Contains(first, "✓") || !strings.Contains(first, "[1] meet.example") {
		t.Errorf("выбранный блок без маркеров: %q", first)
	}
	if got := segFg(t, agendaSegs(s.agenda.items[0]), "Синк"); got != colorOnAccent {
		t.Errorf("выбранный блок не onAccent: %v", got)
	}
	if got := segFg(t, agendaSegs(s.agenda.items[1]), "Разбор"); got != colorTextPrimary {
		t.Errorf("невыбранный блок не primary: %v", got)
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

	// Пустой день — без событий и выбора («Встреч нет» рисует Draw), ссылок
	// нет, Enter ничего не открывает.
	row, col = cellOf(grid, 18)
	s.onDaySelected(row, col)
	if got := len(s.agenda.items); got != 0 {
		t.Fatalf("событий пустой повестки = %d, хочу 0", got)
	}
	if got := s.agenda.Selected(); got != -1 {
		t.Errorf("пустой день выбрал событие %d", got)
	}
	if s.dayLinks != nil {
		t.Errorf("пустой день: dayLinks = %v, хочу nil", s.dayLinks)
	}
	s.openEventDetails(s.agenda.Selected()) // Enter на пустом дне — без действия
}

// TestAgendaListStyles: sim — повестка в стиле писем (V-34/V-51): событие —
// две строки с зазором, выбранный блок залит на всю ширину акцентом, время
// будущей встречи primary, прошедшая — muted, полоса прокрутки — колонка
// справа за рамкой панели (30 встреч > видимых).
func TestAgendaListStyles(t *testing.T) {
	a := newTestApp(t)
	fixAgendaNow(t, nov(t, 22, 9, 1))
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
	// Раскладка V-51: событие — 2 строки (шапка, место), между событиями зазор.
	// Событие 0 — строки iy/iy+1, зазор iy+2, событие 1 — iy+3/iy+4, ...
	// Выбранный блок первой встречи: заливка на всю ширину (обе строки), текст
	// onAccent.
	if _, bg, _ := styleAt(ix, iy).Decompose(); !same(bg, colorSelectedBg) {
		t.Errorf("выбранная строка: bg=%v, хочу %v", bg, colorSelectedBg)
	}
	sawSel := false
	for x := ix; x < ix+iw; x++ {
		if r, _, st, _ := sim.GetContent(x, iy); r != ' ' {
			sawSel = true
			if fg, _, _ := st.Decompose(); !same(fg, colorSelectedFg) {
				t.Errorf("текст выбранного блока: fg=%v, хочу %v", fg, colorSelectedFg)
			}
		}
	}
	if !sawSel {
		t.Error("в выбранном блоке нет текста")
	}
	if _, bg, _ := styleAt(ix+iw-1, iy).Decompose(); !same(bg, colorSelectedBg) {
		t.Errorf("выбранная строка залита не целиком: правый край bg=%v", bg)
	}
	// Пустая строка места выбранного события остаётся в блоке (заливка accent).
	if _, bg, _ := styleAt(ix, iy+1).Decompose(); !same(bg, colorSelectedBg) {
		t.Errorf("пустая строка места не в блоке: bg=%v, хочу %v", bg, colorSelectedBg)
	}
	// Зазор: пустая строка на фоне панели, без текста.
	if _, bg, _ := styleAt(ix, iy+2).Decompose(); !same(bg, colorPanelBg) {
		t.Errorf("зазор: bg=%v, хочу %v", bg, colorPanelBg)
	}
	for x := ix; x < ix+iw; x++ {
		if r, _, _, _ := sim.GetContent(x, iy+2); r != ' ' {
			t.Errorf("в зазоре между событиями есть текст: %q", r)
			break
		}
	}
	// Прошедшая встреча (1, 8:30–9:00 при now=9:01) — всё muted.
	if fg, bg, _ := styleAt(ix+listPadLeft, iy+3).Decompose(); !same(fg, colorTextSecondary) || same(bg, colorSelectedBg) {
		t.Errorf("время прошедшей встречи: fg/bg=%v/%v, хочу %v", fg, bg, colorTextSecondary)
	}
	sawPast := false
	for x := ix; x < ix+iw; x++ {
		if r, _, st, _ := sim.GetContent(x, iy+3); r == 'В' {
			if fg, _, _ := st.Decompose(); !same(fg, colorTextSecondary) {
				t.Errorf("тема прошедшей встречи: fg=%v, хочу muted", fg)
			}
			sawPast = true
			break
		}
	}
	if !sawPast {
		t.Error("прошедшей встречи нет в повестке")
	}
	// Будущая встреча (2, 9:00–9:30) — время ярче (primary).
	if fg, _, _ := styleAt(ix+listPadLeft, iy+6).Decompose(); !same(fg, colorTextPrimary) {
		t.Errorf("время будущей встречи: fg=%v, хочу primary", fg)
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
