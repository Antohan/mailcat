package ui

// V-32: monthGrid растягивает 7 колонок и 6 недель на всю панель,
// навигация зажата границами месяца, клик выбирает только день месяца.

import (
	"fmt"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestDayCellRectGeometry(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {220, 60}} {
		w, h := size.w, size.h
		mx, my := gridMargins(w, h)
		// V-48: сетка прижата к левому верхнему углу — поля нулевые.
		if mx != 0 || my != 0 {
			t.Errorf("%dx%d: сетка не прижата к углу: mx=%d my=%d", w, h, mx, my)
		}
		// Шапка дней недели — в строке my, первая неделя — через линию под ней.
		if _, top, _, _ := dayCellRect(0, 0, w, h); top != my+2 {
			t.Errorf("%dx%d: шапка не под полем: top=%d, ждали %d", w, h, top, my+2)
		}
		prevBottom, weekHeight := my+2, -1
		for row := 0; row < 6; row++ {
			_, top, _, ch := dayCellRect(row, 0, w, h)
			if top != prevBottom || ch <= 0 {
				t.Errorf("%dx%d: неделя %d не стыкуется: top=%d bottom=%d", w, h, row, top, prevBottom)
			}
			if weekHeight >= 0 && ch != weekHeight {
				t.Errorf("%dx%d: недели разной высоты: %d != %d", w, h, ch, weekHeight)
			}
			weekHeight = ch
			prevBottom = top + ch + 1 // плюс линия между неделями
		}
		// Остаток высоты — снизу, меньше высоты ячейки: сетка прижата к верху.
		if bottom := h - (prevBottom - 1); bottom < 0 || bottom > 5 {
			t.Errorf("%dx%d: остаток снизу %d вне 0…5 при верхнем поле %d", w, h, bottom, my)
		}
		x, width := mx, -1
		for col := 0; col < 7; col++ {
			cx, _, cw, _ := dayCellRect(0, col, w, h)
			if cx != x || cw <= 0 {
				t.Errorf("%dx%d: колонка %d не стыкуется: x=%d w=%d", w, h, col, cx, cw)
			}
			if width >= 0 && cw != width {
				t.Errorf("%dx%d: колонки разной ширины: %d != %d", w, h, cw, width)
			}
			width = cw
			x += cw + 1 // плюс вертикальная линия
		}
		// Остаток ширины — справа, меньше ширины ячейки: сетка прижата влево.
		if right := w - (x - 1); right < 0 || right > 6 {
			t.Errorf("%dx%d: остаток справа %d вне 0…6 при левом поле %d", w, h, right, mx)
		}
	}
}

// TestDayAtMargins: клик в остатке справа/снизу и в строке шапки — мимо ячеек.
func TestDayAtMargins(t *testing.T) {
	const w, h = 100, 30
	mx, my := gridMargins(w, h)
	// V-48: сетка прижата к левому верхнему углу, остаток — справа/снизу.
	if mx != 0 || my != 0 {
		t.Fatalf("сетка не у угла: mx=%d my=%d", mx, my)
	}
	cw, ch := gridCellSize(w, h)
	for _, p := range []struct {
		x, y int
		name string
	}{
		{w - 1, my + 3, "правое поле"},
		{w - 1, h - 1, "нижний правый остаток"},
		{mx, my, "строка шапки"},
		{mx, my + 1, "линия под шапкой"},
		{mx + 1, h - 1, "нижнее поле"},
	} {
		if _, _, ok := dayAt(p.x, p.y, w, h); ok {
			t.Errorf("%s: точка (%d,%d) попала в ячейку", p.name, p.x, p.y)
		}
	}
	if _, _, ok := dayAt(mx, my+2, w, h); !ok {
		t.Errorf("первая ячейка сетки не находится")
	}
	// Левая колонка начинается от рамки: заливка вплотную к ней (V-48).
	if x, y, _, _ := dayCellRect(0, 0, w, h); x != 0 || y != 2 {
		t.Errorf("первая ячейка не у левого верхнего угла: x=%d y=%d", x, y)
	}
	// Вертикальная линия между колонками — не ячейка (V-38).
	if _, _, ok := dayAt(mx+cw, my+2, w, h); ok {
		t.Errorf("вертикальная линия попала в ячейку")
	}
	// Первая ячейка второй недели — тоже от рамки, под линией.
	if _, _, ok := dayAt(0, 2+ch+1, w, h); !ok {
		t.Errorf("ячейка второй недели не находится")
	}
}

func TestMonthGridMoveClamped(t *testing.T) {
	g := newMonthGrid()
	g.SetCells(MonthGrid(2026, time.November, nil), 0) // 1 ноября — воскресенье
	g.SelectDay(1)
	for _, delta := range []int{-1, -7} {
		g.Move(delta)
		if day := g.cells[g.selRow][g.selCol].Day; day != 1 {
			t.Fatalf("Move(%d) вышел за 1-е: %d", delta, day)
		}
	}
	g.Move(7)
	if day := g.cells[g.selRow][g.selCol].Day; day != 8 {
		t.Fatalf("Move(7) с 1-го: %d, ждали 8", day)
	}
	g.SelectDay(30)
	for _, delta := range []int{1, 7} {
		g.Move(delta)
		if day := g.cells[g.selRow][g.selCol].Day; day != 30 {
			t.Fatalf("Move(%d) вышел за 30-е: %d", delta, day)
		}
	}
	g.Move(-7)
	if day := g.cells[g.selRow][g.selCol].Day; day != 23 {
		t.Fatalf("Move(-7) с 30-го: %d, ждали 23", day)
	}
}

func TestMonthGridMouseSelectsOnlyDays(t *testing.T) {
	g := newMonthGrid()
	g.SetBorder(true)
	g.SetRect(0, 0, 40, 20)
	g.SetCells(MonthGrid(2026, time.November, nil), 0)
	g.SelectDay(1)
	x0, y0, w, h := g.GetInnerRect()
	click := func(row, col int) (bool, bool) {
		cx, cy, cw, ch := dayCellRect(row, col, w, h)
		ev := tcell.NewEventMouse(x0+cx+cw/2, y0+cy+ch/2, tcell.Button1, tcell.ModNone)
		focused := false
		consumed, _ := g.MouseHandler()(tview.MouseLeftClick, ev, func(p tview.Primitive) {
			focused = p == g
		})
		return consumed, focused
	}
	// Пустая ячейка первой недели (соседний месяц) невыбираема.
	if consumed, focused := click(0, 0); consumed || focused {
		t.Fatal("клик по пустой ячейке выбран")
	}
	if day := g.cells[g.selRow][g.selCol].Day; day != 1 {
		t.Fatalf("клик по пустой ячейке сменил день: %d", day)
	}
	// День 1 стоит в последней колонке первой недели.
	if consumed, focused := click(0, 6); !consumed || !focused {
		t.Fatal("клик по дню не выбран")
	}
	if day := g.cells[g.selRow][g.selCol].Day; day != 1 {
		t.Fatalf("клик выбрал не тот день: %d", day)
	}
	if consumed, _ := click(4, 6); !consumed {
		t.Fatal("клик по дню 29 не выбран")
	}
	if day := g.cells[g.selRow][g.selCol].Day; day != 29 {
		t.Fatalf("клик выбрал не тот день: %d, ждали 29", day)
	}
}

func TestCalendarGridFillsPanel(t *testing.T) {
	a := newTestApp(t)
	a.calendar.year, a.calendar.month = 2026, time.November
	a.calendar.render()
	a.pages.SwitchToPage(pageCalendar)

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	const w, h = 100, 30
	a.tapp.SetScreen(sim)
	sim.SetSize(w, h)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = a.tapp.Run()
	}()
	a.tapp.QueueUpdateDraw(func() {})

	// Последняя неделя (день 30, понедельник) целиком в панели: шесть недель
	// одного размера, остаток высоты уходит вниз (V-48).
	_, iy, iw, ih := a.calendar.grid.GetInnerRect()
	row, col := cellOf(a.calendar.grid.cells, 30)
	_, cy, _, _ := dayCellRect(row, col, iw, ih)
	if runeIndex(simRow(sim, iy+cy+1), "30") < 0 {
		t.Errorf("последняя неделя не у низа панели: %q", string(simRow(sim, iy+cy+1)))
	}

	a.tapp.Stop()
	<-done
}

// TestMonthGridStyles: палитра ячеек (V-33, V-40, V-45) — выбранный день залит
// целиком colorSelectedBg, «сегодня» — акцентный номер на фоне панели (без
// заливки), Сб/Вс — colorWeekend, «·N» — colorTextSecondary (на акценте —
// colorSelectedFg). Цвета — RGB-роли, поэтому на 256-цветном терминале
// и в truecolor они одни и те же.
func TestMonthGridStyles(t *testing.T) {
	g := newMonthGrid()
	g.SetBorder(true)
	const w, h = 72, 20
	g.SetRect(0, 0, w, h)
	cells := MonthGrid(2026, time.November, nil)
	for _, ev := range []struct{ day, events int }{{17, 2}, {20, 3}} {
		row, col := cellOf(cells, ev.day)
		cells[row][col].Events = ev.events
	}
	g.SetCells(cells, 17) // «сегодня» — 17 ноября
	g.SelectDay(20)       // выбранный — 20 ноября (пятница)

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	sim.SetSize(w, h)
	g.Draw(sim)

	ix, iy, iw, ih := g.GetInnerRect()
	cell := func(day int) (x, y, cw int) {
		row, col := cellOf(cells, day)
		cx, cy, cw, _ := dayCellRect(row, col, iw, ih)
		return ix + cx, iy + cy, cw
	}
	styleAt := func(x, y int) tcell.Style {
		_, _, st, _ := sim.GetContent(x, y)
		return st
	}

	// Шапка дней недели — второстепенный текст.
	mx, my := gridMargins(iw, ih)
	if fg, _, _ := styleAt(ix+mx, iy+my).Decompose(); fg != colorTextSecondary {
		t.Errorf("дни недели: fg=%v, хочу colorTextSecondary", fg)
	}

	// Выбранный день: сплошная заливка на всю ширину до линий, текст — с
	// отступом 1 от края, onAccent, жирный (V-44).
	x, y, cw := cell(20)
	if r, _, _, _ := sim.GetContent(x, y); r != ' ' {
		t.Errorf("в углу выбранной ячейки %q, ждали заливку без текста", r)
	}
	if r, _, _, _ := sim.GetContent(x+1, y); r != '2' {
		t.Errorf("в выбранной ячейке %q, ждали число 20", r)
	}
	fg, bg, attr := styleAt(x+1, y).Decompose()
	if fg != colorSelectedFg || bg != colorSelectedBg || attr&tcell.AttrBold == 0 {
		t.Errorf("выбранный день: fg/bg/bold=%v/%v/%v, хочу %v/%v/bold",
			fg, bg, attr&tcell.AttrBold != 0, colorSelectedFg, colorSelectedBg)
	}
	if _, blankBg, _ := styleAt(x+cw-1, y).Decompose(); blankBg != colorSelectedBg {
		t.Errorf("выбранный день залит не целиком: пустая клетка bg=%v", blankBg)
	}
	if fg, _, cattr := styleAt(x+3, y).Decompose(); fg != colorSelectedFg || cattr&tcell.AttrBold != 0 {
		t.Errorf("счётчик на акценте: fg=%v bold=%v, хочу %v/regular",
			fg, cattr&tcell.AttrBold != 0, colorSelectedFg)
	}

	// «Сегодня»: акцентный номер на фоне панели (V-45), счётчик — secondary;
	// заливки нет — угловая клетка остаётся цветом панели.
	x, y, cw = cell(17)
	fg, bg, attr = styleAt(x+1, y).Decompose()
	if fg != colorAccent || bg != colorPanelBg || attr&tcell.AttrBold == 0 {
		t.Errorf("сегодня: fg=%v bg=%v bold=%v, хочу %v/%v/bold",
			fg, bg, attr&tcell.AttrBold != 0, colorAccent, colorPanelBg)
	}
	if _, blankBg, _ := styleAt(x+cw-1, y).Decompose(); blankBg != colorPanelBg {
		t.Errorf("«сегодня» с заливкой: фон пустой клетки %v", blankBg)
	}
	fg, bg, cattr := styleAt(x+3, y).Decompose()
	if fg != colorTextSecondary || bg != colorPanelBg || cattr&tcell.AttrBold != 0 {
		t.Errorf("счётчик сегодня: fg=%v bg=%v bold=%v, хочу secondary/%v/regular",
			fg, bg, cattr&tcell.AttrBold != 0, colorPanelBg)
	}

	// Выходной (21 ноября — суббота) — цвет выходного, номер тоже жирный.
	x, y, _ = cell(21)
	if fg, _, attr := styleAt(x+1, y).Decompose(); fg != colorWeekend || attr&tcell.AttrBold == 0 {
		t.Errorf("выходной: fg=%v bold=%v, хочу colorWeekend/bold", fg, attr&tcell.AttrBold != 0)
	}

	// V-39/V-44: номер любого дня жирный и с отступом в 1 клетку от левого
	// края — без %2d-паддинга однозначное и двузначное число начинаются
	// одинаково.
	for _, day := range []int{5, 12} {
		x, y, _ := cell(day)
		r, _, st, _ := sim.GetContent(x+1, y)
		bold := func() tcell.AttrMask { _, _, a := st.Decompose(); return a & tcell.AttrBold }
		if r != []rune(fmt.Sprint(day))[0] || bold() == 0 {
			t.Errorf("день %d: первый символ %q bold=%v, ждали цифру с отступом 1 жирным", day, r, bold() != 0)
		}
	}

	// V-45: при совпадении «сегодня» и выбора побеждает выбранный — сплошной
	// акцент/onAccent, а не акцентный номер на фоне панели.
	g.SelectDay(17)
	g.Draw(sim)
	x, y, _ = cell(17)
	if fg, bg, attr := styleAt(x+1, y).Decompose(); fg != colorSelectedFg || bg != colorSelectedBg || attr&tcell.AttrBold == 0 {
		t.Errorf("сегодня+выбор: fg/bg=%v/%v, хочу %v/%v", fg, bg, colorSelectedFg, colorSelectedBg)
	}
}

// TestMonthGridSeparators (V-38, V-43): линии между колонками и неделями и под
// шапкой дней недели — colorGridLine (светлее чисел, темнее hairline);
// заливка выбранного лежит внутри линий и не затирает число дня.
func TestMonthGridSeparators(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {220, 60}} {
		w, h := size.w, size.h
		g := newMonthGrid()
		g.SetBorder(true)
		g.SetRect(0, 0, w, h)
		g.SetCells(MonthGrid(2026, time.November, nil), 0)
		g.SelectDay(20) // пятница, 4-я колонка
		sim := tcell.NewSimulationScreen("UTF-8")
		if err := sim.Init(); err != nil {
			t.Fatalf("инициализация симуляции: %v", err)
		}
		sim.SetSize(w, h)
		g.Draw(sim)

		ix, iy, iw, ih := g.GetInnerRect()
		mx, my := gridMargins(iw, ih)
		cw, ch := gridCellSize(iw, ih)
		content := func(x, y int) (rune, tcell.Color) {
			r, _, st, _ := sim.GetContent(ix+x, iy+y)
			fg, _, _ := st.Decompose()
			return r, fg
		}
		// Вертикаль между 1-й и 2-й колонками — на всю высоту сетки, включая шапку.
		for yy := my; yy < my+6*ch+7; yy++ {
			if r, fg := content(mx+cw, yy); (r != tview.Borders.Vertical && r != tview.Borders.Cross) || fg != colorGridLine {
				t.Fatalf("%dx%d: вертикаль (%d,%d) = %q/%v", w, h, mx+cw, yy, r, fg)
			}
		}
		// Горизонталь под шапкой дней недели — на всю ширину сетки.
		for xx := mx; xx < mx+7*cw+6; xx++ {
			if r, fg := content(xx, my+1); (r != tview.Borders.Horizontal && r != tview.Borders.Cross) || fg != colorGridLine {
				t.Fatalf("%dx%d: горизонталь (%d,%d) = %q/%v", w, h, xx, my+1, r, fg)
			}
		}
		// V-44: выбранный день залит от линии до линии — левый верхний угол
		// ячейки тоже заливка (пробел), число — с отступом 1 вправо/вниз;
		// слева от ячейки — вертикальная линия, а не заливка выбора.
		row, col := cellOf(g.cells, 20)
		cx, cy, cw2, ch2 := dayCellRect(row, col, iw, ih)
		textY := cy + 1 // высота ячейки здесь > 1
		if r, _, st, _ := sim.GetContent(ix+cx, iy+cy); r != ' ' || !isSelectedBg(st) {
			t.Fatalf("%dx%d: угол выбранной ячейки (%d,%d) = %q/%v, ждали заливку",
				w, h, cx, cy, r, st)
		}
		if r, fg := content(cx+1, textY); r != '2' || fg != colorSelectedFg {
			t.Fatalf("%dx%d: число выбранного (%d,%d) = %q/%v", w, h, cx+1, textY, r, fg)
		}
		if _, _, st, _ := sim.GetContent(ix+cx+cw2-1, iy+cy+ch2-1); !isSelectedBg(st) {
			t.Fatalf("%dx%d: низ-право выбранной ячейки не залит", w, h)
		}
		// V-48: край заливки проходит там, где стоит глиф линии: клетки линий
		// вокруг выбора — полублоки/четвертьблоки, в клетку соседа не лезем.
		blockAt := func(x, y int) (rune, tcell.Color, tcell.Color) {
			r, _, st, _ := sim.GetContent(ix+x, iy+y)
			fg, bg, _ := st.Decompose()
			return r, fg, bg
		}
		for _, p := range []struct {
			x, y int
			r    rune
		}{
			{cx - 1, cy + 1, '\u2590'}, {cx + cw2, cy + 1, '\u258c'},
			{cx + 1, cy - 1, '\u2584'}, {cx + 1, cy + ch2, '\u2580'},
			{cx - 1, cy - 1, '\u2597'}, {cx + cw2, cy - 1, '\u2596'},
			{cx - 1, cy + ch2, '\u259d'}, {cx + cw2, cy + ch2, '\u2598'},
		} {
			if r, fg, bg := blockAt(p.x, p.y); r != p.r || fg != colorSelectedBg || bg != colorPanelBg {
				t.Fatalf("%dx%d: край выбора (%d,%d) = %q/%v/%v, ждали %q",
					w, h, p.x, p.y, r, fg, bg, p.r)
			}
		}
		// Далёкая линия остаётся линией на фоне панели.
		if r, fg := content(mx+cw, cy+1); r != tview.Borders.Vertical || fg != colorGridLine {
			t.Fatalf("%dx%d: линия вне выбора задета: %q/%v", w, h, r, fg)
		}
		// V-48: крайняя левая колонка залита от самой рамки (поля нет),
		// заливка занимает все четыре угла ячейки.
		g.SelectDay(16) // понедельник — колонка 0
		g.Draw(sim)
		row, col = cellOf(g.cells, 16)
		cx, cy, cw2, ch2 = dayCellRect(row, col, iw, ih)
		if cx != 0 {
			t.Fatalf("%dx%d: левая колонка не у рамки: x=%d", w, h, cx)
		}
		for _, p := range []struct{ x, y int }{
			{cx, cy}, {cx + cw2 - 1, cy},
			{cx, cy + ch2 - 1}, {cx + cw2 - 1, cy + ch2 - 1},
		} {
			if _, _, st, _ := sim.GetContent(ix+p.x, iy+p.y); !isSelectedBg(st) {
				t.Fatalf("%dx%d: угол выбранной ячейки (%d,%d) не залит", w, h, p.x, p.y)
			}
		}
		// Клетка рамки вплотную к выбору — тоже половина заливки; глиф рамки
		// там замещён полублоком (иначе край выбора не совпал бы с рамкой).
		if r, fg, bg := blockAt(-1, cy+1); r != '\u2590' || fg != colorSelectedBg || bg != colorPanelBg {
			t.Fatalf("%dx%d: рамка у выбора = %q/%v/%v", w, h, r, fg, bg)
		}
		// Вне выбора рамка не тронута.
		if r, _, _ := blockAt(-1, 3); r != tview.Borders.Vertical {
			t.Fatalf("%dx%d: рамка вне выбора испорчена: %q", w, h, r)
		}
	}
}

// isSelectedBg — фон ячейки равен colorSelectedBg (V-44).
func isSelectedBg(st tcell.Style) bool {
	_, bg, _ := st.Decompose()
	return bg == colorSelectedBg
}
