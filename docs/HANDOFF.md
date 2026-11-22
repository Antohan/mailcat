# mailcat — HANDOFF

## Дальше
**V-32** (волна 5, `docs/PLAN-ITER3.1.md`): сетка календаря заполняет панель —
`tview.Table` → `monthGrid` в новом `calendar_grid.go`; далее V-33…V-36.

## Сделано
- iter31 V-31 (811622b): шапки и верхний бар календаря — без капса, счётчики справа.
- iter31 V-30 (80c83e1): календарь без ASCII-баннера.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31 на живом терминале; «ок» скриншотов V-31 до старта V-32.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-31: шапки календаря — `SetListTitle` (без капса; капс только UI-надписи, V-2); слева `MonthTitle`/`AgendaTitle`, справа `meetingsCount` (muted всегда); верхний бар строит `updateTopBar` по фронт-странице, зовётся из `globalKeys` 1/2 и `shiftMonth`; разделитель колонок — `newDivider`, как на почте.
- Волна 5: панели/списки календаря — theme-роли, `colorSelectedBg/Fg`, фокус — только accent-рамка (V-22 отменён); сетка — `monthGrid`, не `Table` (V-32).
- V-29: разделители папок не выбираемы — `SetInputCapture`+`foldersMouse`; «Ещё» — по тексту (`isFolderRow`); после `SetCurrentItem` — `applyActiveFolderStyle`.
- V-28: `formatBody` — Sanitize + tview.Escape ПОСТРОЧНО (SECURITY §3); заголовок — `[accent::b]`; ссылки/вложения/шапка V-15…V-17 не трогать.
- V-27: `wrappedLines` — по `tview.WordWrap`, не `GetWrappedLineCount`; →/l/h поглощает `SetInputCapture` панели чтения и `a.detail`; ← — «к списку»; `h/l` в черновике — тот же TextView.
- V-26: ширина папок — только из DrawFunc (`setFoldersWidth`, один `SetItemText`).
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`); [ВНЕШН] у каждой строки (SECURITY §4).
- V-24/V-23/V-21: `hintFolders` — все сегменты обязательные; панели переключает только cycleFocus; кэш `a.hintSegs` пишет лишь showHint; старт — фокус `messages`.
- V-17: пункт — две строки, вторая — paintItemTail; поля — listPadLeft/Right; Enter — readMessage.
