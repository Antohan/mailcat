package ui

// Тесты санитизации на границе UI по итогам security-ревью этапа 2:
// 1) текст ошибок EWS (EWSError.MessageText/faultstring) — серверный,
//    до терминала только через Sanitize;
// 2) строки для tview.Table/List — tview.Escape (теги [red]/["region"]
//    интерпретируются этими виджетами безусловно).

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rivo/tview"

	"mailcat/internal/config"
	"mailcat/internal/mail"
)

// newTestApp — App без запуска цикла событий (сеть не трогается).
func newTestApp(t *testing.T) *App {
	t.Helper()
	return New(config.Defaults(), nil, nil)
}

// TestShowErrorSanitized: OSC-инъекция в тексте ошибки не доходит до статус-бара.
func TestShowErrorSanitized(t *testing.T) {
	a := newTestApp(t)
	a.showError(errors.New("EWS ErrorInternalServerError: \x1b]52;c;aGVsbG8=\x07упс \x1b]8;;http://evil\x1b\\ссылка"))
	text := a.status.GetText(false)
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("в статус-бар прошёл ESC: %q", text)
	}
}

// TestShowErrorRunesTruncation: усечение длинного текста не рвёт русские руны.
func TestShowErrorRunesTruncation(t *testing.T) {
	a := newTestApp(t)
	long := strings.Repeat("ж", 500)
	a.showError(errors.New(long))
	text := a.status.GetText(false)
	runes := []rune(text)
	// Без усечения было бы ~530 рун; с усечением — ~230, и все целые.
	if len(runes) > 250 {
		t.Errorf("длинная ошибка не усечена: %d рун", len(runes))
	}
	for _, r := range runes {
		if r == 0xFFFD {
			t.Errorf("усечение разорвало руну: %q", text)
			break
		}
	}
	if !strings.Contains(text, "ж") {
		t.Errorf("текст ошибки потерян: %q", text)
	}
}

// TestFormatMessageListItemEscaped: теги tview в теме/имени отправителя
// экранируются (Table интерпретирует [red] даже без SetDynamicColors;
// tview.Escape вставляет "[]" внутрь тега — семантика tview).
func TestFormatMessageListItemEscaped(t *testing.T) {
	m := mail.MessageSummary{
		Subject: "[red]СРОЧНО[-] проверка",
		From:    mail.Address{Email: `["evil"]@x.example`},
	}
	row := FormatMessageListItem(m, []string{"company.example"}, time.Now())
	if !strings.Contains(row[2], "[red[]") {
		t.Errorf("тег [red] в теме не экранирован: %q", row[2])
	}
	if !strings.Contains(row[0], `["evil"[]`) {
		t.Errorf("region-тег в адресе не экранирован: %q", row[0])
	}
}

// TestFormatFolderEscaped: имя папки с тегом экранируется для tview.List.
// Кириллица в скобках ([ВХОД]) тегом не является — проходит как есть.
func TestFormatFolderEscaped(t *testing.T) {
	got := FormatFolder(mail.Folder{DisplayName: `["x"]Входящие`}, false, false)
	if !strings.Contains(got, `["x"[]`) {
		t.Errorf("имя папки не экранировано: %q", got)
	}
	if got2 := FormatFolder(mail.Folder{DisplayName: `[ВХОД]ящик`}, false, false); !strings.Contains(got2, `[ВХОД]ящик`) {
		t.Errorf("безопасные скобки искажены: %q", got2)
	}
}

// TestPaintFocusFramesWired (UI-6): все пять панелей почты/календаря
// зарегистрированы под акцентную рамку; хук возвращает false — отрисовка
// продолжается.
func TestPaintFocusFramesWired(t *testing.T) {
	a := newTestApp(t)
	if len(a.focusPanels) != 5 {
		t.Fatalf("панелей с рамкой фокуса = %d, хочу 5", len(a.focusPanels))
	}
	if a.paintFocusFrames(nil) {
		t.Error("paintFocusFrames прервал отрисовку (вернул true)")
	}
}

// TestPageHint (UI-7): подсказка статус-бара соответствует экрану —
// у почты и календаря свои наборы клавиш, модалы — свои.
func TestPageHint(t *testing.T) {
	a := newTestApp(t)
	hints := map[string]string{
		pageMail:        formatHint(hintMail),
		pageCalendar:    formatHint(hintCalendar),
		pageDetails:     formatHint(hintDetails),
		pageEvent:       formatHint(hintEvent),
		pageLinks:       formatHint(hintLinks),
		pageAttachments: formatHint(hintAttachments),
	}
	// Модальные страницы не существуют до первого открытия — добавляем
	// заглушку, SwitchToPage непроверенной страницы ничего не меняет.
	a.pages.AddPage(pageLinks, tview.NewBox(), true, false)
	a.pages.AddPage(pageAttachments, tview.NewBox(), true, false)
	for page, want := range hints {
		a.pages.SwitchToPage(page)
		if got := a.pageHint(); got != want {
			t.Errorf("страница %s: подсказка %q, хочу %q", page, got, want)
		}
	}
}

// TestFormatHint (V-7, RD-6): клавиша — в бейдже `[клавиша]` акцентом
// жирным, подпись — второстепенным цветом; текст наш (не из EWS), теги
// динамических цветов допустимы, «[» бейджа экранирован от парсера tview.
func TestFormatHint(t *testing.T) {
	got := formatHint([]hintSeg{{"Esc", "назад"}, {"q", "выход"}})
	want := fmt.Sprintf("[#%06x::b]%s [#%06x]назад [#%06x]· [#%06x::b]%s [#%06x]выход",
		colorAccent.Hex(), tview.Escape("[Esc]"),
		colorTextSecondary.Hex(), colorTextSecondary.Hex(),
		colorAccent.Hex(), tview.Escape("[q]"), colorTextSecondary.Hex())
	if got != want {
		t.Errorf("formatHint = %q, хочу %q", got, want)
	}
}

// TestErrorAutoClear (UI-7/UI-9): ошибка живёт в строке сообщений и не
// трогает строку подсказок; после таймаута сообщение гаснет; таймер с
// устаревшим поколением свежий статус не затирает.
func TestErrorAutoClear(t *testing.T) {
	a := newTestApp(t)
	a.showError(errors.New("сеть недоступна"))
	if text := a.status.GetText(false); !strings.Contains(text, "Ошибка") {
		t.Fatalf("ошибка не показана: %q", text)
	}
	if got := a.hint.GetText(false); got != formatHint(hintMail) {
		t.Errorf("ошибка затронула строку подсказок: %q", got)
	}
	a.clearError(a.statusGen) // то, что таймер прислал бы в QueueUpdateDraw
	if text := a.status.GetText(false); text != "" {
		t.Errorf("сообщение не погасло после таймаута: %q", text)
	}
	// Статус сменился после ошибки — таймер должен устареть.
	a.showError(errors.New("вторая ошибка"))
	a.setStatus("Загрузка…")
	a.clearError(a.statusGen - 1)
	if text := a.status.GetText(false); text != "Загрузка…" {
		t.Errorf("устаревший таймер затёр статус: %q", text)
	}
}

// TestLoadIndicator (UI-8/UI-9): индикатор загрузки живёт в строке сообщений
// и сосуществует с подсказками — смена страницы и автоочистка ошибки надпись
// не затирают; последняя endLoad гасит строку сообщений, подсказки остаются.
func TestLoadIndicator(t *testing.T) {
	a := newTestApp(t)
	id := a.beginLoad("Загрузка писем…")
	if text := a.status.GetText(false); text != spinnerFrames[0:3]+" Загрузка писем…" {
		t.Fatalf("индикатор не показан: %q", text)
	}
	if got := a.hint.GetText(false); got != formatHint(hintMail) {
		t.Fatalf("подсказки пропали во время загрузки: %q", got)
	}
	a.showHint() // смена страницы во время загрузки
	if text := a.status.GetText(false); text != spinnerFrames[0:3]+" Загрузка писем…" {
		t.Errorf("подсказка затёрла индикатор загрузки: %q", text)
	}
	// Перекрытие загрузок: завершилась не последняя — остаётся надпись идущей.
	id2 := a.beginLoad("Загрузка письма…")
	a.endLoad(id2)
	if text := a.status.GetText(false); text != "Загрузка писем…" {
		t.Errorf("надпись ещё идущей загрузки потеряна: %q", text)
	}
	// Ошибка во время загрузки: автоочистка возвращает индикатор, не пустую строку.
	a.showError(errors.New("сбой"))
	a.clearError(a.statusGen) // то, что таймер прислал бы в QueueUpdateDraw
	if text := a.status.GetText(false); text != "Загрузка писем…" {
		t.Errorf("после ошибки не вернулся индикатор загрузки: %q", text)
	}
	a.endLoad(id)
	if text := a.status.GetText(false); text != "" {
		t.Errorf("после последней загрузки строка сообщений не погасла: %q", text)
	}
	if got := a.hint.GetText(false); got != formatHint(hintMail) {
		t.Errorf("подсказки после загрузки не на месте: %q", got)
	}
}

// TestHintLineFollowsPages (UI-9): нижняя строка — подсказки видимого экрана,
// видны ВСЕГДА: сообщения, ошибки и загрузки их не меняют.
func TestHintLineFollowsPages(t *testing.T) {
	a := newTestApp(t)
	a.pages.SwitchToPage(pageCalendar)
	a.showHint()
	if got := a.hint.GetText(false); got != formatHint(hintCalendar) {
		t.Errorf("строка подсказок не сменилась со страницей: %q", got)
	}
	id := a.beginLoad("Загрузка календаря…")
	a.showError(errors.New("сбой"))
	if got := a.hint.GetText(false); got != formatHint(hintCalendar) {
		t.Errorf("сообщение/ошибка затронули строку подсказок: %q", got)
	}
	a.endLoad(id)
	if got := a.hint.GetText(false); got != formatHint(hintCalendar) {
		t.Errorf("конец загрузки затронул строку подсказок: %q", got)
	}
}
