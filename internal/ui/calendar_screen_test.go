package ui

// V-30: у календаря нет ASCII-баннера месяца — месяц живёт в шапке панели,
// сетка начинается сразу под шапкой и занимает всю высоту колонки.

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestCalendarWithoutBanner(t *testing.T) {
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

	// Строка 0 — верхний бар, строка 1 — шапка панели с месяцем (V-31: без капса).
	if row := string(simRow(sim, 1)); runeIndex(simRow(sim, 1), "Ноябрь 2026") < 0 {
		t.Errorf("месяца нет в шапке панели: %q", row)
	}
	// V-31: справа в шапке сетки — счётчик встреч месяца.
	if row := string(simRow(sim, 1)); runeIndex(simRow(sim, 1), "0 встреч") < 0 {
		t.Errorf("счётчика встреч нет в шапке сетки: %q", row)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r, _, _, _ := sim.GetContent(x, y); r == '█' {
				t.Fatalf("баннер вернулся: '█' в (%d, %d)", x, y)
			}
		}
	}
	// Сетка начинается сразу под шапкой: рамка на y=2, а дни недели — через
	// верхнее поле сетки (V-37: остаток высоты уходит в поля, не в недели).
	if r, _, _, _ := sim.GetContent(0, 2); r != '┌' {
		t.Errorf("рамка сетки не под шапкой: (%d, 2) = %q", 0, r)
	}
	_, iy, iw, ih := a.calendar.grid.GetInnerRect()
	_, my := gridMargins(iw, ih)
	if row := simRow(sim, iy+my); runeIndex(row, "Пн") < 0 || runeIndex(row, "Вс") < runeIndex(row, "Пт") {
		t.Errorf("дни недели не в первой строке сетки: %q", string(row))
	}
	// Сетка тянется до низа страницы (над строкой сообщений).
	if r, _, _, _ := sim.GetContent(0, h-3); r != '└' {
		t.Errorf("низ сетки не у нижней границы страницы: (%d, %d) = %q", 0, h-3, r)
	}

	a.tapp.Stop()
	<-done
}
