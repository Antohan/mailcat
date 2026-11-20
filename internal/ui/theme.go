package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Светлая тема: фон панелей белый, текст тёмный.
// WHY: tview по умолчанию красит панели в чёрный, а список писем брал
// ColorDefault (цвет терминала) — на светлом терминале это чёрный шрифт
// на чёрном фоне. Тема задаётся явно, чтобы не зависеть от профиля терминала.
func applyLightTheme() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorWhite
	tview.Styles.ContrastBackgroundColor = colorSelectedBg
	tview.Styles.BorderColor = tcell.ColorGray
	tview.Styles.TitleColor = tcell.ColorBlack
	tview.Styles.GraphicsColor = tcell.ColorBlack
	tview.Styles.PrimaryTextColor = tcell.ColorBlack
	tview.Styles.SecondaryTextColor = tcell.ColorDarkGray
	tview.Styles.TertiaryTextColor = tcell.ColorDarkGreen
	tview.Styles.InverseTextColor = tcell.ColorWhite
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorWhite
}

// Палитра на белом фоне: жёлтый/зелёный из тёмной темы здесь нечитаемы.
var (
	colorUnread     = tcell.ColorBlack // непрочитанные: жирный чёрный
	colorRead       = tcell.ColorGray  // прочитанные: приглушённые
	colorWeekend    = tcell.ColorRed   // Сб/Вс в календаре
	colorOtherMonth = tcell.ColorGray  // дни соседних месяцев
	colorSelectedBg = tcell.ColorBlue  // выделенная строка/пункт
	colorSelectedFg = tcell.ColorWhite
	// Рамка активной панели (UI-6). WHY: бирюзовый, а не синий — синий уже
	// занят выделением строки (colorSelectedBg): «активная панель» — это не
	// «выбранная строка», рамка не должна притворяться выделением.
	colorFocusBorder = tcell.ColorDarkCyan
	// Ошибка в статус-баре (UI-7). Красный на белом фоне читаем; смысловые
	// цвета не пересекаются: синий — выделение, бирюза — рамка фокуса.
	colorError = tcell.ColorRed
)
