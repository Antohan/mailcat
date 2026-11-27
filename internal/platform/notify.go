package platform

// Уведомления macOS через osascript (PLAN.md этап 6, SECURITY.md §9):
// текст писем попадает в AppleScript-строку — обязательное экранирование,
// звук — только системное имя "default".

import (
	"fmt"
	"os/exec"
	"strings"
)

// appleScriptEscape готовит строку для вставки в AppleScript-литерал в двойных
// кавычках: " → \", \ → \\, переводы строк → пробел (SECURITY.md §9).
func appleScriptEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// notifyCmd строит osascript-команду display notification (отдельно — для теста).
// Кавычки ставим сами: appleScriptEscape уже дал AppleScript-экранирование,
// %q добавил бы второе.
func notifyCmd(title, body string, sound bool) *exec.Cmd {
	script := fmt.Sprintf("display notification \"%s\" with title \"%s\"",
		appleScriptEscape(body), appleScriptEscape(title))
	if sound {
		script += " sound name \"default\""
	}
	return exec.Command("/usr/bin/osascript", "-e", script) // #nosec G204 -- бинарник фиксирован, shell не используется
}

// Notify показывает уведомление в Центре уведомлений macOS. Ошибки выполнения
// возвращаются вызывающему (логируются, но не прерывают TUI — SECURITY.md §9).
func Notify(title, body string, sound bool) error {
	if err := notifyCmd(title, body, sound).Run(); err != nil {
		return fmt.Errorf("уведомление macOS: %w", err)
	}
	return nil
}
