---
description: Загрузить задачу из активного плана итерации и сразу приступить
agent: go-implementer
---

Read `docs/CURRENT` — the active plan's path on its first non-comment line
(`#` lines and blanks are skipped, as `close-task.sh` parses it). If it is
`none`, say there is no active plan and stop: the next iteration is composed
per `docs/PLAN-TEMPLATE.md`, not here.

Preconditions — check before anything else. Any failure: say what failed and
stop without touching the tree.
- the branch is `dev`, the working tree is clean (`git status`);
- the plan exists and the task $ARGUMENTS («V-1» or «next unchecked») has
  exactly one spec in it — zero or several → stop;
- every other plan in `docs/` is finished history: do not open them, do not add
  tasks there.

Then read `docs/HANDOFF.md` (state, next task) and ONLY that one spec.

1. Check the task against the actual code with a quick `grep` inside the
   package — never a repo-wide read.
2. EWS/SOAP/XML work → consult the `ews-specialist` subagent before
   implementing.
3. If the task is fully described in the plan, implement it right away (rules
   in AGENTS.md, including "Token budget rules" and "Code minimalism").
4. Ask the user only on real ambiguity.
5. **Visual task → screenshot before commit**: render the changed screen and
   attach before/after screenshots to the report; the next visual task does
   not start without the user's «ок» on the previous one.
6. Close the task in this order: `/check` → security task: `security-reviewer`
   reviews the task's diff before the commit, fix the findings, re-run
   `/check` → commit the code `<iter>:<ID>: <done>` (prefix from the plan) →
   add the CHANGELOG.md entry at the top (format: the plan, «Журнал решений»)
   → rewrite `docs/HANDOFF.md` (≤25 lines, «Дальше» = the next task) →
   `./scripts/close-task.sh <ID>`. The script ticks the checkbox and commits
   the journal, the plan and the handoff; do not tick or commit these three by
   hand.
7. Report in at most five lines: what changed, what was verified, what is
   next. Then say the task is closed and the next one needs a new session —
   and stop.
