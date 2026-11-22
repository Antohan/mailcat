# mailcat — HANDOFF

## Дальше
**V-30** (волна 5, `docs/PLAN-ITER3.1.md`): календарь без ASCII-баннера —
`calendar.go`/`banner.go`; далее V-31…V-36 по плану.

## Сделано
- iter31 V-29 (36bb752): разделители групп папок пропускаются курсором и кликом.
- iter31 V-28 (499e03b): заголовки тела — жирный accent, пустые строки схлопнуты.
- iter31 V-27 (a1d2d85): →/l/h не прокручивают текст панели чтения по горизонтали.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте (ложные заголовки
  best-effort); О-3 в Terminal.app/VS Code; V-25/V-26/V-29 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-29: разделители не выбираемы — клавиши поглощает `SetInputCapture` панели (`moveFolderCursor`), клик — `foldersMouse` в mouse capture (событие гасится, позиция НЕ подменяется для List); «Ещё» — по тексту (`isFolderRow`), не по `moreIdx`; после `SetCurrentItem` — `applyActiveFolderStyle`; Enter разделителя и вид строк не трогаются.
- V-28: `formatBody` — Sanitize + tview.Escape ПОСТРОЧНО (SECURITY §3), не на всё тело; заголовок — `[accent::b]` (`isBodyHeading`, WHY), пустая строка после него убирается; ссылки/вложения/шапка V-15…V-17 и хвост `messageView` не трогать.
- V-27: `wrappedLines` считает строки `tview.WordWrap`, НЕ `GetWrappedLineCount` (тот портит индекс TextView: `t.width==0`); →/l/h поглощает только `SetInputCapture` панели чтения и `a.detail`; ← — «к списку» (V-15), `SetScrollable` и полоса прокрутки V-17 не трогаются; `h/l` поглощены и в черновике — тот же TextView.
- V-26: ширина панели папок — только из DrawFunc (`setFoldersWidth`, один `SetItemText`, без Clear/GetFocus); вид строки — `folderRowText` для обоих путей.
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`: ящик + группа дня); [ВНЕШН] у каждой строки (SECURITY §4).
- V-24/V-23/V-21: `hintFolders` — все сегменты обязательные; панели переключает только cycleFocus (он же зовёт showHint); кэш `a.hintSegs` пишет лишь showHint (V-17); старт — фокус `messages`.
- Выделение — всегда `colorSelectedBg`/`colorSelectedFg`; фокус показывает только accent-рамка (V-22 отменён). Календарь (волна 5) — те же правила: theme-роли, `SetListTitle` без капса данных, списки/бары как в почте; сетка — `monthGrid`, не `Table`.
- V-17: пункт — две строки, вторая — paintItemTail; поля — listPadLeft/Right; в Table нет «│»; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; Enter — readMessage.
