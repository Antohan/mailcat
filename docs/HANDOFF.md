# mailcat — HANDOFF

## Дальше
RD-3: папки — иконки, отступы, акцент, счётчики. Читать `docs/PLAN-ITER3.md`
(RD-3), `internal/ui/format.go` (`FormatFolder`), `mail.go` (`renderFolders`).

## Сделано
- b2c7fd3 — RD-2: каркас (panel.go, пропорции 20/42/38); d26e076 — RD-1:
  палитра и тема; e3e8daf — Release 2.

## Открыто
- TODO пользователя (`TESTING.md §5`): smoke S3–S6, темы, resize вживую.
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции
  (mark-as-read/delete, [write]) отложены — решения у пользователя.

## Не трогать
- RD-2: заголовок — `panelHeader` (капс, accent у активной); линия —
  textSecondary, не hairline; рамка неактивной панели красится `colorPanelBg`,
  не выключается — иначе контент прыгает при смене фокуса.
- RD-1: цвета в виджетах — только роли `theme.go`; фокус и выделение — accent.
- UI-9: подсказки — showHint→a.hint; сообщения/ошибки — setStatus→a.status;
  endLoad(id) ровно один раз на ветку горутины; красный — только colorError;
  ошибки EWS — Sanitize, по рунам; автоочистка по statusGen.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; CalendarView без
  SortOrder/Restriction; read-only; tls только internal/ews.
