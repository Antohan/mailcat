# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: волна 2 и V-17 закрыты → следующий
**V-8** (календарь: шаблон GetItem для CalendarItem в `EWS-REFERENCE.md §11`
через `ews-specialist`, потом код). Старт: `/stage V-8`.

## Сделано
- iter31 V-17 (2430932): плотность списков, заливка папок, подробная шапка.
- iter31 V-16 (6275d36): подписи ПАПКИ/ПРОСМОТР убраны, фикс линии заголовка.
- iter31 V-13…V-15, V-11…V-12; V-10…V-1, UI-9 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- V-17 на живом терминале: воздух списков, выделение папки, подробная шапка
  (скриншот юзера); скругления отложены — tview только прямоугольная заливка.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write. Отложено:
  item:Importance (нет в EWS-REFERENCE), write-операции, Catppuccin, item:Preview.

## Не трогать
- V-17: прокладка — NotSelectable и не в itemRows; папки двухстрочные только при
  foldersTwoLine (DrawFunc); шапка чтения — FormatMessageHeader; в Table нет «│».
- V-16/V-15: header: nil у папок/просмотра; end линии — относительный; Enter —
  readMessage; messageView общий; reading — сброс в showPreview/selectFolder.
- V-7…V-1/UI-9: кейкап — Escape целиком; [ВНЕШН]=8; цвета — theme.go; g — список.
