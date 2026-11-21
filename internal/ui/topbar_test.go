package ui

// Тесты верхнего бара (V-6, V-11).

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
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
