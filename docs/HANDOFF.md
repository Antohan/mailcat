# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`, волна 2: V-12 ✅ → следующий **V-13**
(остаточные полоски при движении по списку; прочитать V-13 и
`appendRows`/`SetSelectedStyle` в `mail.go`).
Старт: `/stage V-13`.

## Сделано
- iter31 V-12 (см. git log): футер сайдбара — «● Connected» / username в две строки, «…» по ширине панели (DrawFunc); V-11…V-1 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code; проверить футер (2 строки, обрезка логина) на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-12: футер — `connText(w)`, обрезка username ДО `tview.Escape` и только в DrawFunc (resize); высота 2 в foldersCol; «подключение…» до первой операции.
- V-11: верхний бар — одна строка, только «mailcat · <папка>» (Sanitize+Escape); дату/время и ● не возвращать, тикер минут не воскрешать.
- V-7: кейкап — `tview.Escape("["+k+"]")` целиком; подпись у КАЖДОЙ клавиши; status остаётся строкой ВЫШЕ hint (UI-9).
- V-6: бар — root-строка 1 над pages; updateTopBar зовёт только selectFolder (имя папки); тесты панелей ждут бар в строке 0.
- V-5: DrawFunc-карточка шапки (гард detailsW); тело в detailsTail; layoutDetails — из before-draw; паддинг — TaggedStringWidth.
- V-4/V-3/V-1/V-2: разделители — пункты списка с перескоком; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go.
