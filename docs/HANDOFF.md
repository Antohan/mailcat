# mailcat — HANDOFF

## Дальше
**V-46** (`docs/PLAN-ITER3.1.md`): дни соседних месяцев — muted числами, сетка
заполнена 42 датами, выбор/клик только текущего месяца (`calendar_format.go` —
`MonthGrid`/`DayCell.InMonth`, `calendar_grid.go` — `Draw`/`cellStyles`/`dayAt`);
далее V-47.

## Сделано
- iter31 V-45 (1828d04): «сегодня» — accent+bold на фоне панели, `colorTodayBg` удалён.
- iter31 V-44: заливка ячейки до линий, число/«·N» — с отступом 1.
- iter31 V-43 (dc883bd): линии сетки — роль `colorGridLine` (светлее чисел).
- iter31 V-42 (b712081): повестка — `agendaList` (Box+Draw), блоки с переносом, клик/скролл/бар.

## Открыто
- «ок» скриншотов: V-45 (`/private/tmp/v45_{before,after}_{light,dark}_{100x30,220x60}.txt` + `.ans`), V-44…V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-45 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-45: «сегодня» — accent+bold на `colorPanelBg`, счётчик `colorTextSecondary`; выбор — сплошной акцент и при совпадении побеждает.
- V-44: заливка — весь `dayCellRect` до линий (не под текстом); текст +1 вправо, +1 вниз при `ch>1` (при `ch==1` вниз нельзя — там линия); счётчик вплотную после числа.
- V-43: линии — только `colorGridLine` (между hairline и textSecondary); пустые/соседние — те же линии.
- V-42: блоки — по строкам в `Draw`/`ScrollTrack` под ширину (не хранить); сегменты `{текст, стиль}` без tview-тегов (Sanitize, без Escape); клик — `MouseHandler`; пустой день — muted «Встреч нет», sel -1.
- V-41/V-40: верхний бар `pageCalendar` — литерал «календарь»; месяц — только шапка сетки; геометрия — `gridCellSize`/`dayCellRect`.
