package ui

// Тесты чистых функций форматирования UI. Все функции прогоняют текст из EWS
// через security.Sanitize (SECURITY.md §3: граница UI) — это фиксируется здесь.

import (
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

// TestFormatMessageRow: колонки [ВНЕШН] | имя отправителя | тема | дата (UI-2).
func TestFormatMessageRow(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	recv := time.Date(2026, 11, 20, 10, 12, 44, 0, time.Local) // сегодня → HH:MM
	m := mail.MessageSummary{
		Subject:  "Отчёт Q3",
		Received: recv,
		From:     mail.Address{Name: "Иванов И.", Email: "ivanov@partner.tld"},
	}
	row := FormatMessageRow(m, orgDomains, now)
	if len(row) != 4 {
		t.Fatalf("колонок %d, хочу 4: %v", len(row), row)
	}
	if row[0] != "[ВНЕШН]" {
		t.Errorf("метка = %q", row[0])
	}
	if row[1] != "Иванов И." {
		t.Errorf("отправитель = %q, хочу имя из t:Name", row[1])
	}
	if row[2] != "Отчёт Q3" {
		t.Errorf("тема = %q", row[2])
	}
	if row[3] != "10:12" {
		t.Errorf("дата = %q, хочу компактную HH:MM для сегодняшнего", row[3])
	}
}

// TestFormatMessageRowFallback: без t:Name — локальная часть адреса (UI-2).
func TestFormatMessageRowFallback(t *testing.T) {
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
		row := FormatMessageRow(mail.MessageSummary{From: tc.from}, orgDomains, now)
		if row[1] != tc.want {
			t.Errorf("отправитель %v = %q, хочу %q", tc.from, row[1], tc.want)
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

// TestFormatMessageRowSanitized: тема с escape-инъекцией sanitized до tview.
func TestFormatMessageRowSanitized(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)
	m := mail.MessageSummary{
		Subject: "плохая\x1b]8;;http://evil\x1b\\тема",
		From:    mail.Address{Email: "a@b.c"},
	}
	row := FormatMessageRow(m, orgDomains, now)
	if row[2] != "плохаятема" {
		t.Errorf("тема не санитизирована: %q", row[2])
	}
}

// TestFormatFolder: имя папки + счётчик непрочитанных.
func TestFormatFolder(t *testing.T) {
	f := mail.Folder{DisplayName: "Входящие", UnreadCount: 23}
	if got := FormatFolder(f); got != "Входящие (23)" {
		t.Errorf("FormatFolder = %q", got)
	}
	f2 := mail.Folder{DisplayName: "Отправленные"}
	if got := FormatFolder(f2); got != "Отправленные" {
		t.Errorf("FormatFolder без непрочитанных = %q", got)
	}
}

// TestFormatMessageHeader: блок заголовков для панели просмотра.
func TestFormatMessageHeader(t *testing.T) {
	m := mail.Message{
		MessageSummary: mail.MessageSummary{
			Subject: "Отчёт Q3",
			From:    mail.Address{Name: "Иванов И.", Email: "ivanov@company.example"},
		},
		To: []mail.Address{{Email: "apetrova@company.example"}, {Email: "vsidorov@company.example"}},
	}
	h := FormatMessageHeader(m)
	for _, want := range []string{"От:", "Иванов И. <ivanov@company.example>", "Кому:", "apetrova@company.example", "vsidorov@company.example", "Тема:", "Отчёт Q3"} {
		if !contains(h, want) {
			t.Errorf("в заголовках нет %q:\n%s", want, h)
		}
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
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
