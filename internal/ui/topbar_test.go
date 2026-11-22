package ui

// Тесты верхнего бара (V-6, V-11).

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestTopBar: одна строка над pages — только «mailcat · <папка>» (имя из EWS
// экранировано). Дата/время и точка связи убраны (V-11): статус связи —
// футер сайдбара (setConn).
func TestTopBar(t *testing.T) {
	a := newTestApp(t)
	a.mail.folderName = `Отчёты["x"]`
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(110, 28)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	a.tapp.QueueUpdateDraw(a.updateTopBar)

	row := strings.TrimRight(string(simRow(sim, 0)), " ")
	if !strings.Contains(row, "mailcat · Отчёты") || !strings.Contains(row, `["x"]`) {
		t.Errorf("строка 0 = %q", row)
	}
	if regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}`).MatchString(row) || strings.Contains(row, "●") {
		t.Errorf("дата/точка верхнего бара не убраны: %q", row)
	}
	a.mail.folderName = ""
	a.tapp.QueueUpdateDraw(a.updateTopBar)
	if row := strings.TrimRight(string(simRow(sim, 0)), " "); strings.Contains(row, " · ") {
		t.Errorf("пустая папка — без разделителя: %q", row)
	}
	a.tapp.Stop()
	<-done
}

// TestTopBarCalendarMonth (V-31): на календаре верхний бар показывает месяц,
// после 1 — снова имя папки. ensureLoaded не ходит в сеть: месяц «загружен».
func TestTopBarCalendarMonth(t *testing.T) {
	a := newTestApp(t)
	a.mail.folderName = "Входящие"
	a.calendar.year, a.calendar.month = 2026, time.November
	a.calendar.loadedKey = a.calendar.monthKey()
	a.globalKeys(tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone))
	if got := a.topLeft.GetText(true); !strings.Contains(got, "Ноябрь 2026") {
		t.Errorf("верхний бар календаря: %q", got)
	}
	a.globalKeys(tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone))
	if got := a.topLeft.GetText(true); !strings.Contains(got, "Входящие") {
		t.Errorf("верхний бар почты: %q", got)
	}
}

// TestConnFooter: футер сайдбара в две строки (V-12) — «● Connected» и
// username под ним; длинный логин обрезан «…» по ширине панели.
func TestConnFooter(t *testing.T) {
	a := newTestApp(t)
	a.cfg.Username = strings.Repeat("u", 40)
	a.mail.setConn(true)
	lines := strings.Split(a.mail.connText(12), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "●") || !strings.Contains(lines[0], "Connected") {
		t.Fatalf("состояние связи = %q", lines)
	}
	if !strings.HasSuffix(lines[1], "…[-]") || tview.TaggedStringWidth(lines[1]) != 12 {
		t.Errorf("username не обрезан по ширине: %q", lines[1])
	}
	// V-17: точка — в колонке левого поля, текст — в колонке имён папок.
	if !strings.HasPrefix(lines[0], " [") || !strings.Contains(lines[0], " Connected") ||
		!strings.HasPrefix(lines[1], "   ") {
		t.Errorf("футер вне колонки папок: %q", lines)
	}
	a.mail.setConn(false)
	if !strings.Contains(a.mail.connText(12), "Нет связи") {
		t.Errorf("ошибка связи не показана: %q", a.mail.connText(12))
	}
}
