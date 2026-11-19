// Package notify — правила уведомлений о новых письмах и фоновый watcher
// (PLAN.md этап 6). Чистая логика решения — здесь, показ — через колбэк.
package notify

import (
	"strings"

	"mailcat/internal/config"
	"mailcat/internal/mail"
	"mailcat/internal/security"
)

// ShouldNotify решает, проходит ли письмо под правила [notify].
// Важность письма недоступна (item:Importance вне whitelist
// docs/EWS-REFERENCE.md §9) — считается normal: min_importance low/normal
// пропускают все письма, high — ни одного.
func ShouldNotify(m mail.MessageSummary, rule config.Notify, orgDomain string) bool {
	if !passesImportance(rule.MinImportance) {
		return false
	}
	if rule.OnlyExternal && !security.IsExternalEmail(m.From.Email, orgDomain) {
		return false
	}
	if len(rule.Senders) > 0 && !matchesSenders(m.From.Email, rule.Senders) {
		return false
	}
	if len(rule.Keywords) > 0 && !matchesKeywords(m.Subject, rule.Keywords) {
		return false
	}
	return true
}

// NotificationTexts возвращает заголовок и тело уведомления. При
// show_subject=false тема не используется вообще (SECURITY.md §9).
func NotificationTexts(m mail.MessageSummary, showSubject bool) (title, body string) {
	from := strings.TrimSpace(m.From.Name)
	if from == "" {
		from = m.From.Email
	}
	title = "Новое письмо от " + from
	if showSubject {
		body = m.Subject
	}
	return title, body
}

func passesImportance(min string) bool {
	return min == "" || min == "low" || min == "normal"
}

func matchesSenders(email string, senders []string) bool {
	e := strings.ToLower(email)
	for _, s := range senders {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "@") {
			if strings.HasSuffix(e, s) {
				return true
			}
			continue
		}
		if e == s {
			return true
		}
	}
	return false
}

func matchesKeywords(subject string, keywords []string) bool {
	s := strings.ToLower(subject)
	for _, k := range keywords {
		if strings.Contains(s, strings.ToLower(k)) {
			return true
		}
	}
	return false
}
