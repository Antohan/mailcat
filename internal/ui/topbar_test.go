package ui

// Тесты верхнего бара (V-6).

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestTopBar: одна строка над pages — слева «mailcat · <папка>» (имя из EWS
// экранировано), справа дата/время и точка связи у самого правого края
// (красная до первого успеха, зелёная после — как футер сайдбара V-4).
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
	if !regexp.MustCompile(`\d{2}\.\d{2}\.\d{4} \d{2}:\d{2} ●$`).MatchString(row) {
		t.Errorf("дата/точка не прижаты вправо: %q", row)
	}
	_, _, st, _ := sim.GetContent(109, 0)
	if fg, _, _ := st.Decompose(); fg != colorError {
		t.Errorf("точка не красная до первого успеха: %v", st)
	}
	a.connOK = true
	a.tapp.QueueUpdateDraw(a.updateTopBar)
	_, _, st, _ = sim.GetContent(109, 0)
	if fg, _, _ := st.Decompose(); fg != colorConnected {
		t.Errorf("точка не зелёная после успеха: %v", st)
	}
	a.mail.folderName = ""
	a.tapp.QueueUpdateDraw(a.updateTopBar)
	if row := strings.TrimRight(string(simRow(sim, 0)), " "); strings.Contains(row, " · ") {
		t.Errorf("пустая папка — без разделителя: %q", row)
	}
	a.tapp.Stop()
	<-done
}
