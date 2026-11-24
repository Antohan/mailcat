package ui

// V-14: g на экране списка открывает реестр ссылок выбранного письма.

import (
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/mail"
)

// TestMessageLinks: реестр из тела — голый URL находится, текст без ссылок
// даёт пустой список.
func TestMessageLinks(t *testing.T) {
	if got := messageLinks(mail.Message{BodyText: "Смотри https://example.com/a"}); len(got) != 1 {
		t.Fatalf("ссылок %d, хочу 1: %+v", len(got), got)
	}
	if got := messageLinks(mail.Message{BodyText: "без ссылок"}); len(got) != 0 {
		t.Fatalf("ссылок %d, хочу 0", len(got))
	}
}

// TestKeyGOnListNoLinks: g на списке догружает письмо (GetItem) и при
// отсутствии ссылок пишет «Ссылок нет» в строку сообщений.
func TestKeyGOnListNoLinks(t *testing.T) {
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
	cfg := config.Defaults()
	cfg.Server = srv.URL
	a := New(cfg, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rows := []mail.MessageSummary{{ItemID: "AAMk-test", Subject: "Отчёт Q3", Received: time.Now()}}
	a.mail.items = append(a.mail.items, rows...)
	a.mail.appendRows(rows)

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(110, 28)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	defer func() { a.tapp.Stop(); <-done }()

	var hook func(*tcell.EventKey) *tcell.EventKey
	_ = a.tapp.QueueUpdateDraw(func() { hook = a.mail.messages.GetInputCapture() })
	if hook == nil {
		t.Fatal("у списка нет обработчика клавиш")
	}
	var got *tcell.EventKey
	_ = a.tapp.QueueUpdateDraw(func() {
		a.tapp.SetFocus(a.mail.messages)
		got = hook(tcell.NewEventKey(tcell.KeyRune, 'g', 0))
	})
	if got != nil {
		t.Errorf("g не перехвачена списком: %v", got)
	}

	var status string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_ = a.tapp.QueueUpdateDraw(func() { status = a.status.GetText(false) })
		if status == "Ссылок нет" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("после g статус = %q, хочу «Ссылок нет»", status)
}
