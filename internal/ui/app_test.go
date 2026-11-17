package ui

// Тесты санитизации на границе UI по итогам security-ревью этапа 2:
// 1) текст ошибок EWS (EWSError.MessageText/faultstring) — серверный,
//    до терминала только через Sanitize;
// 2) строки для tview.Table/List — tview.Escape (теги [red]/["region"]
//    интерпретируются этими виджетами безусловно).

import (
	"errors"
	"strings"
	"testing"

	"mailcat/internal/config"
	"mailcat/internal/mail"
)

// newTestApp — App без запуска цикла событий (сеть не трогается).
func newTestApp(t *testing.T) *App {
	t.Helper()
	return New(config.Defaults(), nil)
}

// TestShowErrorSanitized: OSC-инъекция в тексте ошибки не доходит до статус-бара.
func TestShowErrorSanitized(t *testing.T) {
	a := newTestApp(t)
	a.showError(errors.New("EWS ErrorInternalServerError: \x1b]52;c;aGVsbG8=\x07упс \x1b]8;;http://evil\x1b\\ссылка"))
	text := a.status.GetText(false)
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("в статус-бар прошёл ESC: %q", text)
	}
}

// TestShowErrorRunesTruncation: усечение длинного текста не рвёт русские руны.
func TestShowErrorRunesTruncation(t *testing.T) {
	a := newTestApp(t)
	long := strings.Repeat("ж", 500)
	a.showError(errors.New(long))
	text := a.status.GetText(false)
	runes := []rune(text)
	// Без усечения было бы ~530 рун; с усечением — ~230, и все целые.
	if len(runes) > 250 {
		t.Errorf("длинная ошибка не усечена: %d рун", len(runes))
	}
	for _, r := range runes {
		if r == 0xFFFD {
			t.Errorf("усечение разорвало руну: %q", text)
			break
		}
	}
	if !strings.Contains(text, "ж") {
		t.Errorf("текст ошибки потерян: %q", text)
	}
}

// TestFormatMessageRowEscaped: теги tview в теме/адресе экранируются
// (Table интерпретирует [red] даже без SetDynamicColors; tview.Escape
// вставляет "[]" внутрь тега — семантика tview).
func TestFormatMessageRowEscaped(t *testing.T) {
	m := mail.MessageSummary{
		Subject: "[red]СРОЧНО[-] проверка",
		From:    mail.Address{Email: `["evil"]@x.example`},
	}
	row := FormatMessageRow(m, "company.example")
	if !strings.Contains(row[2], "[red[]") {
		t.Errorf("тег [red] в теме не экранирован: %q", row[2])
	}
	if !strings.Contains(row[1], `["evil"[]`) {
		t.Errorf("region-тег в адресе не экранирован: %q", row[1])
	}
}

// TestFormatFolderEscaped: имя папки с тегом экранируется для tview.List.
// Кириллица в скобках ([ВХОД]) тегом не является — проходит как есть.
func TestFormatFolderEscaped(t *testing.T) {
	got := FormatFolder(mail.Folder{DisplayName: `["x"]Входящие`})
	if !strings.Contains(got, `["x"[]`) {
		t.Errorf("имя папки не экранировано: %q", got)
	}
	if got2 := FormatFolder(mail.Folder{DisplayName: `[ВХОД]ящик`}); got2 != `[ВХОД]ящик` {
		t.Errorf("безопасные скобки искажены: %q", got2)
	}
}
