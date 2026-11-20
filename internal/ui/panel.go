package ui

// Каркас панелей (RD-2): у неактивной панели вместо тяжёлой рамки — строка
// заголовка капсом с hairline-разделителем, у активной — акцентная рамка.
// tview.Box не умеет скрывать отдельные стороны рамки, поэтому заголовок —
// собственный примитив высотой в одну строку.

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// panelHeader — строка «ПАПКИ ─────────» над панелью.
type panelHeader struct {
	*tview.Box
	title  string
	active bool
}

func newPanelHeader(title string) *panelHeader {
	h := &panelHeader{Box: tview.NewBox()}
	// WHY: фон заголовка берём из темы явно — tview запомнил глобальный фон
	// при создании Box, а тёмная тема задаёт свой цвет поверхности.
	// Не делать: оставлять фон по умолчанию — в тёмной теме строка выпадает.
	h.SetBackgroundColor(colorPanelBg)
	h.SetTitle(title)
	return h
}

// SetTitle меняет заголовок: месяц календаря, число встреч в повестке.
func (h *panelHeader) SetTitle(title string) *panelHeader {
	h.title = strings.ToUpper(title)
	return h
}

// setActive переключает цвет заголовка: активная панель — акцент (RD-1).
func (h *panelHeader) setActive(active bool) {
	h.active = active
}

// Draw рисует заголовок и добивает строку hairline-разделителем.
func (h *panelHeader) Draw(screen tcell.Screen) {
	h.Box.DrawForSubclass(screen, h)
	x, y, w, _ := h.GetRect()
	if w <= 0 {
		return
	}
	color := colorTextSecondary
	if h.active {
		color = colorAccent
	}
	_, printed := tview.Print(screen, h.title, x, y, w, tview.AlignLeft, color)
	// WHY: линия — textSecondary, а не hairline: hairline (222,226,230) на
	// белой поверхности не читается, и разделитель выглядел отсутствующим.
	// Не делать: возвращать hairline — заголовки снова «висят в воздухе».
	rule := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	for i := printed + 1; i < w; i++ {
		screen.SetContent(x+i, y, tview.Borders.Horizontal, nil, rule)
	}
}

// panelColumn — вертикальная пара «заголовок + панель». Фокус передаётся
// телу: заголовок неинтерактивен.
func panelColumn(header *panelHeader, body tview.Primitive) *tview.Flex {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(body, 0, 1, true)
}
