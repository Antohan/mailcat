# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-10 ✅, V-1 ✅, V-2 ✅ → следующий
**V-3** (каркас: тонкий вертикальный разделитель ПИСЬМА ↔ ПРОСМОТР (П.4б);
двойная рамка папок → одна тонкая (П.2в); активная панель — акцент, механика
UI-6 сохраняется; разделитель ПАПКИ ↔ ПИСЬМА — согласовать по толщине).
Визуальные задачи: скриншот до/после в отчёте; без «ок» юзера на предыдущей
не стартовать. Старт: `/stage V-3`.

## Сделано
- iter31 V-2: заголовок списка «Входящие · N непрочитанных» + «всего 1234 ·
  стр. 1/25»; panelHeader.SetListTitle, updateListTitle; /check зелёный.
- iter31 V-1: двухстрочные строки списка; FormatMessageListItem; скриншоты.
- iter31 V-10: скобки автолинка поглощены матчем reBareURL (links.go).
- Релиз итерации 2: e3e8daf (merge dev→main); smoke S1–S8 подтверждены.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 баннер/заливка
  в Terminal.app и VS Code.
- НИЗКО (security-ревью): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠).
- Отложено: write-операции (mark-as-read/delete), палитра Catppuccin,
  сниппет item:Preview (риск 500).

## Не трогать
- V-2: SetListTitle без капса (имя папки — данные сервера); правая часть —
  всегда textSecondary (акцент = активная панель, RD-1); счётчики скрыты до
  первой страницы папки; зазоры headerGap=2 с обеих сторон линии (П.4а).
- V-1: точка статуса — тег dotLine сквозь SetSelectedStyle; строка темы
  невыбираема, красит paintSubject (selRow; сброс selRow=-1 в rebuildRows
  обязателен); усечение — только truncateEllipsis в DrawFunc (встроенное
  «…» tview не работает с кириллицей: байты vs клетки); [ВНЕШН] —
  externalMarkCol=8, панель не «дышит».
- V-10: `<?`/`>?` в reBareURL поглощают скобки автолинка; Trim `<<>` в
  RenderBody обязателен (WHY в links.go).
- Статус-бар (UI-9): showHint→a.hint (всегда видна); setStatus→a.status;
  endLoad один раз на ветку; красный — только colorError; ошибки EWS —
  Sanitize.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500 на живом;
  CalendarView без SortOrder/Restriction; read-only; tls только
  internal/ews.
- itemRows — единственный маппинг «строка↔s.items» (О-1); RD-7: спиннер —
  SetText, не setStatus; errorShown стопает тик.
- RD-1/3: цвета — только роли theme.go; активная папка = открытая (ID);
  О-3: заливка «сегодня» — colorTodayBg, баннер SetWrap(false).
