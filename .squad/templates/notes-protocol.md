# Squad Notes Protocol

> Contract for agent state via git notes. Agents write commit-scoped context
> here instead of modifying `.squad/` files in PRs.
>
> **Version:** 1.0
> **Backends:** `git-notes`, `orphan`

---

## Overview

Squad state has two layers:

1. **Git notes layer** (this document) — thin, commit-scoped annotations that
   attach agent context to commits without appearing in PRs or diffs.
2. **Permanent state layer** — long-lived decisions, routing rules, and archives
   stored via the configured state backend (`git-notes` or `orphan` branch).

Agents write notes during their work rounds. Ralph promotes flagged notes to
permanent state after a PR merges.

---

## Namespaces

Each agent writes to its own namespace to prevent conflicts:

| Namespace | Owner | Purpose |
|-----------|-------|---------|
| `refs/notes/squad/data` | Data | Architecture decisions, implementation choices |
| `refs/notes/squad/worf` | Worf | Security reviews, vulnerability assessments |
| `refs/notes/squad/seven` | Seven | Documentation quality, API contract decisions |
| `refs/notes/squad/ralph` | Ralph | Work-round progress, task-state annotations |
| `refs/notes/squad/q` | Q | Devil's advocate findings, risk assessments |
| `refs/notes/squad/research` | Any agent | Research notes that should survive branch deletion |
| `refs/notes/squad/review` | Any agent | Code review context (mirrors Gerrit's pattern) |

**Rule**: Only write to your own namespace. The shared namespaces
(`research`, `review`) use `append` — never `add`.

---

## Note JSON Schema

All notes MUST be valid JSON. Minimum required fields:

```json
{
  "agent": "Data",
  "timestamp": "2026-03-23T14:00:00Z",
  "type": "decision | research | review | progress | security",
  "content": "..."
}
```

### Decision notes

```json
{
  "agent": "Data",
  "timestamp": "2026-03-23T14:00:00Z",
  "type": "decision",
  "decision": "Use JWT RS256 for auth middleware",
  "reasoning": "Existing pattern in codebase — auth.go:47-89.",
  "alternatives_considered": ["HS256", "session tokens"],
  "confidence": "high",
  "promote_to_permanent": true
}
```

Set `"promote_to_permanent": true` to signal Ralph to copy this to
`decisions.md` after the PR merges.

### Research notes

```json
{
  "agent": "Data",
  "timestamp": "2026-03-23T14:00:00Z",
  "type": "research",
  "topic": "JWT vs session tokens",
  "findings": {},
  "effort_hours": 2.5,
  "archive_on_close": true
}
```

Set `"archive_on_close": true` to signal Ralph to archive this to
`state/research/` even if the PR is rejected.

---

## Write Commands

```bash
# Write a decision note on the current commit
git notes --ref=squad/{your-agent} add \
  -m '{"agent":"{Agent}","timestamp":"...","type":"decision","decision":"..."}' \
  HEAD

# Append to an existing note (multiple items on same commit)
git notes --ref=squad/{your-agent} append \
  -m '{"agent":"{Agent}","timestamp":"...","type":"progress","content":"..."}' \
  HEAD

# Read your note
git notes --ref=squad/{your-agent} show HEAD

# List all commits with notes in your namespace
git notes --ref=squad/{your-agent} list
```

Or use the helper script:

```powershell
./scripts/notes/write-note.ps1 -Agent data -Type decision \
  -Content '{"decision":"Use JWT","reasoning":"..."}' \
  [-Commit HEAD] [-Promote] [-Archive]
```

---

## Fetch / Push

**Notes are NOT fetched or pushed by default.** Every clone needs setup.

### One-time setup

```bash
git config --add remote.origin.fetch 'refs/notes/*:refs/notes/*'
git fetch origin 'refs/notes/*:refs/notes/*'
```

Or use the helper:

```powershell
./scripts/notes/fetch.ps1 -Setup
```

### Every work round

1. **Start**: `git fetch origin 'refs/notes/*:refs/notes/*'`
2. **End**: `git push origin 'refs/notes/*:refs/notes/*'`

---

## Conflict Handling

1. **Per-agent namespaces prevent 99% of conflicts.** Only one agent writes to
   `refs/notes/squad/data`, so there are no write conflicts in normal use.

2. **Same agent, two machines:** First push wins. Losing machine should fetch
   and append:
   ```bash
   git fetch origin 'refs/notes/*:refs/notes/*'
   git notes --ref=squad/{agent} append -m '{...}' HEAD
   git push origin 'refs/notes/*:refs/notes/*'
   ```

3. **Shared namespaces** (`research`, `review`): Always use `git notes append`,
   never `git notes add`.

4. **Push conflict recovery:**
   ```bash
   git fetch origin 'refs/notes/*:refs/notes/*'
   git notes merge refs/notes/remotes/origin/squad/{namespace}
   git push origin 'refs/notes/*:refs/notes/*'
   ```

---

## When to Use Notes vs State Backend

| Use git notes | Use state backend |
|---------------|-------------------|
| Why THIS choice on THIS commit | Universal routing rules, conventions |
| Decisions scoped to a feature | Long-lived decisions for all future work |
| Research for a specific investigation | Research archives (promoted from notes) |
| Security sign-offs per commit | Agent history persisting across features |
| Agent-to-agent context for current feature | Team agreements and policies |

When in doubt: **notes first, promote to permanent state later.** Ralph handles
the promotion automatically when `promote_to_permanent` is set.

---

## Ralph Promotion Rules

**After PR merge:**

1. Fetch all notes from remote
2. Traverse commits reachable from the default branch that have notes
3. For each note with `"promote_to_permanent": true` → append to `decisions.md`
4. Push state

**After PR close/rejection:**

1. List notes in `squad/research` on the closed branch's commits
2. For each note with `"archive_on_close": true` → archive to `research/`
3. Push state
4. Notes on rejected commits are NOT promoted — this is the desired behavior
