// Package ui — экраны TUI (tview): почта и календарь. Весь текст из EWS
// проходит security.Sanitize на границе — в функциях форматирования ниже —
// и только затем попадает в виджеты (docs/SECURITY.md §3).
package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rivo/tview"

	"mailcat/internal/mail"
	"mailcat/internal/security"
)

// ExternalMark возвращает "[ВНЕШН]", если домен отправителя не входит ни в
// один домен организации (анти-фишинг, docs/SECURITY.md §4.1). Логика —
// security.IsExternalEmail, здесь только представление.
func ExternalMark(fromEmail string, orgDomains []string) string {
	if security.IsExternalEmail(fromEmail, orgDomains...) {
		return "[ВНЕШН]"
	}
	return ""
}

// FormatMessageRow — колонки строки списка писем: [ВНЕШН] | реальный адрес
// отправителя | тема | дата (локальная зона, короткий формат). Текст из EWS
// проходит Sanitize (§3) и tview.Escape — Table интерпретирует теги цветов/
// регионов безусловно (ревью этапа 2).
func FormatMessageRow(m mail.MessageSummary, orgDomains []string) []string {
	date := ""
	if !m.Received.IsZero() {
		date = m.Received.Local().Format("02.01 15:04")
	}
	return []string{
		ExternalMark(m.From.Email, orgDomains),
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

// standardFolders — фиксированный порядок стандартных папок в панели (UI-1).
var standardFolders = []string{
	"Входящие", "Черновики", "Исходящие", "Отправленные", "Удалённые", "Нежелательная",
}

// technicalFolders — служебные папки, замеченные на живом сервере (обкатка
// 20.11): в основном списке — шум, но клиент read-only и «спрятать» ≠
// «удалить», поэтому они уходят в сворачиваемую секцию «Ещё ▸», а не в никуда.
// WHY: точный список, а не эвристика по имени — незнакомую папку можно
// принять за пользовательскую; additions — через ui.hidden_folders в конфиге.
var technicalFolders = map[string]bool{
	"Conversation Action Settings": true,
	"Корневая папка Yammer":        true,
	"Опции":                        true,
}

// SplitFolders делит ответ FindFolder для панели «Папки» (UI-1): основной
// список — стандартные в порядке standardFolders, затем остальные по
// алфавиту (байтовый порядок stdlib: латиница раньше кириллицы; approved-
// deps не включают x/text/collate, а смешанных алфавитов в одном ящике
// почти не бывает); technicalFolders — отдельным списком для «Ещё ▸».
// hidden — точные имена из ui.hidden_folders, исключаются полностью.
func SplitFolders(folders []mail.Folder, hidden []string) (main, technical []mail.Folder) {
	hiddenSet := make(map[string]bool, len(hidden))
	for _, h := range hidden {
		hiddenSet[h] = true
	}
	main = make([]mail.Folder, 0, len(folders))
	for _, f := range folders {
		switch {
		case hiddenSet[f.DisplayName]:
			// полностью скрыта пользователем
		case technicalFolders[f.DisplayName]:
			technical = append(technical, f)
		default:
			main = append(main, f)
		}
	}
	rank := func(f mail.Folder) int {
		for i, name := range standardFolders {
			if f.DisplayName == name {
				return i
			}
		}
		return len(standardFolders)
	}
	sort.SliceStable(main, func(i, j int) bool {
		ri, rj := rank(main[i]), rank(main[j])
		if ri != rj {
			return ri < rj
		}
		return main[i].DisplayName < main[j].DisplayName
	})
	sort.Slice(technical, func(i, j int) bool {
		return technical[i].DisplayName < technical[j].DisplayName
	})
	return main, technical
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
