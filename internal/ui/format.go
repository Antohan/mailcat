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

// sameLocalDay — те же локальные календарные сутки (граница — местная
// полночь, не 24 часа назад).
func sameLocalDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// FormatShortDate — компактная дата строки списка (UI-2): сегодня → HH:MM,
// давность ≤ 7 дней → сокращённый день недели («Чт»), старше → ДД.ММ.
// now передаётся явно — детерминизм в тестах.
func FormatShortDate(now, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	tl, nl := t.Local(), now.Local()
	if sameLocalDay(tl, nl) {
		return tl.Format("15:04")
	}
	if tl.After(nl.AddDate(0, 0, -7)) {
		return shortDays[tl.Weekday()]
	}
	return tl.Format("02.01")
}

// dayBuckets — заголовки разделителей дней в списке (О-1); индекс — группа
// dayBucket (0 — сегодня).
var dayBuckets = []string{"Сегодня", "Вчера", "Ранее"}

// dayBucket — группа дня для разделителя списка (О-1): 0 — сегодня, 1 —
// вчера, 2 — ранее; нулевая дата (Received неизвестен) — «Ранее».
// now передаётся явно — детерминизм в тестах.
func dayBucket(now, t time.Time) int {
	if t.IsZero() {
		return 2
	}
	tl, nl := t.Local(), now.Local()
	if sameLocalDay(tl, nl) {
		return 0
	}
	if sameLocalDay(tl, nl.AddDate(0, 0, -1)) {
		return 1
	}
	return 2
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

// messageDot — точка статуса в строке 1 пункта списка (V-1). Цвет задаёт
// ячейка в appendRows (colorUnread — непрочитан, colorTextPrimary —
// прочитан; в тёмной теме colorUnread = accent), поэтому знак один для
// обоих состояний.
const messageDot = "●"

// externalMarkCol — фиксированная ширина маркера [ВНЕШН] (П.4г): пустой
// маркер дополняется пробелами, чтобы отправители всех строк начинались с
// одного столбца и панель не «дышала» при появлении внешних писем.
const externalMarkCol = 8

// subjectIndent — отступ строки 2 (темы): «● » (2) + externalMarkCol.
var subjectIndent = strings.Repeat(" ", 2+externalMarkCol)

// FormatMessageListItem — контент двухстрочного пункта списка писем (V-1):
// [строка 1, время, строка 2]. Строка 1 — точка статуса + [ВНЕШН]
// фиксированной ширины + отправитель (t:Name, fallback — локальная часть
// адреса); время — отдельная правая колонка (appendRows ставит AlignRight,
// панели не «дышет»); строка 2 — тема с отступом под отправителя, усечение
// с «…» делает truncateEllipsis в DrawFunc (WHY у неё). Цвета и жирность
// назначает appendRows через SetTextColor, а не теги в тексте: выделенная
// строка Table перекрашивается целиком через SetSelectedStyle, теги бы
// пробивались (исключение — точка статуса, WHY у dotLine). Текст из EWS
// проходит Sanitize (§3) и tview.Escape — Table интерпретирует
// теги цветов/регионов безусловно (ревью этапа 2).
func FormatMessageListItem(m mail.MessageSummary, orgDomains []string, now time.Time) []string {
	mark := fmt.Sprintf("%-*s", externalMarkCol, ExternalMark(m.From.Email, orgDomains))
	return []string{
		messageDot + " " + mark + senderLabel(m.From),
		FormatShortDate(now, m.Received),
		subjectIndent + tview.Escape(security.Sanitize(m.Subject)),
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

// truncateEllipsis — обрезка строки под max клеток с «…» в конце.
// WHY: своё, а не встроенное многоточие tview.Table — tview сравнивает
// байтовую длину напечатанного (end-start) с шириной в клетках, у кириллицы
// байты вдвое больше клеток и условие «текст не влез» всегда ложно.
// Порог по рунам, не по клеткам — same best effort, что и displayWidth.
func truncateEllipsis(s string, max int) string {
	if max < 1 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

// dotLine красит точку статуса тегом: непрочитанный — colorUnread (в тёмной
// теме accent), прочитанный — colorRead (muted, П.1 референса). WHY: тег
// осознанно пробивает SetSelectedStyle — статус письма виден и на выделенной
// строке; «[-]» возвращает остальной текст (отправителя) в цвет ячейки/
// выделения.
func dotLine(text string, unread bool) string {
	c := colorRead
	if unread {
		c = colorUnread
	}
	return "[" + hexTag(c) + "]" + messageDot + "[-]" + strings.TrimPrefix(text, messageDot)
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

// FormatMessageHeader — блок заголовков письма (RD-5): Тема (жирный) → От →
// Кому (если есть) → Дата → линия-разделитель; тело печатает вызывающий.
// Служит и превью (MessageSummary без Кому/вложений), и деталям письма.
// WHY: маркер — в строке темы, а не отдельной строкой: ниже в карточке
// FormatAttachments уже печатает список с заголовком «Вложения:».
// Важность (⚠) не показывается: item:Importance нет в EWS-REFERENCE, с
// сервера не приходит (открытый вопрос в HANDOFF «Открыто»).
// Реальный адрес отправителя показывается всегда (SECURITY.md §4.1).
// Вывод содержит теги tview: получатели включают SetDynamicColors(true),
// поэтому все данные из EWS проходят Sanitize + tview.Escape (инвариант escape).
func FormatMessageHeader(m mail.Message) string {
	var b strings.Builder
	topic := tview.Escape(security.Sanitize(m.Subject))
	if n := len(m.Attachments); n > 0 {
		topic += fmt.Sprintf("  📎 %d", n)
	}
	fmt.Fprintf(&b, "[::b]Тема:  %s[-::-]\n", topic)
	b.WriteString("От:    " + tview.Escape(security.Sanitize(m.From.String())) + "\n")
	if len(m.To) > 0 {
		b.WriteString("Кому:  " + tview.Escape(security.Sanitize(joinAddresses(m.To))) + "\n")
	}
	if !m.Received.IsZero() {
		b.WriteString("Дата:  " + m.Received.Local().Format("02.01.2006 15:04") + "\n")
	}
	fmt.Fprintf(&b, "%s\n", mutedText(strings.Repeat("─", headerRuleWidth)))
	return b.String()
}

// headerRuleWidth — длина линии-разделителя под заголовками письма (RD-5):
// фикс, а не ширина панели — узкое окно TextView переносит сам.
const headerRuleWidth = 32

// mutedText красит статический текст интерфейса (не из EWS — escape не нужен)
// в textSecondary. Заглушки и линия-разделитель письма (RD-2/RD-5):
// линия — textSecondary, не hairline.
func mutedText(s string) string {
	return fmt.Sprintf("[%s]%s[-]", hexTag(colorTextSecondary), s)
}

// FormatAttachments — список вложений с человекочитаемыми размерами
// (модал деталей, этап 3 добавит клавиши действий).
func FormatAttachments(list []mail.Attachment) string {
	if len(list) == 0 {
		return ""
	}
	lines := make([]string, 0, len(list))
	for _, a := range list {
		// escapeForList (Sanitize + tview.Escape), не просто Sanitize: вывод
		// печатается в TextView с dynamic colors (RD-5) — «[» в имени вложения
		// иначе стал бы тегом (находка security-reviewer).
		lines = append(lines, fmt.Sprintf("  • %s (%s)", escapeForList(a.Name), a.SizeLabel()))
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
