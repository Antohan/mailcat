# mailcat — HANDOFF

## Дальше
**V-33** (волна 5, `docs/PLAN-ITER3.1.md`): палитра и типографика сетки —
стили в `calendar_grid.go` (Draw), `FormatDayCell` без скобок `[N]` в
`calendar_format.go`; далее V-34…V-36.

## Сделано
- iter31 V-32 (4202cf3): сетка календаря заполняет панель — `monthGrid`.
- iter31 V-31 (811622b): шапки и верхний бар календаря — без капса, счётчики справа.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31/V-32 на живом терминале; «ок» скриншотов V-32 до старта V-33.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-32: сетка — только `monthGrid` (Box+Draw), не `tview.Table`; геометрия — чистые `dayCellRect`/`dayAt` на каждый Draw (resize бесплатно); седьмая колонка забирает остаток ширины; смена выбора — только `selectCell` (`onSelect` → повестка); текст ячейки — `gridText`, не `tview.Print`.
- V-31: шапки календаря — `SetListTitle` (без капса; капс только UI-надписи, V-2); слева `MonthTitle`/`AgendaTitle`, справа `meetingsCount` (muted всегда); верхний бар строит `updateTopBar` по фронт-странице, зовётся из `globalKeys` 1/2 и `shiftMonth`; разделитель колонок — `newDivider`, как на почте.
- Волна 5: панели/списки календаря — theme-роли, `colorSelectedBg/Fg`, фокус — только accent-рамка (V-22 отменён).
- V-29: разделители папок не выбираемы — `SetInputCapture`+`foldersMouse`; «Ещё» — по тексту (`isFolderRow`); после `SetCurrentItem` — `applyActiveFolderStyle`.
- V-28: `formatBody` — Sanitize + tview.Escape ПОСТРОЧНО (SECURITY §3); заголовок — `[accent::b]`; ссылки/вложения/шапка V-15…V-17 не трогать.
- V-27: `wrappedLines` — по `tview.WordWrap`, не `GetWrappedLineCount`; →/l/h поглощает `SetInputCapture` панели чтения и `a.detail`; ← — «к списку»; `h/l` в черновике — тот же TextView.
- V-26: ширина папок — только из DrawFunc (`setFoldersWidth`, один `SetItemText`).
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`); [ВНЕШН] у каждой строки (SECURITY §4).
- V-24/V-23/V-21: `hintFolders` — все сегменты обязательные; панели переключает только cycleFocus; кэш `a.hintSegs` пишет лишь showHint; старт — фокус `messages`.
