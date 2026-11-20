# mailcat — HANDOFF

## Дальше
Итерация 2 закрыта (UI-1…UI-9, T-1, D-1). Осталось пройти DoD PLAN-ITER2.md
с пользователем: smoke S1–S8 и UI-2..UI-9/T-1 живьём (TESTING.md §5, VPN).
Merge dev→main — одним merge-коммитом по явной просьбе, вне задачной сессии.

## Сделано (итерация 2)
- 1a8e2fc — iter2:UI-9: две строки статуса — подсказки всегда видны.
- f7659c8 D-1 · c28341f T-1 · ed80438 UI-8 · 86cc38c UI-7 · e015c2f UI-6 ·
  c4b4140 UI-5 · a5ca69d UI-4 · b110fbe UI-3 · 184bf25 UI-2 · f7cae6e/9b1d232 UI-1.

## Открыто (нужен VPN — TESTING.md §5)
- Smoke S1–S8 и UI-2..UI-9/T-1; item:Importance в EWS-REFERENCE? (нужно
  для min_importance и ⚠); security-ревью git diff 8fb6a9f..HEAD — по желанию.

## Не трогать
- Статус-бар (UI-9): подсказки — только showHint→a.hint (нижняя строка,
  видна всегда); сообщения/ошибки/индикатор — setStatus→a.status; endLoad/
  clearError гасят a.status или возвращают идущую загрузку — a.hint не
  трогают. Загрузки: endLoad(id) ровно один раз на ветку горутины
  (QueueUpdateDraw); loads — только главный поток TUI; красный — только
  colorError; автоочистка по statusGen; ошибки EWS — Sanitize, по рунам.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only; tls только internal/ews.
