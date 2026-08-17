---
name: "e2e-template-testing"
description: "End-to-end validation of coordinator and agent template changes"
domain: "development"
confidence: "high"
source: "manual"
---

## Context

Squad's coordinator prompt (`squad.agent.md`) and agent charters (e.g.
`scribe-charter.md`) are shipped as templates in `.squad-templates/`. Changes to
these files affect how every squad session behaves — but unit tests can't catch
prompt-level regressions because the prompts are interpreted by an LLM at
runtime.

This skill describes how to validate template changes end-to-end by running real
squad sessions against a locally-built CLI that includes your modified templates.

## When To Use

- You changed `.squad-templates/squad.agent.md` (coordinator prompt)
- You changed `.squad-templates/scribe-charter.md` or other agent charters
- You changed `.squad-templates/notes-protocol.md` or helper scripts
- You added new conditional blocks (e.g. state-backend-aware spawn templates)
- You modified the init scaffolding that writes templates to target repos

## Prerequisites

- **Node.js** ≥20, **npm** ≥10
- **Git** CLI
- **GitHub Copilot CLI** (`copilot` or `ghcs`) installed
- A local clone of the squad repo on your feature branch

## Workflow

### Step 0 — Post initial tracking comment (**FIRST action — before anything else**)

If `PR_NUMBER` and `REPO` are both set, **the absolute first thing you do** — before
fast-fail checks, before building, before creating any repos — is post the initial
tracking comment with all steps marked as `:hourglass_flowing_sand: Pending`.

This gives reviewers immediate visibility that a run is in progress and what to expect.

```powershell
$runStart = Get-Date
$body = @"
## E2E Progress - PR $env:PR_NUMBER

| Step | Status | Started | Duration |
|---|---|---|---|
| 1. Fast-fail checks (build :cd: link :cd: ``squad version``) | :hourglass_flowing_sand: Pending | --:-- | -- |
| 2. Create test repo(s) | :hourglass_flowing_sand: Pending | --:-- | -- |
| 3. ``squad init`` + file verification | :hourglass_flowing_sand: Pending | --:-- | -- |
| 4. Run sessions | :hourglass_flowing_sand: Pending | --:-- | -- |
| 5. Verify outcomes | :hourglass_flowing_sand: Pending | --:-- | -- |
| 6. Record verdicts + post final comment | :hourglass_flowing_sand: Pending | --:-- | -- |

| Symbol | Meaning |
|---|---|
| :hourglass_flowing_sand: | Not started |
| :arrows_counterclockwise: | Running |
| :white_check_mark: | Passed |
| :x: | Failed |
| :warning: | Passed with caveats |

*Run started: $($runStart.ToString('HH:mm')) — all steps pending*
"@

$tmpFile = [System.IO.Path]::GetTempFileName()
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($tmpFile, $body, $utf8NoBom)
$response = gh api "repos/$env:REPO/issues/$env:PR_NUMBER/comments" --method POST --field "body=@$tmpFile" | ConvertFrom-Json
$env:COMMENT_ID = $response.id
Remove-Item $tmpFile -Force
Write-Host "Progress comment posted — ID: $($response.id)"
```

After posting, immediately update the comment to mark Step 1 as `:arrows_counterclockwise: Running` (do NOT wait — this is a two-step sequence: post all-pending, then immediately update to Step 1 running). Then proceed to Step 1.

See **Progress Reporting** for the full comment lifecycle and update patterns.

**⚠️ STOP before continuing:** If posting the comment fails (network error, auth error), abort the run and report the failure. Do not proceed silently without a tracking comment.

### Step 1 — Build the CLI from your branch

```bash
cd /path/to/squad          # your feature branch
npm install
npm run build -w packages/squad-sdk && npm run build -w packages/squad-cli

# Link so `squad` command uses your local build (workspace flag — no cd required)
npm link -w packages/squad-cli
```

Verify: `squad version` output includes the `-preview` suffix (e.g., `x.y.z-preview`),
confirming the local dev build is active. If the output shows a plain semver without
`-preview`, the globally-installed npm package is still in use — re-check the link step.
See [CONTRIBUTING.md — Making the `squad` Command Use Your Local Build](../../../CONTRIBUTING.md)
for the full guidance on local dev versioning.

### Step 2 — Create a disposable test repo

```bash
mkdir /tmp/sq-test-1 && cd /tmp/sq-test-1
git init
echo "# Test Project" > README.md
echo '{"name":"test-project","version":"1.0.0"}' > package.json
mkdir src
echo "export function hello() { return 'world' }" > src/index.ts
git add -A && git commit -m "init: test project"
```

Keep the project small — you only need enough for the coordinator to recognize a
codebase and hire a team.

### Step 3 — Init a squad with your modified templates

```bash
squad init
# If testing a specific feature (e.g. state backends):
# squad init --state-backend git-notes
```

Verify the init produced the expected files:
```bash
ls -la .squad/
cat .squad/team.md          # should have ## Members with 3+ agents
cat .squad/config.json      # should reflect any CLI flags you passed
```

### Step 4 — Run a real session and capture output

Use the Copilot CLI's `-p` flag with `--allow-all-tools` for non-interactive sessions.
`--allow-all-tools` is **required** for automated/non-interactive runs — without it,
tool calls (including file writes) prompt for confirmation and block.

```powershell
# PowerShell (Windows)
copilot --agent squad --allow-all-tools -p "Picard, decide what testing framework to use. Write your decision." `
  2>&1 | Tee-Object evidence/session-task.log
```

```bash
# Bash (macOS/Linux)
copilot --agent squad --allow-all-tools -p "Picard, decide what testing framework to use. Write your decision." \
  2>&1 | tee evidence/session-task.log
```

Alternatively, set the `COPILOT_ALLOW_ALL=1` environment variable instead of the flag.

For multi-turn workflows, run sequential sessions:
```powershell
# Session A: give the team a task
copilot --agent squad --allow-all-tools -p "prompt A" 2>&1 | Tee-Object evidence/session-A.log

# Session B: verify state persisted
copilot --agent squad --allow-all-tools -p "What decisions has the team made?" 2>&1 | Tee-Object evidence/session-B.log
```

### Step 5 — Verify the outcome

Check that your template change had the expected effect. Common checks:

```bash
# State location (for state-backend changes)
git notes --ref=squad list              # git-notes backend
git ls-tree -r squad-state              # orphan backend
ls .squad/agents/*/history.md           # worktree backend

# Coordinator behavior (grep session log)
grep "STATE_BACKEND" evidence/session-task.log
grep "spawn" evidence/session-task.log

# File tree diff
git diff --stat HEAD~1                  # what changed on working branch
git log --all --oneline                 # commits across all branches
```

### Step 6 — Record the verdict

Create an `evidence/verdict.md` in each test repo:

```markdown
## Test: [scenario name]
**Backend:** worktree | git-notes | orphan | two-layer
**Branch:** [your feature branch]
**Result:** PASS | PARTIAL | FAIL
**Duration:** Xm Ys

### What was verified
- [ ] Coordinator identified feature correctly (from session log)
- [ ] Agent was spawned via `task` tool (not simulated)
- [ ] team.md has ## Members with 3+ agents
- [ ] State landed in correct location
- [ ] No unexpected side effects

### Evidence files
- session-task.log — full session output
- git-log.txt — `git log --all --oneline`

### Notes
[anything unusual or noteworthy]
```

Record the wall-clock time from the start of Step 1 (fast-fail checks) to the end
of Step 6 (verdict posted). This is the full E2E run duration for this scenario.

## Progress Reporting

Use this section only when you are running E2E validation for an open PR. If
`PR_NUMBER` and `REPO` are both set, post and maintain a live tracking comment
in the PR thread. If either value is missing (for example, a local-only run),
skip progress reporting silently.

### Start the tracking comment (Step 0 — see Workflow above)

The initial comment must be posted as **Step 0** — the absolute first action before
anything else. See the Step 0 block in the Workflow section for the exact code.

The subsections below describe how to **update** the comment at each step boundary.
For reference, here is the initial all-pending comment body posted in Step 0:

1. Post a PR comment before Step 1 begins:

```bash
gh pr comment "$PR_NUMBER" --repo "$REPO" --body "## E2E Progress\n\n| Step | Status | Started | Duration |
|---|---|---|---|
| 1. Fast-fail checks (build · link · \\`squad version\\`) | ⏳ Pending | --:-- | -- |
| 2. Create test repo(s) | ⏳ Pending | --:-- | -- |
| 3. \\`squad init\\` + file verification | ⏳ Pending | --:-- | -- |
| 4. Run sessions | ⏳ Pending | --:-- | -- |
| 5. Verify outcomes | ⏳ Pending | --:-- | -- |
| 6. Record verdicts + post final comment | ⏳ Pending | --:-- | -- |
\n| Symbol | Meaning |
|---|---|
| ⏳ | Not started |
| 🔄 | Running |
| ✅ | Passed |
| ❌ | Failed |
| ⚠️ | Passed with caveats |"
```

2. Capture the comment ID immediately after posting it:

```bash
COMMENT_ID=$(gh api "repos/$REPO/issues/$PR_NUMBER/comments" --jq '.[-1].id')
```

3. Treat Step 1 as in progress as soon as the comment exists. Update the body so
   Step 1 shows `🔄 Running` and every later step remains `⏳ Pending`.

### Update the tracking comment after every step boundary

1. When marking a step `🔄 Running`, record `$startTime = Get-Date` and store the
   `HH:MM` start time in that row's **Started** column.
2. Edit the existing comment in place; do not post a new progress comment:

```bash
gh api --method PATCH "repos/$REPO/issues/comments/$COMMENT_ID" --field body="..."
```

3. When marking a step `✅`, `❌`, or `⚠️`, compute
   `$duration = (Get-Date) - $startTime` and format it as
   `"{0}m {1}s" -f [int]$duration.TotalMinutes, $duration.Seconds`.
4. Update the completed step row to `✅`, `❌`, or `⚠️`, keep its original
   `HH:MM` value in **Started**, and write the formatted duration in **Duration**.
5. Keep all previously completed rows unchanged.
6. Mark the next step as `🔄 Running` and set its **Started** value.
7. Leave later steps as `⏳ Pending` with `--:--` for **Started** and `--` for
   **Duration**.
8. If a step fails and you stop early, still update the comment so the failed step
   shows `❌` with its original start time and computed duration, and Step 6
   becomes `🔄 Running` while you prepare the final verdict.

### Use this status legend in the comment

| Symbol | Meaning |
|---|---|
| ⏳ | Not started |
| 🔄 | Running |
| ✅ | Passed |
| ❌ | Failed |
| ⚠️ | Passed with caveats |

### Use exact step names and order

Keep these six rows in this exact order every time you update the comment:

1. Fast-fail checks (build · link · `squad version`)
2. Create test repo(s)
3. `squad init` + file verification
4. Run sessions
5. Verify outcomes
6. Record verdicts + post final comment

### Handle Windows comment bodies safely

On Windows PowerShell 5.1, use the **`--field body=@file`** pattern to post comment
bodies. Write the content to a temp file using UTF-8 **without BOM**, then pass
`--field "body=@$tmpFile"` to `gh api`. This is more reliable than piping JSON
through `--input -` on PS 5.1, which can silently corrupt multi-byte characters
even with `[Console]::OutputEncoding = UTF8`.

Key rules:
- Use `New-Object System.Text.UTF8Encoding $false` (the `$false` disables the BOM).
  `[System.Text.Encoding]::UTF8` writes a BOM which GitHub renders as a stray
  character (`﻿`) at the start of the comment.
- Use `--field "body=@$tmpFile"`, NOT `--input -` or `--input filename`, for
  comment body updates. The `@` prefix tells `gh` to read the field value from
  the file rather than treating the path as a literal string.
- Clean up the temp file after posting.
- Scrub any local absolute paths from the body before posting (see PII Protection
  section).

```powershell
$step1StartTime = Get-Date
$step1Started = $step1StartTime.ToString('HH:mm')
$step1Duration = (Get-Date) - $step1StartTime
$step1DurationText = "{0}m {1}s" -f [int]$step1Duration.TotalMinutes, $step1Duration.Seconds
$step2StartTime = Get-Date
$step2Started = $step2StartTime.ToString('HH:mm')
$body = @"
## E2E Progress

| Step | Status | Started | Duration |
|---|---|---|---|
| 1. Fast-fail checks (build · link · `squad version`) | :white_check_mark: Passed | $step1Started | $step1DurationText |
| 2. Create test repo(s) | :arrows_counterclockwise: Running | $step2Started | -- |
| 3. `squad init` + file verification | :hourglass_flowing_sand: Pending | --:-- | -- |
| 4. Run sessions | :hourglass_flowing_sand: Pending | --:-- | -- |
| 5. Verify outcomes | :hourglass_flowing_sand: Pending | --:-- | -- |
| 6. Record verdicts + post final comment | :hourglass_flowing_sand: Pending | --:-- | -- |

| Symbol | Meaning |
|---|---|
| :hourglass_flowing_sand: | Not started |
| :arrows_counterclockwise: | Running |
| :white_check_mark: | Passed |
| :x: | Failed |
| :warning: | Passed with caveats |
"@

$tmpFile = "$env:TEMP\e2e-comment-body.md"
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($tmpFile, $body, $utf8NoBom)
gh api --method PATCH "repos/$env:REPO/issues/comments/$env:COMMENT_ID" --field "body=@$tmpFile"
Remove-Item $tmpFile -Force
```

### Progressive Verdicting — Post After Each Scenario (Critical)

**Do NOT batch all scenario results to the end.** This is the most common cause
of lost verdicts. After each scenario completes, **immediately** PATCH the
tracking comment with that scenario's result before moving to the next one.

The pattern for each scenario:

```powershell
# After scenario N completes — PATCH immediately, before starting scenario N+1
$scenarioNDuration = "{0}m {1}s" -f [int]((Get-Date) - $scenarioNStartTime).TotalMinutes, ((Get-Date) - $scenarioNStartTime).Seconds
# ...rebuild the full comment body with this scenario updated to PASS/FAIL/PARTIAL...
$tmpFile = [System.IO.Path]::GetTempFileName()
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($tmpFile, $body, $utf8NoBom)
gh api --method PATCH "repos/$env:REPO/issues/comments/$env:COMMENT_ID" --field "body=@$tmpFile"
Remove-Item $tmpFile -Force
Write-Host "Scenario N verdict posted"
```

This guarantees that even if the AI model connection drops mid-run, the last
successfully PATCHed state is always visible in the PR.

### Agent Run Time Budget

**⚠️ Critical: Background agents lose their AI model connection after ~15 minutes
of continuous execution.** This is a platform limit, not a bug in your code.
The verdict stage appears to "hang" because the connection drops right at the end
when the agent has been running too long.

**Per-agent scenario budget:**

| Scenario type | Estimated time | Budget |
|---|---|---|
| Static checks only (file existence, grep, size) | 1-3 min | 4 per agent |
| `squad init` + file verification (no copilot session) | 3-5 min | 3 per agent |
| `squad init` + one `copilot --agent squad` session | 8-15 min | **1 per agent** |
| Build + link + one copilot session | 12-20 min | **1 per agent** |

**Rule: Limit yourself to 1 scenario that includes a `copilot --agent squad` session
per agent run.** For a plan with multiple copilot-session scenarios, run them in
separate agents — not in sequence within a single agent.

If your scenario plan has N copilot-session scenarios, request N separate sims
agents to run them in parallel (one scenario each). Static scenarios may be
batched up to 4 per agent.

**If you are running a scenario with a `copilot --agent squad` session:**
- Run the build and link ONCE at the start (shared across all static scenarios)
- Run the copilot session immediately after the repo is set up
- PATCH the comment with the result immediately after the session ends
- Then proceed to static scenarios while you still have connection budget

### Replace the tracking comment with the final verdict

When you reach Step 6, replace the tracking comment body entirely with the final
structured verdict table. Do not post a separate final comment. The tracking
comment is the final verdict comment.

Include a summary row at the bottom of the final table showing the total elapsed
time for the full run:

```text
| **Total** | — | HH:MM | Xm Ys |
```

**If the connection drops before Step 6:** The last progressive verdict PATCH
already shows the partial state. The next agent run should read the existing
comment, pick up where it left off, and add remaining scenario rows rather than
starting fresh.

## Test Matrix Template

Use this matrix when planning validation for a template change. Not every change
needs every row — pick the scenarios relevant to your modification.

| # | Scenario | What to verify | Duration |
|---|----------|----------------|----------|
| 1 | Basic init + task | Templates applied, agent spawned, work produced | — |
| 2 | Cross-branch persistence | State survives `git checkout` (if state-backend) | — |
| 3 | Scribe behavior | Scribe commits to correct target | — |
| 4 | PR cleanliness | Feature branch PR has no leaked state files | — |
| 5 | Migration path | Existing squad picks up new template behavior | — |
| 6 | Edge case: empty repo | Init works in repo with single commit | — |
| 7 | Edge case: monorepo | Init works in subdirectory of monorepo | — |

Note: Keep `—` during planning, then replace it with the actual elapsed time when
recording the verdict for each scenario.

## Tips

- **Name test repos descriptively:** `sq-test-notes-crossbranch`, not `test1`.
- **Always capture session logs.** Without logs, you can't debug failures.
- **One scenario per repo.** Don't reuse repos across unrelated tests — state
  leaks between tests make results unreliable.
- **Clean up after.** Delete test repos when done. They accumulate fast.
- **Windows users:** Use PowerShell. `Tee-Object` replaces `tee`. Paths use `\`.

## Fast-Fail Rules

These checks must pass before running any scenario. If any fail, stop
immediately and report the failure — do **not** attempt workarounds or mark
scenarios as SKIPPED.

0. **Clean stale SDK before building.** Before running `npm run build`, remove any
   stale published copy of `@bradygaster/squad-sdk` that may have been installed
   into `packages/squad-cli/node_modules/`. This local copy shadows the workspace
   symlink in the root `node_modules/` and causes TypeScript to see the published
   version instead of the local source. Run from the repo root:
   ```powershell
   $stale = "packages\squad-cli\node_modules\@bradygaster\squad-sdk"
   if (Test-Path $stale) { Remove-Item -Recurse -Force $stale; Write-Host "Cleaned stale SDK" }
   ```
   This is safe to run unconditionally — if the path doesn't exist, the command is
   a no-op. The root `node_modules\@bradygaster\squad-sdk` workspace symlink remains
   intact and npm will use it automatically.
1. **Build must succeed.** Run `npm run build` from the repo root. A build
   failure blocks all scenarios; report `BUILD_FAILED` and stop.
   - If the error is `tsc: not found` or similar missing-binary errors, run
     `npm install` first to reconcile `node_modules` with the lock file, then
     retry. This can happen after `git checkout HEAD -- package-lock.json`
     restores the lock file without reinstalling.
2. **CLI must link successfully.** `cd packages/squad-cli && npm link` must exit
   0. If it fails, report `LINK_FAILED` and stop.
3. **`squad version` must run.** After linking, `squad version` must output a
   version string. If not, report `CLI_NOT_FOUND` and stop.

Do **not** mark scenarios as SKIPPED due to build or environment errors —
that obscures real failures from reviewers. SKIPPED is only acceptable when the
user explicitly requests it.

## PII Protection — Mandatory

When posting evidence to PR comments, issues, or any shared document:

- **Never include absolute paths** that contain a local username (e.g.,
  `C:\Users\username\...` or `/home/username/...`).
- **Use `~` notation** for home-relative paths: `~/AppData/Local/Temp/...`
  or `~/tmp/sq-test-1`.
- **Repo-internal paths use `<repo-root>` as the prefix.** If evidence files live
  inside the repository (e.g. `.e2e/`, `tmp/`, or any subdirectory of the repo),
  write them as `<repo-root>\.e2e\pr-1035\evidence` — not as `~\..\..\...` backward
  navigation. `<repo-root>` is a clear, portable placeholder for the repository root
  that does not expose the machine's directory layout.
- **Scrub before posting.** Replace any occurrence of the local machine path
  prefix (everything up to and including the username segment) with `~`.

Example — ❌ wrong: `C:\Users\johndoe\AppData\Local\Temp\sq-e2e-pr1035\evidence`
Example — ✅ right: `~/AppData/Local/Temp/sq-e2e-pr1035/evidence`
Example — ❌ wrong: `~\..\..\repos\squad\.e2e\pr-1035\evidence`
Example — ✅ right: `<repo-root>\.e2e\pr-1035\evidence`

This applies to all evidence tables, verdict files, and PR comments.

## Anti-Patterns

- **Skipping the local build.** If you test with the published CLI, you're
  testing the old templates, not your changes.
- **Posting absolute paths in PR comments.** Always scrub to `~`-relative paths
  before sharing. See PII Protection above.
- **Marking scenarios SKIPPED due to environment issues.** Fix the environment
  (use fast-fail rules above) or report BUILD_FAILED — never silently skip.
- **Testing only the happy path.** Template changes often break edge cases (empty
  repos, monorepos, cross-branch). Test at least 2-3 scenarios.
- **Trusting session output alone.** Always verify git state independently —
  agents can claim they wrote something without actually doing it.
- **Reusing test repos.** Prior state bleeds into later tests. Start fresh.
- **Batching all scenario verdicts to the end.** The AI model connection drops
  after ~15 minutes. Always PATCH the comment after each scenario so partial
  results are never lost. See Progressive Verdicting above.
- **Running multiple `copilot --agent squad` sessions in one agent.** Each
  session takes 5-15 minutes; combined with build time, you'll hit the ~15-minute
  connection budget. One copilot session per agent — split into parallel agents
  if your plan has more.

## Sandbox / Permission Notes

### Always pass `--allow-all-tools` in non-interactive mode

The Copilot CLI requires explicit permission to run tools automatically. In
interactive mode the user approves each tool call; in non-interactive mode (`-p`)
those prompts cannot be displayed and writes fail silently or with a "Permission
denied and could not request permission from user" error.

Fix: always include `--allow-all-tools` (or `--yolo` / `--allow-all`) in Step 4
commands, or export `COPILOT_ALLOW_ALL=1` before running E2E sessions.

This also applies when `copilot --agent squad` is launched as a subprocess from
inside a Copilot CLI background agent (e.g. Sims running via the `task` tool) —
the flag is still needed.

### `--allow-all-paths` for repos outside the CWD

By default the CLI restricts file access to the current directory tree. If the
coordinator needs to read files from a parent repo while running in a disposable
test repo, add `--allow-all-paths`:

```powershell
copilot --agent squad --allow-all-tools --allow-all-paths -p "..."
```

Or use the combined shorthand: `--allow-all` / `--yolo`.

## Confidence

high — Validated through 12 real E2E test sessions during state-backend
development (PR #1004). `--allow-all-tools` requirement confirmed in PR #1035.
