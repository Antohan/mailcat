// Package config читает TOML-конфигурацию mailcat (~/.config/mailcat/config.toml),
// подставляет значения по умолчанию и валидирует поля.
// Пароль в конфиге не хранится — только в Keychain (docs/SECURITY.md §2).
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config — корневая структура конфигурации; поля соответствуют config.example.toml.
type Config struct {
	Server          string   `toml:"server"`
	Username        string   `toml:"username"`
	CAPath          string   `toml:"ca_path"`
	OrgDomain       string   `toml:"org_domain"`  // один домен (простой случай)
	OrgDomains      []string `toml:"org_domains"` // несколько: приоритетнее org_domain
	PageSize        int      `toml:"page_size"`
	MaxAttachmentMB int      `toml:"max_attachment_mb"`

	UI     UI     `toml:"ui"`
	Notify Notify `toml:"notify"`
	Log    Log    `toml:"log"`
}

// UI — настройки интерфейса.
type UI struct {
	Language string `toml:"language"`
	// Theme — палитра интерфейса: "light" (по умолчанию) или "dark".
	Theme         string   `toml:"theme"`
	HiddenFolders []string `toml:"hidden_folders"` // точные имена папок: не показывать в панели вообще (iter2 UI-1)
}

// Notify — правила уведомлений о новых письмах.
type Notify struct {
	Enabled       bool     `toml:"enabled"`
	IntervalSec   int      `toml:"interval_sec"`
	MinImportance string   `toml:"min_importance"`
	OnlyExternal  bool     `toml:"only_external"`
	Senders       []string `toml:"senders"`
	Keywords      []string `toml:"keywords"`
	ShowSubject   bool     `toml:"show_subject"`
	Sound         bool     `toml:"sound"`
}

// Log — параметры файлового лога (без секретов).
type Log struct {
	File string `toml:"file"`
}

// Defaults возвращает конфигурацию со значениями по умолчанию.
func Defaults() *Config {
	return &Config{
		PageSize:        50,
		MaxAttachmentMB: 100,
		UI:              UI{Language: "ru", Theme: "light"},
		Notify: Notify{
			Enabled:       true,
			IntervalSec:   60,
			MinImportance: "normal",
			ShowSubject:   true,
			Sound:         true,
		},
	}
}

// Load читает TOML-файл поверх значений по умолчанию. Валидация — отдельным
// вызовом Validate, чтобы UI/doctor могли показать все проблемы сразу.
func Load(path string) (*Config, error) {
	path = ExpandHome(path)
	raw, err := os.ReadFile(path) // #nosec G304 -- путь конфига задаёт пользователь
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать конфигурацию %s: %w", path, err)
	}
	cfg := Defaults()
	md, err := toml.Decode(string(raw), cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка разбора TOML в %s: %w", path, err)
	}
	// WHY: неизвестный ключ — не ошибка парсера, но молча отключает настройку:
	// опечатка в org_domains стирает метку [ВНЕШН] без сообщения.
	// Not to do: игнорировать Undecoded — тихая потеря настройки дороже отказа.
	if keys := unknownKeys(md); len(keys) > 0 {
		return nil, fmt.Errorf("неизвестные ключи в %s: %s — сверьтесь с config.example.toml",
			path, strings.Join(keys, ", "))
	}
	return cfg, nil
}

// unknownKeys — ключи, которых нет в Config. Родительская таблица
// побеждает ключи внутри неё, чтобы [foo] не печатался как «foo, foo.bar».
func unknownKeys(md toml.MetaData) []string {
	undecoded := md.Undecoded()
	seen := make(map[string]bool, len(undecoded))
	for _, k := range undecoded {
		seen[k.String()] = true
	}
	out := make([]string, 0, len(undecoded))
	for _, k := range undecoded {
		if len(k) > 1 && seen[k[:len(k)-1].String()] {
			continue
		}
		out = append(out, k.String())
	}
	return out
}

// InternalDomains — эффективный список внутренних доменов: org_domains,
// если задан и не пуст, иначе единственный org_domain. Оба необязательны:
// без доменов метка [ВНЕШН] не ставится. Регистр нормализуется, пустые
// элементы отбрасываются.
func (c *Config) InternalDomains() []string {
	domains := make([]string, 0, len(c.OrgDomains)+1)
	for _, d := range c.OrgDomains {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) > 0 {
		return domains
	}
	if d := strings.ToLower(strings.TrimSpace(c.OrgDomain)); d != "" {
		return []string{d}
	}
	return nil
}

// Validate проверяет обязательные поля и диапазоны значений.
func (c *Config) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("поле server обязательно (адрес Exchange, например https://mail.company.example)")
	}
	u, err := url.Parse(c.Server)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("server должен быть https-адресом, получено %q", c.Server)
	}
	if u.User != nil {
		return fmt.Errorf("server не должен содержать учётные данные (user:pass@host) — пароль хранится в Keychain, а не в конфиге")
	}
	if c.Username == "" {
		return fmt.Errorf("поле username обязательно (DOMAIN\\user или user@company.example)")
	}
	if c.PageSize <= 0 {
		return fmt.Errorf("page_size должен быть больше нуля, получено %d", c.PageSize)
	}
	if c.PageSize > 500 {
		return fmt.Errorf("page_size: максимум 500, получено %d", c.PageSize)
	}
	if c.MaxAttachmentMB <= 0 {
		return fmt.Errorf("max_attachment_mb должен быть больше нуля, получено %d", c.MaxAttachmentMB)
	}
	// WHY: 380 МиБ в base64 не влезает в потолок ответа 512 МиБ
	// (internal/ews/soap.go, maxResponseBytes) — скачивание упадёт непонятно.
	// Not to do: поднимать 380, не пересчитав потолок ответа EWS.
	if c.MaxAttachmentMB > 380 {
		return fmt.Errorf("max_attachment_mb: максимум 380 — большее вложение в base64 не поместится в ответ Exchange, получено %d", c.MaxAttachmentMB)
	}
	if c.UI.Language != "ru" {
		return fmt.Errorf("ui.language: MVP поддерживает только \"ru\", получено %q", c.UI.Language)
	}
	switch c.UI.Theme {
	case "light", "dark":
	default:
		return fmt.Errorf("ui.theme: допустимо light|dark, получено %q", c.UI.Theme)
	}
	if c.Notify.Enabled && c.Notify.IntervalSec < 30 {
		return fmt.Errorf("notify.interval_sec: минимум 30 секунд, получено %d", c.Notify.IntervalSec)
	}
	switch c.Notify.MinImportance {
	case "low", "normal", "high":
	default:
		return fmt.Errorf("notify.min_importance: допустимо low|normal|high, получено %q", c.Notify.MinImportance)
	}
	return nil
}

// DefaultPath возвращает путь к конфигу по умолчанию (~/.config/mailcat/config.toml).
func DefaultPath() string {
	return ExpandHome("~/.config/mailcat/config.toml")
}

// ExpandHome заменяет ведущую "~" на домашний каталог пользователя.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	switch {
	case path == "~":
		return home
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[2:])
	default:
		// Например "~user/..." — не раскрываем.
		return path
	}
}
