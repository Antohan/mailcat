package ui

// Сетка месяца (V-32): вместо tview.Table — собственный Box + Draw, чтобы
// 7 равных колонок и 6 недель растягивались на всю высоту панели, а день
// и счётчик встреч стояли в левом верхнем углу ячейки.

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// weekdayNames — шапка сетки, Пн — первая колонка.
var weekdayNames = [...]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

// weekColWidth — ширина колонки номеров недель (V-52): отступ 1 и две цифры.
const weekColWidth = 3

// monthGrid — месячная сетка: данные ([][]DayCell), ISO-номера недель строк и
// состояние выбора. onSelect зовётся при каждой смене выбранного дня —
// повестка перестраивается.
type monthGrid struct {
	*tview.Box
	cells    [][]DayCell
	weeks    []int
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

// SetCells раскладывает новую сетку; weeks — ISO-номера недель её строк,
// today — число месяца для маркера «сегодня» (0 — месяц не текущий).
func (g *monthGrid) SetCells(cells [][]DayCell, weeks []int, today int) {
	g.cells = cells
	g.weeks = weeks
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
	if last := g.maxDay(); day > last {
		day = last
	}
	g.SelectDay(day)
}

func (g *monthGrid) maxDay() int {
	last := 0
	for _, row := range g.cells {
		for _, c := range row {
			if c.InMonth && c.Day > last {
				last = c.Day
			}
		}
	}
	return last
}

// gridCellSize — размер ячейки: колонка номеров недель (V-52) и 7 колонок
// дней разделены 7 вертикальными линиями, а по высоте — шапка дней недели,
// линия под ней и 5 линий между неделями.
func gridCellSize(w, h int) (cw, ch int) {
	return (w - weekColWidth - 7) / 7, (h - 7) / 6
}

// gridMargins — внешние поля сетки: остаток ширины/высоты делится поровну
// по краям, чтобы колонка недель, все 7 колонок и 6 недель были одного размера.
func gridMargins(w, h int) (mx, my int) {
	cw, ch := gridCellSize(w, h)
	return (w - (weekColWidth + 7*cw + 7)) / 2, (h - (6*ch + 7)) / 2
}

// dayCellRect — внутренний прямоугольник ячейки недели row (0–5) и колонки
// col (0–6) в координатах сетки: над неделями — поле my, строка шапки дней
// недели и линия под ней; слева — колонка номеров недель и линия после неё;
// между неделями и колонками — по клетке линии (V-38, V-52).
// WHY: остаток ширины/высоты уходит в поля, а не в седьмую колонку/низ —
// иначе ячейки разного размера и число дня стоит в разных точках.
// Не делать: отдавать остаток последней колонке/неделе (так было до V-37).
func dayCellRect(row, col, w, h int) (x, y, cw, ch int) {
	cw, ch = gridCellSize(w, h)
	mx, my := gridMargins(w, h)
	return mx + weekColWidth + 1 + col*(cw+1), my + 2 + row*(ch+1), cw, ch
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
	g.DrawForSubclass(screen, g)
	x, y, w, h := g.GetInnerRect()
	if w <= 0 || len(g.cells) == 0 {
		return
	}
	if cw, ch := gridCellSize(w, h); cw <= 0 || ch <= 0 {
		return
	}
	headStyle := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
	mx, my := gridMargins(w, h)
	for col, name := range weekdayNames {
		cx, _, cw, _ := dayCellRect(0, col, w, h)
		gridText(screen, x+cx, y+my, cw, name, headStyle)
	}
	todayRow := g.todayWeekRow()
	for row := 0; row < 6 && row < len(g.cells); row++ {
		if row < len(g.weeks) {
			// V-52: ISO-номер недели строки — muted, текущая неделя акцентом;
			// отступ как у числа дня: слева 1 клетка, вниз 1 при ch>1.
			weekStyle := tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg)
			if row == todayRow {
				weekStyle = weekStyle.Foreground(colorAccent)
			}
			label := fmt.Sprintf("%d", g.weeks[row])
			_, cy, _, ch := dayCellRect(row, 0, w, h)
			ty := cy
			if ch > 1 {
				ty = cy + 1
			}
			gridText(screen, x+mx+1, y+ty, weekColWidth-1, label, weekStyle)
		}
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
}

// drawGridLines — разделители сетки (V-38): вертикали после колонки номеров
// недель и между колонками дней (включая строку шапки), горизонтали под
// шапкой дней недели и между неделями. Заливка выбранного лежит внутри
// dayCellRect, поэтому линии её не затирают.
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
	right := mx + weekColWidth + 7*cw + 7
	// Семь вертикалей: после колонки недель и шесть между днями.
	verts := make([]int, 7)
	for i := range verts {
		verts[i] = mx + weekColWidth + i*(cw+1)
	}
	for _, lx := range verts {
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
	for _, lx := range verts {
		for i := 0; i <= 5; i++ {
			screen.SetContent(x+lx, y+my+1+i*(ch+1), tview.Borders.Cross, nil, rule)
		}
	}
}

// todayWeekRow — строка сетки, в которой стоит «сегодня», или -1 (месяц не
// текущий): акцентом красится номер именно её недели (V-52).
// WHY: сравнивать номер недели с time.Now().ISOWeek() нельзя — в чужом году
// неделя 44 тоже бывает; искать строку по today детерминированно.
func (g *monthGrid) todayWeekRow() int {
	if g.today == 0 {
		return -1
	}
	for row, cells := range g.cells {
		for _, c := range cells {
			if c.InMonth && c.Day == g.today {
				return row
			}
		}
	}
	return -1
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
