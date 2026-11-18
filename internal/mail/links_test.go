package mail

// Тесты реестра ссылок (ARCHITECTURE.md §4, PLAN.md этап 3): нумерация [N],
// показ реального хоста, ⚠ при тексте≠адресе, IDN/punycode-флаг.

import (
	"strings"
	"testing"
)

// TestRenderBodyPlainTextLinks: голые URL в тексте нумеруются, в тексте —
// [N] хост; реестр хранит полные URL.
func TestRenderBodyPlainTextLinks(t *testing.T) {
	reg := NewLinkRegistry()
	body := reg.RenderBody("Смотрите https://a.example/doc и http://b.example/x здесь")
	if !strings.Contains(body, "[1] a.example") || !strings.Contains(body, "[2] b.example") {
		t.Errorf("ссылки не пронумерованы: %q", body)
	}
	if strings.Contains(body, "https://a.example") {
		t.Errorf("полный URL остался в теле: %q", body)
	}
	list := reg.List()
	if len(list) != 2 || list[0].URL != "https://a.example/doc" || list[1].URL != "http://b.example/x" {
		t.Fatalf("реестр: %+v", list)
	}
}

// TestRenderBodyHTMLAnchorMismatch: якорь с текстом-обманкой — в теле текст
// и реальный хост с ⚠; флаг Mismatch в реестре.
func TestRenderBodyHTMLAnchorMismatch(t *testing.T) {
	reg := NewLinkRegistry()
	body := reg.RenderBodyHTML(`<p>Проверьте <a href="http://evil.tld/login">sberbank.ru</a> срочно</p>`)
	if !strings.Contains(body, "sberbank.ru") {
		t.Errorf("текст ссылки потерян: %q", body)
	}
	if !strings.Contains(body, "[1] evil.tld") {
		t.Errorf("реальный хост не показан: %q", body)
	}
	if !strings.Contains(body, "⚠") {
		t.Errorf("нет пометки ⚠ при тексте≠адресе: %q", body)
	}
	list := reg.List()
	if len(list) != 1 || !list[0].Mismatch || list[0].Text != "sberbank.ru" || list[0].Host != "evil.tld" {
		t.Fatalf("реестр: %+v", list)
	}
}

// TestRenderBodyHTMLAnchorHonest: текст-слово на честный хост — без ⚠.
func TestRenderBodyHTMLAnchorHonest(t *testing.T) {
	reg := NewLinkRegistry()
	body := reg.RenderBodyHTML(`<p><a href="https://company.example/doc">отчёт</a></p>`)
	if strings.Contains(body, "⚠") {
		t.Errorf("ложное ⚠: %q", body)
	}
	if !strings.Contains(body, "[1] company.example") {
		t.Errorf("хост не показан: %q", body)
	}
}

// TestRenderBodyIDN: punycode/IDN-хост — punycode-форма и флаг [IDN].
func TestRenderBodyIDN(t *testing.T) {
	reg := NewLinkRegistry()
	body := reg.RenderBodyHTML(`<a href="https://пример.рф/вход">банк</a>`)
	if !strings.Contains(body, "[IDN]") {
		t.Errorf("нет флага IDN: %q", body)
	}
	if !strings.Contains(body, "xn--") {
		t.Errorf("punycode-форма не показана: %q", body)
	}
	list := reg.List()
	if len(list) != 1 || !list[0].IDN {
		t.Fatalf("реестр: %+v", list)
	}
}

// TestRenderBodyNoLinks: текст без ссылок не меняется.
func TestRenderBodyNoLinks(t *testing.T) {
	reg := NewLinkRegistry()
	body := reg.RenderBody("Обычный текст без ссылок.")
	if body != "Обычный текст без ссылок." {
		t.Errorf("текст искажён: %q", body)
	}
	if len(reg.List()) != 0 {
		t.Errorf("реестр не пуст: %+v", reg.List())
	}
}

// TestRenderBodySanitized: escape-инъекция в тексте ссылки остаётся для
// санитайзера UI (реестр не глотает её молча).
func TestRenderBodySanitized(t *testing.T) {
	reg := NewLinkRegistry()
	html := "<a href=\"https://x.example/a\">текст\x1b]8;;http://evil\x1b\\инъекция</a>"
	body := reg.RenderBodyHTML(html)
	if !strings.ContainsRune(body, '\x1b') {
		t.Errorf("реестр проглотил escape-инъекцию — Sanitize на UI её не увидит: %q", body)
	}
}
