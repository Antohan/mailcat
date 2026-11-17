package security

// Тесты санитайзера терминала — docs/SECURITY.md §3 (порядок: escape →
// управляющие → ToValidUTF8). Кейсы §3 обязательны полностью уже на этапе 2
// (DoD этапа 2: весь текст из EWS санитизирован до передачи в tview).

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// OSC 8 гиперссылка-инъекция
		{"OSC 8 ссылка", "\x1b]8;;http://evil\x1b\\текст\x1b]8;;\x1b\\", "текст"},
		// OSC 52 — запись буфера обмена
		{"OSC 52 буфер", "\x1b]52;c;aGVsbG8=\x07x", "x"},
		// Подмена заголовка окна
		{"OSC 0 title", "\x1b]0;взлом\x07осталось", "осталось"},
		// CSI (цвет/курсор)
		{"CSI", "\x1b[31mкрасный\x1b[0m", "красный"},
		{"CSI курсор", "a\x1b[2Jb", "ab"},
		// Двухсимвольная ESC-команда
		{"ESC M", "x\x1bMy", "xy"},
		// C1 (0x80–0x9f, включая 8-битный CSI-старт): сам символ удаляется
		// как управляющий; параметры без стартера неисполнимы и безопасны.
		{"C1 старт CSI", "a\x9b31mb", "a31mb"},
		{"C1 одинокий", "a\x9bb", "ab"},
		// Прочие управляющие C0 (кроме \t \n \r), DEL
		{"C0 и DEL", "a\x00\x01\x0b\x0c\x0e\x1f\x7fb", "ab"},
		// Таб/перевод строк/возврат каретки сохраняются
		{"TNR сохраняются", "a\tb\nc\rd", "a\tb\nc\rd"},
		// Битый UTF-8: каждый невалидный байт → U+FFFD (семантика ToValidUTF8)
		{"битый UTF-8", "a\xff\xfeb", "a\uFFFD\uFFFDb"},
		{"битый UTF-8 двойной", "\xff\xfe", "\uFFFD\uFFFD"},
		// Нормальный текст не искажается
		{"кириллица+эмодзи", "Привет, мир 🎉 — всё ок", "Привет, мир 🎉 — всё ок"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sanitize(tc.in); got != tc.want {
				t.Errorf("Sanitize(%q) = %q, хочу %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeNoEscapeLeft: после санитизации не остаётся ни одного ESC-байта.
func TestSanitizeNoEscapeLeft(t *testing.T) {
	nasty := "\x1b]8;;x\x1b\\\x1b[A\x1bM\x9b31m\x1b]0;t\x07"
	if got := Sanitize(nasty); strings.ContainsRune(got, '\x1b') {
		t.Errorf("в результате остался ESC: %q", got)
	}
}
