package notify

// Тесты правил уведомлений (PLAN.md этап 6): важность/отправители/ключевые
// слова/only_external/show_subject. Важность письма недоступна (item:Importance
// вне whitelist EWS-REFERENCE) — считается normal.

import (
	"testing"

	"mailcat/internal/config"
	"mailcat/internal/mail"
)

func msg(from, subject string) mail.MessageSummary {
	return mail.MessageSummary{Subject: subject, From: mail.Address{Name: "Иванов И.", Email: from}}
}

func TestShouldNotify(t *testing.T) {
	cases := []struct {
		name string
		rule config.Notify
		m    mail.MessageSummary
		want bool
	}{
		{"нет правил — все письма", config.Notify{Enabled: true}, msg("a@corp.ru", "Отчёт"), true},
		{"senders адрес", config.Notify{Enabled: true, Senders: []string{"boss@corp.ru"}}, msg("boss@corp.ru", "x"), true},
		{"senders чужой", config.Notify{Enabled: true, Senders: []string{"boss@corp.ru"}}, msg("a@corp.ru", "x"), false},
		{"senders домен @hr.corp.ru", config.Notify{Enabled: true, Senders: []string{"@hr.corp.ru"}}, msg("news@hr.corp.ru", "x"), true},
		{"senders домен не матчится подстрокой", config.Notify{Enabled: true, Senders: []string{"@corp.ru"}}, msg("x@notcorp.ru", "x"), false},
		{"keyword в теме", config.Notify{Enabled: true, Keywords: []string{"срочно"}}, msg("a@corp.ru", "СРОЧНО: сервер лежит"), true},
		{"keyword нет", config.Notify{Enabled: true, Keywords: []string{"инцидент"}}, msg("a@corp.ru", "СРОЧНО"), false},
		{"only_external внешний", config.Notify{Enabled: true, OnlyExternal: true}, msg("a@partner.tld", "x"), true},
		{"only_external внутренний", config.Notify{Enabled: true, OnlyExternal: true}, msg("a@corp.ru", "x"), false},
		{"min_importance low пропускает", config.Notify{Enabled: true, MinImportance: "low"}, msg("a@corp.ru", "x"), true},
		{"min_importance normal пропускает", config.Notify{Enabled: true, MinImportance: "normal"}, msg("a@corp.ru", "x"), true},
		{"min_importance high режет всех", config.Notify{Enabled: true, MinImportance: "high"}, msg("a@corp.ru", "x"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldNotify(tc.m, tc.rule, []string{"corp.ru"}); got != tc.want {
				t.Errorf("ShouldNotify(%+v) = %v, хочу %v", tc.m, got, tc.want)
			}
		})
	}
}

// TestNotificationTexts: заголовок/тело уведомления; show_subject=false —
// тема не используется вообще (§9 приватность).
func TestNotificationTexts(t *testing.T) {
	m := msg("a@corp.ru", "Секретная тема")
	title, body := NotificationTexts(m, true)
	if title == "" || body != "Секретная тема" {
		t.Errorf("с темой: %q / %q", title, body)
	}
	title, body = NotificationTexts(m, false)
	if body != "" {
		t.Errorf("тема утекла при show_subject=false: %q", body)
	}
	if title == "" || "Секретная тема" == title {
		t.Errorf("заголовок без темы: %q", title)
	}
}
