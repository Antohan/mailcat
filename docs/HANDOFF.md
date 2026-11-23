# mailcat — HANDOFF

## Дальше
**V-41** (волна 6, `docs/PLAN-ITER3.1.md`): верхний бар на календаре —
«mailcat · календарь» вместо месяца, месяц только в шапке сетки; 1/2 обновляет.
`app.go` (`updateTopBar`), `app_test.go`.

## Сделано
- iter31 V-40 (f3294cb): «сегодня» — акцентный номер на tint; совпадение → выбор.
- iter31 V-39 (ff5564f): числа bold, счётчик regular, без `%2d`-паддинга, flush left.
- iter31 V-38 (5903303): разделители ячеек сетки — Vertical/Horizontal/Cross, textSecondary.
- iter31 V-37 (7553acc): ячейки сетки одного размера, остаток — внешние поля.

## Открыто
- «ок» скриншотов: V-40 (`/tmp/v40_{before,after}_{100,220}_sel{20,17}.txt`), V-39, V-38, V-37, V-36, V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-40 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-40: «сегодня» — номер `colorAccent` на tint `colorTodayBg`; выбор — сплошной акцент/onAccent; порядок наложения выходной→сегодня→выбор (совпадение побеждает выбранный).
- V-39: число дня — `AttrBold` у всех состояний (выходной/сегодня/выбранный), счётчик — regular textSecondary (на акценте onAccent); `FormatDayCell` — `%d`, без паддинга; шапка и числа от левого края `dayCellRect`.
- V-38: геометрия — только `gridCellSize` (`cw=(w-6)/7`, `ch=(h-7)/6`) и `dayCellRect`; линии — клетка, `Cross` на пересечениях, `colorTextSecondary`, после ячеек; заливки — внутри `dayCellRect`.
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading` гасит двойной Enter; шапка и тело — один реестр ссылок (маркеры [N]); тело только через `formatBody` (Sanitize+Escape построчно); GetItem — §11, без `BodyType`, разбор `t:CalendarItem`.
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`, не capture; 1/2 — `ensureFocus` в сетку, фокус повестки сохраняется.
- V-33/V-32: стили ячейки — только `cellStyles` (выходной→сегодня→выбор); счётчик — отдельный стиль, на акценте `colorSelectedFg`; сетка — только `monthGrid` (Box+Draw); смена выбора — только `selectCell`.
