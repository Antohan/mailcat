// Package ui — экраны TUI (tview): почта и календарь. Весь текст из EWS
// проходит security.Sanitize на границе — в функциях форматирования ниже —
// и только затем попадает в виджеты (docs/SECURITY.md §3).
package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
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

// shortDays — сокращения дней недели для компактной даты списка (UI-2);
// индексация по time.Weekday() (воскресенье = 0).
var shortDays = []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}

// FormatShortDate — компактная дата строки списка (UI-2): сегодня → HH:MM,
// давность ≤ 7 дней → сокращённый день недели («Чт»), старше → ДД.ММ.
// now передаётся явно — детерминизм в тестах.
func FormatShortDate(now, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	tl, nl := t.Local(), now.Local()
	if tl.Year() == nl.Year() && tl.YearDay() == nl.YearDay() {
		return tl.Format("15:04")
	}
	if tl.After(nl.AddDate(0, 0, -7)) {
		return shortDays[tl.Weekday()]
	}
	return tl.Format("02.01")
}

// senderLabel — колонка отправителя (UI-2): имя из t:Name, при отсутствии —
// локальная часть адреса (youtrack@… ×20 — нечитаемо). Полный адрес остаётся
// в заголовке выбранного письма (SECURITY.md §4.1 — адрес доступен всегда).
func senderLabel(a mail.Address) string {
	if name := strings.TrimSpace(a.Name); name != "" {
		return tview.Escape(security.Sanitize(name))
	}
	email := a.Email
	if i := strings.IndexByte(email, '@'); i > 0 {
		email = email[:i]
	}
	return tview.Escape(security.Sanitize(email))
}

// FormatMessageRow — колонки строки списка писем (UI-2): [ВНЕШН] | имя
// отправителя (t:Name, fallback — локальная часть адреса) | тема |
// компактная дата (FormatShortDate). Текст из EWS
// проходит Sanitize (§3) и tview.Escape — Table интерпретирует теги цветов/
// регионов безусловно (ревью этапа 2).
func FormatMessageRow(m mail.MessageSummary, orgDomains []string, now time.Time) []string {
	return []string{
		ExternalMark(m.From.Email, orgDomains),
		senderLabel(m.From),
		tview.Escape(security.Sanitize(m.Subject)),
		FormatShortDate(now, m.Received),
	}
}

// folderIcons — unicode-иконки стандартных папок (RD-3); остальные папки —
// без иконки, с отступом под иконку. Точный список, а не эвристика по
// подстроке: «Архив Входящие» не должен получить иконку входящих.
var folderIcons = map[string]string{
	"Входящие":      "📥",
	"Отправленные":  "📤",
	"Черновики":     "📝",
	"Удалённые":     "🗑",
	"Нежелательная": "🚫",
}

// folderNameCol — ширина колонки имени (иконка/отступ + имя): 3 клетки на
// иконку-эмодзи (двойная ширина) и 13 на «Нежелательная». Счётчик прижат
// вправо выравниванием на эту колонку; у более длинных имён — минимальный
// отступ в 1 клетку.
const folderNameCol = 16

// displayWidth — наивная ширина в клетках терминала: руны по одной, плюс
// лишняя клетка на каждый эмодзи из folderIcons (в Terminal.app они двойной
// ширины — это и проверяем). WHY: считаем только собственные иконки, а не
// общий west-asian-алгоритм — зависимость uniseg не одобрена (AGENTS.md §6),
// а имена с CJK дают заниженную ширину; выравнивание best effort.
func displayWidth(s string) int {
	w := utf8.RuneCountInString(s)
	for _, icon := range folderIcons {
		if strings.Contains(s, icon) {
			w += utf8.RuneCountInString(icon)
		}
	}
	return w
}

// hexTag — цвет роли темы в виде тега tview («#2563eb»), чтобы цвет строки
// менялся вместе с темой, а не был захардкожен в тексте.
func hexTag(c tcell.Color) string {
	return fmt.Sprintf("#%06x", c.Hex())
}

// FormatFolder — пункт списка папок: иконка/отступ, имя, счётчик непрочитанных
// цветом textSecondary (прижат вправо на folderNameCol). active — папка
// открыта: текст accent + жирный. Sanitize + tview.Escape сохраняются (List
// интерпретирует теги безусловно — данные экранируются, теги наши).
func FormatFolder(f mail.Folder, active bool) string {
	indent, hasIcon := folderIcons[f.DisplayName]
	if !hasIcon {
		indent = "   "
	} else {
		indent += " "
	}
	pad := folderNameCol - displayWidth(indent) - utf8.RuneCountInString(f.DisplayName)
	if pad < 1 {
		pad = 1
	}
	row := indent + tview.Escape(security.Sanitize(f.DisplayName)) + strings.Repeat(" ", pad)
	if active {
		row = fmt.Sprintf("[%s::b]%s[-::-]", hexTag(colorAccent), row)
	}
	if f.UnreadCount > 0 {
		return fmt.Sprintf("%s[%s](%d)", row, hexTag(colorTextSecondary), f.UnreadCount)
	}
	return row
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
// От/Кому/Дата/Тема + 📎 N при вложениях (UI-3, метаданные GetItem).
// WHY: маркер — в строке темы, а не отдельной строкой: ниже в карточке
// FormatAttachments уже печатает список с заголовком «Вложения:».
// Важность (⚠) не показывается: item:Importance нет в EWS-REFERENCE, с
// сервера не приходит (открытый вопрос в HANDOFF «Открыто»).
// Реальный адрес отправителя показывается всегда (SECURITY.md §4.1).
func FormatMessageHeader(m mail.Message) string {
	var b strings.Builder
	b.WriteString("От:    " + security.Sanitize(m.From.String()) + "\n")
	if len(m.To) > 0 {
		b.WriteString("Кому:  " + security.Sanitize(joinAddresses(m.To)) + "\n")
	}
	if !m.Received.IsZero() {
		b.WriteString("Дата:  " + m.Received.Local().Format("02.01.2006 15:04") + "\n")
	}
	topic := "Тема:  " + security.Sanitize(m.Subject)
	if n := len(m.Attachments); n > 0 {
		topic += fmt.Sprintf("  📎 %d", n)
	}
	b.WriteString(topic + "\n")
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
