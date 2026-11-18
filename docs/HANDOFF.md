# HANDOFF — состояние после этапа 5

Перезаписывается в конце каждого этапа: ≤25 строк, новое сверху, закрытое
удаляется. Код — истина, этот файл — навигация.

Сделано: 309332b, 31b956e, 2507572, 3bbff28 — этап 5 (doctor, очистка
вложений, perm-warnings, in-flight + gen-защита пагинации, README).

Дальше: этап 6, задача 6.2 (`internal/notify/watcher.go`). Читать PLAN.md §этап 6
и `internal/notify/rules.go`.

Начато, не закоммичено (рабочее дерево):
- задача 6.1 вчерне готова: `internal/notify/` (rules.go + rules_test.go, новые);
- изменены `internal/security/phishing.go`, `internal/ui/format.go`,
  добавлены `internal/security/policy_test.go`, `internal/platform/notify_test.go`;
- `./scripts/check.sh` красный: `internal/platform/notify_test.go:19` —
  undefined `appleScriptEscape`. До коммита дописать функцию в
  `internal/platform` и получить зелёный прогон.

Открыто для пользователя:
- smoke по TESTING.md §5 (два экрана, n/p, ошибка при выключенном VPN) — живой
  сервер здесь недоступен;
- чекбоксы в PLAN.md не отмечены ни для одного этапа — привести в соответствие.

Не трогать: `internal/ui/links.go` — URL в реестре ссылок показывается
намеренно, экранируется отдельно (SECURITY §3).
