package ui

// Тесты чистых функций форматирования UI. Все функции прогоняют текст из EWS
// через security.Sanitize (SECURITY.md §3: граница UI) — это фиксируется здесь.

import (
	"strings"
	"testing"
	"time"

	"mailcat/internal/mail"
)

const orgDomain = "company.example"

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
		if got := ExternalMark(tc.email, orgDomain); got != tc.want {
			t.Errorf("ExternalMark(%q) = %q, хочу %q", tc.email, got, tc.want)
		}
	}
}

// TestFormatMessageRow: колонки [ВНЕШН] | реальный адрес | тема | дата.
func TestFormatMessageRow(t *testing.T) {
	recv := time.Date(2026, 11, 17, 10, 12, 44, 0, time.UTC)
	m := mail.MessageSummary{
		Subject:  "Отчёт Q3",
		Received: recv,
		From:     mail.Address{Name: "Иванов И.", Email: "ivanov@partner.tld"},
	}
	row := FormatMessageRow(m, orgDomain)
	if len(row) != 4 {
		t.Fatalf("колонок %d, хочу 4: %v", len(row), row)
	}
	if row[0] != "[ВНЕШН]" {
		t.Errorf("метка = %q", row[0])
	}
	if row[1] != "ivanov@partner.tld" {
		t.Errorf("отправитель = %q, хочу реальный адрес", row[1])
	}
	if row[2] != "Отчёт Q3" {
		t.Errorf("тема = %q", row[2])
	}
	if row[3] == "" || row[3] == recv.Format(time.RFC3339) {
		t.Errorf("дата = %q, хочу локальный короткий формат", row[3])
	}
}

// TestFormatMessageRowSanitized: тема с escape-инъекцией sanitized до tview.
func TestFormatMessageRowSanitized(t *testing.T) {
	m := mail.MessageSummary{
		Subject: "плохая\x1b]8;;http://evil\x1b\\тема",
		From:    mail.Address{Email: "a@b.c"},
	}
	row := FormatMessageRow(m, orgDomain)
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
