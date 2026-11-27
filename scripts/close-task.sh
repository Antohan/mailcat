#!/bin/sh
# Закрытие задачи: галочка в активном плане + коммит журнала, плана и handoff.
# Аргумент — ID задачи (например V-16); план — из docs/CURRENT.
# Успех: одна строка OK. Провал: причина, exit 1. Код задачи должен быть
# закоммичен заранее: всё незакоммиченное вне трёх файлов — отказ.
set -u
cd "$(dirname "$0")/.." || exit 2

die() { echo "FAIL: $*"; exit 1; }

# WHY: журнальный коммит делается только на dev. Один запуск на main оставил бы
# там запись итерации до релиза; detached HEAD — та же потеря привязки.
# Не делать: проверять «не main» — закрытие на случайной ветке тоже ошибка.
git rev-parse --git-dir >/dev/null 2>&1 || die "не Git-репозиторий — закрывать нечего"
branch=$(git symbolic-ref --quiet --short HEAD) || die "detached HEAD — переключитесь на dev"
[ "$branch" = dev ] || die "ветка «${branch}», а закрывать задачи можно только на dev"

id="${1:-}"
# WHY: ID — заглавные буквы, дефис и цифры, шаблон общий с token-report.py.
# Он попадает в регулярные выражения ниже: с «|» в хвосте поиск по журналу
# совпадал с чужой задачей (V-1|x → V-1).
# Не делать: проверку через case с «*» в конце — тот пропускает любой хвост.
if [ -z "$id" ] || ! printf '%s\n' "$id" | grep -qE '^[A-Z]{1,5}-[0-9]+$'; then
    echo "FAIL: нет ID задачи — например ./scripts/close-task.sh V-16"
    exit 1
fi
plan=$(grep -vE '^[[:space:]]*(#|$)' docs/CURRENT | head -n 1)
if [ -z "$plan" ] || [ "$plan" = none ] || [ ! -f "$plan" ]; then
    echo "FAIL: в docs/CURRENT нет активного плана — закрывать нечего ($id)"
    exit 1
fi
[ -f docs/HANDOFF.md ] || { echo 'FAIL: нет docs/HANDOFF.md'; exit 1; }

# Запись в журнале обязательна: без неё задача не закрыта.
entry=$(head -n 150 CHANGELOG.md | grep -E "^## .*:$id([^0-9]|$)" | head -n 1)
[ -n "$entry" ] || { echo "FAIL: в верхних 150 строках CHANGELOG.md нет записи для $id"; exit 1; }

# Код задачи должен быть закоммичен: грязными могут быть только три файла.
# WHY: код возврата git status проверяется до разбора вывода. Иначе сбой Git
# (lock, повреждённый индекс) давал пустой список «чужих» изменений, и скрипт
# ставил галочку в план, падая лишь на git add.
# Читаемые имена: core.quotePath=false; переименование раскладывается на
# удаление и добавление, чтобы старая и новая ветки пути не прятались в «R».
if ! status=$(git -c core.quotePath=false status --porcelain --no-renames); then
    die "git status завершился с ошибкой — изменения не проверены, ничего не изменено"
fi
other=$(printf '%s\n' "$status" | sed -E 's/^.. //' \
    | grep -vxF -e CHANGELOG.md -e "$plan" -e docs/HANDOFF.md)
[ -z "$other" ] || {
    echo 'FAIL: незакоммичено вне журнала, плана и handoff — сначала закоммитьте код:'
    echo "$other"
    exit 1
}

lines=$(wc -l < docs/HANDOFF.md | tr -d ' ')
[ "$lines" -le 25 ] || { echo "FAIL: docs/HANDOFF.md — $lines строк, лимит 25"; exit 1; }

if grep -qE "^- \[ \] \*\*$id\." "$plan"; then
    # WHY: временный файл — mktemp рядом с планом, чтобы mv был атомарным и на
    # другой файловой системе не копировался; trap убирает его при любом выходе.
    # Замена через awk, а не sed -i — одна запись и на BSD, и на GNU.
    tmp=$(mktemp "$plan.XXXXXX") || die "не удалось создать временный файл рядом с $plan"
    trap 'rm -f "$tmp"' EXIT HUP INT TERM
    awk -v id="$id" '{
        if (!d && $0 ~ ("^- \\[ \\] \\*\\*" id "[.]")) { sub(/^- \[ \]/, "- [x]"); d = 1 }
        print
    }' "$plan" > "$tmp" && mv "$tmp" "$plan" || die "не удалось поставить галочку в $plan"
elif ! grep -qE "^- \[x\] \*\*$id\." "$plan"; then
    echo "FAIL: в $plan нет задачи $id"
    exit 1
fi

iter=$(printf '%s\n' "$entry" | grep -oE 'iter[0-9]+(\.[0-9]+)?' | head -n 1)
git add CHANGELOG.md "$plan" docs/HANDOFF.md || die 'git add завершился с ошибкой'
if git diff --cached --quiet; then
    echo "OK: $id уже закрыт, коммитить нечего"
    exit 0
fi
git commit -q -m "docs: ${iter:-iter} $id — changelog, чекбокс, handoff" || {
    die "git commit не удался (хук или подпись) — причина выше; повторный запуск безопасен: задача с галочкой распознаётся как закрытая"
}
echo "OK: $id закрыт — $(git log -1 --format=%h)"
