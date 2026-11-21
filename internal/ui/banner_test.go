package ui

// Тесты ASCII-баннера месяца (О-3): форма, ширина строк, покрытие всех 12
// месяцев шрифтом bannerGlyphs.

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestMonthBanner(t *testing.T) {
	for m := time.January; m <= time.December; m++ {
		banner := MonthBanner(m)
		if len(banner) != 5 {
			t.Fatalf("%v: строк баннера = %d, хочу 5", m, len(banner))
		}
		width := utf8.RuneCountInString(banner[0])
		for row, line := range banner {
			if utf8.RuneCountInString(line) != width {
				t.Errorf("%v: строка %d шириной %d, хочу %d", m, row, len(line), width)
			}
			for _, r := range line {
				if r != '█' && r != ' ' {
					t.Errorf("%v: строка %d содержит %q — только █ и пробел", m, row, r)
				}
			}
		}
		if width == 0 || width > 60 {
			t.Errorf("%v: ширина баннера = %d, жду 1–60", m, width)
		}
		if !strings.Contains(banner[0], "█") {
			t.Errorf("%v: верхняя строка пуста — буквы не отрисованы", m)
		}
	}
}

func TestMonthBannerSpecific(t *testing.T) {
	// Ноябрь: Н О Я Б Р Ь — 6 букв × 4 колонки + 5 разделителей = 29.
	banner := MonthBanner(time.November)
	for _, line := range banner {
		if utf8.RuneCountInString(line) != 29 {
			t.Fatalf("ширина «НОЯБРЬ» = %d рун, хочу 29", utf8.RuneCountInString(line))
		}
	}
	want := []string{
		"█  █  ██   ██  ███  ███  █   ",
		"█  █ █  █ █  █ █    █  █ █   ",
		"████ █  █  ███ ███  ███  ███ ",
		"█  █ █  █  █ █ █  █ █    █  █",
		"█  █  ██  █  █ ███  █    ███ ",
	}
	for row := range want {
		if banner[row] != want[row] {
			t.Errorf("строка %d = %q, хочу %q", row, banner[row], want[row])
		}
	}
}
