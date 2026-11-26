---
description: Release procedure — verify iteration DoD, security review, usage report, merge dev→main after user confirmation
agent: go-implementer
---

You are running the release procedure for the current iteration. The release
is a human decision: your job is the checklist and the mechanics; the merge
happens only after the user explicitly confirms. If any precondition fails —
stop and report; do not fix things on your own inside a release session.

1. Preconditions (all must hold, otherwise stop and report what is missing):
   - `docs/CURRENT` names an active plan (not `none`); branch is `dev`, working
     tree clean (`git status`);
   - every task checkbox in that plan is ticked and every DoD point of the
     plan is met;
   - `/check` (`./scripts/check.sh`) is green;
   - `CHANGELOG.md` has an entry for every task of the plan.
2. The manual smoke scenarios (`docs/TESTING.md §5`) need VPN and the live
   server — ask the user whether they have been run for this iteration and
   wait for the answer. Do not proceed on an assumption.
3. Spawn `security-reviewer` over the iteration range: from the last release
   (`git log main -1 --format=%h`) to `dev`. A verdict containing КРИТИЧНО
   findings stops the release. If the subagent is not in this session's list,
   run the same review through a general subagent — give it pointers only
   (`docs/SECURITY.md`, invariants in `HANDOFF.md → Не трогать`), require
   read-only behaviour and the ≤10-line verdict, and mark the final report
   «ревью деградировано: general вместо security-reviewer».
4. Run `./scripts/token-report.py` and keep its three headline numbers for the
   final report: sessions cut by the step limit, sessions per task, cost per
   task. «Отчёт недоступен» is reported as is — it does not block the release.
5. Ask: «Релизим? Merge dev → main». Only on an explicit «да», on `dev`:
   - mark the plan header «ВЫПОЛНЕНА» (the release is the `Release: <итерация>`
     merge commit on `main`);
   - set the first line of `docs/CURRENT` to `none`, or to the next plan if
     the user names one;
   - rewrite `docs/HANDOFF.md`: «Сделано» = the released iteration, «Дальше» =
     the next plan or «план не составлен», keep «Открыто» and «Не трогать»;
   - commit `docs: релиз <итерация> — план заморожен, CURRENT=none`;
   - `git checkout main` → `git merge --no-ff dev -m "Release: <итерация> —
     <тема>"` → `git switch dev`. Checkout and merge are permission-guarded
     (ask) — the prompts are the confirmation. Never push: pushing is the
     user's action. Tagging (`git tag -a v0.X.0`) is optional and only on
     explicit request.
6. Report in at most five lines: what was released, the merge hash, the three
   numbers from step 4, what is next. The session ends here.
