# mailcat — HANDOFF

## Дальше
**V-47** (`docs/PLAN-ITER3.1.md`): клавиша `t` — выбор сегодняшнего дня, переход
на текущий месяц; подсказки «t сегодня» (`calendar.go` — `monthHook`,
`jumpToToday`, `app.go` — `hintCalendarGrid`/`hintCalendarAgenda`);
далее V-48.

## Сделано
- iter31 V-46: 42 даты, соседние месяцы muted без счётчиков, выбор только in-month.
- iter31 V-45 (1828d04): «сегодня» — accent+bold на фоне панели, `colorTodayBg` удалён.
- iter31 V-44: заливка ячейки до линий, число/«·N» — с отступом 1.
- iter31 V-43 (dc883bd): линии сетки — роль `colorGridLine` (светлее чисел).

## Открыто
- «ок» скриншотов: V-46 (`/private/tmp/v46_{before,after}_{light,dark}_{100x30,220x60}.txt` + `.ans`), V-35…V-45.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-46 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-46: `InMonth` — единственный признак своего дня (выбор/клик/`cellOf`/`maxDay`); `Day` у соседних — настоящий номер; счётчики только in-month; muted без bold.
- V-45: «сегодня» — accent+bold на `colorPanelBg`, счётчик `colorTextSecondary`; выбор — сплошной акцент и при совпадении побеждает.
- V-44: заливка — весь `dayCellRect` до линий (не под текстом); текст +1 вправо, +1 вниз при `ch>1` (при `ch==1` вниз нельзя — там линия); счётчик вплотную после числа.
- V-43: линии — только `colorGridLine` (между hairline и textSecondary); пустые/соседние — те же линии.
- V-42: блоки — по строкам в `Draw`/`ScrollTrack` под ширину (не хранить); сегменты `{текст, стиль}` без tview-тегов (Sanitize, без Escape); клик — `MouseHandler`; пустой день — muted «Встреч нет», sel -1.
