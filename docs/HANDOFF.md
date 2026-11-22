# mailcat — HANDOFF

## Дальше
Итерация 3.1: V-21 закрыт → следующий **V-22** (приглушение выделения без
фокуса), `/stage V-22`; перед стартом — «ок» пользователя по скриншотам V-21.

## Сделано
- iter31 V-21 (9104f4b): стартовый фокус — список писем (флаг колонки + SetFocus в New).
- iter31 V-20 (d8ee71a): SetChangedFunc не пишет folderName, accent стоит на открытой папке.
- iter31 V-19 (8539aa4): renderFolders сохраняет курсор через restoreFolderCursor.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; О-3 в Terminal.app/VS Code; V-17 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-21: стартовый фокус — `messages`: флаг `focus` у колонки писем и `SetFocus` в `New`.
- V-20: folderName/accent задаёт только selectFolder; SetChangedFunc — лишь applyActiveFolderStyle (по имени: inbox с пустым ID).
- V-19: renderFolders до Clear помнит папку/«Ещё», после — restoreFolderCursor (ID, при пустом — имя).
- V-18: счётчик watcher'а — только refreshFolderRow (SetItemText, без renderFolders/Clear).
- V-17: пункт — две строки, вторая — paintItemTail; scrollItemIntoView двигает offset у нижнего края; поля — listPadLeft/Right в appendRows и clampMessageColumns; папки — по одной; шапка — FormatMessageHeader; в Table нет «│»; смена папки сбрасывает restoreID/row; connText: ● =1, текст =3; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; полосы — scrollbar.go после панели; header: nil; Enter — readMessage.
