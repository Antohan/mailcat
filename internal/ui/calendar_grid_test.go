package ui

// V-32: monthGrid растягивает 7 колонок и 6 недель на всю панель,
// навигация зажата границами месяца, клик выбирает только день месяца.

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestDayCellRectGeometry(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 30}, {220, 60}} {
		w, h := size.w, size.h
		if _, top, _, _ := dayCellRect(0, 0, w, h); top != 1 {
			t.Errorf("%dx%d: шапка выше одной строки: top=%d", w, h, top)
		}
		prevBottom := 1
		for row := 0; row < 6; row++ {
			_, top, _, ch := dayCellRect(row, 0, w, h)
			if top != prevBottom || ch <= 0 {
				t.Errorf("%dx%d: неделя %d не стыкуется: top=%d bottom=%d", w, h, row, top, prevBottom)
			}
			prevBottom = top + ch
		}
		if prevBottom != h {
			t.Errorf("%dx%d: недели не доходят до низа: %d != %d", w, h, prevBottom, h)
		}
		x := 0
		for col := 0; col < 7; col++ {
			cx, _, cw, _ := dayCellRect(0, col, w, h)
			if cx != x || cw <= 0 {
				t.Errorf("%dx%d: колонка %d не стыкуется: x=%d w=%d", w, h, col, cx, cw)
			}
			x += cw
		}
		if x != w {
			t.Errorf("%dx%d: колонки не покрывают ширину: %d != %d", w, h, x, w)
		}
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

	// Последняя неделя (день 30, понедельник) — у самого низа внутренней
	// области сетки: 6 недель делят остаток высоты, а не жмутся к шапке.
	if runeIndex(simRow(sim, h-7), "30") < 0 {
		t.Errorf("последняя неделя не у низа панели: %q", string(simRow(sim, h-7)))
	}

	a.tapp.Stop()
	<-done
}
