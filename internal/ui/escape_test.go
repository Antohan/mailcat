package ui

// Тесты экранирования строк для tview.List/tview.Modal по итогам
// security-ревью этапа 3: List И Modal интерпретируют style-теги безусловно —
// URL/имя вложения с [green]-фрагментом отображались бы искажённо
// («показанный URL ≠ открываемому»).

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rivo/tview"

	"mailcat/internal/mail"
)

// TestEscapeForList: Sanitize + tview.Escape в одном шаге для List/Modal.
func TestEscapeForList(t *testing.T) {
	raw := "http://evil.tld/a?x=[green]скрыто[-]"
	got := escapeForList(raw)
	if got == raw {
		t.Fatalf("строка с тегами не изменена: %q", got)
	}
	// tview.Escape вставляет "[]" внутрь тега — рендер покажет литеральный [green].
	if !strings.Contains(got, "[green[]") {
		t.Errorf("тег [green] не экранирован: %q", got)
	}
	// Управляющие символы вырезаются санитайзером.
	if strings.ContainsRune(escapeForList("a\x1b[31mb"), '\x1b') {
		t.Error("ESC прошёл через escapeForList")
	}
}

// TestLinkModalUsesEscape: главный текст пункта модала ссылок строится
// через escapeForList (показанный URL = открываемому).
func TestLinkModalUsesEscape(t *testing.T) {
	l := mail.LinkInfo{N: 1, Host: "evil.tld", URL: "http://evil.tld/a?x=[green]"}
	main, secondary := linkListEntry(l)
	if !strings.Contains(main, "evil.tld") {
		t.Errorf("хост потерян: %q", main)
	}
	if !strings.Contains(secondary, "[green[]") {
		t.Errorf("URL в secondary не экранирован: %q", secondary)
	}
}

// TestAttachmentModalUsesEscape: имя вложения в списке и подтверждении —
// через escapeForList.
func TestAttachmentModalUsesEscape(t *testing.T) {
	a := mail.Attachment{Name: "[red]отчёт.pdf", ContentLength: 5}
	main, _ := attachmentListEntry(a)
	if !strings.Contains(main, "[red[]") {
		t.Errorf("имя вложения не экранировано: %q", main)
	}
	confirm := dangerousConfirmText("[red]отчёт.pdf")
	if !strings.Contains(confirm, "[red[]") {
		t.Errorf("имя в подтверждении не экранировано: %q", confirm)
	}
	if strings.Contains(confirm, "\x1b") {
		t.Error("ESC в тексте подтверждения")
	}
}

// Проверка поведения tview.Escape напрямую (регрессия семантики).
func TestTviewEscapeSemantics(t *testing.T) {
	if got := tview.Escape("[green]"); got == "[green]" {
		t.Errorf("tview.Escape не экранирует [green]: %q", got)
	}
}

// TestFormatMessageListItemTruncateBeforeEscape (C-4): тема усекается до
// tview.Escape. Обрезка escaped-строки рвала тег — на экране одиночная
// скобка и ширина больше maxW.
func TestFormatMessageListItemTruncateBeforeEscape(t *testing.T) {
	m := mail.MessageSummary{Subject: "[red]" + strings.Repeat("длинная тема ", 10)}
	const maxW = 16 // subjectIndent (10) + "[red]" (5) + «…»
	row := FormatMessageListItem(m, orgDomains, time.Now(), false, maxW)
	shown := tview.Unescape(row[2])
	if n := utf8.RuneCountInString(shown); n > maxW {
		t.Errorf("ширина %d > %d: %q", n, maxW, shown)
	}
	if strings.Count(shown, "[") != strings.Count(shown, "]") {
		t.Errorf("одиночные скобки после усечения: %q", shown)
	}
	if !strings.HasSuffix(shown, "…") {
		t.Errorf("нет многоточия: %q", shown)
	}
}
