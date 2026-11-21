# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-10, V-1…V-7 ✅ → следующий **V-8**
(календарь: убрать ASCII-баннер, сетка на панель, Enter→детали, GetItem Body;
СПЕКА сначала — `EWS-REFERENCE.md §11`, скриншот обязателен).
Старт: `/stage V-8` — только после «ок» юзера на V-6 и V-7.

## Сделано
- iter31 V-7: кейкапы `[клавиша]` акцентом + подпись у каждой (v7-*.txt); V-6: бар «mailcat · папка»/дата+● (v6-*.txt); iter31:fix (1c633d0): «u» при пустом сайдбаре; V-5: шапка/колонка 90; V-4…V-10 — git log.

## Открыто
- **Ждёт «ок» юзера**: V-6 (верхний бар) и V-7 (кейкапы) на живом терминале; без «ок» V-8 не стартует.
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-7: кейкап — `tview.Escape("["+k+"]")` целиком (иначе буквенные бейджи съедает парсер тегов); подпись у КАЖДОЙ клавиши; «1/2»→«экран» ради 100 колонок; status остаётся строкой ВЫШЕ hint (UI-9).
- V-6: бар — root-строка 1 над pages; правые 18 клеток фикс. шириной; тикер минут — только через ctx сессии; setConn/selectFolder зовут updateTopBar; тесты панелей ждут бар в строке 0.
- fix: refreshFolder при пустом сайдбаре зовёт loadFolders — обычный «u» не менять.
- V-5: DrawFunc-карточка шапки (гард detailsW); тело в detailsTail; layoutDetails — из before-draw; паддинг — TaggedStringWidth; FormatMessageHeader — только превью.
- V-4/V-3/V-1/V-2: разделители — пункты списка с перескоком; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis; selRow=-1; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go («сегодня» — colorTodayBg).
