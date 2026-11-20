# mailcat — HANDOFF

## Дальше
RD-4: список писем — статус прочитанности. Читать `docs/PLAN-ITER3.md` (RD-4),
`internal/ui/format.go` (`FormatMessageRow`), `mail.go` (таблица писем, стили
`col != 3`).

## Сделано
- 7941f3a — RD-3: папки (иконки, отступ, акцент открытой, счётчики);
  c37695a — RD-2: каркас; d26e076 — RD-1: палитра; e3e8daf — Release 2.

## Открыто
- TODO пользователя (TESTING.md §5): smoke S3–S6, темы, resize; иконки/выравнивание счётчиков папок в Terminal.app (эмодзи двойной ширины).
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции (mark-as-read/delete, [write]) отложены — решения у пользователя.

## Не трогать
- RD-3: активная папка = открытая (ID; до первого переключения — имя); accent-теги строки снимаются, когда курсор на ней — иначе accent на accent-фоне инверсии невидим; счётчик watcher'а — через mainFolders, не по тексту строки (там теги+иконка).
- RD-2: заголовок — `panelHeader` (капс, accent у активной); линия — textSecondary, не hairline; рамка неактивной красится `colorPanelBg`, не выключается.
- RD-1: цвета в виджетах — только роли `theme.go`; фокус и выделение — accent.
- UI-9: подсказки — showHint→a.hint; ошибки — setStatus→a.status; endLoad(id) один раз на ветку; красный — только colorError; ошибки EWS — Sanitize по рунам; очистка по statusGen.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; CalendarView без SortOrder/Restriction; read-only; tls только internal/ews.
