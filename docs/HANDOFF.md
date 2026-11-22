# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: волна 2 и V-17 закрыты → следующий
**V-8** (календарь: шаблон GetItem для CalendarItem в `EWS-REFERENCE.md §11`
через `ews-specialist`, потом код). Старт: `/stage V-8`.

## Сделано
- iter31 V-17 (2430932, dc211b6, 05326c7): поля списка, папки, шапка.
- iter31 V-16 (6275d36): подписи ПАПКИ/ПРОСМОТР убраны, фикс линии заголовка.
- iter31 V-13…V-15, V-11…V-12; V-10…V-1, UI-9 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- V-17 на живом терминале: поля списка, выделение папки, подробная шапка
  (скриншот юзера); скругления отложены — tview только прямоугольная заливка.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview.

## Не трогать
- V-17: «подушка» — NotSelectable и не в itemRows; заливку красят 3 строки и
  оба столбца (paintItemTail); поля — listPadLeft/Right в appendRows И
  clampMessageColumns (иначе resize срежет), заливка отступает от рамки (padX);
  папки — по одной; шапка — FormatMessageHeader; в Table нет «│».
- V-16/V-15: header: nil; end линии — относительный; Enter — readMessage; messageView/reading — как было.
- V-7…V-1/UI-9: кейкап — Escape целиком; [ВНЕШН]=8; цвета — theme.go; g — список.
