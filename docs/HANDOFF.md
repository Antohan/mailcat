# mailcat — HANDOFF

## Дальше
Итерация 3.1: следующий **V-26** (счётчики папок в правый столбец),
`/stage V-26`; читать `format.go: FormatFolder`, `mail.go: renderFolders`,
`applyActiveFolderStyle`, `foldersW`, DrawFunc списка папок. Визуальная волна —
после «ок» пользователя по V-25 (серии отправителя).

## Сделано
- iter31 V-25 (536293f): серия отправителя — имя у первого письма, дальше точка и [ВНЕШН].
- iter31 V-24 (b2059df): при фокусе на папках — свой набор подсказок (`hintFolders`).
- iter31 V-23 (022c0e2): Tab/Shift-Tab зовёт showHint после SetFocus.
- iter31 V-22: отменён (revert 657378a) — выделение без фокуса не приглушается.
- iter31 V-21 (9104f4b): стартовый фокус — список писем.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; О-3 в Terminal.app/VS Code; V-25 на живом терминале и живой почте.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`: ящик + группа дня); [ВНЕШН] у каждой строки (SECURITY §4); append и clamp считают серию одинаково.
- V-24: `hintFolders` — все сегменты обязательные, ≤100 колонок без opt; у папок нет v/g, есть Enter/→.
- V-23/V-21: панели переключает только cycleFocus (он же зовёт showHint); кэш `a.hintSegs` пишет лишь showHint (V-17); старт — фокус `messages`.
- Выделение — всегда `colorSelectedBg`/`colorSelectedFg`; фокус показывает только accent-рамка (V-22 отменён).
- V-20/V-19/V-18: folderName/accent — только selectFolder; renderFolders помнит папку/«Ещё» (restoreFolderCursor); watcher — только refreshFolderRow.
- V-17: пункт — две строки, вторая — paintItemTail; поля — listPadLeft/Right в appendRows и clampMessageColumns; папки — по одной; в Table нет «│»; connText: ● =1, текст =3; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; header: nil; Enter — readMessage.
