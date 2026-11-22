package ui

// Полоса прокрутки панели: в tview своей полосы нет — рисуем одноколонную
// (жёлоб + бегунок) по данным панели. Высота бегунка — доля видимого в общем;
// полоса рисуется после панели, поэтому берёт её свежий rect на этом кадре.

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// scrollThumb — позиция и высота бегунка в полосе visible клеток:
// total <= visible — прокручивать нечего (0, 0); бегунок не меньше клетки.
func scrollThumb(offset, total, visible int) (top, height int) {
	if visible <= 0 || total <= visible {
		return 0, 0
	}
	// Округление вверх: бегунок не меньше честной доли, иначе на 10 видимых
	// из 40 он терял бы клетку (100/40 = 2 вместо 2.5).
	height = (visible*visible + total - 1) / total
	if height < 1 {
		height = 1
	}
	if height > visible {
		height = visible
	}
	maxOff := total - visible
	if offset < 0 {
		offset = 0
	}
	if offset > maxOff {
		offset = maxOff
	}
	top = (visible - height) * offset / maxOff
	return top, height
}

// scrollbar — одноколонная полоса прокрутки: geom отдаёт экранные координаты
// и высоту внутренней области панели (рисуем после панели — rect свежий),
// track — offset и total прокручиваемого содержимого.
type scrollbar struct {
	*tview.Box
	geom  func() (x, y, h int)
	track func() (offset, total int)
}

func newScrollbar() *scrollbar {
	sb := &scrollbar{Box: tview.NewBox()}
	// WHY: фон колонки — поверхность панели: при пустой полосе не остаётся
	// полосы терминального цвета (та же причина, что у panelHeader, V-16).
	sb.SetBackgroundColor(colorPanelBg)
	return sb
}

// Draw рисует жёлоб «│» hairline и бегунок «█» secondary; рамку не рисует.
func (sb *scrollbar) Draw(screen tcell.Screen) {
	sb.Box.Draw(screen)
	if sb.geom == nil || sb.track == nil {
		return
	}
	x, y, h := sb.geom()
	if h <= 0 {
		return
	}
	offset, total := sb.track()
	top, thumb := scrollThumb(offset, total, h)
	if thumb == 0 {
		return
	}
	trackStyle := tcell.StyleDefault.Foreground(colorHairline)
	thumbStyle := tcell.StyleDefault.Foreground(colorTextSecondary)
	for i := 0; i < h; i++ {
		r, style := '│', trackStyle
		if i >= top && i < top+thumb {
			r, style = '█', thumbStyle
		}
		screen.SetContent(x, y+i, r, nil, style)
	}
}
