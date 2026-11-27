package mail

// Реестр ссылок (ARCHITECTURE.md §4): при рендере тела письма URL извлекаются
// в реестр [1..N]; в теле ссылка показывается как «текст [N] host» (для
// текста≠адреса — с ⚠); IDN-хосты — punycode-формой и флагом [IDN]. Открытие —
// только явным действием из модала ссылок (этап 3, SECURITY.md §4.2).

import (
	"fmt"
	"regexp"
	"strings"

	"mailcat/internal/security"
)

// LinkInfo — запись реестра ссылок одного письма.
type LinkInfo struct {
	N        int    // номер [N] в теле
	URL      string // полный URL (для модала g)
	Text     string // текст якоря (может быть пуст для голой ссылки)
	Host     string // отображаемый хост (punycode-форма для IDN)
	Mismatch bool   // текст ссылки претворяется другим адресом (⚠)
	IDN      bool   // xn--/не-ASCII хост ([IDN])
}

// LinkRegistry собирает ссылки одного письма; живёт в рамках его рендера.
type LinkRegistry struct {
	links []LinkInfo
}

// NewLinkRegistry — пустой реестр ссылок одного письма.
func NewLinkRegistry() *LinkRegistry { return &LinkRegistry{} }

// List возвращает зарегистрированные ссылки по порядку.
func (r *LinkRegistry) List() []LinkInfo { return r.links }

// WHY: `<?`/`>?` поглощают скобки автолинка <https://…> в матч — иначе после
// замены URL маркером остаются огрызки «<[1] host>» (баг V-10).
// Не трогать: без них в RenderBody делали бы Trim по краям матча.
var reBareURL = regexp.MustCompile(`<?https?://[^\s<>"'\x60\x7f]+>?`)

// reAnchor — грубый разбор <a href="...">...</a> (тела писем не являются
// валидным XML — парсеры encoding/xml непригодны). Значение — в двойных или
// одинарных кавычках, либо без кавычек: почтовые клиенты шлют и такое (T-1).
// WHY: href обязан следовать за пробелом — иначе в <a data-href="трекер"
// href="настоящий"> совпал бы data-href и в реестр попал бы URL трекера.
var reAnchor = regexp.MustCompile(`(?is)<a\s(?:[^>]*\s)?href=(?:"([^"]*)"|'([^']*)'|([^\s>]+))[^>]*>(.*?)</a>`)

// register добавляет ссылку и возвращает маркер для вставки в текст.
func (r *LinkRegistry) register(rawURL, text string) string {
	facts := security.HostInfo(rawURL)
	info := LinkInfo{
		N:        len(r.links) + 1,
		URL:      rawURL,
		Text:     text,
		Host:     facts.ASCII,
		Mismatch: security.LinkMismatch(text, rawURL),
		IDN:      facts.IDN,
	}
	r.links = append(r.links, info)

	label := fmt.Sprintf("[%d] %s", info.N, info.Host)
	if info.IDN {
		label += " [IDN]"
	}
	if info.Mismatch {
		label += " ⚠"
	}
	return label
}

// RenderBody нумерует голые URL в текстовом теле; возвращает текст с [N] host.
func (r *LinkRegistry) RenderBody(text string) string {
	if text == "" {
		return ""
	}
	return reBareURL.ReplaceAllStringFunc(text, func(url string) string {
		// Скобки автолинка уже в матче (reBareURL) — срезаем их с URL.
		return r.register(strings.TrimRight(strings.Trim(url, "<>"), ".,);!?"), "")
	})
}

// RenderBodyHTML рендерит HTML-тело: якоря — «текст [N] host», остальное —
// через HTMLToText с нумерацией голых URL. Реестр при этом наполняется.
func (r *LinkRegistry) RenderBodyHTML(html string) string {
	if html == "" {
		return ""
	}
	text := reAnchor.ReplaceAllStringFunc(html, func(m string) string {
		groups := reAnchor.FindStringSubmatch(m)
		if len(groups) < 5 {
			return m
		}
		// Ровно одна альтернатива кавычек участвует в матче — конкатенация
		// даёт значение href без разбора, какая именно сработала.
		rawURL, anchorText := groups[1]+groups[2]+groups[3], stripTags(groups[4])
		if rawURL == "" {
			return m // href="" — не ссылка: разметку вычистит html2text
		}
		label := r.register(rawURL, anchorText)
		if anchorText != "" {
			return anchorText + " " + label
		}
		return label
	})
	return r.RenderBody(HTMLToText(text))
}

// stripTags убирает разметку из текста якоря (может содержать <b> и т.п.).
var reTag = regexp.MustCompile(`(?s)<[^>]*>`)

func stripTags(s string) string {
	return strings.TrimSpace(reTag.ReplaceAllString(s, ""))
}
