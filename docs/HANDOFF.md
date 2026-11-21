# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`, волна 2: V-13 ✅, V-14 ✅ → следующий
**V-15** (чтение письма в правой панели; прочитать V-15 в плане и
`preview`/`showPreview`/`openDetails` в `mail.go`). Старт: `/stage V-15`.

## Сделано
- iter31 V-13, V-14 (см. git log): полоски сняты (`SetTransparency`), g на списке — реестр ссылок; V-12…V-1 — git log.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code; футер (2 строки) и g на письме со ссылками/без на живом Exchange.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-14: g — только панель списка; GetItem общий (`loadMessage`), `messageLinks` — тот же реестр, что у деталей; ссылки в модал напрямую, `currentLinks` не трогать; нет ссылок — `setStatus`.
- V-13: снятие выделения темы — `SetTransparency(true)`, не `ColorDefault`.
- V-12: футер — `connText(w)`, обрезка username ДО `tview.Escape` только в DrawFunc; высота 2; «подключение…» до первой операции.
- V-11: верхний бар — одна строка «mailcat · <папка>» (Sanitize+Escape); дату/время и ● не возвращать.
- V-7: кейкап — `tview.Escape("["+k+"]")` целиком; подпись у КАЖДОЙ клавиши; status выше hint (UI-9).
- V-6: бар — root-строка 1 над pages; updateTopBar зовёт только selectFolder; тесты панелей ждут бар в строке 0.
- V-5: DrawFunc-карточка шапки (гард detailsW); тело в detailsTail; layoutDetails из before-draw; паддинг TaggedStringWidth.
- V-4…V-1: разделители — пункты списка; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go.
