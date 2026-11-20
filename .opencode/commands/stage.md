---
description: Загрузить задачу из текущего плана итерации и сразу приступить
---

Read `docs/HANDOFF.md` first (state, next task), then ONLY the task $ARGUMENTS
(«RD-1» or «next unchecked») in `docs/PLAN-ITER3.md` — the current iteration.
`docs/PLAN.md` and `docs/PLAN-ITER2.md` are finished history: do not open them,
do not add tasks there.

1. Check the task against the actual code with a quick `grep` inside the
   package — never a repo-wide read.
2. If the task is fully described in the plan, implement it right away (rules
   in AGENTS.md, including "Token budget rules").
3. Ask the user only on real ambiguity.
4. Close the task: `/check` → commit `iter3:RD-N: <done>` → CHANGELOG.md entry
   (format: PLAN-ITER3.md, «Журнал решений») → tick the checkbox → rewrite
   `docs/HANDOFF.md` (≤25 lines, «Дальше» = the next iteration task).
5. Report in at most five lines: what changed, what was verified, what is
   next. Then say the task is closed and the next one needs a new session —
   and stop.
