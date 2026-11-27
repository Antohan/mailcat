# mailcat — HANDOFF

## Дальше
Итерация 4 — `docs/PLAN-ITER4.md`, задача A-5 (среда агента: `scripts/env-check.sh`).
Читать: задачу A-5, `.opencode/agents/*.md`, `opencode.json`, `docs/PLAN-ITER4.md` A-5; старт `/stage A-5`.

## Сделано
- a69840d — iter4:C-8: транспорт запоминает NTLM, лог схемы только при успехе.
- b400340 — iter4:C-7: частичный файл вложения удаляется при сбое записи.

## Открыто
- Живой smoke, TODO: C-8 — реальный Exchange: после первого 401+NTLM дальнейшие запросы идут без Basic; C-5 — CA 0644 даёт предупреждение; C-4 — скриншот до/после (`[red[…` → `[red]…`), «ок» пользователя; C-3 — `.command`/`.app`/`.dmg`; A-4 — `/stage` на `main`; A-3 — `/release`. «ок» на V-35…V-53 нет; S3–S7 — TODO.

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схему аутентификации выбирает сервер, после C-8 транспорт лишь её помнит (`ntlmChosen`, `atomic.Bool`), память по ошибке NTLM не сбрасывается; «схема basic принята» — лог только при 2xx.
- Вложения — только через `security.CreateUnique` (O_EXCL); имя — без ведущих точек/пробелов (`TrimLeft(". \t")`), лимит 10 000.
- Запись вложения — только `writeAttachment`: частичный файл удаляется после Close и лишь при совпадении `Name()` писателя с путём (C-7).
- Скилл EWS: `item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`; `policy_test.go` не ослаблять.
- Ревьюер без shell: diff и stat путями; `#nosec` — только семь мест A-8; релиз стоит на `КРИТИЧНО`/`ВАЖНО`, M < N — не принято.
- Сетка календаря — тонкие линии 55/45; клик по колонке недель день не выбирает, ширина 3 (V-52).
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; порядок шагов задачи один, CURRENT — первая непустая строка без `#`.
- Вывод ошибок doctor — только `security.Sanitize(err.Error())`; ключи конфига — строго по Config, `org_domain` допустим, 500/380 не поднимать.
- Опасные расширения — Windows/Office + 13 macOS-типов (C-3; `SECURITY.md §4` — A-9); список писем — `truncateEllipsis` до `tview.Escape`, `senderLabel` без Escape (C-4).
- Права `ca_path` — только `platform.CheckFilePerm` (каталог не проверяется); в doctor CA проверяет шаг 2, стартовая проверка при `doctor` пропущена (C-5).
- Редирект https→http на том же хосте уже покрыт `TestRedirectSchemeDowngradeRejected` (`review1_fix_test.go`) — дубль не добавлять.
