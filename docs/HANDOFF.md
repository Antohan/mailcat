# mailcat — HANDOFF

## Дальше
План не составлен — следующая итерация планируется по `docs/PLAN-TEMPLATE.md`, старт — `/stage <ID>`.

## Сделано
- Релиз итерации 4: merge `Release: итерация 4` в `main` (iter4: A-1…A-9, C-1…C-8; C-9 — вне плана `6c7c1f2`).

## Открыто
- Нет открытых пунктов: smoke S1–S7 и визуал V-35…V-53 итерации 4 подтверждены пользователем.
- Ревью iter4: 4 МЕЛКО не блокировали (класс C-1 и права скриптов); правила — в «Не трогать».

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схема аутентификации: выбирает сервер, транспорт лишь помнит (`ntlmChosen`), «basic принята» — только при 2xx.
- Вложения — только `security.CreateUnique` (O_EXCL), имя без ведущих точек/пробелов, лимит 10 000, запись только `writeAttachment` (C-7), недописанный файл удаляется.
- Усечение списка — `truncateEllipsis` до `tview.Escape` (C-4); скилл EWS повторяет справочник (`item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`), `policy_test.go` не ослаблять; календарь — тонкие линии 55/45, колонка недель день не выбирает, ширина 3 (V-52).
- Опасные расширения — список в `SECURITY.md §4` и `dangerousExtensions`; проверяется последнее расширение.
- Ревьюер и `ews-specialist` без shell (эксперт — без edit/webfetch); модель эксперта — `zai/glm-5.3`, ревьюер наследует сессию.
- `#nosec` — ровно семь мест A-8 с причиной; инструменты ищутся PATH → `$GOBIN` → первый `$(go env GOPATH)/bin`; нет gosec/govulncheck — FAIL с командой установки.
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; CURRENT — первая непустая строка без `#`.
- Ошибки doctor — только `security.Sanitize`; ключи конфига по Config; `ca_path` — `platform.CheckFilePerm` (doctor — шаг 2); `env-check.sh` вне `check.sh`; числа `token-report.py` — в §2 AI-AGENT-PRACTICES.
- Ревью iter4: в `opencode.json` права скриптов без `*` (glob авто-подтверждает цепочку); имена ключей конфига и локальные значения `doctor` — через `Sanitize`; `env-check.sh` проверяет у `security-reviewer` и deny `shell`.
