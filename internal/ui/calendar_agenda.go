package ui

// Повестка дня (V-42): вместо tview.List — собственный Box + Draw, чтобы
// событие было блоком строк (V-51: две строки — время+тема / место), обёрнутых
// по ширине панели (tview.List обрезает длинную строку). Между событиями —
// пустая строка; прошедшие muted, выделение — весь блок акцентом; ↑↓ двигают
// события, Enter открывает детали, клик выбирает событие; полоса прокрутки —
// scrollbar.go.

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// agendaNow — текущее время повестки: по нему прошедшие встречи становятся
// muted (V-51); тесты закрепляют часы, чтобы стили не зависели от даты прогона.
var agendaNow = time.Now

// agendaSeg — кусок строки повестки одним стилем. Текст уже прошёл Sanitize;
// tview.Escape не применяется: сегменты рисуются через tcell, а не парсятся
// как разметка tview (WHY у AgendaSegments).
type agendaSeg struct {
	text  string
	style tcell.Style
}

// agendaRow — строка раскладки: сегменты одной линии и индекс события.
type agendaRow struct {
	segs []agendaSeg
	ev   int
}

// agendaList — повестка дня: items — строки событий без переноса (V-51: две
// строки на событие), раскладка по строкам считается под текущую ширину панели
// (Draw и ScrollTrack).
type agendaList struct {
	*tview.Box
	items  [][][]agendaSeg
	sel    int
	offset int // первая видимая строка раскладки

	rows  []agendaRow // раскладка последнего Draw/ScrollTrack
	rowW  int
	onSel func(int)
}

func newAgendaList() *agendaList {
	l := &agendaList{Box: tview.NewBox(), sel: -1}
	l.SetBackgroundColor(colorPanelBg)
	return l
}

// SetItems задаёт события дня (строки каждого) и выбранное. Прокрутка не
// сбрасывается: смена выбора пересобирает те же строки, а на новый день
// scrollToSel в Draw всё равно приведёт её к первой строке выбранного блока.
func (l *agendaList) SetItems(items [][][]agendaSeg, sel int) {
	l.items, l.sel = items, sel
	if l.sel >= len(items) {
		l.sel = len(items) - 1
	}
	if len(items) == 0 {
		l.sel, l.offset = -1, 0
	}
	l.rows = nil // раскладка от items устарела
}

// SetSelectFunc — подписка на смену выбранного события (↑↓, клик).
func (l *agendaList) SetSelectFunc(fn func(sel int)) { l.onSel = fn }

// Selected — индекс выбранного события (-1 — пустой день).
func (l *agendaList) Selected() int { return l.sel }

// Move сдвигает выбор на delta событий с зажимом; смена зовёт onSel.
func (l *agendaList) Move(delta int) {
	if len(l.items) == 0 || l.sel < 0 {
		return
	}
	next := l.sel + delta
	if next < 0 {
		next = 0
	}
	if next >= len(l.items) {
		next = len(l.items) - 1
	}
	if next == l.sel {
		return
	}
	l.sel = next
	if l.onSel != nil {
		l.onSel(next)
	}
}

// contentWidth — ширина текста: поля строк как у списка писем (listPadLeft/
// listPadRight), заливка выделения — на всю ширину панели.
func contentWidth(w int) int {
	if w -= listPadLeft + listPadRight; w < 1 {
		return 1
	}
	return w
}

// layout раскладывает строки событий в строки экрана под ширину width: между
// событиями — пустая строка (ev -1, не выбирается), пустая вторая строка
// события остаётся строкой (двухстрочный ритм V-51).
func (l *agendaList) layout(width int) {
	l.rowW = width
	l.rows = l.rows[:0]
	for i, lines := range l.items {
		if i > 0 {
			l.rows = append(l.rows, agendaRow{ev: -1})
		}
		for _, segs := range lines {
			wrapped := wrapSegments(segs, width)
			if len(wrapped) == 0 {
				wrapped = [][]agendaSeg{nil}
			}
			for _, line := range wrapped {
				l.rows = append(l.rows, agendaRow{segs: line, ev: i})
			}
		}
	}
}

// wrapSegments переносит сегменты по словам в строки ширины width (в рунах):
// слово длиннее строки режется принудительно. Пробелы нормализуются в один.
func wrapSegments(segs []agendaSeg, width int) [][]agendaSeg {
	var lines [][]agendaSeg
	var cur []agendaSeg
	curW := 0
	add := func(seg agendaSeg) {
		if n := len(cur); n > 0 && cur[n-1].style == seg.style {
			cur[n-1].text += seg.text
			return
		}
		cur = append(cur, seg)
	}
	flush := func() {
		if len(cur) > 0 {
			lines = append(lines, cur)
			cur, curW = nil, 0
		}
	}
	for _, seg := range segs {
		for _, word := range strings.Fields(seg.text) {
			runes := []rune(word)
			if curW > 0 && curW+1+len(runes) <= width {
				add(agendaSeg{" ", seg.style})
				curW++
			} else if curW > 0 {
				flush()
			}
			for len(runes) > 0 {
				avail := width - curW
				if avail <= 0 {
					flush()
					continue
				}
				if len(runes) <= avail {
					add(agendaSeg{string(runes), seg.style})
					curW += len(runes)
					break
				}
				add(agendaSeg{string(runes[:avail]), seg.style})
				flush()
				runes = runes[avail:]
			}
		}
	}
	flush()
	return lines
}

// scrollToSel держит выбранный блок в видимой области высотой h.
func (l *agendaList) scrollToSel(h int) {
	if l.sel < 0 {
		return
	}
	first, last := -1, -1
	for i, row := range l.rows {
		if row.ev == l.sel {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return
	}
	if first < l.offset {
		l.offset = first
	}
	if last >= l.offset+h {
		l.offset = last - h + 1
	}
	l.clampOffset(h)
}

// clampOffset не даёт прокрутке выйти за раскладку.
func (l *agendaList) clampOffset(h int) {
	maxOff := len(l.rows) - h
	if maxOff < 0 {
		maxOff = 0
	}
	if l.offset > maxOff {
		l.offset = maxOff
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

// ScrollTrack — прокрутка повестки для полосы: offset и всего строк.
func (l *agendaList) ScrollTrack() (offset, total int) {
	_, _, w, _ := l.GetInnerRect()
	l.layout(contentWidth(w))
	return l.offset, len(l.rows)
}

// emptyAgendaText — заглушка пустого дня (UI-строка, не из EWS).
const emptyAgendaText = "Встреч нет"

// Draw рисует видимые строки повестки: выбранный блок залит акцентом на всю
// ширину; пустой день — muted «Встреч нет».
func (l *agendaList) Draw(screen tcell.Screen) {
	l.Box.DrawForSubclass(screen, l)
	x, y, w, h := l.GetInnerRect()
	if w <= 0 || h <= 0 {
		return
	}
	width := contentWidth(w)
	l.layout(width)
	if len(l.rows) == 0 {
		l.offset = 0
		l.drawText(screen, x+listPadLeft, y, emptyAgendaText,
			tcell.StyleDefault.Foreground(colorTextSecondary).Background(colorPanelBg), width)
		return
	}
	l.scrollToSel(h)
	l.clampOffset(h)
	for i := 0; i < h && l.offset+i < len(l.rows); i++ {
		row := l.rows[l.offset+i]
		bg := colorPanelBg
		if row.ev == l.sel {
			bg = colorSelectedBg
		}
		for xx := 0; xx < w; xx++ {
			screen.SetContent(x+xx, y+i, ' ', nil, tcell.StyleDefault.Background(bg))
		}
		cx := x + listPadLeft
		for _, seg := range row.segs {
			style := seg.style.Background(bg)
			for _, r := range seg.text {
				if cx >= x+listPadLeft+width {
					break
				}
				screen.SetContent(cx, y+i, r, nil, style)
				cx++
			}
		}
	}
}

// drawText рисует строку без разметки tview, обрезая по ширине.
func (l *agendaList) drawText(screen tcell.Screen, x, y int, text string, style tcell.Style, width int) {
	for i, r := range []rune(text) {
		if i >= width {
			return
		}
		screen.SetContent(x+i, y, r, nil, style)
	}
}

// MouseHandler — клик выбирает событие под курсором (весь его блок строк) и
// передаёт фокус повестке; пустое место панели только фокусирует.
func (l *agendaList) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	return func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}
		mx, my := event.Position()
		x, y, w, h := l.GetInnerRect()
		if mx < x || mx >= x+w || my < y || my >= y+h {
			return false, nil
		}
		setFocus(l)
		l.layout(contentWidth(w))
		if row := my - y + l.offset; row >= 0 && row < len(l.rows) {
			// Зазор между событиями (ev -1) только фокусирует панель.
			if ev := l.rows[row].ev; ev >= 0 && ev != l.sel {
				l.sel = ev
				if l.onSel != nil {
					l.onSel(l.sel)
				}
			}
		}
		return true, nil
	}
}
