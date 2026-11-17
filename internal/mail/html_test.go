package mail

// Тесты HTML→текст (docs/TESTING.md §2: таблицы, ссылки, стили).

import (
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string // подстроки, которые должны быть
		bad  []string // чего быть не должно
	}{
		{
			name: "простой абзац",
			in:   "<html><body><p>Привет, мир!</p></body></html>",
			want: []string{"Привет, мир!"},
		},
		{
			name: "ссылка: показывается реальный URL (анти-фишинг)",
			in:   `<p>Смотрите <a href="https://x.example">отчёт</a> здесь</p>`,
			want: []string{"https://x.example"},
			bad:  []string{"<a", "href"},
		},
		{
			name: "таблица — содержимое ячеек",
			in:   "<table><tr><td>Сумма</td><td>100</td></tr></table>",
			want: []string{"Сумма", "100"},
			bad:  []string{"<td", "<table"},
		},
		{
			name: "стили и скрипты выбрасываются",
			in:   `<style>body{color:red}</style><script>alert(1)</script><p>Текст</p>`,
			want: []string{"Текст"},
			bad:  []string{"color:red", "alert", "<script", "<style"},
		},
		{
			name: "сущности декодируются",
			in:   "<p>A &amp; B &laquo;цитата&raquo;</p>",
			want: []string{"A & B", "«цитата»"},
		},
		{
			name: "кириллица и эмодзи не искажаются",
			in:   "<p>Отчёт готов 🎉</p>",
			want: []string{"Отчёт готов 🎉"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := HTMLToText(tc.in)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("нет %q в %q", w, got)
				}
			}
			for _, b := range tc.bad {
				if strings.Contains(got, b) {
					t.Errorf("%q не должен попасть в %q", b, got)
				}
			}
		})
	}
}

// TestHTMLToTextEmpty: пустой вход — пустой выход, без паники.
func TestHTMLToTextEmpty(t *testing.T) {
	if got := HTMLToText(""); got != "" {
		t.Errorf("HTMLToText(\"\") = %q, хочу пусто", got)
	}
}
