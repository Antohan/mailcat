# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: волна 2 закрыта (V-11…V-16 ✅) →
следующий **V-8** (календарь: убрать ASCII-баннер, сетка на панель, Enter →
детали встречи, описание через GetItem). Сначала — шаблон GetItem для
CalendarItem в `EWS-REFERENCE.md §11` (consult `ews-specialist`), потом код.
Старт: `/stage V-8`.

## Сделано
- iter31 V-16 (6275d36): подписи ПАПКИ и ПРОСМОТР убраны, фикс вылета линии заголовка.
- iter31 V-13…V-15, V-11…V-12 (см. git log); V-10…V-1, UI-9 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code; футер (2 строки), g на письме, v/g/Esc из панели чтения, отсутствие подписей ПАПКИ/ПРОСМОТР — на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-16: у папок/просмотра `header: nil` (paintFocusFrames пропускает nil); линия заголовка — относительный `end` (w-wRight-headerGap), не `x+w`; тест панелей ждёт подпись ПИСЬМА в средней колонке.
- V-15: Enter — `readMessage` (не `showDetails`); `messageView` общий для панели и деталей; `reading` сбрасывать в showPreview/selectFolder; вложения/ссылки — `back func()` на источник.
- V-14: g — только панель списка; GetItem общий (`loadMessage`); нет ссылок — `setStatus`.
- V-12…V-11: футер — `connText(w)`, обрезка username ДО `tview.Escape`, высота 2; верхний бар — одна строка «mailcat · <папка>» (Sanitize+Escape), дату/время и ● не возвращать.
- V-7…V-1: кейкап — `tview.Escape("["+k+"]")` целиком, подпись у КАЖДОЙ клавиши, status выше hint; бар — root-строка 1 над pages; DrawFunc-карточка шапки (гард detailsW); разделители — пункты списка; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go.
