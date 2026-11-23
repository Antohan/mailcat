# mailcat — HANDOFF

## Дальше
**V-40** (волна 6, `docs/PLAN-ITER3.1.md`): «сегодня» — номер `colorAccent` +
bold, фон `colorTodayBg`; выбранный — сплошной акцент/onAccent; совпадение —
выбранный. `calendar_grid.go` (`cellStyles`), `calendar_grid_test.go`/sim.

## Сделано
- iter31 V-39 (ff5564f): числа дней bold, счётчик regular, без `%2d`-паддинга, flush left.
- iter31 V-38 (5903303): разделители ячеек сетки — Vertical/Horizontal/Cross, textSecondary.
- iter31 V-37 (7553acc): ячейки сетки одного размера, остаток — внешние поля.

## Открыто
- «ок» скриншотов: V-39 (`/tmp/v39_{before,after}_{100,220}.txt`), V-38, V-37, V-36, V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-39 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-39: число дня — `AttrBold` у всех состояний (выходной/сегодня/выбранный), счётчик — regular textSecondary (на акценте onAccent); `FormatDayCell` — `%d`, без паддинга; шапка и числа от левого края `dayCellRect`.
- V-38: геометрия — только `gridCellSize` (`cw=(w-6)/7`, `ch=(h-7)/6`) и `dayCellRect` (внутренний прямоугольник: +2 над неделями, +1 на линию между); линии — клетка, `Cross` на пересечениях, `colorTextSecondary`, после ячеек; заливки — внутри `dayCellRect`.
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading` гасит двойной Enter; шапка и тело — один реестр ссылок (маркеры [N]); тело только через `formatBody` (Sanitize+Escape построчно); GetItem — §11, без `BodyType`, разбор `t:CalendarItem`.
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`, не capture; 1/2 — `ensureFocus` в сетку, фокус повестки сохраняется.
- V-34: повестка — `SetItemText` при смене курсора (не Clear); changed идёт до сдвига и на AddItem — зажим по `GetItemCount` в `restyleAgenda`; бар — колонка за рамкой.
- V-33/V-32: стили ячейки — только `cellStyles` (выходной→сегодня→выбор); счётчик — отдельный стиль, на акценте `colorSelectedFg`; сетка — только `monthGrid` (Box+Draw); смена выбора — только `selectCell`.
