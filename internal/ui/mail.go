package ui

// Экран почты: Flex[ папки 20% | список 40% | просмотр 40% ] (PLAN.md этап 2).
// Изменения состояния — только из главного потока tview (обработчики и
// QueueUpdateDraw), поэтому мьютекс не нужен (ARCHITECTURE.md §3).

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mailcat/internal/ews"
	"mailcat/internal/mail"
	"mailcat/internal/security"
)

const pageSizeCap = 100 // защита от нулевого page_size в конфиге

// Маркеры сворачиваемой секции техпапок (UI-1). Не проходят FormatFolder —
// статический текст, не из EWS.
const (
	moreFoldersClosed = "Ещё ▸"
	moreFoldersOpen   = "Ещё ▾"
)

type mailScreen struct {
	app  *App
	root *tview.Flex

	folders  *tview.List
	messages *tview.Table
	preview  *tview.TextView

	// Состояние экрана.
	folder     ews.FolderID // выбранная папка (пустая = distinguished inbox)
	folderName string
	items      []mail.MessageSummary
	offset     int  // Offset следующей страницы FindItem
	lastPage   bool // IncludesLastItemInRange последней страницы
	loading    bool // запрос страниц в полёте (ревью этапа 2: защита от дублей)

	// Папки панели (UI-1): основной список и техпапки секции «Ещё ▸».
	mainFolders []mail.Folder
	techFolders []mail.Folder
	techOpen    bool
	// WHY: поколение выбора папки — ответ, пришедший после переключения,
	// относится к прежней папке и не должен стирать уже загруженный список.
	// Не делать: сравнивать имя папки вместо поколения — при возврате A→B→A
	// старый ответ A снова станет «свежим».
	gen int

	pageSize int
}

func newMailScreen(a *App) *mailScreen {
	s := &mailScreen{app: a, pageSize: a.cfg.PageSize}
	if s.pageSize < 1 || s.pageSize > pageSizeCap {
		s.pageSize = 50
	}

	s.folders = tview.NewList().
		ShowSecondaryText(false).
		SetMainTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(colorSelectedBg).
		SetSelectedTextColor(colorSelectedFg)
	s.folders.SetBorder(true).SetTitle(" Папки ")

	s.messages = tview.NewTable().
		SetSelectable(true, false).
		SetSeparator('\u2502')
	s.messages.SetBorder(true).SetTitle(" Письма ")
	// WHY: tview считает ширину колонки максимумом по ячейкам и выбрасывает
	// колонки, не влезающие целиком, — длинная тема выталкивала дату за край
	// панели (обкатка 20.11). Капы пересчитываются на каждом Draw, до
	// раскладки колонок, — переживают resize терминала.
	s.messages.SetDrawFunc(s.clampMessageColumns)

	s.preview = tview.NewTextView().
		SetWrap(true).
		SetScrollable(true).
		SetDynamicColors(false) // dynamic colors — этап 3 (с tview.Escape)
	s.preview.SetBorder(true).SetTitle(" Просмотр ")

	s.root = tview.NewFlex().
		AddItem(s.folders, 0, 1, false).
		AddItem(s.messages, 0, 2, false).
		AddItem(s.preview, 0, 2, true)

	s.folders.SetChangedFunc(func(_ int, mainText, _ string, _ rune) {
		// «Ещё ▸/▾» — не папка: смещение курсора через пункт секции
		// не должно подменять имя текущей папки.
		if mainText == moreFoldersClosed || mainText == moreFoldersOpen {
			return
		}
		s.folderName = mainText
	})
	s.folders.SetSelectedFunc(func(_ int, _ string, _ string, _ rune) {
		s.app.tapp.SetFocus(s.messages)
	})
	s.messages.SetSelectionChangedFunc(func(row, _ int) {
		s.showPreview(row)
		s.maybeLoadNextPage(row)
	})
	s.messages.SetSelectedFunc(func(row, _ int) {
		s.openDetails(row)
	})

	// Tab/BackTab — циклический фокус панелей; ←/→ — соседняя панель (PLAN
	// этап 2 DoD: «←/→ или Tab — между панелями»); u — обновить папку.
	tabHook := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			s.cycleFocus()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'u' || event.Rune() == 'U' {
				s.refreshFolder()
				return nil
			}
		}
		return event
	}
	s.folders.SetInputCapture(tabHook)
	s.messages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			s.app.tapp.SetFocus(s.folders)
			return nil
		case tcell.KeyRight:
			s.app.tapp.SetFocus(s.preview)
			return nil
		}
		return tabHook(event)
	})
	s.preview.SetInputCapture(tabHook)
	return s
}

// openDetails открывает полноэкранный экран деталей письма (Enter): все
// заголовки, полный текст, список вложений (PLAN.md этап 2); Esc — назад.
func (s *mailScreen) openDetails(row int) {
	if row < 0 || row >= len(s.items) {
		return
	}
	ref := s.items[row]
	s.app.setStatus("Загрузка письма…")
	go func() {
		detail, err := s.app.client.GetMessage(s.app.ctx, ref.ItemID, ref.ChangeKey)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() { s.app.showError(err) })
			return
		}
		full := mail.FullMessage(detail)
		full.From = ref.From // From не приходит в GetItem — берём из строки списка
		s.app.tapp.QueueUpdateDraw(func() {
			s.app.showDetails(full)
		})
	}()
}

// loadFolders заполняет панель папок (FindFolder от msgfolderroot): основной
// список + техпапки в секции «Ещё ▸» (UI-1) — и открывает «Входящие»
// (distinguished inbox — первая страница писем).
func (s *mailScreen) loadFolders() {
	s.app.setStatus("Загрузка папок…")
	go func() {
		list, err := s.app.client.FindFolders(s.app.ctx)
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() { s.app.showError(err) })
			return
		}
		folders := mail.Folders(list)
		s.app.tapp.QueueUpdateDraw(func() {
			s.mainFolders, s.techFolders = SplitFolders(folders, s.app.cfg.UI.HiddenFolders)
			s.renderFolders()
			s.app.showHint()
			// Стартуем с входящих (distinguished) — отдельным запросом.
			s.selectFolder(ews.FolderID{}, "Входящие")
		})
	}()
}

// renderFolders перестраивает панель целиком: основной список, затем пункт
// «Ещё ▸/▾» и, когда секция развернута, техпапки. Целиком — потому что
// сворачивание/разворачивание меняет хвост списка одним махом.
func (s *mailScreen) renderFolders() {
	s.folders.Clear()
	add := func(f mail.Folder) {
		display := FormatFolder(f)
		s.folders.AddItem(display, "", 0, func() {
			s.selectFolder(ews.FolderID{ID: f.ID, ChangeKey: f.ChangeKey}, display)
		})
	}
	for _, f := range s.mainFolders {
		add(f)
	}
	if len(s.techFolders) == 0 {
		return
	}
	mark := moreFoldersClosed
	if s.techOpen {
		mark = moreFoldersOpen
	}
	s.folders.AddItem(mark, "", 0, s.toggleTech)
	if s.techOpen {
		for _, f := range s.techFolders {
			add(f)
		}
	}
}

// toggleTech сворачивает/разворачивает секцию техпапок. Курсор возвращается
// на пункт «Ещё», чтобы повторное Enter срабатывало как сворачивание.
func (s *mailScreen) toggleTech() {
	s.techOpen = !s.techOpen
	s.renderFolders()
	s.folders.SetCurrentItem(len(s.mainFolders))
}

// selectFolder переключает текущую папку и сбрасывает список.
func (s *mailScreen) selectFolder(id ews.FolderID, name string) {
	s.folder = id
	s.folderName = name
	s.items = nil
	s.offset = 0
	s.lastPage = false
	s.messages.Clear()
	s.preview.SetText("")
	s.loadMessages()
}

// refreshFolder перечитывает текущую папку (клавиша u).
func (s *mailScreen) refreshFolder() {
	s.selectFolder(s.folder, s.folderName)
}

// loadMessages догружает страницу писем (FindItem, offset — курсор пагинации).
// Ответ устаревшего запроса (папка/offset сменились) отбрасывается.
func (s *mailScreen) loadMessages() {
	if s.loading {
		return
	}
	s.loading = true
	s.gen++
	gen := s.gen
	s.app.setStatus("Загрузка писем…")
	offset := s.offset
	folder := s.folder
	go func() {
		var page ews.Page
		var err error
		if folder.ID == "" {
			page, err = s.app.client.FindDistinguishedMessages(s.app.ctx, "inbox", s.pageSize, offset)
		} else {
			page, err = s.app.client.FindMessages(s.app.ctx, folder, s.pageSize, offset)
		}
		if err != nil {
			s.app.tapp.QueueUpdateDraw(func() {
				s.loading = false
				s.app.showError(err)
			})
			return
		}
		added := mail.MessageSummaries(page.Items)
		s.app.tapp.QueueUpdateDraw(func() {
			s.loading = false
			if s.gen != gen {
				return // папка уже сменилась — ответ устарел
			}
			s.items = append(s.items, added...)
			s.offset = page.NextOffset
			s.lastPage = page.IncludesLast
			s.appendRows(added)
			if len(s.items) == 0 {
				s.messages.SetCell(0, 0, tview.NewTableCell("(писем нет)"))
			}
			s.app.showHint()
		})
	}()
}

// clampMessageColumns ограничивает ширину колонок «отправитель» и «тема»
// текущей шириной панели, чтобы колонка даты всегда оставалась видимой
// (UI-2). Возврат — стандартный внутренний прямоугольник Box с рамкой.
func (s *mailScreen) clampMessageColumns(_ tcell.Screen, x, y, w, h int) (int, int, int, int) {
	const markW, dateW = len("[ВНЕШН]"), 5 // «15:04» / «02.01»
	inner := w - 2                         // рамка панели
	if inner > 0 {
		sender := inner / 4
		if sender > 18 {
			sender = 18
		}
		if sender < 8 {
			sender = 8
		}
		subject := inner - (markW + 1) - (sender + 1) - (dateW + 1) - 1
		if subject < 4 {
			subject = 4 // уже панель без внешних писем — tview сам ужмёт тему
		}
		for row := 0; row < s.messages.GetRowCount(); row++ {
			if c := s.messages.GetCell(row, 1); c != nil {
				c.SetMaxWidth(sender)
			}
			if c := s.messages.GetCell(row, 2); c != nil {
				c.SetMaxWidth(subject)
			}
		}
	}
	return x + 1, y + 1, w - 2, h - 2
}

// appendRows дописывает строки в таблицу; непрочитанные — ярким белым.
func (s *mailScreen) appendRows(added []mail.MessageSummary) {
	base := len(s.items) - len(added)
	for i, m := range added {
		row := base + i
		for col, text := range FormatMessageRow(m, s.app.cfg.InternalDomains(), time.Now()) {
			expansion := 0
			if col == 2 {
				expansion = 1 // тема забирает свободную ширину
			}
			cell := tview.NewTableCell(text).SetExpansion(expansion)
			if col != 3 && !m.IsRead {
				cell.SetTextColor(colorUnread).SetAttributes(tcell.AttrBold)
			}
			if col == 3 {
				cell.SetTextColor(colorRead)
			}
			s.messages.SetCell(row, col, cell)
		}
	}
}

// maybeLoadNextPage подгружает следующую страницу при достижении конца списка;
// повторный запуск при загрузке в полёте игнорируется.
func (s *mailScreen) maybeLoadNextPage(row int) {
	if s.loading || s.lastPage || row < len(s.items)-2 {
		return
	}
	if _, more := ews.NextOffset(ews.Page{NextOffset: s.offset, IncludesLast: s.lastPage}); more {
		s.loadMessages()
	}
}

// ewsAttachment конвертирует доменное вложение в метаданные для GetAttachment.
func ewsAttachment(a mail.Attachment) ews.AttachmentMeta {
	return ews.AttachmentMeta{
		AttachmentID:  a.AttachmentID,
		Name:          a.Name,
		ContentType:   a.ContentType,
		ContentLength: a.ContentLength,
	}
}

// showPreview показывает черновик письма в панели просмотра (без загрузки).
func (s *mailScreen) showPreview(row int) {
	if row < 0 || row >= len(s.items) {
		return
	}
	m := s.items[row]
	text := "Тема:  " + security.Sanitize(m.Subject) + "\n" +
		"От:    " + security.Sanitize(m.From.String()) + "\n\n" +
		"Enter — прочитать письмо…"
	s.preview.SetText(text)
	s.preview.ScrollToBeginning()
}

// updateInboxUnread обновляет счётчик непрочитанных «Входящих» из watcher'а
// (этап 6: QueueUpdateDraw). n<0 — счётчик неизвестен, пункт не трогаем.
func (s *mailScreen) updateInboxUnread(n int) {
	if n < 0 {
		return
	}
	for i := 0; i < s.folders.GetItemCount(); i++ {
		main, _ := s.folders.GetItemText(i)
		if strings.HasPrefix(main, "Входящие") {
			s.folders.SetItemText(i, FormatFolder(mail.Folder{DisplayName: "Входящие", UnreadCount: n}), "")
			return
		}
	}
}

// cycleFocus перебирает панели почты по кругу (Tab).
func (s *mailScreen) cycleFocus() {
	focus := []tview.Primitive{s.folders, s.messages, s.preview}
	for i, p := range focus {
		if s.app.tapp.GetFocus() == p {
			s.app.tapp.SetFocus(focus[(i+1)%len(focus)])
			return
		}
	}
	s.app.tapp.SetFocus(s.messages)
}
