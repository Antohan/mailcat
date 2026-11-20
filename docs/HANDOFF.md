# mailcat — HANDOFF

## Дальше
Итерация 3 — редизайн интерфейса: `docs/PLAN-ITER3.md` (RD-1…RD-7;
опциональные О-1…О-3 — согласовать с пользователем до старта). Тема светлая
по умолчанию, тёмная — конфиг `[ui] theme`. Формат планов —
`docs/PLAN-TEMPLATE.md`. Старт: `/stage RD-1`.

## Сделано
- e3e8daf — Release: итерация 2 — доработка интерфейса (merge dev→main).
- Итерация 2: UI-1…UI-9, T-1, D-1; smoke S1–S8 подтверждены пользователем.
- Замер практик: hit-rate кэша 96%, все 15 сессий — флагман variant high
  ($5.52), EN-файлы не сэкономили токены (дека mailcat-token-economy.pptx).

## Открыто
- НИЗКО (security-ревью релиза): attachments.go ~120 — частичный файл
  при ошибке Write (косметика, не безопасность).
- item:Importance нет в EWS-REFERENCE — нужно для min_importance и ⚠.
- Write-операции (mark-as-read/delete, конфиг [write], MoveToDeletedItems
  only) — отложены; решения зафиксированы в обсуждении с пользователем.

## Не трогать
- Статус-бар (UI-9): подсказки — только showHint→a.hint (нижняя строка,
  видна всегда); сообщения/ошибки/индикатор — setStatus→a.status; endLoad/
  clearError гасят a.status или возвращают идущую загрузку — a.hint не
  трогают. Загрузки: endLoad(id) ровно один раз на ветку горутины
  (QueueUpdateDraw); loads — только главный поток TUI; красный — только
  colorError; автоочистка по statusGen; ошибки EWS — Sanitize, по рунам.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only; tls только internal/ews.
