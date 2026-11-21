package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Палитра приложения. Инвариант: цвета в виджетах берутся только отсюда
// (rd-1) — сырые tcell-цвета в mail.go/calendar.go/app.go не появляются.
// Роли семантические: основной/второстепенный текст, акцент, hairline,
// ошибка, выходные. Значения ниже — светлая тема (по умолчанию).
var (
	colorTextPrimary   = tcell.NewRGBColor(31, 41, 55)    // основной текст, не чёрный
	colorTextSecondary = tcell.NewRGBColor(107, 114, 128) // даты, счётчики, подписи
	colorAccent        = tcell.NewRGBColor(37, 99, 235)   // единственный акцент
	colorHairline      = tcell.NewRGBColor(222, 226, 230) // тонкие разделители
	colorError         = tcell.NewRGBColor(201, 64, 58)   // приглушённый красный
	colorWeekend       = tcell.NewRGBColor(199, 115, 110) // Сб/Вс в календаре
	colorConnected     = tcell.NewRGBColor(26, 127, 55)   // зелёная точка связи (V-4)

	colorPanelBg  = tcell.ColorWhite
	colorOnAccent = tcell.ColorWhite // текст на акцентной заливке

	// WHY (О-3): «сегодня» — светлый оттенок акцента, а не сам акцент:
	// сплошной акцент уже занят выделением таблицы, совпадение сделало бы
	// выбранный день неотличимым от «сегодня».
	// Не делать: заливать «сегодня» colorAccent.
	colorTodayBg = tcell.NewRGBColor(219, 234, 254)
)

// Производные роли для виджетов; заполняются apply*Theme.
var (
	colorUnread      = colorTextPrimary
	colorRead        = colorTextSecondary
	colorOtherMonth  = colorTextSecondary // дни соседних месяцев
	colorSelectedBg  = colorAccent
	colorSelectedFg  = colorOnAccent
	colorFocusBorder = colorAccent
)

// applyTheme выбирает палитру по cfg.UI.Theme ("light" по умолчанию).
// Вызывается до создания виджетов: значения ниже читаются при отрисовке.
func applyTheme(theme string) {
	if theme == "dark" {
		applyDarkTheme()
		return
	}
	applyLightTheme()
}

// applyLightTheme — светлые поверхности, тёмный (не чёрный) текст.
// WHY: tview по умолчанию красит панели в чёрный, а список писем брал
// ColorDefault (цвет терминала) — на светлом терминале это чёрный шрифт
// на чёрном фоне. Тема задаётся явно, чтобы не зависеть от профиля терминала.
func applyLightTheme() {
	colorTextPrimary = tcell.NewRGBColor(31, 41, 55)
	colorTextSecondary = tcell.NewRGBColor(107, 114, 128)
	colorAccent = tcell.NewRGBColor(37, 99, 235)
	colorHairline = tcell.NewRGBColor(222, 226, 230)
	colorError = tcell.NewRGBColor(201, 64, 58)
	colorWeekend = tcell.NewRGBColor(199, 115, 110)
	colorConnected = tcell.NewRGBColor(26, 127, 55)
	colorPanelBg = tcell.ColorWhite
	colorOnAccent = tcell.ColorWhite
	colorTodayBg = tcell.NewRGBColor(219, 234, 254) // светлый оттенок акцента
	applyDerivedRoles()
	applyStyles()
}

// applyDarkTheme — те же роли на тёмных поверхностях.
func applyDarkTheme() {
	colorTextPrimary = tcell.NewRGBColor(229, 231, 235)
	colorTextSecondary = tcell.NewRGBColor(161, 161, 170)
	colorAccent = tcell.NewRGBColor(96, 165, 250)
	colorHairline = tcell.NewRGBColor(63, 63, 70)
	colorError = tcell.NewRGBColor(248, 113, 113)
	colorWeekend = tcell.NewRGBColor(252, 165, 165)
	colorConnected = tcell.NewRGBColor(63, 185, 80)
	colorPanelBg = tcell.NewRGBColor(24, 24, 27)
	colorOnAccent = tcell.NewRGBColor(17, 24, 39)
	colorTodayBg = tcell.NewRGBColor(30, 58, 138) // тёмный оттенок акцента
	applyDerivedRoles()
	applyStyles()
}

// applyDerivedRoles связывает роли с виджетами.
// WHY (RD-1): рамка фокуса и выделение строки — один акцент; различие
// только в поверхности (фон строки vs рамка). Прежний бирюзовый фокус
// отменён сознательно, чтобы акцент в интерфейсе был один.
// Не делать: возвращать отдельный цвет рамки — интерфейс снова запестрит.
func applyDerivedRoles() {
	// WHY (RD-4): непрочитанные — жирным единственным акцентом: он выделяет
	// их в обеих палитрах; отдельный «цвет непрочитанных» снова бы запестрил
	// (см. RD-1). Не делать: свой цвет на каждую смысловую группу строк.
	colorUnread = colorAccent
	colorRead = colorTextSecondary
	colorOtherMonth = colorTextSecondary
	colorSelectedBg = colorAccent
	colorSelectedFg = colorOnAccent
	colorFocusBorder = colorAccent
}

// applyStyles переносит палитру в глобальные стили tview.
func applyStyles() {
	tview.Styles.PrimitiveBackgroundColor = colorPanelBg
	tview.Styles.ContrastBackgroundColor = colorAccent
	tview.Styles.BorderColor = colorHairline
	// WHY (V-3, П.2в): рамка панели с фокусом в tview рисуется двойными
	// линиями (╔═╗) — для каркаса слишком тяжело. Все шесть focus-символов
	// сводим к одинарным; цвет рамки (акцент активной панели) ставит
	// paintFocusFrames, механика UI-6 не меняется.
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Styles.TitleColor = colorTextSecondary
	tview.Styles.GraphicsColor = colorTextPrimary
	tview.Styles.PrimaryTextColor = colorTextPrimary
	tview.Styles.SecondaryTextColor = colorTextSecondary
	tview.Styles.TertiaryTextColor = colorTextSecondary
	tview.Styles.InverseTextColor = colorOnAccent
	tview.Styles.ContrastSecondaryTextColor = colorOnAccent
}
