package security

// Тесты анти-фишинга (docs/SECURITY.md §4, PLAN.md этап 3):
// IDN/punycode-хосты, опасные расширения вложений, «текст ссылки ≠ адрес».

import "testing"

// TestHostInfoPunycode: хосты с xn-- и не-ASCII определяются как IDN,
// punycode-форма выдаётся через golang.org/x/net/idna.
func TestHostInfoPunycode(t *testing.T) {
	cases := []struct {
		host     string
		isIDN    bool
		asciiSub string // подстрока ожидаемой punycode-формы ("" — сам хост)
	}{
		{"xn--80ak6aa92e.com", true, "xn--80ak6aa92e.com"},
		{"пример.рф", true, "xn--"},
		{"mail.company.example", false, "mail.company.example"},
		{"localhost", false, "localhost"},
	}
	for _, tc := range cases {
		info := HostInfoOf(tc.host)
		if info.IDN != tc.isIDN {
			t.Errorf("HostInfoOf(%q).IDN = %v, хочу %v", tc.host, info.IDN, tc.isIDN)
		}
		if tc.asciiSub != "" && !containsStr(info.Ascii, tc.asciiSub) {
			t.Errorf("HostInfoOf(%q).Ascii = %q, хочу подстроку %q", tc.host, info.Ascii, tc.asciiSub)
		}
	}
}

// TestHostInfoFromURL: вариант для полной ссылки берёт хост из URL.
func TestHostInfoFromURL(t *testing.T) {
	info := HostInfo("https://xn--80ak6aa92e.com/login")
	if !info.IDN || info.Ascii != "xn--80ak6aa92e.com" {
		t.Errorf("HostInfo(url) = %+v", info)
	}
	if got := HostInfo("http://mail.company.example/"); got.IDN || got.Ascii != "mail.company.example" {
		t.Errorf("HostInfo(url) = %+v", got)
	}
}

// TestIsDangerousExtension: список PLAN.md этап 3; двойное расширение ловится
// по последнему; регистр не важен.
func TestIsDangerousExtension(t *testing.T) {
	dangerous := []string{
		"virus.exe", "script.js", "macro.vbs", "ps.ps1", "run.bat", "cmd.cmd",
		"link.lnk", "page.hta", "screen.scr", "app.jar", "setup.msi",
		"doc.docm", "sheet.xlsm", "REPORT.PDF.EXE", "Отчёт.exe",
	}
	for _, name := range dangerous {
		if !IsDangerousExtension(name) {
			t.Errorf("IsDangerousExtension(%q) = false, хочу true", name)
		}
	}
	safe := []string{"report.pdf", "данные.csv", "photo.jpeg", "archive.zip", "doc.docx", "sheet.xlsx"}
	for _, name := range safe {
		if IsDangerousExtension(name) {
			t.Errorf("IsDangerousExtension(%q) = true, хочу false", name)
		}
	}
}

// TestLinkMismatch: ⚠ только когда текст ссылки притворяется адресом и ведёт
// на другой хост; обычный текст («Сбербанк») — не расхождение (реальный хост
// и так показывается в реестре ссылок).
func TestLinkMismatch(t *testing.T) {
	cases := []struct {
		text string
		url  string
		want bool
	}{
		{"https://sberbank.ru", "http://evil.tld/x", true},
		{"sberbank.ru", "http://evil.tld", true},
		{"Сбербанк", "http://evil.tld", false},
		{"sberbank.ru", "https://sberbank.ru/ib", false},
		{"www.sberbank.ru", "https://sberbank.ru", false}, // www — не расхождение
		{"читайте здесь", "https://company.example/doc", false},
		{"", "https://company.example", false},
	}
	for _, tc := range cases {
		if got := LinkMismatch(tc.text, tc.url); got != tc.want {
			t.Errorf("LinkMismatch(%q, %q) = %v, хочу %v", tc.text, tc.url, got, tc.want)
		}
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexStr(s, sub) >= 0
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
