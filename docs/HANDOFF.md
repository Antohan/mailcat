# mailcat — HANDOFF

## Дальше
Итерация 3.1: следующий **V-27** (перенос текста панели чтения без
горизонтального скролла), `/stage V-27`; читать `mail.go`: `SetInputCapture`,
`showReading`/`showPreview`; `app.go`: `a.detail`, `SetWrap`.

## Сделано
- iter31 V-26 (176068c): счётчик папок у правого края, длинное имя усекается «…».
- iter31 V-25 (536293f): серия отправителя — имя у первого письма, дальше точка и [ВНЕШН].
- iter31 V-24 (b2059df): при фокусе на папках — свой набор подсказок (`hintFolders`).
- iter31 V-22: отменён (revert 657378a) — выделение без фокуса не приглушается.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; О-3 в Terminal.app/VS Code; V-25/V-26 на живом терминале и живой почте.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-26: ширина панели папок — только из DrawFunc (`setFoldersWidth`, один `SetItemText`, без Clear/GetFocus); `width≤0` — прежний зазор до `folderNameCol`; вид строки — `folderRowText` для обоих путей; `refreshFolderRow` берёт `f` (новый счётчик), не копию из `rowByItem`.
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`: ящик + группа дня); [ВНЕШН] у каждой строки (SECURITY §4); append и clamp считают серию одинаково.
- V-24: `hintFolders` — все сегменты обязательные, ≤100 колонок без opt; у папок нет v/g, есть Enter/→.
- V-23/V-21: панели переключает только cycleFocus (он же зовёт showHint); кэш `a.hintSegs` пишет лишь showHint (V-17); старт — фокус `messages`.
- Выделение — всегда `colorSelectedBg`/`colorSelectedFg`; фокус показывает только accent-рамка (V-22 отменён).
- V-20/V-19/V-18: folderName/accent — только selectFolder; renderFolders помнит папку/«Ещё» (restoreFolderCursor); watcher — только refreshFolderRow.
- V-17: пункт — две строки, вторая — paintItemTail; поля — listPadLeft/Right в appendRows и clampMessageColumns; папки — по одной; в Table нет «│»; connText: ● =1, текст =3; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; header: nil; Enter — readMessage.
