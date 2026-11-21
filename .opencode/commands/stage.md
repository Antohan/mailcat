---
description: Загрузить задачу из текущего плана итерации и сразу приступить
agent: go-implementer
---

Read `docs/HANDOFF.md` first (state, next task), then ONLY the task $ARGUMENTS
(«V-1» or «next unchecked») in `docs/PLAN-ITER3.1.md` — the current iteration.
`docs/PLAN.md`, `docs/PLAN-ITER2.md`, `docs/PLAN-ITER3.md` are finished
history: do not open them, do not add tasks there.

1. Check the task against the actual code with a quick `grep` inside the
   package — never a repo-wide read.
2. If the task is fully described in the plan, implement it right away (rules
   in AGENTS.md, including "Token budget rules" and "Code minimalism").
3. Ask the user only on real ambiguity.
4. **Visual task → screenshot before commit**: render the changed screen and
   attach before/after screenshots to the report; the next visual task does
   not start without the user's «ок» on the previous one.
5. Close the task: `/check` → commit `iter31:V-N: <done>` → CHANGELOG.md entry
   (format: PLAN-ITER3.1.md, «Журнал решений») → tick the checkbox → rewrite
   `docs/HANDOFF.md` (≤25 lines, «Дальше» = the next iteration task).
6. Report in at most five lines: what changed, what was verified, what is
   next. Then say the task is closed and the next one needs a new session —
   and stop.
