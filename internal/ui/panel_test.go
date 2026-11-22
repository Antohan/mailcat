package ui

// Тест-сторож каркаса (RD-2): раскладка почты на 100×30 / 140×40 / 220×60
// отрисовывается без паники, заголовки панелей — капсом, пропорции 20/42/38.

import (
	"fmt"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"mailcat/internal/mail"
)

// simRow — строка y экрана симуляции (по руне на ячейку).
func simRow(sim tcell.SimulationScreen, y int) []rune {
	w, _ := sim.Size()
	row := make([]rune, w)
	for x := 0; x < w; x++ {
		row[x], _, _, _ = sim.GetContent(x, y)
	}
	return row
}

// runeIndex — позиция подстроки в рунах (строки.Index считает байты UTF-8,
// а нам нужны экранные столбцы).
func runeIndex(row []rune, sub string) int {
	want := []rune(sub)
	for i := 0; i+len(want) <= len(row); i++ {
		if string(row[i:i+len(want)]) == sub {
			return i
		}
	}
	return -1
}

func TestPanelLayoutSizes(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{100, 30}, {140, 40}, {220, 60}} {
		a := newTestApp(t)
		sim := tcell.NewSimulationScreen("UTF-8")
		if err := sim.Init(); err != nil {
			t.Fatalf("%dx%d: инициализация симуляции: %v", sz.w, sz.h, err)
		}
		// WHY: SetScreen сам вызывает screen.Init() и сбрасывает размер на
		// 80×25 — SetSize должен идти после SetScreen, иначе тест проверяет
		// не тот размер.
		a.tapp.SetScreen(sim)
		sim.SetSize(sz.w, sz.h)
		// Draw()/ForceDraw() вне цикла событий не работают: первый идёт через
		// QueueUpdate (вечное ожидание), второй требует мьютекса цикла.
		// Поэтому поднимаем цикл и ждём отрисовки через QueueUpdateDraw.
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = a.tapp.Run()
		}()
		a.tapp.QueueUpdateDraw(func() {})

		// Строка 0 — верхний бар (V-6), подпись списка — строкой ниже.
		// V-16: у ПАПОК и ПРОСМОТРА подписи нет, осталась только ПИСЬМА.
		row := simRow(sim, 1)
		messages := runeIndex(row, "ПИСЬМА")
		if messages < 0 {
			t.Fatalf("%dx%d: подпись «ПИСЬМА» не найдена: %q", sz.w, sz.h, string(row))
		}
		if runeIndex(row, "ПАПКИ") >= 0 || runeIndex(row, "ПРОСМОТР") >= 0 {
			t.Errorf("%dx%d: подписи ПАПКИ/ПРОСМОТР вернулись: %q", sz.w, sz.h, string(row))
		}
		// Подпись списка — в средней колонке, между сайдбаром и просмотром.
		lo, hi := sz.w*pctFolders/100, sz.w*(pctFolders+pctMessages)/100+1
		if messages < lo || messages >= hi {
			t.Errorf("%dx%d: подпись списка в колонке %d, хочу %d…%d", sz.w, sz.h, messages, lo, hi)
		}
		a.tapp.Stop()
		<-done // Run сам вызывает screen.Fini()
	}
}

// TestListTitle (V-2): левая часть — «Папка · N непрочитанных» (счётчик из
// кэша FindFolder, без него — только имя), правая — «всего T · стр. p/P»;
// до первой страницы папки правой части нет.
func TestListTitle(t *testing.T) {
	s := &mailScreen{
		folderName:     "Входящие",
		pageSize:       50,
		mainFolders:    []mail.Folder{{ID: "f1", DisplayName: "Входящие", UnreadCount: 6}},
		messagesHeader: newPanelHeader("Письма"),
	}
	check := func(left, right string) {
		t.Helper()
		s.updateListTitle()
		if s.messagesHeader.title != left || s.messagesHeader.right != right {
			t.Errorf("заголовок = %q | %q, хочу %q | %q", s.messagesHeader.title, s.messagesHeader.right, left, right)
		}
	}
	check("Входящие · 6 непрочитанных", "") // страницы ещё не грузились
	s.totalInView, s.offset = 1234, 50
	check("Входящие · 6 непрочитанных", "всего 1234 · стр. 1/25")
	s.offset = 1234 // неполная последняя страница
	check("Входящие · 6 непрочитанных", "всего 1234 · стр. 25/25")
	s.offset, s.totalInView, s.mainFolders[0].UnreadCount = 6, 6, 0
	check("Входящие", "всего 6 · стр. 1/1")
}

// TestPanelHeaderRightPart (V-2, П.4а): правая часть прижата к правому краю,
// линия между частями не прилипает ни к одной из них (зазор headerGap).
func TestPanelHeaderRightPart(t *testing.T) {
	h := newPanelHeader("Папка").SetListTitle("Папка", "всего 2 · стр. 1/2")
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	defer sim.Fini()
	sim.SetSize(40, 1)
	h.SetRect(0, 0, 40, 1)
	h.Draw(sim)

	const (
		line      = '─'
		rightText = "всего 2 · стр. 1/2"
	)
	rightStart := 40 - len([]rune(rightText))
	runes := simRow(sim, 0)
	if string(runes[rightStart:]) != rightText {
		t.Fatalf("правая часть не у правого края: %q", string(runes))
	}
	if runes[5+1] == line { // зазор слева: сразу за «Папка» линии быть не должно
		t.Errorf("линия прилипла к левой подписи: %q", string(runes))
	}
	if runes[5+headerGap] != line {
		t.Errorf("линия не началась после зазора: %q", string(runes))
	}
	for _, i := range []int{rightStart - headerGap, rightStart - 1} {
		if runes[i] == line {
			t.Errorf("линия прилипла к правой части (колонка %d): %q", i, string(runes))
		}
	}
}

// TestMessageRowReadState (RD-4, V-1): непрочитанное — жирный accent на
// строке 1 пункта, прочитанное — textPrimary без жирного; время (колонка 1)
// и тема (строка 2) всегда textSecondary — приглушены в обеих группах;
// [ВНЕШН] отдельного цвета не имеет.
func TestMessageRowReadState(t *testing.T) {
	a := newTestApp(t)
	rows := []mail.MessageSummary{
		{Subject: "Непрочитанное", From: mail.Address{Name: "Иван Петров", Email: "ivan.petrov@example.com"}, Received: time.Now()},
		{Subject: "Прочитанное", From: mail.Address{Name: "Иван Петров", Email: "ivan.petrov@example.com"}, Received: time.Now(), IsRead: true},
	}
	a.mail.items = append(a.mail.items, rows...)
	a.mail.appendRows(rows)

	// WHY: tview хранит цвет ячейки в Style (Color/Attributes — legacy-поля,
	// в которые сеттеры не пишут, если Style задан NewTableCell).
	rowStyle := func(row, col int) (tcell.Color, tcell.AttrMask) {
		fg, _, attr := a.mail.messages.GetCell(row, col).Style.Decompose()
		return fg, attr
	}

	fg, attr := rowStyle(a.mail.itemRows[0], 0)
	if fg != colorUnread || attr&tcell.AttrBold == 0 {
		t.Errorf("строка 1 непрочитанного: цвет %v, атрибуты %v; хочу %v + жирный", fg, attr, colorUnread)
	}
	fg, attr = rowStyle(a.mail.itemRows[1], 0)
	if fg != colorTextPrimary || attr&tcell.AttrBold != 0 {
		t.Errorf("строка 1 прочитанного: цвет %v, атрибуты %v; хочу %v без жирного", fg, attr, colorTextPrimary)
	}
	for i := range rows {
		if fg, _ = rowStyle(a.mail.itemRows[i], 1); fg != colorRead {
			t.Errorf("время строки %d: цвет %v, хочу textSecondary %v", i, fg, colorRead)
		}
		if fg, _ = rowStyle(a.mail.itemRows[i]+1, 0); fg != colorTextSecondary {
			t.Errorf("тема строки %d: цвет %v, хочу textSecondary %v", i, fg, colorTextSecondary)
		}
	}
}

// TestMessageListDensity (RD-2, DoD; V-17): на 140×40 в списке видно ≥10
// писем. С V-17 пункт занимает три строки (отправитель, тема, пустая
// прокладка), поэтому считаем письма по itemRows, а не строки с текстом.
func TestMessageListDensity(t *testing.T) {
	a := newTestApp(t)
	rows := make([]mail.MessageSummary, 40)
	for i := range rows {
		rows[i] = mail.MessageSummary{
			Subject:  fmt.Sprintf("Тема письма номер %d", i),
			From:     mail.Address{Name: "Иван Петров", Email: "ivan.petrov@example.com"},
			Received: time.Now(),
			IsRead:   true,
		}
	}
	a.mail.items = append(a.mail.items, rows...)
	a.mail.appendRows(rows)

	const width, height = 140, 40
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(width, height)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = a.tapp.Run()
	}()
	a.tapp.QueueUpdateDraw(func() {})

	if x0 := runeIndex(simRow(sim, 1), "ПИСЬМА"); x0 < 0 {
		t.Fatal("заголовок «ПИСЬМА» не найден")
	}
	_, _, _, innerH := a.mail.messages.GetInnerRect()
	offset, _ := a.mail.messages.GetOffset()
	visible := 0
	for _, row := range a.mail.itemRows {
		if row >= offset && row < offset+innerH {
			visible++
		}
	}
	if visible < 10 {
		t.Errorf("в списке видно %d писем на 140×40, нужно ≥10", visible)
	}
	a.tapp.Stop()
	<-done
}

// TestMessageListItemTail (V-17): пункт — две строки (отправитель, тема);
// тема невыбираема (курсор пропускает), в itemRows только отправитель (О-1),
// а заливка выделения тянется на оба столбца второй строки — правый край
// блока ровный, дата не «ступенькой».
func TestMessageListItemTail(t *testing.T) {
	a := newTestApp(t)
	now := time.Now()
	rows := []mail.MessageSummary{
		{Subject: "Первое", From: mail.Address{Name: "Иван", Email: "ivan@example.com"}, Received: now},
		{Subject: "Второе", From: mail.Address{Name: "Иван", Email: "ivan@example.com"}, Received: now},
	}
	a.mail.items = append(a.mail.items, rows...)
	a.mail.appendRows(rows)

	if got := a.mail.itemRows[1] - a.mail.itemRows[0]; got != 2 {
		t.Fatalf("между пунктами %d строк, хочу 2 (отправитель + тема)", got)
	}
	subj := a.mail.itemRows[0] + 1
	if c := a.mail.messages.GetCell(subj, 0); c == nil || !c.NotSelectable {
		t.Errorf("строка темы %d должна быть невыбираемой: %+v", subj, c)
	}
	if c := a.mail.messages.GetCell(subj, 1); c == nil || !c.NotSelectable {
		t.Errorf("ячейка времени в строке темы %d должна быть невыбираемой: %+v", subj, c)
	}
	a.mail.paintItemTail(subj, true)
	if _, bg, _ := a.mail.messages.GetCell(subj, 1).Style.Decompose(); bg != colorSelectedBg {
		t.Errorf("правый край темы не залит: %v, хочу %v", bg, colorSelectedBg)
	}
}

// TestScrollOffsetForItem (V-17): при выделении у нижнего края offset
// сдвигается, чтобы тема пункта не обрезалась; видимый пункт и уход выше
// видимой области offset не трогают (там клампит tview).
func TestScrollOffsetForItem(t *testing.T) {
	if got := scrollOffsetForItem(0, 5, 4); got != 1 {
		t.Errorf("у нижнего края offset = %d, хочу 1 (видна и тема)", got)
	}
	if got := scrollOffsetForItem(0, 5, 3); got != 0 {
		t.Errorf("видимый пункт сдвинул offset на %d", got)
	}
	if got := scrollOffsetForItem(10, 5, 2); got != 10 {
		t.Errorf("выделение выше области сдвинуло offset на %d", got)
	}
}

// TestV3ThinFrame — сторож каркаса V-3: рамка панели с фокусом одинарная
// (не двойные focus-линии tview), между колонками — по одному тонкому
// вертикальному разделителю.
func TestV3ThinFrame(t *testing.T) {
	a := newTestApp(t)
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatalf("инициализация симуляции: %v", err)
	}
	a.tapp.SetScreen(sim)
	sim.SetSize(110, 28)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = a.tapp.Run()
	}()
	a.tapp.QueueUpdateDraw(func() {})
	a.tapp.SetFocus(a.mail.messages)
	a.tapp.QueueUpdateDraw(func() {})

	for y := 1; y <= 2; y++ {
		row := simRow(sim, y)
		if runeIndex(row, "╔") >= 0 || runeIndex(row, "═") >= 0 || runeIndex(row, "║") >= 0 {
			t.Errorf("двойные линии рамки фокуса вернулись (строка %d): %q", y, string(row))
		}
	}
	// Разделители: на строке 1 ровно две тонкие вертикальные линии,
	// не совпадающие с краями рамок. Рамки неактивных панелей красит в цвет
	// поверхности paintFocusFrames (RD-2) — их символы на экране невидимы,
	// поэтому считаем только видимые (V-16: рамка панели теперь на строке 1).
	w, _ := sim.Size()
	divs := 0
	for x := 0; x < w; x++ {
		r, _, style, _ := sim.GetContent(x, 1)
		if fg, _, _ := style.Decompose(); fg == colorPanelBg {
			continue
		}
		if r == '│' {
			divs++
		}
	}
	if divs != 2 {
		t.Errorf("видимых разделителей на строке 1: %d, хочу 2", divs)
	}
	a.tapp.Stop()
	<-done
}

// TestPaintSubjectClearsHighlight (V-13): снятие выделения со строки темы
// возвращает ячейке прозрачность, а не цветной/терминальный фон — иначе на
// каждой посещённой строке остаётся полоса.
func TestPaintSubjectClearsHighlight(t *testing.T) {
	a := newTestApp(t)
	rows := []mail.MessageSummary{{
		Subject:  "Тема",
		From:     mail.Address{Name: "Иван Петров", Email: "ivan.petrov@example.com"},
		Received: time.Now(),
	}}
	a.mail.items = append(a.mail.items, rows...)
	a.mail.appendRows(rows)

	subj := a.mail.itemRows[0] + 1
	a.mail.paintItemTail(subj, true)
	if c := a.mail.messages.GetCell(subj, 0); c.Transparent {
		t.Fatal("выделенная тема: ячейка прозрачна, заливки нет")
	}
	a.mail.paintItemTail(subj, false)
	if c := a.mail.messages.GetCell(subj, 0); !c.Transparent {
		t.Errorf("снятая тема: ячейка не прозрачна (Transparent=false) — останется полоса")
	}
}
