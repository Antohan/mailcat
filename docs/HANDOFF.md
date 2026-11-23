# mailcat — HANDOFF

## Дальше
**V-45** (`docs/PLAN-ITER3.1.md`): «сегодня» без tint — номер `colorAccent`+bold
на фоне панели, `colorTodayBg` удалить, если не нужен (`calendar_grid.go` —
`cellStyles`); далее V-46, V-47.

## Сделано
- iter31 V-44: заливка ячейки до линий, число/«·N» — с отступом 1.
- iter31 V-43 (dc883bd): линии сетки — роль `colorGridLine` (светлее чисел).
- iter31 V-42 (b712081): повестка — `agendaList` (Box+Draw), блоки с переносом, клик/скролл/бар.
- iter31 V-41/V-40/V-39/V-38 (04907e9/f3294cb/ff5564f/5903303): бар, «сегодня», bold, геометрия.

## Открыто
- «ок» скриншотов: V-44 (`/tmp/v44_{before,after}_{light,dark}_{100x30,220x60}.txt` + `.ans`), V-43…V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-44 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-44: заливка — весь `dayCellRect` до линий (не под текстом); текст +1 вправо, +1 вниз при `ch>1` (при `ch==1` вниз нельзя — там линия); счётчик вплотную после числа.
- V-43: линии — только `colorGridLine` (между hairline и textSecondary); пустые/соседние — те же линии.
- V-42: блоки — по строкам в `Draw`/`ScrollTrack` под ширину (не хранить); сегменты `{текст, стиль}` без tview-тегов (Sanitize, без Escape); клик — `MouseHandler`; пустой день — muted «Встреч нет», sel -1.
- V-41: на `pageCalendar` верхний бар — литерал «календарь»; месяц — только шапка сетки.
- V-40: «сегодня» — `colorAccent` на `colorTodayBg`; выбор — акцент/onAccent, совпадение побеждает выбранный.
- V-38: геометрия — только `gridCellSize`/`dayCellRect`; линии — после ячеек; заливки — внутри `dayCellRect`.
