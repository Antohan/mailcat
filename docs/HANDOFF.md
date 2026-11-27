# mailcat — HANDOFF

## Дальше
Итерация 4 — `docs/PLAN-ITER4.md`, задача A-9 (доки и инварианты). Читать:
задачу A-9, `docs/PLAN-TEMPLATE.md`, `docs/SECURITY.md §4/§6`, `docs/TESTING.md`;
старт `/stage A-9`.

## Сделано
- 7bd6572 A-8 · ec01223 A-7 · 627653c A-6 · f7137e5 A-5 (iter4).

## Открыто
- Живой smoke, TODO: C-8 — после 401+NTLM нет Basic; C-5 — CA 0644; C-4 — скриншот до/после и «ок»;
  C-3 — `.command`/`.app`/`.dmg`; A-4 — `/stage` на `main`; A-3 — `/release`; A-5/A-6 —
  `env-check.sh` на машине пользователя. «ок» на V-35…V-53 нет; S3–S7 — TODO.

## Не трогать
- `docs/ARCHITECTURE.md §4` целиком; схему аутентификации выбирает сервер, транспорт лишь помнит (`ntlmChosen`), «basic принята» — только при 2xx.
- Вложения — только `security.CreateUnique` (O_EXCL), имя без ведущих точек/пробелов, лимит 10 000, запись — только `writeAttachment` (C-7).
- Скилл EWS повторяет справочник (`item:IsRead` — только в ответе `Default`, CalendarView без `SortOrder`), `policy_test.go` не ослаблять; календарь — тонкие линии 55/45, колонка недель день не выбирает, ширина 3 (V-52).
- Ревьюер и `ews-specialist` без shell (эксперт — без edit/webfetch), diff/stat путями; релиз стоит на КРИТИЧНО/ВАЖНО, M<N не принято.
- `#nosec` — ровно семь мест A-8 с причиной; инструменты ищутся PATH → `$GOBIN` → первый `$(go env GOPATH)/bin`; нет gosec/govulncheck — FAIL с командой установки, не пропуск; `check.sh` отличает синтаксис `gofmt` от неформата, в `go test` видны `panic:`/`fatal error:`.
- close-task.sh: журнал только на dev; сбой git — отказ до записи; temp — mktemp рядом с планом; CURRENT — первая непустая строка без `#`.
- Ошибки doctor — только `security.Sanitize`; ключи конфига по Config; опасные расширения Windows/Office + 13 macOS; список писем — `truncateEllipsis` до `tview.Escape`, `senderLabel` без Escape; `ca_path` — только `platform.CheckFilePerm` (doctor CA — шаг 2).
- `env-check.sh` проверяет только вывод opencode (`x-opencode-directory`, stderr вне JSON), в `check.sh` не входит; редирект https→http покрыт `TestRedirectSchemeDowngradeRejected` — дубль не добавлять.
- `token-report.py`: корень от `__file__`, без `main`/ошибки git — «Отчёт недоступен»; обрыв — по последнему ответу без отрицания; цена — по распознанным задачам.
