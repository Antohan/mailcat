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
			if c.InMonth && c.Day > max {
				max = c.Day
			}
		}
	}
	return max
}

// gridCellSize — размер ячейки: 7 колонок разделены 6 вертикальными линиями,
// а по высоте — шапка дней недели, линия под ней и 5 линий между неделями.
func gridCellSize(w, h int) (cw, ch int) {
	return (w - 6) / 7, (h - 7) / 6
}

// gridMargins — внешние поля сетки: сетка прижата к левому верхнему углу
// панели, остаток ширины/высоты остаётся справа/снизу.
// WHY: деление остатка по краям (до V-48) оставляло полосу фона панели между
// заливкой крайней колонки и рамкой, а сверху — пустые строки.
// Не делать: возвращать центрирование — выбор снова отрывается от рамки.
func gridMargins(w, h int) (mx, my int) {
	return 0, 0
}

// dayCellRect — внутренний прямоугольник ячейки недели row (0–5) и колонки
// col (0–6) в координатах сетки: над неделями — поле my, строка шапки дней
// недели и линия под ней; между неделями и колонками — по клетке линии (V-38).
// WHY: остаток ширины/высоты уходит в поля, а не в седьмую колонку/низ —
// иначе ячейки разного размера и число дня стоит в разных точках.
// Не делать: отдавать остаток последней колонке/неделе (так было до V-37).
func dayCellRect(row, col, w, h int) (x, y, cw, ch int) {
	cw, ch = gridCellSize(w, h)
	mx, my := gridMargins(w, h)
	return mx + col*(cw+1), my + 2 + row*(ch+1), cw, ch
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

// Draw рисует шапку дней недели и дни: число (жирным) и «·N» — с отступом
// 1 клетка от левого/верхнего края ячейки; выбранный день залит
// colorSelectedBg от линии до линии, «сегодня» — акцентный номер без заливки
// (V-33, V-39, V-44, V-45).
func (g *monthGrid) Draw(screen tcell.Screen) {
	g.Box.DrawForSubclass(screen, g)
	x, y, w, h := g.GetInnerRect()
	if w <= 0 || len(g.cells) == 0 {
		return
	}
	if cw, ch := gridCellSize(w, h); cw <= 0 || ch <= 0 {
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
			// WHY: заливка — весь dayCellRect (до линий), а текст с отступом 1:
			// вплотную к линии число сливалось с сеткой, выбранный блок
			// выглядел обрезанным. Отступ общий для всех дней — числа в одной
			// колонке стоят ровно.
			// Не делать: заливать только под текстом — «сегодня»/выбор теряли
			// форму дня между линиями.
			tx, ty := cx+1, cy
			// Ячейка высотой в 1 клетку: отступ вниз пришёлся бы на линию
			// недели, поэтому текст остаётся в единственной строке.
			if ch > 1 {
				ty = cy + 1
			}
			gridText(screen, x+tx, y+ty, cw-1, dayText, dayStyle)
			if count != "" {
				dx := len([]rune(dayText))
				gridText(screen, x+tx+dx, y+ty, cw-1-dx, "·"+count, countStyle)
			}
		}
	}
	g.drawGridLines(screen, x, y, w, h)
	// V-48: обводка выбора — поверх линий, иначе они её затирают.
	g.fillSelRing(screen, x, y, w, h)
}

// selRing — клетки-линии вокруг выбранного дня, включая пересечения (V-48).
func (g *monthGrid) selRing(w, h int) (x0, y0, x1, y1 int, ok bool) {
	if !g.hasSel {
		return 0, 0, 0, 0, false
	}
	cx, cy, cw, ch := dayCellRect(g.selRow, g.selCol, w, h)
	return cx - 1, cy - 1, cx + cw, cy + ch, true
}

// fillSelRing — обводка выбора «вровень» с линиями (V-48): клетки-линий вокруг
// выбранной заливаются наполовину, поэтому край заливки проходит там, где стоит
// глиф линии/рамки, и не наезжает на клетку соседа.
// WHY: линия живёт в своей клетке и рисуется по центру — целая заливка этой
// клетки давала выступ, а без неё — полосу фона панели до линии.
// Не делать: заливать клетки линий целиком — выделение вылезает за рамки.
func (g *monthGrid) fillSelRing(screen tcell.Screen, x, y, w, h int) {
	x0, y0, x1, y1, ok := g.selRing(w, h)
	if !ok {
		return
	}
	cx, cy, cw, ch := dayCellRect(g.selRow, g.selCol, w, h)
	edge := tcell.StyleDefault.Foreground(colorSelectedBg).Background(colorPanelBg)
	put := func(xx, yy int, r rune) {
		if xx < 0 || yy < 0 || xx >= w || yy >= h {
			return
		}
		screen.SetContent(x+xx, y+yy, r, nil, edge)
	}
	for xx := cx; xx < cx+cw; xx++ {
		put(xx, y0, '▄') // верхняя линия: заливка в нижнюю половину
		put(xx, y1, '▀')
	}
	for yy := cy; yy < cy+ch; yy++ {
		put(x0, yy, '▐') // левая линия: заливка в правую половину
		put(x1, yy, '▌')
	}
	put(x0, y0, '▗')
	put(x1, y0, '▖')
	put(x0, y1, '▝')
	put(x1, y1, '▘')
	// Рамка панели лежит вне inner rect — её клетку красим так же.
	if cx == 0 {
		for yy := y0; yy <= y1; yy++ {
			screen.SetContent(x-1, y+yy, '▐', nil, edge)
		}
	}
	if cx+cw == w {
		for yy := y0; yy <= y1; yy++ {
			screen.SetContent(x+w, y+yy, '▌', nil, edge)
		}
	}
}

// drawGridLines — разделители сетки (V-38): вертикали между колонками (включая
// строку шапки), горизонтали под шапкой дней недели и между неделями. Заливка
// выбранного лежит внутри dayCellRect, поэтому линии её не затирают.
// WHY: отдельная роль colorGridLine (V-43) — hairline на белой поверхности
// не читается, а textSecondary (цвет чисел и подписей) слишком тяжёлый:
// линии спорили с числами. Роль светлее чисел и темнее hairline; пустые и
// соседние ячейки получают те же тихие линии — без отдельного акцента.
// Не делать: возвращать textSecondary — сетка снова перевесит числа.
func (g *monthGrid) drawGridLines(screen tcell.Screen, x, y, w, h int) {
	mx, my := gridMargins(w, h)
	cw, ch := gridCellSize(w, h)
	rule := tcell.StyleDefault.Foreground(colorGridLine).Background(colorPanelBg)
	bottom := my + 6*ch + 7
	right := mx + 7*cw + 6
	for col := 0; col < 6; col++ {
		lx := mx + (col+1)*cw + col
		for yy := my; yy < bottom; yy++ {
			screen.SetContent(x+lx, y+yy, tview.Borders.Vertical, nil, rule)
		}
	}
	for i := 0; i <= 5; i++ {
		ly := my + 1 + i*(ch+1)
		for xx := mx; xx < right; xx++ {
			screen.SetContent(x+xx, y+ly, tview.Borders.Horizontal, nil, rule)
		}
	}
	// Пересечения — крестиком, иначе линия недели рвёт вертикаль.
	for col := 0; col < 6; col++ {
		lx := mx + (col+1)*cw + col
		for i := 0; i <= 5; i++ {
			ly := my + 1 + i*(ch+1)
			screen.SetContent(x+lx, y+ly, tview.Borders.Cross, nil, rule)
		}
	}
}

// cellStyles — стили числа и счётчика «·N» одной ячейки. Состояния
// накладываются по порядку: выходной → «сегодня» → выбранный.
// WHY: номер дня жирный у всех дней (V-39) — шрифт терминала не масштабируется.
// Счётчик — обычный: «·N» второстепенен; на акценте берёт colorSelectedFg
// (colorTextSecondary на акценте нечитаем).
// Не делать: один стиль на всю строку — счётчик перестал бы отличаться.
func (g *monthGrid) cellStyles(row, col int) (day, count tcell.Style) {
	day = tcell.StyleDefault.Foreground(colorTextPrimary).Background(colorPanelBg).Bold(true)
	count = tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	if col >= 5 {
		day = day.Foreground(colorWeekend) // Сб/Вс
	}
	if !g.cells[row][col].InMonth {
		// V-46: дни соседних месяцев — только отображение: muted и без
		// жирности, чтобы числа текущего месяца оставались главными.
		// Не делать: weekend-цвет для соседних — muted уже заглушает.
		day = day.Foreground(colorOtherMonth).Bold(false)
		count = count.Foreground(colorOtherMonth)
	}
	if g.cells[row][col].InMonth && g.cells[row][col].Day == g.today {
		// О-3/V-45: «сегодня» — акцентный номер на фоне панели; сплошной
		// акцент занят выбором (после совпадения выбор перекрывает этот стиль).
		// WHY: tint-подложка под номером читалась как second selection и
		// спорила с сеткой; без фона «сегодня» держится цветом и жирностью.
		// Не делать: возвращать заливку colorTodayBg — два залитых дня подряд.
		day = day.Foreground(colorAccent).Bold(true).Background(colorPanelBg)
		count = count.Foreground(colorTextSecondary).Background(colorPanelBg)
	}
	if g.hasSel && row == g.selRow && col == g.selCol {
		day = tcell.StyleDefault.Foreground(colorSelectedFg).Background(colorSelectedBg).Bold(true)
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

// MouseHandler — клик выбирает ячейку дня (V-32). Дни соседних месяцев
// невыбираемы (V-46); фокус переходит панели сетки.
func (g *monthGrid) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}
		mx, my := event.Position()
		x, y, w, h := g.GetInnerRect()
		row, col, ok := dayAt(mx-x, my-y, w, h)
		if !ok || row >= len(g.cells) || !g.cells[row][col].InMonth {
			return false, nil
		}
		setFocus(g)
		g.selectCell(row, col)
		return true, nil
	}
}
