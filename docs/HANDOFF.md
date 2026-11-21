# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-10, V-1…V-5 ✅ → следующий **V-6**
(верхний бар: «mailcat · <папка>» слева; дата/время и ● справа; одна строка,
минимум высоты; скриншот обязателен). Старт: `/stage V-6`.

## Сделано
- iter31 V-5: шапка деталей — аватар-инициалы, «Имя <email>», дата справа, «Кому», линия, тема жирным; колонка чтения 90 (центр) / вся минус поля (<100); «Вложения N»; 📎 убран; скриншоты opencode/v5-*.txt. iter31 V-4…V-10 и релиз 2 — `git log --oneline` (b56a77d…, e3e8daf).

## Открыто
- **Ждёт «ок» юзера на V-5** (шапка/колонка чтения) на живом терминале.
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено: write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-5: DrawFunc пересобирает карточку шапки по ширине, гард `a.detailsW`; тело кэшируется в `detailsTail` (resize не перегенерирует реестр ссылок); `layoutDetails` — только из before-draw; паддинг даты — `tview.TaggedStringWidth`; усечение — `truncateEllipsis`; `FormatMessageHeader` — только превью.
- V-4: разделители — пункты списка с перескоком; «Ещё» — `s.moreIdx`; футер SetWrap(false); applyActiveFolderStyle пересчитывает ВСЕ строки.
- V-3/V-1/V-2: рамки одинарные; разделители textSecondary; неактивные рамки невидимы (RD-2); truncateEllipsis («…» tview ломает кириллицу); selRow=-1 в rebuildRows; [ВНЕШН] externalMarkCol=8; headerGap=2.
- Статус-бар (UI-9): showHint→a.hint; ошибки Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only; itemRows — единственный маппинг (О-1).
- RD-1/3: цвета — только роли theme.go; «сегодня» colorTodayBg (О-3).
