#!/bin/sh
# Проверка среды агента: агенты, команды, скиллы, плагин, steps, docs/CURRENT.
# Ловит то, что ломается без ошибки: агент с нерезолвящейся моделью молча
# исчезает из списка, скилл или плагин не загружается молча.
# Формат: строка на проверку (OK / FAIL: …); код выхода 1 при провале.
set -u
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd) || exit 2
cd "$root" || exit 2

failed=0
fail_line() {
    printf 'FAIL: %s\n' "$1"
    failed=1
}

# WHY: stderr CLI отделён от JSON-потока: предупреждение в stdout сломало бы
# разбор, а traceback python не дал бы строки FAIL. На сбое печатается его
# первая строка.
errfile=$(mktemp) || exit 2
trap 'rm -f "$errfile"' EXIT HUP INT TERM

# WHY: сервис opencode без заголовка x-opencode-directory отвечает в контексте
# домашнего каталога — глобальные скиллы находятся, проектные и команды нет.
# Не делать: полагаться на текущий каталог процесса — CLI его не передаёт.
api() {
    opencode api get "$1" --header "x-opencode-directory: $root" 2>"$errfile"
}

if ! command -v opencode >/dev/null 2>&1; then
    fail_line "opencode не найден в PATH"
    exit 1
fi

if agents=$(opencode debug agents 2>"$errfile"); then
    printf '%s' "$agents" | python3 -c '
import json, sys
failed = False

def check(name, ok, detail=""):
    global failed
    if ok:
        print("OK: %s" % name)
    else:
        failed = True
        print("FAIL: %s" % name + (" — %s" % detail if detail else ""))

def last_effect(agent, action):
    # Последнее совпавшее правило побеждает: user-config дописывает deny в конец.
    effect = None
    for p in agent.get("permissions", []):
        if p.get("action") in (action, "*") and p.get("resource") == "*":
            effect = p.get("effect")
    return effect

try:
    agents = {a["id"]: a for a in json.load(sys.stdin)}
except (ValueError, TypeError) as e:
    print("FAIL: ответ opencode debug agents — неожиданный JSON: %s" % e)
    sys.exit(1)
gi = agents.get("go-implementer")
check("агент go-implementer загружен", gi is not None)
if gi:
    model = gi.get("model") or {}
    check("модель go-implementer задана", bool(model.get("id") and model.get("providerID")),
          "без providerID модель не резолвится")
    steps = gi.get("steps")
    check("steps go-implementer >= 120", isinstance(steps, int) and steps >= 120,
          "steps=%s" % steps)
check("агент ews-specialist загружен", "ews-specialist" in agents)
sr = agents.get("security-reviewer")
check("агент security-reviewer загружен", sr is not None)
if sr:
    eff = last_effect(sr, "edit")
    check("security-reviewer без права правки", eff == "deny",
          "эффективное право edit: %s" % eff)
# shortcut: правку ews-specialist проверяет A-6, после её запрета добавить сюда,
# как у security-reviewer — сегодня консультант править ещё может.

sys.exit(1 if failed else 0)
' || failed=1
else
    fail_line "opencode debug agents — $(head -n 1 "$errfile")"
fi

if commands=$(api /api/command); then
    printf '%s' "$commands" | python3 -c '
import json, sys
failed = False
try:
    names = {c["name"] for c in json.load(sys.stdin)["data"]}
except (ValueError, KeyError, TypeError) as e:
    print("FAIL: ответ opencode /api/command — неожиданный JSON: %s" % e)
    sys.exit(1)
for name in ("check", "release", "stage"):
    if name in names:
        print("OK: команда %s" % name)
    else:
        failed = True
        print("FAIL: команда %s не загружена" % name)
sys.exit(1 if failed else 0)
' || failed=1
else
    fail_line "opencode api get /api/command — $(head -n 1 "$errfile")"
fi

if skills=$(api /api/skill); then
    printf '%s' "$skills" | python3 -c '
import json, os, sys
failed = False
root = sys.argv[1]
try:
    data = {s["id"]: s for s in json.load(sys.stdin)["data"]}
except (ValueError, KeyError, TypeError) as e:
    print("FAIL: ответ opencode /api/skill — неожиданный JSON: %s" % e)
    sys.exit(1)
ews = data.get("ews-reference")
if ews is None:
    failed = True
    print("FAIL: скилл ews-reference не загружен")
else:
    path = os.path.abspath(ews.get("path", ""))
    if path.startswith(root + os.sep):
        print("OK: скилл ews-reference (из репозитория)")
    else:
        failed = True
        print("FAIL: скилл ews-reference не из репозитория — %s" % path)
for name in ("tui-fundamentals", "golang-tui"):
    if name in data:
        print("OK: скилл %s (глобальный)" % name)
    else:
        failed = True
        print("FAIL: глобальный скилл %s не загружен" % name)
sys.exit(1 if failed else 0)
' "$root" || failed=1
else
    fail_line "opencode api get /api/skill — $(head -n 1 "$errfile")"
fi

if plugins=$(api /api/plugin); then
    printf '%s' "$plugins" | python3 -c '
import json, sys
failed = False
try:
    data = {p["id"]: p for p in json.load(sys.stdin)["data"]}
except (ValueError, KeyError, TypeError) as e:
    print("FAIL: ответ opencode /api/plugin — неожиданный JSON: %s" % e)
    sys.exit(1)
sp = data.get("superpowers")
if sp is None:
    failed = True
    print("FAIL: плагин superpowers не загружен")
elif (sp.get("state") or {}).get("status") != "active":
    failed = True
    print("FAIL: плагин superpowers не активен — %s"
          % (sp.get("state") or {}).get("status"))
else:
    print("OK: плагин superpowers")
sys.exit(1 if failed else 0)
' || failed=1
else
    fail_line "opencode api get /api/plugin — $(head -n 1 "$errfile")"
fi

plan=$(grep -vE '^[[:space:]]*(#|$)' docs/CURRENT 2>/dev/null | head -n 1)
if [ "$plan" = none ]; then
    echo "OK: docs/CURRENT — none"
elif [ -n "$plan" ] && [ -f "$plan" ]; then
    echo "OK: docs/CURRENT — $plan"
else
    fail_line "docs/CURRENT не называет существующий файл: ${plan:-пусто}"
fi

exit "$failed"
