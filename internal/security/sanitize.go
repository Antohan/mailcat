// Package security — санитизация непроверенного текста из EWS до любого
// вывода в терминал (docs/SECURITY.md §3) и анти-фишинг (этап 3).
//
// Весь текст из Exchange считается заражённым: темы, имена, адреса, тела,
// календарные поля, имена вложений.
package security

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Шаг 1 §3 — escape-последовательности: CSI (\x1b[), OSC (включая гиперссылки
// OSC 8 и запись буфера обмена OSC 52), двухсимвольные ESC-команды (ESC M).
// C1-формы (0x80–0x9f, включая 8-битный CSI-старт 0x9b) удаляются на шаге 2
// как класс управляющих: сами по себе параметры без стартера неисполнимы.
var (
	reCSI = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]")
	reOSC = regexp.MustCompile("\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)?")
	reESC = regexp.MustCompile("\x1b[@-Z\\\\-_]")
	reCtl = regexp.MustCompile("[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]") // C0 (кроме \t\n\r) и DEL
)

// Sanitize вычищает терминальные escape-последовательности и управляющие
// символы из непроверенной строки. Табуляция, перевод строки и возврат
// каретки сохраняются; каждый невалидный байт UTF-8 становится U+FFFD
// (семантика strings.ToValidUTF8).
func Sanitize(s string) string {
	s = reCSI.ReplaceAllString(s, "")
	s = reOSC.ReplaceAllString(s, "")
	s = reESC.ReplaceAllString(s, "")
	s = reCtl.ReplaceAllString(s, "")
	return sanitizeUTF8(s)
}

// sanitizeUTF8 завершает шаги 2–3 §3 на уровне рун:
//   - легитимно закодированные C1-руны (U+0080–U+009F) удаляются;
//   - невалидные байты UTF-8 в диапазоне 0x80–0x9F (C1-подобные) удаляются;
//   - прочие невалидные байты заменяются на U+FFFD.
//
// Прямое сравнение рун здесь корректно отличает C1 от кириллицы:
// «Л» = 0xD0 0x9B декодируется как руна U+041B (не C1), а одиночный байт 0x9B
// невалиден и попадает в ветку RuneError.
func sanitizeUTF8(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if c := s[i]; c >= 0x80 && c <= 0x9f {
				i++ // C1-подобный невалидный байт — удалить
				continue
			}
			b.WriteString("\uFFFD")
			i++
			continue
		}
		if r >= 0x80 && r <= 0x9f { // легитимная C1-руна
			i += size
			continue
		}
		b.WriteString(s[i : i+size])
		i += size
	}
	return b.String()
}
