# mailcat — HANDOFF

## Дальше
**V-9** (доки, `docs/PLAN-ITER3.1.md`): README (скриншоты/структура),
HANDOFF «Не трогать» — инвариант «цвета/стили — только theme.go/константы UI-6»;
policy_test не задет (чтение Body в whitelist). Читать: `README.md`, план V-9.

## Сделано
- iter31 V-42 (b712081): повестка — `agendaList` (Box+Draw), блоки с переносом, клик/скролл/бар.
- iter31 V-41 (04907e9): верхний бар календаря — «mailcat · календарь», месяц только в шапке.
- iter31 V-40 (f3294cb): «сегодня» — акцентный номер на tint; совпадение → выбор.
- iter31 V-39/V-38 (ff5564f/5903303): bold-числа, flush left; линии ячеек сетки.

## Открыто
- «ок» скриншотов: V-42 (`/tmp/v42_{before,after}_{80x30,100x30,220x60}.txt`), V-41, V-40, V-39, V-38, V-37, V-36, V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-42 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-42: блоки — по строкам в `Draw`/`ScrollTrack` под текущую ширину (не хранить строки); выбор/скролл — индексы событий/строк; сегменты `{текст, стиль}` без tview-тегов (Sanitize, без Escape); клик — `MouseHandler`, выбор + фокус; пустой день — muted «Встреч нет», sel -1.
- V-41: на `pageCalendar` верхний бар — литерал «календарь»; месяц — только шапка сетки.
- V-40: «сегодня» — `colorAccent` на `colorTodayBg`; выбор — акцент/onAccent, совпадение побеждает выбранный.
- V-39: число дня — bold у всех состояний, счётчик — regular textSecondary; `FormatDayCell` — `%d`, без паддинга.
- V-38: геометрия — только `gridCellSize`/`dayCellRect`; линии — после ячеек; заливки — внутри `dayCellRect`.
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading`; тело — только `formatBody`; GetItem — §11.
- V-35: подсказки — два набора по фокусу (`pageHintSegs`); Enter на повестке — `SetSelectedFunc`; 1/2 — `ensureFocus` в сетку.
