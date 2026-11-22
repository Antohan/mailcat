# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-18 закрыт → следующий **V-19**
(`renderFolders` сохраняет курсор, `restoreFolderCursor`, moreIdx). Старт: `/stage V-19`.

## Сделано
- iter31 V-18 (6b21b29): watcher обновляет строку «Входящих» через refreshFolderRow.
- iter31 V-17 (2430932 … 8ab12a0, 0a15523): поля, папки, шапка, подсказки, полосы.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code;
  V-17 на живом терминале (поля, выделение папки, подробная шапка).
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено:
  item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-18: счётчик watcher'а — только refreshFolderRow (SetItemText по строке,
  формула FormatFolder; без renderFolders/Clear — они сбрасывают курсор).
- V-17: пункт — две строки; вторая красится paintItemTail; scrollItemIntoView
  двигает offset у нижнего края; поля — listPadLeft/Right в appendRows И
  clampMessageColumns; папки — по одной; шапка — FormatMessageHeader; в Table
  нет «│»; смена папки сбрасывает restoreID/row; connText: ● =1, текст =3;
  подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; полосы —
  scrollbar.go (Draw после панели); V-16/V-15: header: nil; Enter — readMessage.
