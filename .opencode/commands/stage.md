---
description: Загрузить задачу этапа N и сразу приступить к реализации
---

Read `docs/HANDOFF.md` first (state, next task, open items), then ONLY the
section of stage $ARGUMENTS in `docs/PLAN.md`. Do not open other documents.

1. Take the first unchecked task of the stage; check it against the actual code
   with a quick `grep`/`ls` inside the package — never a repo-wide read.
2. If the task is fully described in PLAN.md, implement it right away (rules in
   AGENTS.md, including "Token budget rules").
3. Ask the user only on real ambiguity.
4. Close the task: `/check` → commit `stage<N>: <done>` → tick the checkbox in
   PLAN.md → rewrite `docs/HANDOFF.md` (≤25 lines, newest on top).
5. Report in at most five lines: what changed, what was verified, what is next.
   Then say that the task is closed and the next one needs a new session — and
   stop.
