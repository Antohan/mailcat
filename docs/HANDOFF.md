# mailcat — HANDOFF

## Дальше
RD-6: нижняя панель — клавиши отделены от описаний. Читать `docs/PLAN-ITER3.md`
(RD-6), `internal/ui/app.go` (константы `hint*`, `showHint`, `status`/`hint`).

## Сделано
- 45e14ec — RD-5: просмотр письма (структура, dynamic colors + Escape, пустые
  состояния); 8a924bd — RD-4: список писем; 1c39486/7941f3a — RD-3: папки;
  c37695a — RD-2: каркас; d26e076 — RD-1: палитра; e3e8daf — Release 2.

## Открыто
- TODO пользователя (TESTING.md §5): smoke S3–S6 (в т.ч. превью/детали RD-5),
  темы, resize; визуал в Terminal.app/VS Code.
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции
  (mark-as-read/delete, [write]) отложены — решения у пользователя.

## Не трогать
- RD-5: preview/details TextView — SetDynamicColors(true): любой текст из EWS в
  них только через Sanitize + tview.Escape (FormatMessageHeader / showDetails /
  FormatAttachments — escapeForList; находка security-reviewer), «просто
  SetText» = тег-инъекция; a.event остаётся dynamic=false (showEventDetails
  пишет без Escape). Разделитель письма — фикс 32 «─» textSecondary; превью
  строит FormatMessageHeader от MessageSummary, свой форматтер не заводить.
- RD-4: цвет ячейки tview живёт в `Style` (сеттеры не пишут в legacy
  Color/Attributes при заданном Style; в тестах `Style.Decompose()`);
  непрочитанные — accent; [ВНЕШН] — цвет своей группы.
- RD-3: активная папка = открытая (ID; до первого переключения — имя);
  accent-теги снимаются под курсором; счётчик watcher'а — через mainFolders.
- RD-2: заголовок — `panelHeader` (капс, accent у активной); линия —
  textSecondary, не hairline; рамка неактивной красится `colorPanelBg`.
- RD-1: цвета в виджетах — только роли `theme.go`; фокус и выделение — accent.
- UI-9: подсказки — showHint→a.hint; ошибки — setStatus→a.status; endLoad(id)
  один раз на ветку; красный — только colorError; очистка по statusGen.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; CalendarView без
  SortOrder/Restriction; read-only; tls только internal/ews.
