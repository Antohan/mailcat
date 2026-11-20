package platform

import (
	"testing"
)

// TestAppleScriptEscape: SECURITY.md §9 — текст писем попадает в osascript:
// " → \", \ → \\, переводы строк → пробел.
func TestAppleScriptEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{`просто текст`, `просто текст`},
		{`он сказал "привет"`, `он сказал \"привет\"`},
		{`путь C:\temp`, `путь C:\\temp`},
		{"строка1\nстрока2", "строка1 строка2"},
		{"CR\rтоже", "CR тоже"},
		{`" + injected`, `\" + injected`},
	}
	for _, tc := range cases {
		if got := appleScriptEscape(tc.in); got != tc.want {
			t.Errorf("appleScriptEscape(%q) = %q, хочу %q", tc.in, got, tc.want)
		}
	}
}

// TestNotifyCmd: команда — osascript -e display notification с экранированием
// и звуком только "default" (SECURITY.md §9).
func TestNotifyCmd(t *testing.T) {
	cmd := notifyCmd(`Титул "x"`, "Тема \n новая", true)
	if cmd.Path != "/usr/bin/osascript" {
		t.Errorf("Path = %q", cmd.Path)
	}
	script := cmd.Args[len(cmd.Args)-1]
	for _, want := range []string{"display notification", "новая", `\"x\"`, "with title", "sound name \"default\""} {
		if !containsSub(script, want) {
			t.Errorf("в скрипте нет %q: %s", want, script)
		}
	}
	if containsSub(script, "\n") {
		t.Errorf("перевод строки не экранирован: %q", script)
	}
	cmd2 := notifyCmd("t", "b", false)
	script2 := cmd2.Args[len(cmd2.Args)-1]
	if containsSub(script2, "sound") {
		t.Errorf("звук при sound=false: %s", script2)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
