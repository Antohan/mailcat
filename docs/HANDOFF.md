# mailcat — HANDOFF

## Дальше
Итерация 3.1 — `docs/PLAN-ITER3.1.md`: V-10 ✅, V-1…V-4 ✅ → следующий **V-5**
(просмотр письма, `app.go` + `format.go` FormatMessageHeader, П.1 ширина
чтения ~90: аватар-инициалы, «Имя <email>», дата, «Кому: …», вложения N;
v/s/o и RD-5 сохраняются). Флагшайп для лейаута. Скриншот до/после обязателен;
**без «ок» юзера на V-4 не стартовать**. Старт: `/stage V-5`.

## Сделано
- iter31 V-4: без иконок (отступ 2, folderNameCol=15); линии-разделители блоков
  (Enter перепрыгивает); футер «● Connected · user» (setConn, точка зелёная/
  красная); счётчик на выделении colorSelectedFg (П.2б); скриншоты
  /private/var/.../opencode/v4-before.txt, v4-after.txt (110×28).
- iter31 V-3: рамка фокуса одинарная; divider между колонками; v3-*.txt.
- iter31 V-2/V-1: заголовок «Папка · N непрочитанных · всего · стр.»;
  двухстрочные строки списка. Релиз итерации 2: e3e8daf; smoke S1–S8 ок.

## Открыто
- **Ждёт «ок» юзера на V-4** (сайдбар) на живом терминале — скриншоты в отчёте.
- TODO юзера (TESTING.md §5): smoke S3–S7 после 3.1; О-3 в Terminal.app/VS Code.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write.
- item:Importance нет в EWS-REFERENCE (min_importance, ⚠). Отложено:
  write-операции, палитра Catppuccin, сниппет item:Preview (риск 500).

## Не трогать
- V-4: разделители — пункты списка с перескоком (не TextView); индекс «Ещё» —
  s.moreIdx (разделители сдвигают индексы); applyActiveFolderStyle пересчитывает
  ВСЕ строки; футер SetWrap(false); на красной точке — «Нет связи».
- V-3: focus-символы Borders одинарные глобально; разделители textSecondary
  (не hairline); рамки неактивных панелей невидимы (RD-2).
- V-1: усечение — только truncateEllipsis («…» tview ломает кириллицу);
  selRow=-1 в rebuildRows обязателен; [ВНЕШН] externalMarkCol=8.
- V-2: SetListTitle без капса; правая часть — textSecondary (RD-1); headerGap=2.
- Статус-бар (UI-9): showHint→a.hint; endLoad один раз на ветку; ошибки —
  Sanitize. EWS: IsRead в FindItem / BodyType в GetItem → 500; read-only.
- itemRows — единственный маппинг «строка↔s.items» (О-1); RD-7: спиннер SetText.
- RD-1/3: цвета — только роли theme.go; О-3: «сегодня» colorTodayBg.
