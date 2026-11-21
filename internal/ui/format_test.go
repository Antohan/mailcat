package ui

// Тесты чистых функций форматирования UI. Все функции прогоняют текст из EWS
// через security.Sanitize (SECURITY.md §3: граница UI) — это фиксируется здесь.

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

// TestFormatFolder: иконка стандартной папки, выравнивание счётчика на
// колонку folderNameCol, счётчик — цветовой тег textSecondary (RD-3).
// Тема в тестах переключаема, поэтому проверяем структуру, а не hex цвета.
func TestFormatFolder(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	got := FormatFolder(f, false)
	if !strings.HasPrefix(got, "📥 Входящие") {
		t.Errorf("нет иконки входящих: %q", got)
	}
	if !strings.Contains(got, "[#") || !strings.HasSuffix(got, "](23)") {
		t.Errorf("счётчик без цвета textSecondary/не прижат: %q", got)
	}
	if width := displayWidth("📥 Входящие"); !strings.Contains(got, strings.Repeat(" ", folderNameCol-width)) {
		t.Errorf("счётчик не выровнен на колонку %d: %q", folderNameCol, got)
	}
	f2 := mail.Folder{DisplayName: "Отправленные"}
	if got2 := FormatFolder(f2, false); !strings.HasPrefix(got2, "📤 Отправленные") || strings.Contains(got2, "(0)") {
		t.Errorf("FormatFolder без непрочитанных = %q", got2)
	}
	// Нестандартная папка: отступ вместо иконки, тегов в имени нет.
	f3 := mail.Folder{DisplayName: "Архив"}
	if got3 := FormatFolder(f3, false); !strings.HasPrefix(got3, "   Архив") {
		t.Errorf("нет отступа у нестандартной папки: %q", got3)
	}
}

// TestFormatFolderActive: открытая папка — текст accent + жирный (RD-3);
// счётчик остаётся вторичным, акцентируется только имя.
func TestFormatFolderActive(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	got := FormatFolder(f, true)
	if !strings.Contains(got, "::b]") {
		t.Errorf("активная папка не жирная: %q", got)
	}
	if !strings.Contains(got, fmt.Sprintf("[%s::b]", hexTag(colorAccent))) {
		t.Errorf("активная папка не accent: %q", got)
	}
	if got := FormatFolder(f, false); strings.Contains(got, "::b]") {
		t.Errorf("неактивная папка жирная: %q", got)
	}
}

// TestFormatMessageHeader: блок заголовков для панели просмотра;
// UI-3 — 📎 N при вложениях (метаданные GetItem), без вложений маркера нет.
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
	for _, want := range []string{"От:", "Иванов И. <ivanov@company.example>", "Кому:", "apetrova@company.example", "vsidorov@company.example", "Тема:", "Отчёт Q3", "📎 2"} {
		if !contains(h, want) {
			t.Errorf("в заголовках нет %q:\n%s", want, h)
		}
	}
	bare := FormatMessageHeader(mail.Message{
		MessageSummary: mail.MessageSummary{Subject: "Пусто", From: mail.Address{Email: "a@b.c"}},
	})
	if contains(bare, "📎") {
		t.Errorf("маркер 📎 без вложений:\n%s", bare)
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

// TestFormatAttachments: имя + человекочитаемый размер.
func TestFormatAttachments(t *testing.T) {
	list := []mail.Attachment{
		{Name: "report.pdf", ContentLength: 2097152},
		{Name: "данные.csv", ContentLength: 1024},
	}
	got := FormatAttachments(list)
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
