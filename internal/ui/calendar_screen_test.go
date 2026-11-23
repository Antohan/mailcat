package ui

// V-30: у календаря нет ASCII-баннера месяца — месяц живёт в шапке панели,
// сетка начинается сразу под шапкой и занимает всю высоту колонки.

import (
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"mailcat/internal/config"
	"mailcat/internal/ews"
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

// TestCalendarJumpToToday (V-47): t в текущем месяце выбирает сегодняшний
// день без сети; из чужого месяца — переход на текущий с перезагрузкой, после
// ответа выбран сегодня. Подсказки сетки и повестки содержат «t сегодня».
func TestCalendarJumpToToday(t *testing.T) {
	today := time.Now()

	// Текущий месяц: месяц «загружен», сеть не нужна.
	a := newTestApp(t)
	a.calendar.year, a.calendar.month = today.Year(), today.Month()
	a.calendar.loadedKey = a.calendar.monthKey()
	a.calendar.render()
	away := 1
	if today.Day() == 1 {
		away = 2
	}
	a.calendar.grid.SelectDay(away)
	hook := a.calendar.grid.GetInputCapture()
	if hook == nil {
		t.Fatal("у сетки нет обработчика клавиш")
	}
	if got := hook(tcell.NewEventKey(tcell.KeyRune, 't', tcell.ModNone)); got != nil {
		t.Fatal("t не поглощена сеткой")
	}
	g := a.calendar.grid
	if day := g.cells[g.selRow][g.selCol].Day; day != today.Day() {
		t.Fatalf("t выбрал день %d, хочу %d", day, today.Day())
	}

	// Чужой месяц: t переключает месяц и перезагружает его через EWS-мок.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(mockFixture(t, string(b)))
	}))
	defer srv.Close()
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caPath, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	client, err := ews.NewClient(srv.URL, ews.Credentials{Username: "u", Password: "p"}, caPath,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	b := New(config.Defaults(), client, nil)
	b.calendar.year, b.calendar.month = today.Year()-1, today.Month()
	b.calendar.loadedKey = b.calendar.monthKey()
	b.calendar.render()

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	b.tapp.SetScreen(sim)
	sim.SetSize(100, 30)
	done := make(chan struct{})
	go func() { defer close(done); _ = b.tapp.Run() }()
	defer func() { b.tapp.Stop(); <-done }()
	b.pages.SwitchToPage(pageCalendar)

	b.tapp.QueueUpdateDraw(func() {
		if ev := b.calendar.grid.GetInputCapture()(tcell.NewEventKey(tcell.KeyRune, 't', tcell.ModNone)); ev != nil {
			t.Error("t не поглощена сеткой (чужой месяц)")
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	var ok bool
	for time.Now().Before(deadline) && !ok {
		_ = b.tapp.QueueUpdateDraw(func() {
			g := b.calendar.grid
			ok = b.calendar.loadedKey == b.calendar.monthKey() && g.cells[g.selRow][g.selCol].Day == today.Day()
		})
		if !ok {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !ok {
		t.Fatal("после t сегодняшний день не выбран")
	}
	if want := fmt.Sprintf("%04d-%02d", today.Year(), int(today.Month())); b.calendar.monthKey() != want {
		t.Errorf("месяц не текущий: %s, хочу %s", b.calendar.monthKey(), want)
	}

	for _, segs := range [][]hintSeg{hintCalendarGrid, hintCalendarAgenda} {
		text := formatHintFit(segs, 100)
		if !strings.Contains(text, "сегодня") {
			t.Errorf("в подсказке нет «t сегодня»: %q", text)
		}
	}
}
