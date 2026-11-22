# mailcat — HANDOFF

## Дальше
Итерация 3.1: следующий **V-24** (подсказки панели папок), `/stage V-24`;
читать `app.go: hintMail/pageHintSegs/showHint`, `mail.go: paintFocusFrames`.
Старт визуальной волны — после «ок» пользователя по выделению (откат V-22).

## Сделано
- iter31 V-23 (022c0e2): Tab/Shift-Tab зовёт showHint после SetFocus.
- iter31 V-22: отменён — приглушение выделения без фокуса откачено (revert 657378a).
- iter31 V-21 (9104f4b): стартовый фокус — список писем (флаг колонки + SetFocus в New).
- iter31 V-20 (d8ee71a): SetChangedFunc не пишет folderName, accent стоит на открытой папке.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; О-3 в Terminal.app/VS Code; V-17 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-23: панели переключает только cycleFocus, и он же после SetFocus зовёт showHint — кэш `a.hintSegs` пишет лишь showHint (V-17), поэтому Tab без него оставлял чужие подсказки.
- Выделение — всегда `colorSelectedBg`/`colorSelectedFg`, независимо от фокуса; фокус показывает только accent-рамка (V-22 отменён).
- V-21: стартовый фокус — `messages`: флаг `focus` у колонки писем и `SetFocus` в `New`.
- V-20: folderName/accent задаёт только selectFolder; SetChangedFunc — лишь applyActiveFolderStyle (по имени: inbox с пустым ID).
- V-19: renderFolders до Clear помнит папку/«Ещё», после — restoreFolderCursor (ID, при пустом — имя).
- V-18: счётчик watcher'а — только refreshFolderRow (SetItemText, без renderFolders/Clear).
- V-17: пункт — две строки, вторая — paintItemTail; scrollItemIntoView двигает offset у нижнего края; поля — listPadLeft/Right в appendRows и clampMessageColumns; папки — по одной; шапка — FormatMessageHeader; в Table нет «│»; смена папки сбрасывает restoreID/row; connText: ● =1, текст =3; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; полосы — scrollbar.go после панели; header: nil; Enter — readMessage.
