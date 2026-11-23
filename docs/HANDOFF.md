# mailcat — HANDOFF

## Дальше
**V-44** (волна 7, `docs/PLAN-ITER3.1.md`): заливка ячейки до линий, текст числа
с отступом 1 (`calendar_grid.go` — `Draw`/`cellStyles`); далее V-45…V-47.

## Сделано
- iter31 V-43 (3c626a2): линии сетки — роль `colorGridLine` (светлее чисел).
- iter31 V-42 (b712081): повестка — `agendaList` (Box+Draw), блоки с переносом, клик/скролл/бар.
- iter31 V-41 (04907e9): верхний бар календаря — «mailcat · календарь», месяц только в шапке.
- iter31 V-40 (f3294cb): «сегодня» — акцентный номер на tint; совпадение → выбор.
- iter31 V-39/V-38 (ff5564f/5903303): bold-числа, flush left; линии ячеек сетки.

## Открыто
- «ок» скриншотов: V-43 (`/tmp/v43_{before,after}_{100x30,220x60}.txt` + `.ans`,
  тёмная — `/tmp/v43_after_dark_100x30.txt`), V-42, V-41, V-40, V-39–V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-43 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-43: линии сетки — только `colorGridLine` (между hairline и textSecondary); не hairline (не видно) и не textSecondary (перевешивает числа).
- V-42: блоки — по строкам в `Draw`/`ScrollTrack` под ширину (не хранить); сегменты `{текст, стиль}` без tview-тегов (Sanitize, без Escape); клик — `MouseHandler`; пустой день — muted «Встреч нет», sel -1.
- V-41: на `pageCalendar` верхний бар — литерал «календарь»; месяц — только шапка сетки.
- V-40: «сегодня» — `colorAccent` на `colorTodayBg`; выбор — акцент/onAccent, совпадение побеждает выбранный.
- V-39: число дня — bold у всех состояний, счётчик — regular textSecondary; `FormatDayCell` — `%d`, без паддинга.
- V-38: геометрия — только `gridCellSize`/`dayCellRect`; линии — после ячеек; заливки — внутри `dayCellRect`.
