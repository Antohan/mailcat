# mailcat — HANDOFF

## Дальше
Итерация 3 не спланирована — сессия планирования: docs/PLAN-ITER3.md
(спеки задач с указателями на код, образец — PLAN-ITER2.md).

## Сделано
- e3e8daf — Release: итерация 2 — доработка интерфейса (merge dev→main).

## Открыто
- НИЗКО (security-ревью релиза): attachments.go ~120 — частичный файл
  при ошибке Write (косметика, не безопасность).
- item:Importance нет в EWS-REFERENCE — нужно для min_importance и ⚠.

## Не трогать
- Статус-бар (UI-9): подсказки — только showHint→a.hint (нижняя строка,
  видна всегда); сообщения/ошибки/индикатор — setStatus→a.status; endLoad/
  clearError гасят a.status или возвращают идущую загрузку — a.hint не
  трогают. Загрузки: endLoad(id) ровно один раз на ветку горутины
  (QueueUpdateDraw); loads — только главный поток TUI; красный — только
  colorError; автоочистка по statusGen; ошибки EWS — Sanitize, по рунам.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only; tls только internal/ews.
