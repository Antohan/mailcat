package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestLightThemeContrast: фон панелей белый, весь текст на нём — тёмный
// (чёрный шрифт на чёрном фоне — WHY в theme.go), выделение — инверсная пара.
func TestLightThemeContrast(t *testing.T) {
	applyLightTheme()

	if got := tview.Styles.PrimitiveBackgroundColor; got != tcell.ColorWhite {
		t.Fatalf("фон панелей = %v, хочу White", got)
	}
	pairs := map[string][2]tcell.Color{
		"основной текст":             {tview.Styles.PrimaryTextColor, tview.Styles.PrimitiveBackgroundColor},
		"заголовки панелей":          {tview.Styles.TitleColor, tview.Styles.PrimitiveBackgroundColor},
		"рамки":                      {tview.Styles.BorderColor, tview.Styles.PrimitiveBackgroundColor},
		"текст выделения":            {tview.Styles.InverseTextColor, tview.Styles.ContrastBackgroundColor},
		"непрочитанные письма":       {colorUnread, tcell.ColorWhite},
		"прочитанные письма":         {colorRead, tcell.ColorWhite},
		"выходные в календаре":       {colorWeekend, tcell.ColorWhite},
		"дни соседних месяцев":       {colorOtherMonth, tcell.ColorWhite},
		"текст в выделенной строчке": {colorSelectedFg, colorSelectedBg},
		"рамка активной панели":      {colorFocusBorder, tcell.ColorWhite},
	}
	for name, p := range pairs {
		if p[0] == p[1] {
			t.Errorf("%s: текст %v совпадает с фоном %v — нечитаемо", name, p[0], p[1])
		}
	}
	// RD-1: рамка фокуса и выделение строки — один акцент; различие только
	// в поверхности (WHY в theme.go).
	if colorFocusBorder != colorAccent || colorSelectedBg != colorAccent {
		t.Errorf("фокус %v / выделение %v должны совпадать с акцентом %v",
			colorFocusBorder, colorSelectedBg, colorAccent)
	}
	if colorTextPrimary == tcell.ColorBlack {
		t.Error("основной текст не должен быть чёрным")
	}
}

// TestDarkThemeContrast: тёмная тема — те же роли, поверхности тёмные.
func TestDarkThemeContrast(t *testing.T) {
	applyTheme("dark")

	if tview.Styles.PrimitiveBackgroundColor == tcell.ColorWhite {
		t.Fatal("фон тёмной темы остался белым")
	}
	if colorTextPrimary == tcell.ColorBlack {
		t.Error("основной текст не должен быть чёрным")
	}
	if colorTextPrimary == tview.Styles.PrimitiveBackgroundColor {
		t.Error("основной текст совпадает с фоном — нечитаемо")
	}
	if colorTextSecondary == tview.Styles.PrimitiveBackgroundColor {
		t.Error("второстепенный текст совпадает с фоном — нечитаемо")
	}
	applyLightTheme() // вернуть палитру по умолчанию для других тестов
}
