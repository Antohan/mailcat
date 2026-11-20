# mailcat — HANDOFF

## Дальше
RD-2: каркас (разделители, пропорции, активная панель, resize). Читать
`docs/PLAN-ITER3.md` (RD-2), `internal/ui/mail.go`, `app.go` (bordered).

## Сделано
- 66d789c — RD-1: палитра и роли theme.go, `[ui] theme`; e3e8daf — Release 2.

## Открыто
- Smoke S3–S6, светлая/тёмная темы в Terminal.app и iTerm2/VSCode — TODO
  пользователя (`TESTING.md §5`): живого Exchange здесь нет.
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write (косметика).
- item:Importance нет в EWS-REFERENCE; Write-операции (mark-as-read/delete,
  [write], MoveToDeletedItems only) отложены — решения у пользователя.

## Не трогать
- RD-1: цвета в виджетах — только роли `theme.go`, без сырых tcell-цветов;
  фокус и выделение строки — один accent.
- Статус-бар (UI-9): подсказки — только showHint→a.hint; сообщения/ошибки/
  индикатор — setStatus→a.status; endLoad/clearError гасят a.status, a.hint
  не трогают. endLoad(id) ровно один раз на ветку горутины (QueueUpdateDraw);
  loads — главный поток TUI; красный — только colorError; автоочистка
  statusGen; ошибки EWS — Sanitize, по рунам.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; CalendarView без SortOrder/Restriction; read-only; tls только internal/ews.
