# mailcat — HANDOFF

## Дальше
Итерация 4 — `docs/PLAN-ITER4.md`, задача C-5 (права файла CA).
Читать: задачу C-5, `internal/platform/platform.go` (`CheckFilePerms` 73), `cmd/mailcat/main.go:89–92`, `cmd/mailcat/doctor.go` шаг 2.
Старт: `/stage C-5`.

## Сделано
- d635d9d — iter4:C-4: усечение строк списка до tview.Escape, тест на `[red]`.
- 7ff57b5 — iter4:C-3: macOS-типы вложений в списке опасных.

## Открыто
- C-4: скриншот до/после (`[red[…` → `[red]…`) в отчёте, «ок» пользователя — TODO.
- Живой smoke, TODO: C-3 — предупреждение `.command`/`.app`/`.dmg` и `.pdf`; A-4 — `/stage` на `main`; A-3 — `/release`. «ок» на V-35…V-53 нет; S3–S7 — TODO.
- Отложено: item:Importance, write-операции, Catppuccin, Preview; doctor печатает `hp`/`ca_path`/логин без Sanitize (вне C-1).

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схему аутентификации выбирает сервер, после C-8 транспорт лишь её помнит.
- Вложения — только через `security.CreateUnique` (O_EXCL).
- Скилл EWS: `item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`; `policy_test.go` не ослаблять.
- Ревьюер без shell: diff и stat путями; `#nosec` — только семь мест A-8; релиз стоит на `КРИТИЧНО`/`ВАЖНО`, M < N — не принято.
- Сетка календаря — тонкие линии 55/45; клик по колонке недель день не выбирает, ширина 3 (V-52).
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; порядок шагов задачи один, CURRENT — первая непустая строка без `#`.
- Вывод ошибок doctor — только `security.Sanitize(err.Error())`; ключи конфига — строго по Config, `org_domain` допустим, 500/380 не поднимать.
- Опасные расширения — Windows/Office + 13 macOS-типов (C-3; `SECURITY.md §4` — A-9); список писем — `truncateEllipsis` до `tview.Escape`, `senderLabel` без Escape (C-4).
