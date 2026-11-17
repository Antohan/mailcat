package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// TestDefaults: пустой TOML-файл -> все значения по умолчанию из config.example.toml.
func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PageSize != 50 {
		t.Errorf("PageSize = %d, хочу 50", cfg.PageSize)
	}
	if cfg.MaxAttachmentMB != 100 {
		t.Errorf("MaxAttachmentMB = %d, хочу 100", cfg.MaxAttachmentMB)
	}
	if cfg.UI.Language != "ru" {
		t.Errorf("UI.Language = %q, хочу ru", cfg.UI.Language)
	}
	if !cfg.Notify.Enabled {
		t.Error("Notify.Enabled = false, хочу true")
	}
	if cfg.Notify.IntervalSec != 60 {
		t.Errorf("Notify.IntervalSec = %d, хочу 60", cfg.Notify.IntervalSec)
	}
	if cfg.Notify.MinImportance != "normal" {
		t.Errorf("Notify.MinImportance = %q, хочу normal", cfg.Notify.MinImportance)
	}
	if cfg.Notify.OnlyExternal {
		t.Error("Notify.OnlyExternal = true, хочу false")
	}
	if !cfg.Notify.ShowSubject {
		t.Error("Notify.ShowSubject = false, хочу true")
	}
	if !cfg.Notify.Sound {
		t.Error("Notify.Sound = false, хочу true")
	}
	if cfg.Log.File != "" {
		t.Errorf("Log.File = %q, хочу пусто (не логировать)", cfg.Log.File)
	}
}

// TestLoadExample: реальный config.example.toml из корня репозитория читается
// и даёт ожидаемые значения.
func TestLoadExample(t *testing.T) {
	cfg, err := Load("../../config.example.toml")
	if err != nil {
		t.Fatalf("Load(config.example.toml): %v", err)
	}
	if cfg.Server != "https://mail.company.example" {
		t.Errorf("Server = %q", cfg.Server)
	}
	if cfg.Username != `COMPANY\ivanov` {
		t.Errorf("Username = %q", cfg.Username)
	}
	if cfg.CAPath != "~/.config/mailcat/company-root-ca.pem" {
		t.Errorf("CAPath = %q", cfg.CAPath)
	}
	if cfg.OrgDomain != "company.example" {
		t.Errorf("OrgDomain = %q", cfg.OrgDomain)
	}
	if cfg.PageSize != 50 || cfg.MaxAttachmentMB != 100 {
		t.Errorf("PageSize/MaxAttachmentMB = %d/%d", cfg.PageSize, cfg.MaxAttachmentMB)
	}
	if cfg.Notify.IntervalSec != 60 {
		t.Errorf("Notify.IntervalSec = %d", cfg.Notify.IntervalSec)
	}
	if cfg.Log.File != "~/.local/state/mailcat/mailcat.log" {
		t.Errorf("Log.File = %q", cfg.Log.File)
	}
}

// TestInvalidTOML: битый синтаксис -> понятная ошибка со словом TOML.
func TestInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("server = \"https://x\nbroken =="), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("ожидал ошибку, получил nil")
	}
	if !contains(err.Error(), "TOML") {
		t.Errorf("ошибка без упоминания TOML: %v", err)
	}
}

// TestMissingFile: нет файла -> ошибка содержит путь.
func TestMissingFile(t *testing.T) {
	_, err := Load("/no/such/mailcat-config.toml")
	if err == nil {
		t.Fatal("ожидал ошибку, получил nil")
	}
	if !contains(err.Error(), "/no/such/mailcat-config.toml") {
		t.Errorf("ошибка без пути к конфигу: %v", err)
	}
}

// TestValidate: таблица некорректных значений -> ошибки на русском.
func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
		want string
	}{
		{"пустой server", func(c *Config) { c.Server = "" }, "server"},
		{"server с userinfo", func(c *Config) {
			c.Server = "https://ivanov:secret@corp.example"
		}, "учётные данные"},
		{"пустой username", func(c *Config) { c.Username = "" }, "username"},
		{"server не https", func(c *Config) { c.Server = "http://mail.example" }, "https"},
		{"page_size ноль", func(c *Config) { c.PageSize = 0 }, "page_size"},
		{"max_attachment_mb ноль", func(c *Config) { c.MaxAttachmentMB = 0 }, "max_attachment_mb"},
		{"interval_sec меньше 30", func(c *Config) { c.Notify.IntervalSec = 10 }, "interval_sec"},
		{"min_importance мусор", func(c *Config) { c.Notify.MinImportance = "urgent" }, "min_importance"},
		{"language не ru", func(c *Config) { c.UI.Language = "en" }, "language"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Defaults()
			cfg.Server = "https://mail.example"
			cfg.Username = `COMPANY\ivanov`
			tc.mut(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("ожидал ошибку с %q, получил nil", tc.want)
			}
			if !contains(err.Error(), tc.want) {
				t.Errorf("ошибка %q не упоминает %q", err.Error(), tc.want)
			}
		})
	}
}

// TestValidateOK: валидный конфиг проходит без ошибок.
func TestValidateOK(t *testing.T) {
	cfg := Defaults()
	cfg.Server = "https://mail.company.example"
	cfg.Username = `COMPANY\ivanov`
	cfg.OrgDomain = "company.example"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestExpandHome: ~ раскрывается в $HOME, обычный путь не меняется.
func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	if got := ExpandHome("~/conf.toml"); got != "/home/tester/conf.toml" {
		t.Errorf("ExpandHome(~/conf.toml) = %q", got)
	}
	if got := ExpandHome("/abs/conf.toml"); got != "/abs/conf.toml" {
		t.Errorf("ExpandHome(/abs/conf.toml) = %q", got)
	}
	if got := ExpandHome(""); got != "" {
		t.Errorf("ExpandHome(\"\") = %q, хочу пусто", got)
	}
}

// TestDefaultPath: путь по умолчанию внутри ~/.config/mailcat/.
func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	if got := DefaultPath(); got != "/home/tester/.config/mailcat/config.toml" {
		t.Errorf("DefaultPath() = %q", got)
	}
}
