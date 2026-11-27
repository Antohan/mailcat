# mailcat — HANDOFF

## Дальше
Итерация 4 — `docs/PLAN-ITER4.md`, задача C-7 (частичный файл вложения).
Читать: задачу C-7, `internal/ui/attachments.go` (`saveAttachment` 103–150).
Старт: `/stage C-7`.

## Сделано
- 64ac8bd — iter4:C-6: безопасные имена файлов, лимит перебора.
- d1a0a34 — iter4:C-5: права `ca_path` (0644 → предупреждение при старте и в doctor).

## Открыто
- Живой smoke, TODO: C-5 — CA 0644 даёт предупреждение; C-4 — скриншот до/после (`[red[…` → `[red]…`), «ок» пользователя; C-3 — `.command`/`.app`/`.dmg`; A-4 — `/stage` на `main`; A-3 — `/release`. «ок» на V-35…V-53 нет; S3–S7 — TODO.
- Отложено: item:Importance, write-операции, Catppuccin, Preview; doctor печатает `hp`/`ca_path`/логин без Sanitize (вне C-1).

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схему аутентификации выбирает сервер, после C-8 транспорт лишь её помнит.
- Вложения — только через `security.CreateUnique` (O_EXCL); имя — без ведущих точек/пробелов (`TrimLeft(". \t")`), лимит 10 000.
- Скилл EWS: `item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`; `policy_test.go` не ослаблять.
- Ревьюер без shell: diff и stat путями; `#nosec` — только семь мест A-8; релиз стоит на `КРИТИЧНО`/`ВАЖНО`, M < N — не принято.
- Сетка календаря — тонкие линии 55/45; клик по колонке недель день не выбирает, ширина 3 (V-52).
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; порядок шагов задачи один, CURRENT — первая непустая строка без `#`.
- Вывод ошибок doctor — только `security.Sanitize(err.Error())`; ключи конфига — строго по Config, `org_domain` допустим, 500/380 не поднимать.
- Опасные расширения — Windows/Office + 13 macOS-типов (C-3; `SECURITY.md §4` — A-9); список писем — `truncateEllipsis` до `tview.Escape`, `senderLabel` без Escape (C-4).
- Права `ca_path` — только `platform.CheckFilePerm` (каталог не проверяется); в doctor CA проверяет шаг 2, стартовая проверка при `doctor` пропущена (C-5).
