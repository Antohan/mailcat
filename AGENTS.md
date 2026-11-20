# AGENTS.md — instructions for the AI agent (mailcat)

You are building **mailcat** — a read-only client for corporate mail and calendar
(on-premises Microsoft Exchange) for macOS: Go + TUI (tview). Work follows the
tasks of the current iteration plan, `docs/PLAN-ITER2.md` (`docs/PLAN.md` is
finished history — do not open or modify it).

**Language.** Everything the user sees is Russian: replies, UI strings, errors,
status lines, code comments. This file and the other agent files are written in
English only to spend fewer tokens — that is not permission to answer in English.

## Where to look (one file at a time)

| File | What it gives you |
| --- | --- |
| `docs/HANDOFF.md` | **start here**: state after the last stage, next task, open items |
| `docs/PLAN-ITER2.md` | **current iteration**: tasks, DoD, the CHANGELOG rule |
| `docs/PLAN.md` | iteration 1 history — do not open in sessions |
| `docs/ARCHITECTURE.md` | packages, data flow; §3 is the list of invariants |
| `docs/EWS-REFERENCE.md` | the only source of truth for SOAP/XML |
| `docs/SECURITY.md` | mandatory security checklist |
| `docs/TESTING.md` | what and how to test; manual smoke in §5 |
| `config.example.toml` | configuration format |

## Golden rules (breaking one means rework)

1. **Read-only.** Allowed EWS operations: `FindFolder`, `FindItem`
   (+`CalendarView`), `GetItem`, `GetAttachment`. Any `Create*`, `Update*`,
   `Delete*`, `Send*`, `Move*`, `Copy*`, `MarkAsRead` — never.
2. **No `InsecureSkipVerify`.** TLS strictly per `docs/SECURITY.md §1`: a
   corporate CA goes only into the client's private `x509.CertPool`. The system
   Keychain is not touched.
3. **All text from EWS is hostile.** Before it reaches the terminal:
   `security.Sanitize()` plus `tview.Escape` for dynamic widgets
   (`docs/SECURITY.md §3`).
4. **Password only from Keychain.** Not in config, logs, errors or dumps.
5. **EWS XML only from `docs/EWS-REFERENCE.md`.** Do not invent elements,
   FieldURIs or attributes. If it is not there, ask the user.
6. **Dependencies — approved list only**: `rivo/tview` (+`tcell`),
   `Azure/go-ntlmssp`, `k3a/html2text`, `x/net/idna`, `BurntSushi/toml`,
   `jhillyerd/enmime`. Anything else: ask first. Stdlib for the rest
   (`encoding/xml`, `crypto/tls`, `net/http`, `log/slog`, `os/exec`).
7. **Russian UI.** Key hints, errors, statuses, folder names («Входящие»,
   «Отправленные»).
8. **Scope is the current iteration plan.** No "for later" features that
   `docs/PLAN-ITER2.md` does not describe.

## Work

1. Open a session with `/stage <task>` (e.g. `/stage UI-1`): it loads the
   iteration plan and the task.
2. All work happens on the `dev` branch. `main` is release-only: it receives
   a finished iteration as a single merge commit — a release action performed
   on explicit user request, never inside a task session.
3. One task = one commit (`iter2:UI-1: short description`) plus its
   CHANGELOG.md entry.
4. Finish with `/check` (`./scripts/check.sh`): gofmt, vet, build, test. Red —
   fix the cause.
5. At the end of a task rewrite `docs/HANDOFF.md` (see below).
6. EWS/SOAP/XML work: consult the `ews-specialist` subagent before
   implementing. Security-relevant work: request `security-reviewer` after —
   once per stage.
7. New tasks enter an iteration plan only as specs — touched files/symbols and
   expected behavior written BEFORE the coding session. A task without
   pointers to code costs exploration tokens on every session that opens it.
8. Iteration finished and the user asks for a release → `/release`: it checks
   the DoD, runs the security review over the iteration range and merges
   `dev → main` only after the user explicitly confirms.

## Token budget rules (not optional)

**One session = one task.** After the commit, write a report of at most five
lines and say the task is closed: the next one starts in a new session. Do not
start a second task, do not fix unrelated things you noticed on the way.

**Circuit breaker.** If the same test, build error or command fails three
times in a row, stop: write the ≤5-line report (what you tried, what you
suspect), close the session, and leave the task in `HANDOFF.md → Дальше`
marked «застряло: …». The fourth attempt almost never fixes it but always
costs a window.

**One call instead of five.** Batch independent commands into a single shell
call; filter output (`2>&1 | tail -20`, `-q`, `--stat`). Run `git diff --stat`
first, then read the specific hunks. Never print a whole file.

**Read narrowly.** Locate a symbol with `grep` instead of reading a file. Files
over ~150 lines: use `offset`/`limit`. Do not open `*_test.go` files you are not
editing. Docs: only the referenced section (`SECURITY.md §3`). While iterating
use `go test -run <Test>`; the full suite runs once, in `/check`.
`.superpowers/sdd/PLAN/progress.md` is an archive — never read it at session
start.

**Do not restate.** No file contents, no doc summaries, no play-by-play of
commands. The report is: what changed, what was verified, what is next.

**Subagents.** Give them a pointer, not a dump («прочитай §5–6 EWS-REFERENCE»).
Ask for a verdict of at most ten lines. A subagent that reads everything costs
more than doing the work yourself. Spawn one only if the task touches EWS/XML
(`ews-specialist`) or security (`security-reviewer`) — never for routine edits.
Security review runs on Flash (the checklist is mechanical), once per
stage/iteration over a commit range, not per task; `ews-specialist` stays on
the flagship.

**Stable prefix = cache hits.** Do not edit, inside a coding session, any file
already read earlier in that session — `AGENTS.md`, `SECURITY.md`, plans,
`HANDOFF.md` included. An edit invalidates the cached prompt prefix and the
rest of the session bills at full price; doc edits go to a separate session.
Open sessions identically (HANDOFF first) and never paste large texts into
coding-session prompts.

**No discussions in coding sessions.** Questions, ideas and planning go to a
separate conversation; a coding session receives a task and returns a report.
A ten-message discussion inside a session re-bills its whole context ten
times.

**Models.** Flash by default; flagship (via `/models`) for the `ews-specialist`
subagent and for iteration tasks where layout or link-registry design matters
(UI-2, UI-4). `security-reviewer` runs on Flash on purpose: its checklist is
mechanical.

## Handoff file

`docs/HANDOFF.md` is the next session's entry point: at most 25 lines, rewritten
(never appended), newest on top, finished lines deleted.

Fields: `Сделано` (commit hashes only) / `Дальше` (task and exactly what to
read) / `Открыто` (smoke checks and TODOs for the user) / `Не трогать`
(invariants of the current stage, in triplet form — see below).

The code is the truth, the handoff is navigation. When they disagree, trust the
code and delete the stale line — do not repair it.

## Environment (not fixable in code)

- Exchange is reachable only while the user has VPN on — network errors must
  hint at it («Похоже, VPN выключен»).
- The user is not an Exchange admin: treat `ErrorAccessDenied` and 403 as
  "ask IT", never work around them.
- macOS: open files and links with `/usr/bin/open`; Keychain through `security`.

The iteration is done when every DoD point of `docs/PLAN-ITER2.md` is met.
Live-server checks are unavailable here: mark them as TODO for the user
(`TESTING.md §5`).

## Decisions and invariants

A non-obvious choice gets recorded where the next person would try to "simplify"
it — an inline comment in the code, three lines:

```go
// WHY: response is discarded by generation, not by folder name.
// Why: a late reply from the previous folder overwrites the loaded list.
// Not to do: compare folder names — after A→B→A the old reply looks fresh.
```

A decision made during a stage also gets one line in `HANDOFF.md` → `Не трогать`.
Only what survives its stage moves into `docs/ARCHITECTURE.md §3`; the rest is
deleted. Never read §3 or the handoff as prose — use them to check the invariant
you are about to break. Every finding from `security-reviewer` becomes one such
invariant line, otherwise the same mistake comes back next stage.
