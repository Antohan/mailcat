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

// panelHeader — строка «ПИСЬМА ─────────» над панелью; справа может быть
// вторая часть (счётчик списка писем, V-2). У ПАПОК и ПРОСМОТРА её нет (V-16).
type panelHeader struct {
	*tview.Box
	title  string
	right  string
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

// headerGap — зазор между текстом заголовка и линией (П.4а): линия не
// прилипает ни к левой подписи, ни к правому счётчику.
const headerGap = 2

// SetTitle меняет заголовок: месяц календаря, число встреч в повестке.
func (h *panelHeader) SetTitle(title string) *panelHeader {
	h.title = strings.ToUpper(title)
	h.right = ""
	return h
}

// SetListTitle — заголовок списка писем (V-2): слева «Папка · N непрочитанных»,
// справа «всего T · стр. p/P». Без капса, в отличие от SetTitle: имя папки —
// данные с сервера, капс его искажает.
func (h *panelHeader) SetListTitle(left, right string) *panelHeader {
	h.title = left
	h.right = right
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
	// WHY (V-16): конец линии — смещение внутри заголовка, а не экранный x+w:
	// заголовок ПИСЬМА стоит в средней колонке, и абсолютный конец уводил
	// линию на 20 клеток вправо, поверх разделителя и рамки просмотра.
	// Не делать: end := x+w и SetContent(x+i) — раньше вылет скрывала строка
	// заголовка ПРОСМОТР, которая рисовалась после и затирала его.
	end := w
	if h.right != "" {
		// Правая часть — приглушённая всегда: это служебные счётчики, акцент
		// зарезервирован за активной панелью (RD-1).
		_, wRight := tview.Print(screen, h.right, x, y, w, tview.AlignRight, colorTextSecondary)
		end -= wRight + headerGap
	}
	// WHY: линия — textSecondary, а не hairline: hairline (222,226,230) на
	// белой поверхности не читается, и разделитель выглядел отсутствующим.
	// Не делать: возвращать hairline — заголовки снова «висят в воздухе».
	rule := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	for i := printed + headerGap; i < end; i++ {
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

// divider — тонкий вертикальный разделитель колонок (V-3, П.4б): одна
// линия на весь столбец между панелями, тем же цветом, что и линии
// заголовков.
type divider struct {
	*tview.Box
}

func newDivider() *divider {
	return &divider{Box: tview.NewBox()}
}

// Draw красит колонку символами Borders.Vertical.
// WHY: цвет textSecondary, не hairline — по той же причине, что и линии
// заголовков выше: hairline на белой поверхности не читается.
func (d *divider) Draw(screen tcell.Screen) {
	d.Box.DrawForSubclass(screen, d)
	x, y, w, h := d.GetRect()
	rule := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			screen.SetContent(xx, yy, tview.Borders.Vertical, nil, rule)
		}
	}
}
