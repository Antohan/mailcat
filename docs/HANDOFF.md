# mailcat — HANDOFF

## Дальше
Все задачи `docs/PLAN-ITER4.md` закрыты (A-9 — `eff88e9`). Следующий шаг — `/release`
по команде пользователя: DoD плана, ревью диапазона, merge `dev → main`.

## Сделано
- eff88e9 A-9 · 7bd6572 A-8 · ec01223 A-7 · 627653c A-6 · f7137e5 A-5 (iter4).

## Открыто
- Живой smoke, TODO: C-8 — после 401+NTLM нет Basic; C-5 — CA 0644; C-4 — скриншот до/после и «ок»;
  C-3 — `.command`/`.app`/`.dmg`; A-4 — `/stage` на `main`; A-3 — `/release`; A-5/A-6 —
  `env-check.sh` на машине пользователя. «ок» на V-35…V-53 нет; S3–S7 — TODO.
- A-9 — только документы, живой проверки не требует.

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схема аутентификации: выбирает сервер, транспорт лишь помнит (`ntlmChosen`), «basic принята» — только при 2xx.
- Вложения — только `security.CreateUnique` (O_EXCL), имя без ведущих точек/пробелов, лимит 10 000, запись только `writeAttachment` (C-7), недописанный файл удаляется.
- Усечение списка — `truncateEllipsis` до `tview.Escape` (C-4); скилл EWS повторяет справочник (`item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`), `policy_test.go` не ослаблять; календарь — тонкие линии 55/45, колонка недель день не выбирает, ширина 3 (V-52).
- Опасные расширения — список в `SECURITY.md §4` и `dangerousExtensions`; проверяется последнее расширение.
- Ревьюер и `ews-specialist` без shell (эксперт — без edit/webfetch); модель эксперта — `zai/glm-5.3`, ревьюер наследует сессию.
- `#nosec` — ровно семь мест A-8 с причиной; инструменты ищутся PATH → `$GOBIN` → первый `$(go env GOPATH)/bin`; нет gosec/govulncheck — FAIL с командой установки.
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; CURRENT — первая непустая строка без `#`.
- Ошибки doctor — только `security.Sanitize`; ключи конфига по Config; `ca_path` — `platform.CheckFilePerm` (doctor — шаг 2); `env-check.sh` вне `check.sh`; числа `token-report.py` — в §2 AI-AGENT-PRACTICES.
