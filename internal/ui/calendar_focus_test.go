package ui

// V-35: фокус, клавиши и подсказки календаря — сетка ↔ повестка.

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestCalendarFocusKeys: 1/2 переносит фокус в сетку, Enter уводит в
// повестку, Esc/← возвращают к сетке, Tab/Shift-Tab переключают панели —
// строка подсказок каждый раз соответствует фокусу.
func TestCalendarFocusKeys(t *testing.T) {
	a := newTestApp(t)
	a.calendar.year, a.calendar.month = 2026, time.November
	a.calendar.render()
	a.pages.SwitchToPage(pageCalendar)

	// Фокус остался в скрытом списке писем — ensureFocus ставит его на сетку.
	a.calendar.ensureFocus()
	a.showHint()
	if a.tapp.GetFocus() != a.calendar.grid {
		t.Fatalf("фокус не на сетке календаря: %T", a.tapp.GetFocus())
	}
	if got := a.hint.GetText(false); got != formatHint(hintCalendarGrid) {
		t.Errorf("подсказка сетки: %q", got)
	}

	gridHook := a.calendar.grid.GetInputCapture()
	agendaHook := a.calendar.agenda.GetInputCapture()

	// Enter на сетке — фокус в повестку.
	if got := gridHook(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)); got != nil {
		t.Fatal("Enter на сетке не поглощён")
	}
	if a.tapp.GetFocus() != a.calendar.agenda {
		t.Fatalf("Enter не увёл фокус в повестку: %T", a.tapp.GetFocus())
	}
	if got := a.hint.GetText(false); got != formatHint(hintCalendarAgenda) {
		t.Errorf("подсказка повестки: %q", got)
	}

	// Esc на повестке — назад в сетку.
	if got := agendaHook(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)); got != nil {
		t.Fatal("Esc на повестке не поглощён")
	}
	if a.tapp.GetFocus() != a.calendar.grid {
		t.Fatalf("Esc не вернул фокус сетке: %T", a.tapp.GetFocus())
	}
	if got := a.hint.GetText(false); got != formatHint(hintCalendarGrid) {
		t.Errorf("подсказка после Esc: %q", got)
	}

	// Tab с сетки → повестка, Shift-Tab обратно.
	gridHook(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	if a.tapp.GetFocus() != a.calendar.agenda {
		t.Fatalf("Tab не сменил фокус на повестку: %T", a.tapp.GetFocus())
	}
	agendaHook(tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone))
	if a.tapp.GetFocus() != a.calendar.grid {
		t.Fatalf("Shift-Tab не вернул фокус сетке: %T", a.tapp.GetFocus())
	}

	// ← на повестке — тоже к сетке (без Ctrl; Ctrl-← — прошлый месяц).
	a.tapp.SetFocus(a.calendar.agenda)
	if got := agendaHook(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)); got != nil {
		t.Fatal("← на повестке не поглощён")
	}
	if a.tapp.GetFocus() != a.calendar.grid {
		t.Fatalf("← не вернул фокус сетке: %T", a.tapp.GetFocus())
	}
}

// TestCalendarHintsFit: оба набора целиком укладываются в 100 колонок —
// opt-сегмент «t сегодня» (V-47) там не снимается, иначе пропала бы
// обязательная клавиша (V-17).
func TestCalendarHintsFit(t *testing.T) {
	for name, segs := range map[string][]hintSeg{
		"сетка":    hintCalendarGrid,
		"повестка": hintCalendarAgenda,
	} {
		full := formatHint(segs)
		if w := tview.TaggedStringWidth(full); w > 100 {
			t.Errorf("подсказки (%s) шире 100 колонок: %d", name, w)
		}
		// На 100 колонках formatHintFit не должен ронять ни одного сегмента.
		if got := formatHintFit(segs, 100); got != full {
			t.Errorf("на 100 колонках снят сегмент (%s): %q", name, got)
		}
	}
}
