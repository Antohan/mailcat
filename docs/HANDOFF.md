# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`, волна 2: V-11 ✅ → следующий **V-12**
(футер сайдбара в две строки «● Connected» / username; прочитать V-12 и
`mail.go` — connFooter/setConn).
Старт: `/stage V-12`.

## Сделано
- iter31 V-11 (919d926): верхний бар — только «mailcat · папка»; дата/время, ● и минутный тикер убраны; V-7 (кейкапы), V-6 (бар), V-5, V-4…V-10 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-11: верхний бар — одна строка, только «mailcat · <папка>» (Sanitize+Escape); дату/время и ● не возвращать, тикер минут не воскрешать; статус связи — только футер сайдбара (setConn).
- V-7: кейкап — `tview.Escape("["+k+"]")` целиком; подпись у КАЖДОЙ клавиши; status остаётся строкой ВЫШЕ hint (UI-9).
- V-6: бар — root-строка 1 над pages; updateTopBar зовёт только selectFolder (имя папки); тесты панелей ждут бар в строке 0.
- V-5: DrawFunc-карточка шапки (гард detailsW); тело в detailsTail; layoutDetails — из before-draw; паддинг — TaggedStringWidth.
- V-4/V-3/V-1/V-2: разделители — пункты списка с перескоком; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go.
