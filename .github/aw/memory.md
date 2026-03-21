---
description: Guide for choosing the right persistent memory strategy in agentic workflows — cache-memory, repo-memory, and repo-memory with wiki. cache-memory is the first choice.
---

# Persistent Memory in Agentic Workflows

Consult this file when designing a workflow that needs to **persist state across runs** — deduplication, incremental processing, cross-run context, or knowledge accumulation.

> ⚠️ **`repo-memory` does NOT mean "cache-memory"**. They are two distinct tools with different backends, tradeoffs, and use cases. `cache-memory` is almost always the right first choice.

---

## Quick Decision Guide

| Need | Use |
|---|---|
| Skip already-processed items (deduplication) | `cache-memory` ✅ first choice |
| Round-robin processing across runs | `cache-memory` ✅ first choice |
| Store ephemeral run state, analysis notes, or intermediate results | `cache-memory` ✅ first choice |
| Long-lived knowledge base visible in PRs and code reviews | `repo-memory` |
| Human-readable wiki pages for knowledge accumulation | `repo-memory` with `wiki: true` |

**Default to `cache-memory` unless you have a specific reason to use `repo-memory`.**

---

## `cache-memory` — First Choice

Uses GitHub Actions cache (`actions/cache`) to persist a local filesystem directory populated by the `@modelcontextprotocol/server-memory` MCP server. The directory lives at `/tmp/gh-aw/cache-memory/`.

### When to use

- **Deduplication**: Track which items (issues, PRs, URLs, IDs) have already been processed
- **Round-robin / incremental processing**: Remember where you left off across scheduled runs
- **Ephemeral structured state**: JSON blobs, processing queues, intermediate analysis results
- **Visual regression baselines**: Store screenshots between PR runs (see `visual-regression.md`)
- **Tool call caching**: Avoid redundant expensive API calls across runs

### Configuration

```yaml
tools:
  cache-memory: true
```

Advanced — custom key:

```yaml
tools:
  cache-memory:
    key: dedup-${{ github.event.schedule }}-${{ github.run_id }}
    retention-days: 30
    allowed-extensions: [".json"]
```

Multiple named caches:

```yaml
tools:
  cache-memory:
    - id: processed
      key: processed-items-${{ github.run_id }}
    - id: results
      key: results-${{ github.run_id }}
      retention-days: 14
```

### Storage path

- Single cache: `/tmp/gh-aw/cache-memory/`
- Multiple caches: `/tmp/gh-aw/cache-memory/{id}/`

### Deduplication example (scheduled workflow)

The following pattern lets a scheduled workflow skip items it has already processed:

```markdown
---
on:
  schedule:
    - cron: "0 9 * * *"
permissions:
  issues: read
engine: copilot
tools:
  github:
    toolsets: [issues]
  cache-memory: true
safe-outputs:
  create-issue:
    title-prefix: "[daily-digest] "
    close-older-issues: true
    labels: [automation]
timeout-minutes: 15
---

Fetch the 20 most recently updated open issues.

Load `/tmp/gh-aw/cache-memory/processed.json` if it exists; it contains an array of
issue numbers already included in past digests.

Skip any issue whose number already appears in that array.

Summarize the remaining (new) issues. If there are none, use the `noop` safe output.

Before finishing, write the updated full list of processed issue numbers back to
`/tmp/gh-aw/cache-memory/processed.json` using a filesystem-safe timestamp:
`YYYY-MM-DD-HH-MM-SS` (no colons, no `T`, no `Z`).
```

### Tradeoffs

| ✅ Pros | ❌ Cons |
|---|---|
| Zero repository noise — no commits, no PRs | Evicted when cache expires (default 7 days; use `retention-days` to extend up to 90) |
| Fast: no Git operations required | Not human-readable in GitHub UI |
| Works with Copilot, Claude, and custom engines | Data loss if cache is invalidated or expires |
| Supports multiple isolated caches per workflow | Files are uploaded as GitHub Actions artifacts — **no colons in filenames** |
| Scoped to workflow by default | |

### Filename safety

Cache-memory files are uploaded as GitHub Actions artifacts. **Artifact filenames must not contain colons** (NTFS limitation on Windows-hosted runners).

```bash
# ✅ GOOD — filesystem-safe timestamp
/tmp/gh-aw/cache-memory/state-2026-02-12-11-20-45.json

# ❌ BAD — colon in timestamp breaks artifact upload
/tmp/gh-aw/cache-memory/state-2026-02-12T11:20:45Z.json
```

When instructing the agent to write timestamped files, say explicitly:
> "Use filesystem-safe timestamp format `YYYY-MM-DD-HH-MM-SS` (no colons, no `T`, no `Z`)."

---

## `repo-memory` — Long-lived Repository Knowledge

Uses a dedicated Git branch (default: `memory/agent-notes`) to store files that persist indefinitely until explicitly deleted. The directory lives at `/tmp/gh-aw/repo-memory/`.

### When to use

- The knowledge needs to survive cache expiration
- You want the memory to be **visible in the repository** (auditable via Git history)
- The workflow accumulates a knowledge base that grows over time (e.g., architecture notes, known issues)
- You need changes to appear in diffs and be reviewable

### Configuration

```yaml
tools:
  repo-memory:
    branch-name: memory/agent-notes   # Optional: custom branch name
    target-repo: owner/other-repo     # Optional: store in another repo
    allowed-extensions: [".json", ".md"]
    max-file-size: 10240              # bytes
    max-file-count: 100
permissions:
  contents: write                     # Required: repo-memory writes to the repo
```

### Tradeoffs

| ✅ Pros | ❌ Cons |
|---|---|
| Persists indefinitely (no expiry) | Requires `contents: write` permission |
| Auditable: Git history shows every change | Produces Git commits — repository noise |
| Survives cache invalidation | Slower: requires Git clone + push |
| Human-readable via GitHub branch UI | Not available for Copilot engine (requires GitHub tools) |
| Can target a different repository | More complex setup |

---

## `repo-memory` with `wiki: true` — GitHub Wiki Backend

A variant of `repo-memory` that stores files in the **GitHub Wiki** (a separate Git repository at `<repo>.wiki.git`) instead of a branch.

### When to use

- You want structured, human-readable documentation pages
- The knowledge is intended for **human consumption** (wikis are browsable)
- You're building a living knowledge base or FAQ

### Configuration

```yaml
tools:
  repo-memory:
    wiki: true
    allowed-extensions: [".md"]
permissions:
  contents: write   # Required: wiki writes via push
```

Files follow GitHub Wiki Markdown conventions: use `[[Page Name]]` syntax for internal links, name files with hyphens instead of spaces.

### Tradeoffs

| ✅ Pros | ❌ Cons |
|---|---|
| Browsable in the GitHub Wiki UI | Requires `contents: write` permission |
| Great for human-readable knowledge bases | Produces Git commits to wiki repo |
| Standard Markdown with wiki link syntax | Restricted to `.md` files in practice |
| Separate from main repo history | Less suitable for structured JSON state |

---

## Summary Comparison

| Feature | `cache-memory` | `repo-memory` | `repo-memory` + wiki |
|---|---|---|---|
| **First choice** | ✅ Yes | No | No |
| **Storage backend** | GitHub Actions cache | Git branch | GitHub Wiki |
| **Persistence** | Up to 90 days | Indefinite | Indefinite |
| **Requires `contents: write`** | No | Yes | Yes |
| **Repository noise** | None | Git commits | Wiki commits |
| **Human-readable in GitHub** | No | Via branch UI | Via Wiki UI |
| **Structured data (JSON)** | ✅ Ideal | Possible | Not recommended |
| **Filename restrictions** | No colons in names | None | Hyphens for spaces |
| **Engine compatibility** | Copilot, Claude, custom | Claude, custom | Claude, custom |

---

## Anti-patterns

- ❌ **Do not invent `repo-memory` as a synonym for `cache-memory`** — they are different tools
- ❌ **Do not use `repo-memory` for ephemeral per-run state** — use `cache-memory`
- ❌ **Do not use `cache-memory` when you need indefinite persistence** — use `repo-memory`
- ❌ **Do not include colons in cache-memory filenames** — artifact upload will fail
