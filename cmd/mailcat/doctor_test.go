package main

// Тесты диагностики doctor (PLAN.md этап 5): 4 шага с человеко-понятными
// выводами — TCP (подсказка про VPN), TLS (объяснение про ca_path),
// аутентификация (Keychain), FindFolder. Шаги инжектируются — без сети.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mailcat/internal/config"
	"mailcat/internal/ews"
)

func okDeps() doctorDeps {
	return doctorDeps{
		dialTCP:      func(string) error { return nil },
		tlsHandshake: func(string, string) error { return nil },
		client:       func(*config.Config) (*ews.Client, error) { return &ews.Client{}, nil },
		findFolders: func(context.Context, *ews.Client) ([]ews.Folder, error) {
			return []ews.Folder{{DisplayName: "Входящие"}, {DisplayName: "Отправленные"}}, nil
		},
	}
}

func writeCfg(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "mailcat")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "server = \"https://mail.company.example\"\nusername = \"COMPANY\\\\ivanov\"\norg_domain = \"company.example\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestDoctorAllOK: все шаги зелёные, список папок печатается, код 0.
func TestDoctorAllOK(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	var out bytes.Buffer
	code := runDoctor(&out, loadCfg(t, home), okDeps())
	if code != 0 {
		t.Fatalf("code = %d, вывод: %s", code, out.String())
	}
	text := out.String()
	for _, want := range []string{"[1/4]", "OK", "[2/4]", "[3/4]", "[4/4]", "Входящие", "Отправленные", "найдено папок: 2"} {
		if !strings.Contains(text, want) {
			t.Errorf("в выводе doctor нет %q:\n%s", want, text)
		}
	}
}

// errWriter ломается на первой записи — закрытый stderr или полный диск.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write: broken pipe") }

// TestDoctorWriteError: отчёт не записался — код 1, а не «всё OK».
func TestDoctorWriteError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	if code := runDoctor(errWriter{}, loadCfg(t, home), okDeps()); code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
}

// TestDoctorVPN: TCP недоступен — подсказка про VPN, код 1, шаги 2–4 не идут.
func TestDoctorVPN(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	deps := okDeps()
	deps.dialTCP = func(string) error { return errors.New("dial tcp: i/o timeout") }
	var out bytes.Buffer
	code := runDoctor(&out, loadCfg(t, home), deps)
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(out.String(), "VPN") {
		t.Errorf("нет подсказки про VPN: %s", out.String())
	}
	if strings.Contains(out.String(), "[2/4]") {
		t.Errorf("шаг 2 выполнен после падения шага 1: %s", out.String())
	}
}

// TestDoctorTLS: сертификат не проходит — объяснение про ca_path, код 1.
func TestDoctorTLS(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	deps := okDeps()
	deps.tlsHandshake = func(string, string) error {
		return &ews.TLSProbeError{CertProblem: true, Err: errors.New("x509: certificate signed by unknown authority")}
	}
	var out bytes.Buffer
	code := runDoctor(&out, loadCfg(t, home), deps)
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(out.String(), "ca_path") {
		t.Errorf("нет объяснения про ca_path: %s", out.String())
	}
	if !strings.Contains(out.String(), "[2/4]") {
		t.Errorf("шаг 2 не отмечен: %s", out.String())
	}
}

// TestDoctorAuth: 401 — инструкция про Keychain/пароль, код 1.
func TestDoctorAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	deps := okDeps()
	deps.findFolders = func(context.Context, *ews.Client) ([]ews.Folder, error) {
		return nil, &ews.HTTPError{Status: 401}
	}
	var out bytes.Buffer
	code := runDoctor(&out, loadCfg(t, home), deps)
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	text := out.String()
	if !strings.Contains(text, "Keychain") || !strings.Contains(text, "пароль") {
		t.Errorf("нет инструкции про пароль/Keychain: %s", text)
	}
	if strings.Contains(text, "найдено папок") {
		t.Errorf("шаг 4 отчитался успехом после 401: %s", text)
	}
}

// TestDoctorAccessDenied: ErrorAccessDenied — «спросите ИТ-отдел».
func TestDoctorAccessDenied(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCfg(t, home)
	deps := okDeps()
	deps.findFolders = func(context.Context, *ews.Client) ([]ews.Folder, error) {
		return nil, &ews.ResponseError{Code: "ErrorAccessDenied"}
	}
	var out bytes.Buffer
	code := runDoctor(&out, loadCfg(t, home), deps)
	if code != 1 {
		t.Fatalf("code = %d, хочу 1", code)
	}
	if !strings.Contains(out.String(), "ИТ-отдел") {
		t.Errorf("нет подсказки про ИТ-отдел: %s", out.String())
	}
}

// TestDoctorSanitizesErrors: escape-последовательность в тексте ошибки
// (сеть, TLS, EWS) не доходит до терминала (SECURITY.md §3).
func TestDoctorSanitizesErrors(t *testing.T) {
	const evil = "boom \x1b]8;;http://evil\x1b\\text"
	cases := []struct {
		name string
		mod  func(*doctorDeps)
		hint string // подсказка после ошибки должна уцелеть
	}{
		{"сеть", func(d *doctorDeps) {
			d.dialTCP = func(string) error { return errors.New(evil) }
		}, "VPN"},
		{"TLS", func(d *doctorDeps) {
			d.tlsHandshake = func(string, string) error {
				return &ews.TLSProbeError{CertProblem: true, Err: errors.New(evil)}
			}
		}, "ca_path"},
		{"Keychain", func(d *doctorDeps) {
			d.client = func(*config.Config) (*ews.Client, error) { return nil, errors.New(evil) }
		}, ""},
		{"EWS", func(d *doctorDeps) {
			d.findFolders = func(context.Context, *ews.Client) ([]ews.Folder, error) {
				return nil, errors.New(evil)
			}
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			writeCfg(t, home)
			deps := okDeps()
			tc.mod(&deps)
			var out bytes.Buffer
			runDoctor(&out, loadCfg(t, home), deps)
			if strings.Contains(out.String(), "\x1b") {
				t.Errorf("ESC в выводе: %q", out.String())
			}
			if tc.hint != "" && !strings.Contains(out.String(), tc.hint) {
				t.Errorf("потеряна подсказка %q: %s", tc.hint, out.String())
			}
		})
	}
}

func loadCfg(t *testing.T, home string) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(home, ".config", "mailcat", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
