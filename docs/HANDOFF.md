# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: волна 2 и V-17 закрыты → следующий
**V-8** (календарь: шаблон GetItem для CalendarItem в `EWS-REFERENCE.md §11`
через `ews-specialist`, потом код). Старт: `/stage V-8`.

## Сделано
- iter31 V-17 (2430932 … 8ab12a0, 0a15523): поля, папки, шапка, подсказки, полосы.
- iter31 V-16 (6275d36): подписи ПАПКИ/ПРОСМОТР убраны, фикс линии заголовка.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- V-17 на живом терминале: поля списка, выделение папки, подробная шапка
  (скриншот юзера); скругления отложены — tview только прямоугольная заливка.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview.

## Не трогать
- V-17: пункт — две строки; вторая красится paintItemTail (оба столбца);
  scrollItemIntoView двигает offset у нижнего края (иначе тема обрезается);
  поля — listPadLeft/Right в appendRows И clampMessageColumns; папки — по одной; шапка — FormatMessageHeader; в Table нет «│»; смена папки сбрасывает
  restoreID/row (switched); connText: ● =1, текст/адрес =3; подсказки — opt
  снимает formatHintFit, в DrawFunc НЕ звать GetFocus (a.hintSegs) — мьютекс;
  полосы — scrollbar.go (Draw после панели), wrappedLines кэш.
- V-16/V-15: header: nil; end линии — относительный; Enter — readMessage; messageView/reading — как было.
