---
name: task-preflight
description: Pre-flight feasibility check for container-vulnerability-fix and large-refactor tasks. Confirms the work is actually possible and produces a concrete diff plan before any PR is opened, so blocked tasks report the blocker instead of leaving an empty pull request.
---

# Task Pre-flight

Run this check **first** on tasks that historically produce zero-diff pull requests:

- container image vulnerability fixes (`Fix vulnerabilities in container image ...`, `Update container image to address vulnerabilities`)
- large-function / large-file refactors (`Refactor large functions in pkg/workflow and pkg/cli`)

These tasks are retried repeatedly because an earlier attempt could not reach the
target image or could not scope a refactor, opened a draft PR with no commits,
and was closed with no explanation for the next attempt.

## Rules

1. **Do not push code until pre-flight passes.** Establish the diff plan first.
2. **A task is only "done" with a real diff or an explicit blocker report.** Never
   finish a session with zero additions and zero deletions and no explanation.
3. **When blocked, write the blocker down** in the progress report / PR description
   before stopping: what was attempted, the exact command and its failure output,
   and what would unblock it (network access, image availability, upstream fix).
   The next attempt must be able to read the blocker without re-running the work.
4. **Never open or leave an empty PR as a placeholder.** If the pre-flight fails,
   the deliverable is the blocker report, not a draft PR.

## Container vulnerability tasks

Pre-flight, in order:

1. Reproduce the finding locally:

   ```bash
   make build
   ./gh-aw compile --force-refresh-container-pins --syft --grype --grant
   ```

2. Confirm the image is reachable. If the pull or scan fails (firewall, missing
   registry credentials, rate limit), that is a blocker — report it and stop.
3. Decide which of the two real remediations applies:
   - **A fixed version exists upstream** → refresh the pin (the compile above
     rewrites the pinned digest) and commit the changed `.lock.yml` files.
   - **No fixed version exists** → add a scoped risk acceptance to `.grype.yaml`
     with the vulnerability ID, package name, version, type, and a `reason`
     explaining why it is accepted and when to re-evaluate. Follow the existing
     entries in that file exactly.
4. If neither applies (for example, the CVE is already accepted or already fixed
   by the current pin), the correct outcome is **no PR** plus a report stating
   that the finding is stale, with the scan output that proves it.

## Large-refactor tasks

Pre-flight, in order:

1. Get the actual offender list instead of guessing:

   ```bash
   make golint-custom LINTER_PACKAGES="./pkg/workflow/... ./pkg/cli/..."
   ```

   `largefunc` reports every function body over `MAX_LINES` (default 60).
2. Pick a **small, named subset** — a handful of functions in one package — and
   write down the extraction plan (new helper names, target files) before editing.
   Repository-wide rewrites of thousands of lines are not reviewable and get
   closed; see the focused decomposition ADRs under `docs/adr/`.
3. Refactors must be behaviour-preserving: no signature changes for exported
   APIs, no logic changes. Validate with `make fmt` then `make test-unit`.
4. If the subset cannot be extracted without behaviour change, report that
   analysis as the result instead of pushing an empty branch.

## Reporting a blocker

Use the progress report / PR description with this shape:

```markdown
## Blocked: <task title>

**Attempted**: <commands run>
**Failure**: <exact error output>
**Why it cannot proceed**: <root cause>
**Unblocked by**: <access, upstream fix, scope decision needed from a maintainer>
```

Keep the report factual and short. It exists so the next attempt starts from the
blocker instead of repeating the same zero-diff cycle.
