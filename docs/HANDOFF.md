# mailcat — HANDOFF

## Дальше
Итерация 3.1 — доработка визуала по референсу: `docs/PLAN-ITER3.1.md`
(V-10 ✅ → следующий **V-1** двухстрочные строки списка; порядок как
пронумерован). Формат планов — `docs/PLAN-TEMPLATE.md`. Правило визуальных
задач: скриншот до/после в отчёте; следующая — без «ок» юзера не стартует
(V-1 — визуальная). Старт: `/stage V-1`.

## Сделано
- iter31 V-10: огрызки `<[1] host>` — скобки автолинка поглощены матчем
  reBareURL (internal/mail/links.go ~36) + тест; /check зелёный.
- Релиз итерации 2: e3e8daf (merge dev→main); smoke S1–S8 подтверждены.
- Итерация 3 (dev): d26e076 RD-1 … ad151f9 О-3 — результат не устроил
  пользователя → план 3.1.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 баннер/заливка
  в Terminal.app и VS Code.
- НИЗКО (security-ревью): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠).
- Отложено: write-операции (mark-as-read/delete, конфиг [write]), палитра
  Catppuccin (после структуры), сниппет item:Preview (риск 500).

## Не трогать
- V-10: `<?`/`>?` в reBareURL поглощают скобки автолинка; Trim `<<>` в
  RenderBody обязателен (WHY-комментарий в links.go).
- Статус-бар (UI-9): подсказки — showHint→a.hint (всегда видна); сообщения/
  ошибки/индикатор — setStatus→a.status; endLoad один раз на ветку
  (QueueUpdateDraw); красный — только colorError; ошибки EWS — Sanitize.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only (write — отдельная
  итерация); tls только internal/ews.
- itemRows — единственный маппинг «строка↔s.items» (О-1); RD-7: спиннер —
  SetText, не setStatus; errorShown стопает тик.
- RD-1/3: цвета — только роли theme.go; активная папка = открытая (ID);
  О-3: заливка «сегодня» — colorTodayBg, баннер SetWrap(false)
  (V-8 в 3.1 уберёт баннер — тогда строка снимается).
