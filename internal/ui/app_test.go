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
	row := FormatMessageListItem(m, []string{"company.example"}, time.Now(), false)
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
	got := FormatFolder(mail.Folder{DisplayName: `["x"]Входящие`}, false, false, 0)
	if !strings.Contains(got, `["x"[]`) {
		t.Errorf("имя папки не экранировано: %q", got)
	}
	if got2 := FormatFolder(mail.Folder{DisplayName: `[ВХОД]ящик`}, false, false, 0); !strings.Contains(got2, `[ВХОД]ящик`) {
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
	got := formatHint([]hintSeg{{"Esc", "назад", false}, {"q", "выход", false}})
	want := fmt.Sprintf("[#%06x::b]%s [#%06x]назад [#%06x]· [#%06x::b]%s [#%06x]выход",
		colorAccent.Hex(), tview.Escape("[Esc]"),
		colorTextSecondary.Hex(), colorTextSecondary.Hex(),
		colorAccent.Hex(), tview.Escape("[q]"), colorTextSecondary.Hex())
	if got != want {
		t.Errorf("formatHint = %q, хочу %q", got, want)
	}
}

// TestFormatHintFit (V-17): на узком экране необязательный сегмент «←/→
// панели» снимается, чтобы подсказка влезала в 100 колонок; на широком —
// показывается; без opt-сегментов текст не меняется.
func TestFormatHintFit(t *testing.T) {
	full := formatHint(hintMail)
	if got := formatHintFit(hintMail, 120); got != full {
		t.Errorf("широкий экран: подсказка урезана")
	}
	fullW := tview.TaggedStringWidth(full)
	narrow := formatHintFit(hintMail, fullW-1)
	if strings.Contains(narrow, "панели") {
		t.Errorf("узкий экран: необязательный сегмент не снят: %q", narrow)
	}
	if tview.TaggedStringWidth(narrow) > fullW-1 {
		t.Errorf("узкий экран: подсказка всё ещё шире (%d > %d)",
			tview.TaggedStringWidth(narrow), fullW-1)
	}
	custom := []hintSeg{{"Esc", "назад", false}, {"q", "выход", false}}
	if got := formatHintFit(custom, 3); got != formatHint(custom) {
		t.Errorf("без opt-сегментов текст изменился: %q", got)
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

// TestReadingInPreview (V-15): Enter больше не открывает полноэкранную
// страницу — письмо читается в панели просмотра, фокус и подсказки уходят
// туда; выбор другой строки возвращает панель в режим черновика.
func TestReadingInPreview(t *testing.T) {
	a := newTestApp(t)
	a.mail.showReading(mail.Message{
		MessageSummary: mail.MessageSummary{
			Subject:  "Тема",
			From:     mail.Address{Name: "Иван", Email: "ivan@company.example"},
			Received: time.Date(2026, 11, 20, 21, 25, 0, 0, time.Local),
		},
		To:       []mail.Address{{Name: "everybody", Email: "everybody@company.example"}},
		BodyText: "тело",
	})
	if !a.mail.reading || a.tapp.GetFocus() != a.mail.preview {
		t.Fatalf("письмо не открылось в панели чтения: reading=%v", a.mail.reading)
	}
	// V-17: шапка подробная (Тема/От/Кому/Дата), а не аватар-карточка V-5.
	text := a.mail.preview.GetText(false)
	for _, want := range []string{"Тема:", "От:", "Кому:", "Дата:", "20.11.2026 21:25"} {
		if !strings.Contains(text, want) {
			t.Errorf("в шапке нет %q:\n%s", want, text)
		}
	}
	if got := a.hint.GetText(false); got != formatHint(hintReading) {
		t.Errorf("подсказки панели чтения: %q", got)
	}
	a.mail.showPreview(-1) // курсор ушёл со строки — панель снова черновик
	if a.mail.reading {
		t.Error("reading не сброшен при смене строки — resize вернёт старое письмо")
	}
}

// TestInitialFocusMessages (V-21): приложение стартует с фокусом в списке
// писем — подсказка стартового экрана (hintMail) и акцентная рамка совпадают.
func TestInitialFocusMessages(t *testing.T) {
	a := newTestApp(t)
	if got := a.tapp.GetFocus(); got != a.mail.messages {
		t.Errorf("стартовый фокус не в списке писем: %T", got)
	}
}

// TestCycleFocusUpdatesHint (V-23): Tab/Shift-Tab вместе с фокусом меняет и
// строку подсказок — она не остаётся от прежней панели.
func TestCycleFocusUpdatesHint(t *testing.T) {
	a := newTestApp(t)
	a.mail.showReading(mail.Message{
		MessageSummary: mail.MessageSummary{Subject: "Тема"},
	})
	if got := a.hint.GetText(false); got != formatHint(hintReading) {
		t.Fatalf("после открытия письма подсказки панели чтения: %q", got)
	}
	a.mail.cycleFocus() // просмотр → папки
	if a.tapp.GetFocus() != a.mail.folders || a.hint.GetText(false) != formatHint(hintFolders) {
		t.Errorf("Tab из просмотра не обновил подсказку: %q", a.hint.GetText(false))
	}
	a.mail.cycleFocus() // папки → письма
	a.mail.cycleFocus() // письма → просмотр
	if a.tapp.GetFocus() != a.mail.preview || a.hint.GetText(false) != formatHint(hintReading) {
		t.Errorf("возврат в просмотр не вернул подсказки панели: %q", a.hint.GetText(false))
	}
}

// TestFoldersHint (V-24): фокус на папках даёт набор панели папок, и он
// укладывается в 100 колонок без opt-сегментов.
func TestFoldersHint(t *testing.T) {
	a := newTestApp(t)
	a.tapp.SetFocus(a.mail.folders)
	if got := formatHint(a.pageHintSegs()); got != formatHint(hintFolders) {
		t.Errorf("подсказки фокуса на папках: %q", got)
	}
	if w := tview.TaggedStringWidth(formatHint(hintFolders)); w > 100 {
		t.Errorf("подсказки папок шире 100 колонок: %d", w)
	}
	for _, s := range hintFolders {
		if s.opt {
			t.Errorf("сегмент %q помечен opt — на 100 колонках обязательная клавиша пропадёт", s.key)
		}
	}
}
