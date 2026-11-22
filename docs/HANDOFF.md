# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-20 закрыт → следующий **V-21**
(стартовый фокус — список писем). Старт: `/stage V-21`.

## Сделано
- iter31 V-20 (d8ee71a): SetChangedFunc не пишет folderName, accent стоит на открытой папке.
- iter31 V-19 (8539aa4): renderFolders сохраняет курсор через restoreFolderCursor.
- iter31 V-18 (6b21b29): watcher обновляет строку «Входящих» через refreshFolderRow.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code; V-17 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-20: folderName и accent задаёт только selectFolder; SetChangedFunc лишь
  applyActiveFolderStyle. Активность по имени (distinguished inbox с ID пуст).
- V-19: renderFolders до Clear помнит папку/«Ещё», после — restoreFolderCursor (ID, при пустом — имя).
- V-18: счётчик watcher'а — только refreshFolderRow (SetItemText, без renderFolders/Clear).
- V-17: пункт — две строки; вторая красится paintItemTail; scrollItemIntoView двигает offset у нижнего края; поля — listPadLeft/Right в appendRows И
  clampMessageColumns; папки — по одной; шапка — FormatMessageHeader; в Table
  нет «│»; смена папки сбрасывает restoreID/row; connText: ● =1, текст =3;
  подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; полосы —
  scrollbar.go (Draw после панели); V-16/V-15: header: nil; Enter — readMessage.
