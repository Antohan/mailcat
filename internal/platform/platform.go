// Package platform — обвязка macOS: открытие файлов/ссылок через /usr/bin/open
// и файловое логирование slog. Прикладной логики здесь нет.
package platform

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mailcat/internal/config"
)

// Open открывает путь (файл или URL) системным приложением macOS.
// Используется только по явному действию пользователя (SECURITY.md §4, §8).
func Open(path string) error {
	if err := openCmd(path).Run(); err != nil {
		return fmt.Errorf("не удалось открыть %s: %w", path, err)
	}
	return nil
}

// KeychainPassword читает пароль сервиса mailcat из Keychain macOS.
// Пароль не логируется и не попадает в ошибки (SECURITY.md §2); вывод команды
// возвращается только вызывающему. Приглашение и сохранение при первом
// запуске — этап 5 (PLAN.md).
func KeychainPassword() (string, error) {
	out, err := keychainCmd().Output()
	if err != nil {
		return "", fmt.Errorf("пароль не найден в Keychain (сервис mailcat): %w\n"+
			"добавьте его командой: security add-generic-password -a <логин> -s mailcat -w", err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// keychainCmd собирает команду чтения пароля (отдельно — для теста).
func keychainCmd() *exec.Cmd {
	return exec.Command("security", "find-generic-password", "-s", "mailcat", "-w")
}

// openCmd собирает команду /usr/bin/open path (отдельно — для теста).
func openCmd(path string) *exec.Cmd {
	return exec.Command("/usr/bin/open", path)
}

// NewLogger возвращает slog-логгер, пишущий в файл path: каталог создаётся с
// правами 0700, файл — 0600 (SECURITY.md §6). Пустой path — логгер в никуда
// (io.Discard); closer в этом случае — заглушка.
//
// В лог не должны попадать секреты (пароли, токены) — SECURITY.md §2.
func NewLogger(path string) (*slog.Logger, func(), error) {
	if path == "" {
		noop := func() {}
		return slog.New(slog.NewTextHandler(io.Discard, nil)), noop, nil
	}
	path = config.ExpandHome(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("каталог лога %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("файл лога %s: %w", path, err)
	}
	closer := func() { _ = f.Close() }
	return slog.New(slog.NewTextHandler(f, nil)), closer, nil
}
