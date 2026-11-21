# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-10, V-1…V-6 ✅ → следующий **V-7**
(нижняя панель-кейкапы: клавиша акцентом/бейджем + подпись textSecondary,
набор по активной панели, П.3; UI-9 сохранить; скриншот обязателен).
Старт: `/stage V-7`.

## Сделано
- iter31 V-6: верхний бар «mailcat · папка» / дата+● (тикер 1 мин, скриншоты opencode/v6-*.txt); iter31:fix (1c633d0): «u» при пустом сайдбаре → loadFolders; iter31 V-5: шапка деталей/колонка 90 (v5-*.txt); V-4…V-10 — git log (b56a77d…, e3e8daf).

## Открыто
- **Ждёт «ок» юзера на V-6** (верхний бар) на живом терминале.
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-6: бар — root-строка 1 над pages; правые 18 клеток фикс. шириной (Flex); тикер минут — только через ctx сессии; setConn/selectFolder зовут updateTopBar; тесты панелей ждут бар в строке 0.
- fix: refreshFolder при пустом сайдбаре зовёт loadFolders — обычный «u» не менять.
- V-5: DrawFunc-карточка шапки (гард detailsW); тело в detailsTail; layoutDetails — из before-draw; паддинг — TaggedStringWidth; FormatMessageHeader — только превью.
- V-4/V-3/V-1/V-2: разделители — пункты списка с перескоком; «Ещё» — s.moreIdx; неактивные рамки невидимы (RD-2); truncateEllipsis («…» tview ломает кириллицу); selRow=-1; [ВНЕШН] externalMarkCol=8; headerGap=2.
- UI-9: showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1); цвета — только роли theme.go («сегодня» — colorTodayBg).
