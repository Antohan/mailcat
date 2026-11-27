# AGENTS.md — instructions for the AI agent (mailcat)

You are building **mailcat** — a read-only client for corporate mail and calendar
(on-premises Microsoft Exchange) for macOS: Go + TUI (tview). Work follows the
tasks of the current iteration plan — the path on the first non-comment line of
`docs/CURRENT` (`#` lines and blanks are skipped, as `close-task.sh` parses it).
`none` there means no active plan: say so and stop. Every other
plan in `docs/` is finished history — do not open or modify it.

**Language.** Everything the user sees is Russian: replies, UI strings, errors,
status lines, code comments. This file and the other agent files are written in
English by team choice — a measurement did not confirm token savings. That is
not permission to answer in English.

## Where to look (one file at a time)

| File | What it gives you |
| --- | --- |
| `docs/HANDOFF.md` | **start here**: state after the last stage, next task, open items |
| `docs/CURRENT` | path of the active iteration plan (one line; `none` = no plan) |
| the plan it names | **current iteration**: tasks, DoD, commit prefix, the CHANGELOG rule |
| `docs/PLAN-TEMPLATE.md` | mandatory format for composing a new iteration plan |
| `docs/ARCHITECTURE.md` | packages, UI skeleton, data flow; §4 is the list of invariants |
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
8. **Scope is the current iteration plan.** No "for later" features that the
   current iteration plan does not describe.

## Code minimalism

Write the smallest complete change (adopted from the ponytail ruleset, MIT):

- Ladder, in order: already in the codebase? → stdlib/platform feature? →
  installed dependency (approved list only, rule 6)? → minimum code that works.
  Never add a dependency for a few lines.
- No unrequested features, abstractions, wrappers, config or "for later" code —
  name what you skipped in one line instead.
- The shortest working diff wins, but it must be complete: update all callers,
  tests and fixtures the change touches; a bug fix goes to the root cause in
  shared code, not to each symptom.
- New non-trivial logic (branches, loops, parsers) needs one small test;
  trivial changes don't.
- A deliberate limitation gets a code comment: `shortcut: <the limit>, <when
  to upgrade>` — and one line in `HANDOFF.md → Открыто`.
- Never cut: validation, error handling against data loss, security,
  accessibility, anything the task explicitly asks for.

## UX/UI work: skills

UX/UI work — layout, panels, palette, typography, key hints, the calendar, any
visual change — starts with the `tui-fundamentals` skill: it is the main TUI/UX
skill (architecture, terminal constraints, keyboard-first flows, review). Load
it before writing the task's plan spec (Work, item 7); the spec names the
framework primitives and terminal constraints, not browser-style mockups.

`golang-tui` is a supplementary technical reference: use it for terminal
standards, width/truncation math, and testing/observing techniques where they
apply — its Charm/Bubble Tea prescriptions do not override this repo's stack:
mailcat stays on tview/tcell (golden rule 6).

## Terminal limits (requirement)

When developing the terminal interface, account for the real limitations of
the macOS terminal: ANSI escape sequences, colour capabilities (256 colours vs
truecolor, light/dark themes), Unicode and character widths (wide East Asian
characters, combining marks, emoji), and the character-cell grid (one cell =
one character; no pixel layout, no font size or scaling, no browser-style
hover/animation). Do not design the interface as if it were a browser. Use the
capabilities of the existing TUI framework (tview/tcell: widgets, styles, box
drawing, focus model, layout primitives); do not propose migrating to another
framework without strong reasons (golden rule 6 still applies).

## Work

1. Open a session with `/stage <task>` (e.g. `/stage UI-1`). Entry checks, in
   this order: `docs/CURRENT` names an active plan, the branch is `dev` with a
   clean tree (`git status`), the task has exactly one spec — any failure stops
   the session without touching the tree.
2. EWS/SOAP/XML work: consult the `ews-specialist` subagent before
   implementing. Implement, then run `/check` (`./scripts/check.sh`): gofmt,
   vet, build, test. Red — fix the cause.
3. Security task: `security-reviewer` reviews the task's own diff before the
   commit; fix the findings and re-run `/check`.
4. One task = one code commit (`<iter>:<ID>: short description`; the prefix
   is in the plan) plus a CHANGELOG.md entry. Then rewrite `docs/HANDOFF.md`
   and close the task with `./scripts/close-task.sh <ID>`: it ticks the
   checkbox in the plan and commits the journal, the plan and the handoff.
   Never tick the checkbox or commit these three by hand.
5. The order above is one sequence — entrance, consultation, implementation,
   `/check`, security review, code commit, journal, handoff, close — no step
   moves.
6. All work happens on the `dev` branch. `main` is release-only: it receives
   a finished iteration as a single merge commit — a release action performed
   on explicit user request, never inside a task session.
7. New tasks enter an iteration plan only as specs — touched files/symbols and
   expected behavior written BEFORE the coding session, in the fixed format of
   `docs/PLAN-TEMPLATE.md`. A task without pointers to code costs exploration
   tokens on every session that opens it.
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

**Do not restate.** No file contents, no doc summaries, no play-by-play of
commands. The report is: what changed, what was verified, what is next.

**Subagents.** Give them a pointer, not a dump («прочитай §5–6 EWS-REFERENCE»).
Ask for a verdict of at most ten lines. A subagent that reads everything costs
more than doing the work yourself. Spawn one only if the task touches EWS/XML
(`ews-specialist`) or security (`security-reviewer`) — never for routine edits.
The security reviewer inherits the session model: a security task's diff is
reviewed before its commit, an iteration gets one review of its commit range at
release — not one per task; `ews-specialist` stays on its pinned model.

**Stable prefix = cache hits.** Do not edit, in the middle of a coding
session, a file read at its start — `AGENTS.md`, `SECURITY.md`, references
included: an edit invalidates the cached prefix and the rest of the session
bills at full price; such edits go to a separate session. The plan and
`HANDOFF.md` are written once, at the close step. Open sessions identically
(HANDOFF first) and never paste large texts into coding-session prompts.

**No discussions in coding sessions.** Questions, ideas and planning go to a
separate conversation; a coding session receives a task and returns a report.
A ten-message discussion inside a session re-bills its whole context ten
times.

**Models.** Flash by default — pinned as `deepseek/deepseek-flash` (deepseek
provider, pay-as-you-go: авторизуйте `opencode auth login` → DeepSeek до
первой сессии). The provider prefix is mandatory: a bare id does not resolve
and the agent silently disappears from lists — observed. `ews-specialist` is
the only deliberate exception: it pins the flagship `zai/glm-5.3`, because the
EWS XML schema is the riskiest part of the project. Every other subagent
(`security-reviewer` included) inherits the session model — do not pin a
`model:` in its file (same silent-disappearance failure mode). Flagship in the
main session only on an explicit user decision (layout-heavy tasks).

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

The iteration is done when every DoD point of the current iteration plan is met.
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
Only what survives its stage moves into `docs/ARCHITECTURE.md §4`; the rest is
deleted. Never read §3 or the handoff as prose — use them to check the invariant
you are about to break. Every finding from `security-reviewer` becomes one such
invariant line, otherwise the same mistake comes back next stage.
