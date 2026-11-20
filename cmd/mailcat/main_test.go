package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mailcat/internal/config"
	"mailcat/internal/ews"
)

// изолированный HOME, чтобы тесты не трогали реальный ~/.config
func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// writeConfig кладёт конфиг в ~/.config/mailcat/config.toml.
func writeConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "mailcat")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const validConfig = `
server = "https://mail.company.example"
username = "COMPANY\\ivanov"
org_domain = "company.example"
`

// testRuntime — зависимости для тестов: фейковый Keychain и перехват UI.
type testRuntime struct {
	password string
	passErr  error
	uiURL    string
	uiRan    bool
	uiErr    error
	folders  []ews.Folder
	foldErr  error
}

func (tr *testRuntime) runtime() runtime {
	return runtime{
		keychainPassword: func() (string, error) { return tr.password, tr.passErr },
		newUI: func(_ *config.Config, c *ews.Client, _ *slog.Logger) error {
			tr.uiRan = true
			tr.uiURL = c.URL
			return tr.uiErr
		},
		findFolders: func(context.Context, *ews.Client) ([]ews.Folder, error) {
			return tr.folders, tr.foldErr
		},
	}
}

// TestRunNoConfig: без конфига — ошибка с путём ожидаемого конфига и подсказкой.
func TestRunNoConfig(t *testing.T) {
	home := fakeHome(t)
	var out, errb bytes.Buffer
	tr := &testRuntime{}
	code := run([]string{"run"}, &out, &errb, tr.runtime())
	if code != 1 {
		t.Errorf("code = %d, хочу 1", code)
	}
	msg := errb.String()
	if !strings.Contains(msg, filepath.Join(home, ".config", "mailcat", "config.toml")) {
		t.Errorf("ошибка без пути конфига: %q", msg)
	}
	if !strings.Contains(msg, "config.example.toml") {
		t.Errorf("ошибка без подсказки про config.example.toml: %q", msg)
	}
}

// TestRunInvalidConfig: server не https — код 1, ошибка про https.
func TestRunInvalidConfig(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, "server = \"http://mail.company.example\"\nusername = \"u\"\n")
	var out, errb bytes.Buffer
	tr := &testRuntime{}
	code := run([]string{"run"}, &out, &errb, tr.runtime())
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(errb.String(), "https") {
		t.Errorf("stderr без упоминания https: %q", errb.String())
	}
}

// TestRunTUI: с конфигом и паролем — UI запускается с endpoint
// server + /EWS/Exchange.asmx (EWS-REFERENCE.md §1).
func TestRunTUI(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, validConfig)
	tr := &testRuntime{password: "secret"}
	var out, errb bytes.Buffer
	code := run([]string{"run"}, &out, &errb, tr.runtime())
	if code != 0 {
		t.Fatalf("code = %d, stderr: %q", code, errb.String())
	}
	if !tr.uiRan {
		t.Fatal("UI не запущен")
	}
	if tr.uiURL != "https://mail.company.example/EWS/Exchange.asmx" {
		t.Errorf("URL = %q, хочу https://mail.company.example/EWS/Exchange.asmx", tr.uiURL)
	}
}

// TestRunTUIKeychainMissing: записи в Keychain нет — понятная ошибка, код 1.
func TestRunTUIKeychainMissing(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, validConfig)
	tr := &testRuntime{passErr: errors.New("security: SecKeychainSearch ...")}
	var out, errb bytes.Buffer
	code := run([]string{"run"}, &out, &errb, tr.runtime())
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(errb.String(), "Keychain") {
		t.Errorf("stderr без упоминания Keychain: %q", errb.String())
	}
	if strings.Contains(errb.String(), "secret") {
		t.Errorf("в ошибке утёк пароль: %q", errb.String())
	}
}

// TestRunFolders: список папок печатается (дымовый тест EWS), текст санитизирован.
func TestRunFolders(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, validConfig)
	tr := &testRuntime{password: "p", folders: []ews.Folder{
		{ID: "1", DisplayName: "Входящие", TotalCount: 10, UnreadCount: 2},
		{ID: "2", DisplayName: "Отправ\x1b]0;x\x07ленные"},
	}}
	var out, errb bytes.Buffer
	code := run([]string{"folders"}, &out, &errb, tr.runtime())
	if code != 0 {
		t.Fatalf("code = %d, stderr: %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "Входящие — писем: 10, непрочитанных: 2") {
		t.Errorf("stdout: %q", out.String())
	}
	if !strings.Contains(out.String(), "Отправленные") || strings.Contains(out.String(), "\x1b") {
		t.Errorf("имя папки не санитизировано: %q", out.String())
	}
}

// TestRunFoldersEWSerror: ошибка EWS → код 1, сообщение в stderr, санитизированное.
func TestRunFoldersEWSerror(t *testing.T) {
	home := fakeHome(t)
	writeConfig(t, home, validConfig)
	tr := &testRuntime{password: "p", foldErr: errors.New("нет соединения (похоже, VPN выключен) \x1b]52;c;evil\x07")}
	var out, errb bytes.Buffer
	code := run([]string{"folders"}, &out, &errb, tr.runtime())
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(errb.String(), "VPN") {
		t.Errorf("stderr: %q", errb.String())
	}
	if strings.ContainsRune(errb.String(), '\x1b') {
		t.Errorf("ESC-инъекция прошла в stderr: %q", errb.String())
	}
}

// TestUnknownAndNoArgs: неизвестная команда и вызов без аргументов — usage, код 2.
func TestUnknownAndNoArgs(t *testing.T) {
	fakeHome(t)
	tr := &testRuntime{}
	for _, argv := range [][]string{{"sync"}, {}} {
		var out, errb bytes.Buffer
		code := run(argv, &out, &errb, tr.runtime())
		if code != 2 {
			t.Errorf("%v: code = %d, хочу 2", argv, code)
		}
		msg := errb.String()
		if !strings.Contains(msg, "run") || !strings.Contains(msg, "doctor") || !strings.Contains(msg, "folders") {
			t.Errorf("%v: usage без списка команд: %q", argv, msg)
		}
	}
}

// TestHelp: help/-h печатает usage в stdout с кодом 0.
func TestHelp(t *testing.T) {
	fakeHome(t)
	var out, errb bytes.Buffer
	tr := &testRuntime{}
	code := run([]string{"help"}, &out, &errb, tr.runtime())
	if code != 0 {
		t.Errorf("code = %d, хочу 0", code)
	}
	if !strings.Contains(out.String(), "Использование") {
		t.Errorf("stdout без usage: %q", out.String())
	}
}

// TestRunLogsHostOnly: в файловый лог пишется хост сервера, а не полный URL
// (SECURITY.md §2 — в логах не должно быть учётных данных).
func TestRunLogsHostOnly(t *testing.T) {
	home := fakeHome(t)
	logPath := filepath.Join(home, "mailcat.log")
	writeConfig(t, home, validConfig+"\n[log]\nfile = \""+logPath+"\"\n")
	tr := &testRuntime{password: "p"}
	var out, errb bytes.Buffer
	code := run([]string{"run"}, &out, &errb, tr.runtime())
	if code != 0 {
		t.Fatalf("code = %d, stderr: %q", code, errb.String())
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("чтение лога: %v", err)
	}
	if strings.Contains(string(data), "https://") {
		t.Errorf("в логе полный URL вместо хоста: %q", data)
	}
	if !strings.Contains(string(data), "mail.company.example") {
		t.Errorf("в логе нет хоста сервера: %q", data)
	}
}
