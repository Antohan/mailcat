# mailcat — HANDOFF

## Дальше
RD-5: просмотр письма — структура и пустые состояния. Читать `docs/PLAN-ITER3.md`
(RD-5), `internal/ui/format.go` (`FormatMessageHeader`), `mail.go` (preview),
`app.go` (`showDetails`).

## Сделано
- 8a924bd — RD-4: список писем (непрочитанные — жирный accent, прочитанные —
  textPrimary, дата всегда textSecondary); 1c39486/7941f3a — RD-3: папки;
  c37695a — RD-2: каркас; d26e076 — RD-1: палитра; e3e8daf — Release 2.

## Открыто
- TODO пользователя (TESTING.md §5): smoke S3–S6, темы, resize; иконки/выравнивание
  счётчиков папок и жирный accent непрочитанных в Terminal.app (эмодзи двойной ширины).
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции
  (mark-as-read/delete, [write]) отложены — решения у пользователя.

## Не трогать
- RD-4: цвет ячейки tview живёт в `Style` (Color/Attributes — legacy, сеттеры
  туда не пишут при заданном Style; в тестах читать `Style.Decompose()`);
  непрочитанные — accent, отдельного «цвета непрочитанных» не заводить;
  [ВНЕШН] красится цветом своей группы, своего цвета не имеет.
- RD-3: активная папка = открытая (ID; до первого переключения — имя); accent-теги
  строки снимаются, когда курсор на ней — иначе accent на accent-фоне инверсии
  невидим; счётчик watcher'а — через mainFolders, не по тексту строки.
- RD-2: заголовок — `panelHeader` (капс, accent у активной); линия — textSecondary,
  не hairline; рамка неактивной красится `colorPanelBg`, не выключается.
- RD-1: цвета в виджетах — только роли `theme.go`; фокус и выделение — accent.
- UI-9: подсказки — showHint→a.hint; ошибки — setStatus→a.status; endLoad(id) один
  раз на ветку; красный — только colorError; ошибки EWS — Sanitize по рунам;
  очистка по statusGen.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; CalendarView без
  SortOrder/Restriction; read-only; tls только internal/ews.
