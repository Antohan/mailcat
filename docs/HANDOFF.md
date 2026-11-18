# HANDOFF — после этапа 5

≤25 строк, новое сверху, закрытое удаляется. Код — истина, файл — навигация.
«Не трогать» — инварианты этапа; выжившие уходят в ARCHITECTURE.md §3.

Сделано: 309332b, 31b956e, 2507572, 3bbff28 — этап 5 (doctor, очистка
вложений, perm-warnings, gen + in-flight пагинация, README).

Дальше: 6.2 `internal/notify/watcher.go` — PLAN.md §этап 6, `rules.go`.

Начато, не закоммичено: 6.1 вчерне (`internal/notify/`); изменены
security/phishing.go, ui/format.go; добавлены security/policy_test.go,
platform/notify_test.go. check.sh красный: notify_test.go:19 — undefined
appleScriptEscape; дописать функцию и получить зелёный прогон.

Открыто: smoke TESTING.md §5 (2 экрана, n/p, VPN); чекбоксы PLAN.md не отмечены.

Не трогать:
- links.go — URL показывается целиком: видно, куда идёт (SECURITY §4.2).
  Не делать: обрезать — «показанный ≠ открываемый» и есть фишинг.
- mail.go (`gen`) — ответ отбрасывается по поколению: поздний ответ прежней
  папки затирает список. Не делать: сравнивать имя папки — при A→B→A
  старый ответ станет «свежим».
