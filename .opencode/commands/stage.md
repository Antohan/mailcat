---
description: Загрузить задачу из активного плана итерации и сразу приступить
agent: go-implementer
---

Read `docs/CURRENT` — the active plan's path on its first line. If it is
`none`, say there is no active plan and stop: the next iteration is composed
per `docs/PLAN-TEMPLATE.md`, not here. Then read `docs/HANDOFF.md` (state,
next task) and ONLY the task $ARGUMENTS («V-1» or «next unchecked») in that
plan. Every other plan in `docs/` is finished history: do not open them, do
not add tasks there.

1. Check the task against the actual code with a quick `grep` inside the
   package — never a repo-wide read.
2. If the task is fully described in the plan, implement it right away (rules
   in AGENTS.md, including "Token budget rules" and "Code minimalism").
3. Ask the user only on real ambiguity.
4. **Visual task → screenshot before commit**: render the changed screen and
   attach before/after screenshots to the report; the next visual task does
   not start without the user's «ок» on the previous one.
5. Close the task in this order: `/check` → commit the code
   `<iter>:<ID>: <done>` (prefix from the plan) → add the CHANGELOG.md entry at
   the top (format: the plan, «Журнал решений») → rewrite `docs/HANDOFF.md`
   (≤25 lines, «Дальше» = the next task) → `./scripts/close-task.sh <ID>`. The
   script ticks the checkbox and commits the journal, the plan and the
   handoff; do not tick or commit these three by hand.
6. Report in at most five lines: what changed, what was verified, what is
   next. Then say the task is closed and the next one needs a new session —
   and stop.
