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

		row := simRow(sim, 0)
		folders := runeIndex(row, "ПАПКИ")
		messages := runeIndex(row, "ПИСЬМА")
		preview := runeIndex(row, "ПРОСМОТР")
		if folders < 0 || messages < 0 || preview < 0 {
			t.Fatalf("%dx%d: заголовки панелей не найдены: %q", sz.w, sz.h, string(row))
		}
		if !(folders < messages && messages < preview) {
			t.Errorf("%dx%d: порядок панелей нарушен: %d %d %d", sz.w, sz.h, folders, messages, preview)
		}
		if sz.w == 100 {
			// 20% / 42% / 38% от 100 — ровно 20 и 62.
			if folders != 0 || messages != 20 || preview != 62 {
				t.Errorf("100×30: колонки %d/%d/%d, хочу 0/20/62", folders, messages, preview)
			}
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

// TestMessageListDensity (RD-2, DoD): на 140×40 в списке видно ≥30 писем —
// раскладка с заголовками панелей плотность не съела.
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

	x0 := runeIndex(simRow(sim, 0), "ПИСЬМА")
	if x0 < 0 {
		t.Fatal("заголовок «ПИСЬМА» не найден")
	}
	visible := 0
	for y := 2; y < height-3; y++ {
		line := simRow(sim, y)
		for x := x0 + 2; x < x0+12 && x < len(line); x++ {
			if line[x] != ' ' && line[x] != 0 {
				visible++
				break
			}
		}
	}
	if visible < 30 {
		t.Errorf("в списке видно %d писем на 140×40, нужно ≥30", visible)
	}
	a.tapp.Stop()
	<-done
}
