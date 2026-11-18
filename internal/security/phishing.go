// Анти-фишинг (docs/SECURITY.md §4): IDN/punycode-хосты, опасные расширения
// вложений, расхождение «текст ссылки ≠ адрес». Всё — чистые функции без сети.

package security

import (
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// HostFacts — сведения о хосте ссылки для реестра (PLAN.md этап 3):
// IDN=true для punycode (xn--) и не-ASCII хостов; Ascii — punycode-форма
// (для обычных хостов — сам хост в нижнем регистре).
type HostFacts struct {
	Host  string // исходный хост
	IDN   bool   // требуется внимание: xn-- или не-ASCII
	Ascii string // punycode-форма (показывается пользователю)
}

// HostInfoOf анализирует хост ссылки. Ошибки idna не критичны: ASCII-форма
// тогда остаётся исходным хостом.
func HostInfoOf(host string) HostFacts {
	info := HostFacts{Host: host, Ascii: strings.ToLower(host)}
	if host == "" {
		return info
	}
	if strings.Contains(strings.ToLower(host), "xn--") || !isASCII(host) {
		info.IDN = true
		if ascii, err := idna.ToASCII(host); err == nil {
			info.Ascii = ascii
		}
	}
	return info
}

// HostInfo — универсальный вход для полной ссылки: разбирает и анализирует хост.
func HostInfo(rawURL string) HostFacts {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return HostInfoOf("")
	}
	return HostInfoOf(u.Hostname())
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7e {
			return false
		}
	}
	return true
}

// dangerousExtensions — PLAN.md этап 3 (расширения, требующие предупреждения
// перед открытием вложения).
var dangerousExtensions = map[string]bool{
	".exe": true, ".js": true, ".vbs": true, ".ps1": true, ".bat": true,
	".cmd": true, ".lnk": true, ".hta": true, ".scr": true, ".jar": true,
	".msi": true, ".docm": true, ".xlsm": true,
}

// IsDangerousExtension сообщает, требует ли имя файла предупреждения перед
// открытием. Смотрит на ПОСЛЕДНЕЕ расширение (report.pdf.exe — опасное).
func IsDangerousExtension(name string) bool {
	dot := strings.LastIndex(name, ".")
	if dot < 0 {
		return false
	}
	return dangerousExtensions[strings.ToLower(name[dot:])]
}

// LinkMismatch возвращает true, когда текст ссылки притворяется адресом
// (URL или домен), но ведёт на другой хост (SECURITY.md §4.2 — пометка ⚠).
// Обычный текст («читайте здесь») расхождением не считается: реальный хост
// и так отображается в реестре ссылок.
func LinkMismatch(linkText, rawURL string) bool {
	text := strings.TrimSpace(linkText)
	if text == "" {
		return false
	}
	textHost := hostOfText(text)
	if textHost == "" {
		return false // текст не претендует быть адресом
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}
	return !hostsEqual(textHost, u.Hostname())
}

// hostOfText извлекает хост из текста ссылки, если текст похож на адрес:
// полный URL, «host/path» или голый домен (с точкой, без пробелов).
func hostOfText(text string) string {
	if strings.ContainsAny(text, " \t") {
		return ""
	}
	if u, err := url.Parse(text); err == nil && u.Host != "" && u.Scheme != "" {
		return u.Hostname()
	}
	// Голый домен: минимум одна точка, только DNS-подобные символы.
	if !strings.Contains(text, ".") {
		return ""
	}
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '-' || r == ':' || r == '/':
		default:
			return ""
		}
	}
	host := text
	if i := strings.IndexAny(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.Index(host, ":"); i >= 0 { // порт
		host = host[:i]
	}
	return host
}

// hostsEqual сравнивает хосты без учёта регистра и ведущего www.
func hostsEqual(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	a, b = trimWWW(a), trimWWW(b)
	return a == b
}

func trimWWW(host string) string {
	if strings.HasPrefix(host, "www.") {
		return host[4:]
	}
	return host
}
