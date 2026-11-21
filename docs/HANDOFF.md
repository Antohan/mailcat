# mailcat — HANDOFF

## Дальше
Итерация 3.1 — доработка визуала по референсу: `docs/PLAN-ITER3.1.md`
(V-10 мелкий баг → V-1…V-9). Формат планов — `docs/PLAN-TEMPLATE.md`.
Правило визуальных задач: скриншот до/после в отчёте; следующая — без «ок»
юзера не стартует. Старт: `/stage V-10`.

## Сделано
- Релиз итерации 2: e3e8daf (merge dev→main); smoke S1–S8 подтверждены.
- Итерация 3 (dev): d26e076 RD-1, c37695a RD-2, 1c39486/7941f3a RD-3,
  8a924bd RD-4, 45e14ec RD-5, 9c2356e RD-6, b0f4eb4 RD-7, d969d0b О-1,
  ad151f9 О-3 — выполнена, но результат не устроил пользователя → план 3.1.
- Замер практик: hit-rate кэша 96%, 15 сессий на флагмане ($5.52),
  EN-файлы не сэкономили токены (дека mailcat-token-economy.pptx).

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 баннер/заливка
  в Terminal.app и VS Code.
- НИЗКО (security-ревью): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠).
- Отложено: write-операции (mark-as-read/delete, конфиг [write]), палитра
  Catppuccin (после структуры), сниппет item:Preview (риск 500).

## Не трогать
- Статус-бар (UI-9): подсказки — showHint→a.hint (всегда видна); сообщения/
  ошибки/индикатор — setStatus→a.status; endLoad один раз на ветку
  (QueueUpdateDraw); красный — только colorError; ошибки EWS — Sanitize.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only (write — отдельная
  итерация); tls только internal/ews.
- О-1: itemRows — единственный маппинг «строка↔s.items». RD-7: спиннер —
  SetText, не setStatus; errorShown стопает тик.
- RD-1/3: цвета — только роли theme.go; активная папка = открытая (ID);
  О-3: заливка «сегодня» — colorTodayBg, баннер SetWrap(false)
  (V-8 в 3.1 уберёт баннер — тогда строка снимается).
