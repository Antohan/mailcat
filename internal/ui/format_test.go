package ui

// Тесты чистых функций форматирования UI. Все функции прогоняют текст из EWS
// через security.Sanitize (SECURITY.md §3: граница UI) — это фиксируется здесь.

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rivo/tview"
	"mailcat/internal/mail"
)

var orgDomains = []string{"company.example"}

// TestExternalMark: метка [ВНЕШН] только для чужих доменов (без учёта регистра).
func TestExternalMark(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"ivanov@company.example", ""},
		{"IVANOV@COMPANY.EXAMPLE", ""},
		{"ivanov@sub.company.example", ""},
		{"evil@company.example.evil.tld", "[ВНЕШН]"},
		{"petrov@partner.tld", "[ВНЕШН]"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := ExternalMark(tc.email, orgDomains); got != tc.want {
			t.Errorf("ExternalMark(%q) = %q, хочу %q", tc.email, got, tc.want)
		}
	}
}

// TestFormatMessageListItem: [строка 1 = ● + [ВНЕШН] + отправитель, время,
// строка 2 = тема с отступом под отправителя] (V-1).
func TestFormatMessageListItem(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	recv := time.Date(2026, 11, 20, 10, 12, 44, 0, time.Local) // сегодня → HH:MM
	m := mail.MessageSummary{
		Subject:  "Отчёт Q3",
		Received: recv,
		From:     mail.Address{Name: "Иванов И.", Email: "ivanov@partner.tld"},
	}
	row := FormatMessageListItem(m, orgDomains, now)
	if len(row) != 3 {
		t.Fatalf("частей %d, хочу 3: %v", len(row), row)
	}
	if row[0] != "● [ВНЕШН] Иванов И." {
		t.Errorf("строка 1 = %q", row[0])
	}
	if row[1] != "10:12" {
		t.Errorf("время = %q, хочу компактное HH:MM для сегодняшнего", row[1])
	}
	if !strings.HasPrefix(row[2], subjectIndent) || !strings.HasSuffix(row[2], "Отчёт Q3") {
		t.Errorf("строка 2 = %q, хочу отступ + тему", row[2])
	}
}

// TestFormatMessageListItemFallback: без t:Name — локальная часть адреса (UI-2).
func TestFormatMessageListItemFallback(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	cases := []struct {
		from mail.Address
		want string
	}{
		{mail.Address{Email: "youtrack@company.example"}, "youtrack"},
		{mail.Address{Email: "bot@partner.tld"}, "bot"},
		{mail.Address{Email: "no-at-sign"}, "no-at-sign"},
		{mail.Address{Email: "@host"}, "@host"}, // пустая локальная часть — адрес целиком
	}
	for _, tc := range cases {
		row := FormatMessageListItem(mail.MessageSummary{From: tc.from}, orgDomains, now)
		if !strings.HasSuffix(row[0], tc.want) {
			t.Errorf("отправитель %v = %q, хочу суффикс %q", tc.from, row[0], tc.want)
		}
	}
}

// TestFormatShortDate: сегодня → HH:MM; ≤7 дней → день недели; старше → ДД.ММ.
func TestFormatShortDate(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local) // пятница
	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"сегодня", time.Date(2026, 11, 20, 9, 5, 0, 0, time.Local), "09:05"},
		{"вчера (четверг)", time.Date(2026, 11, 19, 18, 0, 0, 0, time.Local), "Чт"},
		{"6 дней назад (суббота)", time.Date(2026, 11, 14, 0, 0, 0, 0, time.Local), "Сб"},
		{"ровно 7 дней — уже ДД.ММ", time.Date(2026, 11, 13, 12, 0, 0, 0, time.Local), "13.11"},
		{"давно", time.Date(2026, 10, 2, 12, 0, 0, 0, time.Local), "02.10"},
		{"прошлый год", time.Date(2025, 12, 31, 12, 0, 0, 0, time.Local), "31.12"},
	}
	for _, tc := range cases {
		if got := FormatShortDate(now, tc.t); got != tc.want {
			t.Errorf("%s: FormatShortDate = %q, хочу %q", tc.name, got, tc.want)
		}
	}
	if got := FormatShortDate(now, time.Time{}); got != "" {
		t.Errorf("нулевая дата = %q, хочу пусто", got)
	}
}

// TestFormatMessageListItemSanitized: тема с escape-инъекцией sanitized до tview.
func TestFormatMessageListItemSanitized(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	m := mail.MessageSummary{
		Subject: "плохая\x1b]8;;http://evil\x1b\\тема",
		From:    mail.Address{Email: "a@b.c"},
	}
	row := FormatMessageListItem(m, orgDomains, now)
	if !strings.HasSuffix(row[2], "плохаятема") {
		t.Errorf("тема не санитизирована: %q", row[2])
	}
}

// TestTruncateEllipsis: обрезка с «…» ровно в max рун, короткая строка —
// без изменений (V-1: тема строки 2).
func TestTruncateEllipsis(t *testing.T) {
	if got := truncateEllipsis("тема", 10); got != "тема" {
		t.Errorf("короткая строка изменена: %q", got)
	}
	if got, want := truncateEllipsis("очень длинная тема письма", 12), "очень длинн…"; got != want {
		t.Errorf("truncateEllipsis = %q, хочу %q", got, want)
	}
}

// TestPluralUnread (V-2): русская морфология счётчика в заголовке списка.
func TestPluralUnread(t *testing.T) {
	cases := map[int]string{
		1: "1 непрочитанное", 2: "2 непрочитанных", 5: "5 непрочитанных",
		11: "11 непрочитанных", 14: "14 непрочитанных", 21: "21 непрочитанное",
		24: "24 непрочитанных", 100: "100 непрочитанных", 111: "111 непрочитанных",
	}
	for n, want := range cases {
		if got := pluralUnread(n); got != want {
			t.Errorf("pluralUnread(%d) = %q, хочу %q", n, got, want)
		}
	}
}

// TestFormatFolder: единый отступ без иконок (П.2а), выравнивание счётчика
// на колонку folderNameCol, счётчик — цветовой тег textSecondary (RD-3).
// Тема в тестах переключаема, поэтому проверяем структуру, а не hex цвета.
func TestFormatFolder(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	got := FormatFolder(f, false, false)
	if !strings.HasPrefix(got, "  Входящие") {
		t.Errorf("нет отступа перед именем: %q", got)
	}
	if !strings.Contains(got, "[#") || !strings.HasSuffix(got, "](23)") {
		t.Errorf("счётчик без цвета textSecondary/не прижат: %q", got)
	}
	if width := utf8.RuneCountInString("  Входящие"); !strings.Contains(got, strings.Repeat(" ", folderNameCol-width)) {
		t.Errorf("счётчик не выровнен на колонку %d: %q", folderNameCol, got)
	}
	f2 := mail.Folder{DisplayName: "Отправленные"}
	if got2 := FormatFolder(f2, false, false); !strings.HasPrefix(got2, "  Отправленные") || strings.Contains(got2, "(0)") {
		t.Errorf("FormatFolder без непрочитанных = %q", got2)
	}
}

// TestFormatFolderSelected: строка под курсором — счётчик colorSelectedFg,
// чтобы читать его на фоне выделения (П.2б).
func TestFormatFolderSelected(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	got := FormatFolder(f, false, true)
	if !strings.Contains(got, fmt.Sprintf("[%s](23)", hexTag(colorSelectedFg))) {
		t.Errorf("счётчик на выделенной строке не colorSelectedFg: %q", got)
	}
	if got2 := FormatFolder(f, false, false); strings.Contains(got2, hexTag(colorSelectedFg)) {
		t.Errorf("невыделенная строка красится colorSelectedFg: %q", got2)
	}
}

// TestFormatFolderActive: открытая папка — текст accent + жирный (RD-3);
// счётчик остаётся вторичным, акцентируется только имя.
func TestFormatFolderActive(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	got := FormatFolder(f, true, false)
	if !strings.Contains(got, "::b]") {
		t.Errorf("активная папка не жирная: %q", got)
	}
	if !strings.Contains(got, fmt.Sprintf("[%s::b]", hexTag(colorAccent))) {
		t.Errorf("активная папка не accent: %q", got)
	}
	if got := FormatFolder(f, false, false); strings.Contains(got, "::b]") {
		t.Errorf("неактивная папка жирная: %q", got)
	}
}

// TestFormatMessageHeader: блок заголовков превью-панели (V-5: детали письма
// перешли на FormatMessageCard, маркер 📎 заменён секцией «Вложения N»).
func TestFormatMessageHeader(t *testing.T) {
	m := mail.Message{
		MessageSummary: mail.MessageSummary{
			Subject: "Отчёт Q3",
			From:    mail.Address{Name: "Иванов И.", Email: "ivanov@company.example"},
		},
		To:          []mail.Address{{Email: "apetrova@company.example"}, {Email: "vsidorov@company.example"}},
		Attachments: []mail.Attachment{{Name: "report.pdf"}, {Name: "данные.csv"}},
	}
	h := FormatMessageHeader(m)
	for _, want := range []string{"От:", "Иванов И. <ivanov@company.example>", "Кому:", "apetrova@company.example", "vsidorov@company.example", "Тема:", "Отчёт Q3"} {
		if !contains(h, want) {
			t.Errorf("в заголовках нет %q:\n%s", want, h)
		}
	}
	// П.2а (без иконок): маркера 📎 нет ни с вложениями, ни без.
	if contains(h, "📎") {
		t.Errorf("иконка 📎 в заголовках:\n%s", h)
	}
	// RD-5: порядок Тема → От; тема жирная; линия-разделитель присутствует.
	if i, j := strings.Index(h, "Тема:"), strings.Index(h, "От:"); i > j {
		t.Errorf("Тема должна идти перед От:\n%s", h)
	}
	if !contains(h, "[::b]Тема:") {
		t.Errorf("тема не жирная:\n%s", h)
	}
	if !contains(h, "──") {
		t.Errorf("нет линии-разделителя:\n%s", h)
	}
	// Тема-обманка с тегом не рендерится как тег (Sanitize + tview.Escape).
	evil := FormatMessageHeader(mail.Message{
		MessageSummary: mail.MessageSummary{Subject: "[red]скрыто", From: mail.Address{Email: "a@b.c"}},
	})
	if !contains(evil, "[red[]") {
		t.Errorf("тег из темы не экранирован:\n%s", evil)
	}
}

// TestFormatMessageCard: шапка страницы деталей (V-5, референс П.1) —
// аватар-инициалы, «Имя <email>», дата справа (паддинг до width; ширина
// печатных клеток меряет tview.TaggedStringWidth), «Кому: …», линия во всю
// ширину колонки, тема жирным. Узкая колонка усекает имя с «…»
// (truncateEllipsis, инвариант V-1).
func TestFormatMessageCard(t *testing.T) {
	m := mail.Message{
		MessageSummary: mail.MessageSummary{
			Subject:  "Отчёт Q3",
			From:     mail.Address{Name: "Иванов Иван Иванович", Email: "ivanov@company.example"},
			Received: time.Date(2026, 11, 21, 15, 4, 0, 0, time.Local),
		},
		To: []mail.Address{{Email: "apetrova@company.example"}, {Email: "vsidorov@company.example"}},
	}
	card := FormatMessageCard(m, 90)
	lines := strings.Split(card, "\n")
	if len(lines) != 4 {
		t.Fatalf("хочу 4 строки (аватар, кому, линия, тема):\n%s", card)
	}
	avatar := "[" + hexTag(colorOnAccent) + ":" + hexTag(colorAccent) + ":b]"
	if !contains(lines[0], avatar) || !contains(lines[0], " ИИ ") {
		t.Errorf("нет аватара-инициалов:\n%q", lines[0])
	}
	if !contains(lines[0], "Иванов Иван Иванович <ivanov@company.example>") {
		t.Errorf("нет «Имя <email>»:\n%q", lines[0])
	}
	if !contains(lines[0], "21.11.2026 15:04") {
		t.Errorf("нет даты:\n%q", lines[0])
	}
	if w := tview.TaggedStringWidth(lines[0]); w != 90 {
		t.Errorf("строка 1 = %d клеток, хочу 90 (дата справа):\n%q", w, lines[0])
	}
	if !contains(lines[1], "Кому:") || !contains(lines[1], "vsidorov@company.example") {
		t.Errorf("нет «Кому»:\n%q", lines[1])
	}
	if c := strings.Count(lines[2], "─"); c != 90 {
		t.Errorf("линия = %d клеток, хочу 90:\n%q", c, lines[2])
	}
	if i, j := strings.Index(card, "─"), strings.Index(card, "[::b]Отчёт Q3"); i > j {
		t.Errorf("тема должна идти после линии:\n%s", card)
	}

	// Узкая колонка: имя усечено с «…», дата всё равно справа до width.
	narrow := strings.Split(FormatMessageCard(m, 58), "\n")
	if w := tview.TaggedStringWidth(narrow[0]); w != 58 {
		t.Errorf("узкая строка 1 = %d клеток, хочу 58:\n%q", w, narrow[0])
	}
	if !contains(narrow[0], "…") {
		t.Errorf("имя не усечено на узкой колонке:\n%q", narrow[0])
	}

	// Без Кому и даты — строки опускаются (остаются аватар, линия, тема).
	bare := FormatMessageCard(mail.Message{
		MessageSummary: mail.MessageSummary{Subject: "Пусто", From: mail.Address{Email: "a@b.c"}},
	}, 90)
	if got := strings.Count(bare, "\n"); got != 2 {
		t.Errorf("без Кому/даты хочу 3 строки, строк-1 = %d:\n%s", got, bare)
	}

	// Тема/имя-обманки с тегами не рендерятся как теги (Sanitize + Escape).
	evil := FormatMessageCard(mail.Message{
		MessageSummary: mail.MessageSummary{
			Subject: "[red]скрыто",
			From:    mail.Address{Name: "[blue]Зло", Email: "x@y.z"},
		},
	}, 90)
	if !contains(evil, "[red[]") || !contains(evil, "[blue[]") {
		t.Errorf("теги из EWS не экранированы:\n%s", evil)
	}
}

// TestInitials: инициалы аватара (V-5) — до двух первых букв слов имени,
// заглавные; пустое имя — буква из локальной части адреса; нечего брать — «?».
func TestInitials(t *testing.T) {
	cases := []struct{ name, email, want string }{
		{"Иванов Иван", "", "ИИ"},
		{"Иванов", "", "И"},
		{"", "ivanov@corp.example", "I"},
		{"мария попова", "", "МП"},
		{"O'Hara Pty Ltd", "", "OP"},
		{"", "", "?"},
	}
	for _, c := range cases {
		if got := initials(c.name, c.email); got != c.want {
			t.Errorf("initials(%q, %q) = %q, хочу %q", c.name, c.email, got, c.want)
		}
	}
}

// TestFormatAttachments: имя + человекочитаемый размер; V-5 — секция
// «Вложения N» с линией-разделителем.
func TestFormatAttachments(t *testing.T) {
	list := []mail.Attachment{
		{Name: "report.pdf", ContentLength: 2097152},
		{Name: "данные.csv", ContentLength: 1024},
	}
	got := FormatAttachments(list)
	if !contains(got, "Вложения 2") || !contains(got, "──") {
		t.Errorf("нет секции «Вложения N» с линией:\n%s", got)
	}
	if !contains(got, "report.pdf (2.0 МБ)") || !contains(got, "данные.csv (1.0 КБ)") {
		t.Errorf("FormatAttachments = %q", got)
	}
	if got := FormatAttachments(nil); got != "" {
		t.Errorf("пустой список = %q, хочу пусто", got)
	}
	// Сторож (RD-5): имя вложения с тегом не должно пройти в dynamic-colors
	// TextView сырым — только в экранированном виде.
	evil := FormatAttachments([]mail.Attachment{{Name: "[red]x.pdf"}})
	if !contains(evil, "[red[]") || contains(evil, "\x1b") {
		t.Errorf("имя вложения не экранировано: %q", evil)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestDayBucket(t *testing.T) {
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.Local)
	cases := []struct {
		t    time.Time
		want int
	}{
		{time.Date(2026, 11, 21, 0, 1, 0, 0, time.Local), 0},
		{time.Date(2026, 11, 21, 23, 59, 0, 0, time.Local), 0},
		{time.Date(2026, 11, 20, 23, 59, 0, 0, time.Local), 1},
		{time.Date(2026, 11, 19, 12, 0, 0, 0, time.Local), 2},
		{time.Time{}, 2}, // Received неизвестен
	}
	for _, c := range cases {
		if got := dayBucket(now, c.t); got != c.want {
			t.Fatalf("dayBucket(%v) = %d, want %d", c.t, got, c.want)
		}
	}
	if dayBuckets[0] != "Сегодня" || dayBuckets[1] != "Вчера" || dayBuckets[2] != "Ранее" {
		t.Fatalf("dayBuckets = %v", dayBuckets)
	}
}
