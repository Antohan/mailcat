# mailcat — HANDOFF

## Дальше
**V-42** (волна 6, `docs/PLAN-ITER3.1.md`): повестка вместо `tview.List` —
`agendaList` (Box+Draw): перенос строк, выделение блока, скролл/бар, клик, пустой
день; `calendar_agenda.go` (новый), `calendar.go`, `calendar_format.go`, тесты.

## Сделано
- iter31 V-41 (04907e9): верхний бар календаря — «mailcat · календарь», месяц только в шапке сетки.
- iter31 V-40 (f3294cb): «сегодня» — акцентный номер на tint; совпадение → выбор.
- iter31 V-39/V-38 (ff5564f/5903303): числа bold, счётчик regular, flush left; линии ячеек сетки.

## Открыто
- «ок» скриншотов: V-41 (`/tmp/v41_{before,after}_{100x30,220x60}.txt`), V-40, V-39, V-38, V-37, V-36, V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-41 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-41: на `pageCalendar` верхний бар — литерал «календарь»; `shiftMonth` бар не пересобирает; месяц — только шапка сетки (`setListTitle`); почтовый бар — папка.
- V-40: «сегодня» — номер `colorAccent` на tint `colorTodayBg`; выбор — сплошной акцент/onAccent; порядок наложения выходной→сегодня→выбор (совпадение побеждает выбранный).
- V-39: число дня — `AttrBold` у всех состояний, счётчик — regular textSecondary (на акценте onAccent); `FormatDayCell` — `%d`, без паддинга; шапка и числа от левого края `dayCellRect`.
- V-38: геометрия — только `gridCellSize` и `dayCellRect`; линии — клетка, `Cross` на пересечениях, `colorTextSecondary`, после ячеек; заливки — внутри `dayCellRect`.
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading` гасит двойной Enter; шапка и тело — один реестр ссылок; тело только через `formatBody`; GetItem — §11, без `BodyType`, `t:CalendarItem`.
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`; 1/2 — `ensureFocus` в сетку.
- V-33/V-32: стили ячейки — только `cellStyles`; счётчик — отдельный стиль; сетка — только `monthGrid` (Box+Draw); смена выбора — только `selectCell`.
