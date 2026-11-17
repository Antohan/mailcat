// Package ui — экраны TUI (tview): почта и календарь. Весь текст из EWS
// проходит security.Sanitize на границе — в функциях форматирования ниже —
// и только затем попадает в виджеты (docs/SECURITY.md §3).
package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/security"
)

// ExternalMark возвращает "[ВНЕШН]", если домен отправителя не совпадает с
// доменом организации (анти-фишинг, docs/SECURITY.md §4.1). Совпадение — по
// суффиксу домена: sub.company.example считается внутренним,
// company.example.evil.tld — нет.
func ExternalMark(fromEmail, orgDomain string) string {
	if fromEmail == "" || orgDomain == "" {
		return ""
	}
	addr := strings.ToLower(fromEmail)
	domain := addr[strings.LastIndex(addr, "@")+1:]
	org := strings.ToLower(orgDomain)
	if domain == org || strings.HasSuffix(domain, "."+org) {
		return ""
	}
	return "[ВНЕШН]"
}

// FormatMessageRow — колонки строки списка писем: [ВНЕШН] | реальный адрес
// отправителя | тема | дата (локальная зона, короткий формат). Текст из EWS
// проходит Sanitize (§3) и tview.Escape — Table интерпретирует теги цветов/
// регионов безусловно (ревью этапа 2).
func FormatMessageRow(m mail.MessageSummary, orgDomain string) []string {
	date := ""
	if !m.Received.IsZero() {
		date = m.Received.Local().Format("02.01 15:04")
	}
	return []string{
		ExternalMark(m.From.Email, orgDomain),
		tview.Escape(security.Sanitize(m.From.Email)),
		tview.Escape(security.Sanitize(m.Subject)),
		date,
	}
}

// FormatFolder — пункт списка папок с счётчиком непрочитанных; Sanitize +
// tview.Escape (List интерпретирует теги безусловно).
func FormatFolder(f mail.Folder) string {
	name := tview.Escape(security.Sanitize(f.DisplayName))
	if f.UnreadCount > 0 {
		return fmt.Sprintf("%s (%d)", name, f.UnreadCount)
	}
	return name
}

// FormatMessageHeader — блок заголовков письма для панели просмотра:
// От/Кому/Дата/Тема. Реальный адрес отправителя показывается всегда
// (SECURITY.md §4.1).
func FormatMessageHeader(m mail.Message) string {
	var b strings.Builder
	b.WriteString("От:    " + security.Sanitize(m.From.String()) + "\n")
	if len(m.To) > 0 {
		b.WriteString("Кому:  " + security.Sanitize(joinAddresses(m.To)) + "\n")
	}
	if !m.Received.IsZero() {
		b.WriteString("Дата:  " + m.Received.Local().Format("02.01.2006 15:04") + "\n")
	}
	b.WriteString("Тема:  " + security.Sanitize(m.Subject) + "\n")
	return b.String()
}

// FormatAttachments — список вложений с человекочитаемыми размерами
// (модал деталей, этап 3 добавит клавиши действий).
func FormatAttachments(list []mail.Attachment) string {
	if len(list) == 0 {
		return ""
	}
	lines := make([]string, 0, len(list))
	for _, a := range list {
		lines = append(lines, fmt.Sprintf("  • %s (%s)", security.Sanitize(a.Name), a.SizeLabel()))
	}
	return "Вложения:\n" + strings.Join(lines, "\n")
}

func joinAddresses(list []mail.Address) string {
	parts := make([]string, 0, len(list))
	for _, a := range list {
		parts = append(parts, a.String())
	}
	return strings.Join(parts, ", ")
}
