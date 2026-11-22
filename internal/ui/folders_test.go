package ui

// Тесты группировки папок для панели «Папки» (iter2 UI-1): стандартные —
// фиксированный порядок, остальные — алфавит, техпапки — секция «Ещё ▸».

import (
	"encoding/pem"
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
	"github.com/rivo/tview"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/mail"
)

// mockFixture выбирает фикстуру EWS по операции в запросе (мок для
// full-path тестов UI; файлы — ../ews/testdata).
func mockFixture(t *testing.T, reqBody string) []byte {
	t.Helper()
	var file string
	switch {
	case strings.Contains(reqBody, "FindFolder"):
		file = "findfolder_response.xml"
	case strings.Contains(reqBody, "GetItem"):
		file = "getitem_response.xml"
	case strings.Contains(reqBody, "CalendarView"):
		file = "calendar_response.xml"
	case strings.Contains(reqBody, `Offset="0"`), !strings.Contains(reqBody, "Offset="):
		file = "finditem_response_p1.xml"
	default:
		file = "finditem_response_p2.xml"
	}
	b, err := os.ReadFile(filepath.Join("..", "ews", "testdata", file))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func names(folders []mail.Folder) []string {
	out := make([]string, 0, len(folders))
	for _, f := range folders {
		out = append(out, f.DisplayName)
	}
	return out
}

func eqNames(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("получено %v, хочу %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("получено %v, хочу %v", got, want)
		}
	}
}

// TestSplitFolders: вход перемешан; стандартные — в фиксированном порядке,
// прочие — по алфавиту, техпапки — отдельно, hidden — исключены полностью.
func TestSplitFolders(t *testing.T) {
	in := []mail.Folder{
		{DisplayName: "Отправленные"},
		{DisplayName: "Опции"},
		{DisplayName: "Входящие", UnreadCount: 3},
		{DisplayName: "Проекты"},
		{DisplayName: "Личное"}, // hidden
		{DisplayName: "Conversation Action Settings"},
		{DisplayName: "Архив"},
		{DisplayName: "Удалённые"},
	}
	main, tech := SplitFolders(in, []string{"Личное"})
	eqNames(t, names(main), []string{"Входящие", "Отправленные", "Удалённые", "Архив", "Проекты"})
	eqNames(t, names(tech), []string{"Conversation Action Settings", "Опции"})
	if main[0].UnreadCount != 3 {
		t.Errorf("UnreadCount потерян при сортировке: %+v", main[0])
	}
}

// TestSplitFoldersStandardOrder: все шесть стандартных папок выстраиваются
// в порядок плана независимо от порядка в ответе сервера.
func TestSplitFoldersStandardOrder(t *testing.T) {
	in := []mail.Folder{
		{DisplayName: "Удалённые"},
		{DisplayName: "Нежелательная"},
		{DisplayName: "Черновики"},
		{DisplayName: "Отправленные"},
		{DisplayName: "Исходящие"},
		{DisplayName: "Входящие"},
	}
	main, tech := SplitFolders(in, nil)
	eqNames(t, names(main), []string{
		"Входящие", "Черновики", "Исходящие", "Отправленные", "Удалённые", "Нежелательная",
	})
	if len(tech) != 0 {
		t.Errorf("техпапки не ожидались: %v", names(tech))
	}
}

// TestRenderFoldersToggle: техпапки спрятаны за «Ещё ▸», Enter по пункту
// раскрывает секцию («Ещё ▾») и возвращает курсор на сам пункт.
func TestRenderFoldersToggle(t *testing.T) {
	s := &mailScreen{
		app: &App{cfg: &config.Config{}},
		// Список создаётся как в newMailScreen: только текст пунктов.
		folders:        tview.NewList(),
		mainFolders:    []mail.Folder{{DisplayName: "Входящие", UnreadCount: 2}},
		techFolders:    []mail.Folder{{DisplayName: "Опции"}, {DisplayName: "Корневая папка Yammer"}},
		messagesHeader: newPanelHeader("Письма"), // renderFolders обновляет и заголовок (V-2)
	}

	s.renderFolders()
	if got := s.folders.GetItemCount(); got != 3 {
		t.Fatalf("свернуто: пунктов %d, хочу 3 (Входящие + разделитель + Ещё ▸)", got)
	}
	if main, _ := s.folders.GetItemText(2); main != "Ещё ▸" {
		t.Errorf("маркер секции = %q, хочу «Ещё ▸»", main)
	}
	if main, _ := s.folders.GetItemText(1); !strings.Contains(main, strings.Repeat("─", 13)) {
		t.Errorf("второй пункт = %q, хочу линию-разделитель групп (V-4)", main)
	}
	if main, _ := s.folders.GetItemText(0); !strings.HasPrefix(main, "  Входящие") || !strings.HasSuffix(main, "](2)") {
		t.Errorf("первый пункт = %q, хочу «  Входящие» со счётчиком 2 (без иконок, П.2а)", main)
	}
	if _, ok := s.rowByItem[1]; ok {
		t.Errorf("разделитель попал в rowByItem: %v", s.rowByItem)
	}

	s.toggleTech()
	if got := s.folders.GetItemCount(); got != 5 {
		t.Fatalf("развернуто: пунктов %d, хочу 5 (Входящие, разделитель, Ещё ▾, 2 техпапки)", got)
	}
	if main, _ := s.folders.GetItemText(2); main != "Ещё ▾" {
		t.Errorf("маркер секции = %q, хочу «Ещё ▾»", main)
	}
	if got := s.folders.GetCurrentItem(); got != 2 {
		t.Errorf("курсор = %d, хочу 2 (остаться на «Ещё», moreIdx с учётом разделителя)", got)
	}

	s.toggleTech()
	if got := s.folders.GetItemCount(); got != 3 {
		t.Fatalf("после сворачивания: пунктов %d, хочу 3", got)
	}
}

// TestUpdateInboxUnreadKeepsCursor (V-18): тик watcher'а меняет только
// счётчик строки «Входящих» — курсор и состав списка остаются на месте.
func TestUpdateInboxUnreadKeepsCursor(t *testing.T) {
	s := &mailScreen{
		app:            &App{cfg: &config.Config{}},
		folders:        tview.NewList(),
		mainFolders:    []mail.Folder{{DisplayName: "Входящие", UnreadCount: 2}, {DisplayName: "Черновики"}},
		messagesHeader: newPanelHeader("Письма"),
	}
	s.renderFolders()
	s.folders.SetCurrentItem(1)

	s.updateInboxUnread(7)
	if got := s.folders.GetCurrentItem(); got != 1 {
		t.Errorf("курсор = %d, хочу 1 (строка «Входящих» не пересобирает список)", got)
	}
	if main, _ := s.folders.GetItemText(0); !strings.Contains(main, "(7)") {
		t.Errorf("строка «Входящих» = %q, хочу счётчик (7)", main)
	}
	if s.mainFolders[0].UnreadCount != 7 {
		t.Errorf("кэш счётчика = %d, хочу 7", s.mainFolders[0].UnreadCount)
	}
}

// TestRenderFoldersNoTech: без техпапок пункт «Ещё ▸» не добавляется.
func TestRenderFoldersNoTech(t *testing.T) {
	s := &mailScreen{
		app:            &App{cfg: &config.Config{}},
		folders:        tview.NewList(),
		mainFolders:    []mail.Folder{{DisplayName: "Входящие"}},
		messagesHeader: newPanelHeader("Письма"),
	}
	s.renderFolders()
	if got := s.folders.GetItemCount(); got != 1 {
		t.Fatalf("пунктов %d, хочу 1 (без «Ещё ▸»)", got)
	}
}

// TestFolderSwitchSelectsFirst (V-17): при переключении папки выделение — на
// первое письмо, а не на прежнюю позицию; восстановление места по ItemId
// остаётся только для перечитывания той же папки («u»).
func TestFolderSwitchSelectsFirst(t *testing.T) {
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

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(120, 40)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	defer func() { a.tapp.Stop(); <-done }()

	a.mail.loadFolders() // старт с «Входящих»
	waitFor := func(cond func() bool) bool {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			ok := false
			_ = a.tapp.QueueUpdateDraw(func() { ok = cond() })
			if ok {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}
	if !waitFor(func() bool { return len(a.mail.items) >= 2 }) {
		t.Fatal("письма не загрузились")
	}
	_ = a.tapp.QueueUpdateDraw(func() {
		a.mail.messages.Select(a.mail.itemRows[1], 0)
	})
	a.mail.selectFolder(ews.FolderID{ID: "other-folder"}, "Другая")
	if !waitFor(func() bool {
		row, _ := a.mail.messages.GetSelection()
		return len(a.mail.items) > 0 && row == a.mail.itemRows[0]
	}) {
		row, _ := a.mail.messages.GetSelection()
		t.Fatalf("после смены папки выделена строка %d, хочу первую (%d)", row, a.mail.itemRows[0])
	}
}

// TestRefreshRestoresEmptySidebar: «u» (refreshFolder) при пустом сайдбаре —
// состояние после ошибки FindFolders при старте — восстанавливает и папки, а
// не только письма: loadFolders завершается selectFolder («Входящие»).
// До фикса «u» перегружал письма с folderName=="" — заголовок списка
// оставался сброшенным, сайдбар пустым до перезапуска (живой кейс 21.11).
func TestRefreshRestoresEmptySidebar(t *testing.T) {
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
	// Состояние «FindFolders при старте упал»: сайдбар пуст, папка не выбрана.
	a.mail.folderName, a.mail.folder = "", ews.FolderID{}

	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(110, 28)
	done := make(chan struct{})
	go func() { defer close(done); _ = a.tapp.Run() }()
	defer func() { a.tapp.Stop(); <-done }()

	a.mail.refreshFolder() // «u»

	var items, mains int
	var title string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_ = a.tapp.QueueUpdateDraw(func() {
			items = a.mail.folders.GetItemCount()
			mains = len(a.mail.mainFolders)
			title = a.mail.messagesHeader.title
		})
		if items > 0 && mains > 0 && title != "" {
			return // восстановилось
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("«u» не восстановил сайдбар: items=%d mainFolders=%d title=%q", items, mains, title)
}
