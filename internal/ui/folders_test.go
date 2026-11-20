package ui

// Тесты группировки папок для панели «Папки» (iter2 UI-1): стандартные —
// фиксированный порядок, остальные — алфавит, техпапки — секция «Ещё ▸».

import (
	"strings"
	"testing"

	"github.com/rivo/tview"

	"mailcat/internal/config"
	"mailcat/internal/mail"
)

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
		folders:     tview.NewList(),
		mainFolders: []mail.Folder{{DisplayName: "Входящие", UnreadCount: 2}},
		techFolders: []mail.Folder{{DisplayName: "Опции"}, {DisplayName: "Корневая папка Yammer"}},
	}

	s.renderFolders()
	if got := s.folders.GetItemCount(); got != 2 {
		t.Fatalf("свернуто: пунктов %d, хочу 2 (Входящие + Ещё ▸)", got)
	}
	if main, _ := s.folders.GetItemText(1); main != "Ещё ▸" {
		t.Errorf("маркер секции = %q, хочу «Ещё ▸»", main)
	}
	if main, _ := s.folders.GetItemText(0); !strings.Contains(main, "📥 Входящие") || !strings.HasSuffix(main, "](2)") {
		t.Errorf("первый пункт = %q, хочу «📥 Входящие» со счётчиком 2", main)
	}

	s.toggleTech()
	if got := s.folders.GetItemCount(); got != 4 {
		t.Fatalf("развернуто: пунктов %d, хочу 4 (Входящие, Ещё ▾, 2 техпапки)", got)
	}
	if main, _ := s.folders.GetItemText(1); main != "Ещё ▾" {
		t.Errorf("маркер секции = %q, хочу «Ещё ▾»", main)
	}
	if got := s.folders.GetCurrentItem(); got != 1 {
		t.Errorf("курсор = %d, хочу 1 (остаться на «Ещё»)", got)
	}

	s.toggleTech()
	if got := s.folders.GetItemCount(); got != 2 {
		t.Fatalf("после сворачивания: пунктов %d, хочу 2", got)
	}
}

// TestRenderFoldersNoTech: без техпапок пункт «Ещё ▸» не добавляется.
func TestRenderFoldersNoTech(t *testing.T) {
	s := &mailScreen{
		app:         &App{cfg: &config.Config{}},
		folders:     tview.NewList(),
		mainFolders: []mail.Folder{{DisplayName: "Входящие"}},
	}
	s.renderFolders()
	if got := s.folders.GetItemCount(); got != 1 {
		t.Fatalf("пунктов %d, хочу 1 (без «Ещё ▸»)", got)
	}
}
