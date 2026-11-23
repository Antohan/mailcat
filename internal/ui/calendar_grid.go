package ui

// Сетка месяца (V-32): вместо tview.Table — собственный Box + Draw, чтобы
// 7 равных колонок и 6 недель растягивались на всю высоту панели, а день
// и счётчик встреч стояли в левом верхнем углу ячейки.

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// weekdayNames — шапка сетки, Пн — первая колонка.
var weekdayNames = [...]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

// monthGrid — месячная сетка: данные ([][]DayCell) и состояние выбора.
// onSelect зовётся при каждой смене выбранного дня — повестка перестраивается.
type monthGrid struct {
	*tview.Box
	cells    [][]DayCell
	today    int // день месяца, если открыт текущий месяц; иначе 0
	selRow   int
	selCol   int
	hasSel   bool
	onSelect func(row, col int)
}

func newMonthGrid() *monthGrid {
	g := &monthGrid{Box: tview.NewBox()}
	g.SetBackgroundColor(colorPanelBg)
	return g
}

// SetCells раскладывает новую сетку; today — число месяца для маркера
// «сегодня» (0 — месяц не текущий).
func (g *monthGrid) SetCells(cells [][]DayCell, today int) {
	g.cells = cells
	g.today = today
}

// SetSelectFunc — подписка на смену выбранного дня.
func (g *monthGrid) SetSelectFunc(fn func(row, col int)) { g.onSelect = fn }

// SelectDay выбирает день месяца; день, которого нет в сетке, игнорируется.
func (g *monthGrid) SelectDay(day int) {
	row, col := cellOf(g.cells, day)
	if row < 0 {
		return
	}
	g.selectCell(row, col)
}

// selectCell — единственная точка смены выбора: состояние плюс onSelect.
func (g *monthGrid) selectCell(row, col int) {
	g.selRow, g.selCol, g.hasSel = row, col, true
	if g.onSelect != nil {
		g.onSelect(row, col)
	}
}

// Move сдвигает выбор на delta дней (←/→ ±1, ↑/↓ ±7) с зажимом в месяце:
// за 1-е и последнее число не выходим, пустые ячейки недостижимы.
func (g *monthGrid) Move(delta int) {
	if !g.hasSel || len(g.cells) == 0 {
		return
	}
	day := g.cells[g.selRow][g.selCol].Day + delta
	if day < 1 {
		day = 1
	}
	if max := g.maxDay(); day > max {
		day = max
	}
	g.SelectDay(day)
}

func (g *monthGrid) maxDay() int {
	max := 0
	for _, row := range g.cells {
		for _, c := range row {
			if c.Day > max {
				max = c.Day
			}
		}
	}
	return max
}

// gridMargins — внешние поля сетки: остаток ширины/высоты делится поровну
// по краям, чтобы все 7 колонок и 6 недель были одного размера.
func gridMargins(w, h int) (mx, my int) {
	return (w - 7*(w/7)) / 2, (h - 1 - 6*((h-1)/6)) / 2
}

// dayCellRect — прямоугольник ячейки недели row (0–5) и колонки col (0–6)
// в координатах сетки: над неделями — поле my и строка шапки дней недели.
// WHY: остаток ширины/высоты уходит в поля, а не в седьмую колонку/низ —
// иначе ячейки разного размера и число дня стоит в разных точках.
// Не делать: отдавать остаток последней колонке/неделе (так было до V-37).
func dayCellRect(row, col, w, h int) (x, y, cw, ch int) {
	cw = w / 7
	ch = (h - 1) / 6
	mx, my := gridMargins(w, h)
	return mx + col*cw, my + 1 + row*ch, cw, ch
}

// dayAt — ячейка под точкой (px, py) в координатах сетки.
func dayAt(px, py, w, h int) (row, col int, ok bool) {
	for r := 0; r < 6; r++ {
		for c := 0; c < 7; c++ {
			x, y, cw, ch := dayCellRect(r, c, w, h)
			if cw > 0 && ch > 0 && px >= x && px < x+cw && py >= y && py < y+ch {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// Draw рисует шапку дней недели и дни: число и «·N» — в левом верхнем углу
// ячейки; выбранный день залит colorSelectedBg целиком, «сегодня» — tint
// colorTodayBg + bold, Сб/Вс — colorWeekend (V-33).
func (g *monthGrid) Draw(screen tcell.Screen) {
	g.Box.DrawForSubclass(screen, g)
	x, y, w, h := g.GetInnerRect()
	if w <= 0 || h < 2 || len(g.cells) == 0 {
		return
	}
	headStyle := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	_, my := gridMargins(w, h)
	for col, name := range weekdayNames {
		cx, _, cw, _ := dayCellRect(0, col, w, h)
		gridText(screen, x+cx, y+my, cw, name, headStyle)
	}
	for row := 0; row < 6 && row < len(g.cells); row++ {
		for col := 0; col < 7; col++ {
			cell := g.cells[row][col]
			if cell.Day == 0 {
				continue
			}
			cx, cy, cw, ch := dayCellRect(row, col, w, h)
			if cw <= 0 || ch <= 0 {
				continue
			}
			dayStyle, countStyle := g.cellStyles(row, col)
			for yy := cy; yy < cy+ch; yy++ {
				for xx := cx; xx < cx+cw; xx++ {
					screen.SetContent(x+xx, y+yy, ' ', nil, dayStyle)
				}
			}
			label := FormatDayCell(cell)
			dayText, count, _ := strings.Cut(label, "·")
			gridText(screen, x+cx, y+cy, cw, dayText, dayStyle)
			if count != "" {
				dx := len([]rune(dayText))
				gridText(screen, x+cx+dx, y+cy, cw-dx, "·"+count, countStyle)
			}
		}
	}
}

// cellStyles — стили числа и счётчика «·N» одной ячейки. Состояния
// накладываются по порядку: выходной → «сегодня» → выбранный.
// WHY: счётчик — второстепенный текст, но на акцентной заливке он берёт
// colorSelectedFg: colorTextSecondary на акценте нечитаем.
// Не делать: один стиль на всю строку — счётчик перестал бы отличаться.
func (g *monthGrid) cellStyles(row, col int) (day, count tcell.Style) {
	day = tcell.StyleDefault.Foreground(colorTextPrimary).Background(colorPanelBg)
	count = tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	if col >= 5 {
		day = day.Foreground(colorWeekend) // Сб/Вс
	}
	if g.cells[row][col].Day == g.today {
		// О-3: «сегодня» — светлый оттенок акцента, сплошной занят выбором.
		day = day.Bold(true).Background(colorTodayBg)
		count = count.Background(colorTodayBg)
	}
	if g.hasSel && row == g.selRow && col == g.selCol {
		day = tcell.StyleDefault.Foreground(colorSelectedFg).Background(colorSelectedBg)
		count = tcell.StyleDefault.Foreground(colorSelectedFg).Background(colorSelectedBg)
	}
	return day, count
}

// gridText рисует строку без tview-разметки, обрезая по ширине ячейки:
// tview.Print подставил бы терминальный фон и затёр заливку выбранной ячейки.
func gridText(screen tcell.Screen, x, y, width int, text string, style tcell.Style) {
	for i, r := range []rune(text) {
		if i >= width {
			return
		}
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// MouseHandler — клик выбирает ячейку дня (V-32). Пустые ячейки соседних
// месяцев невыбираемы; фокус переходит панели сетки.
func (g *monthGrid) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}
		mx, my := event.Position()
		x, y, w, h := g.GetInnerRect()
		row, col, ok := dayAt(mx-x, my-y, w, h)
		if !ok || row >= len(g.cells) || g.cells[row][col].Day == 0 {
			return false, nil
		}
		setFocus(g)
		g.selectCell(row, col)
		return true, nil
	}
}
