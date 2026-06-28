# GitHub Agentic Workflows (gh-aw)

GitHub Agentic Workflows (`gh-aw`) is a GitHub CLI extension that compiles markdown workflows into GitHub Actions.

## Important Note: gh-aw vs GitHub Copilot CLI

- **gh-aw** is the `gh aw` CLI extension.
- **copilot** is a separate CLI used as one possible runtime engine.
- Use `gh aw` commands for workflow authoring/compilation (`gh aw compile`, `gh aw run`, `gh aw audit`).

## Ambient Context (First Invocation)

To keep first-turn context small, only these repository root instruction files should be considered ambient:

| File | Purpose |
|---|---|
| `AGENTS.md` | Minimal global operating rules and routing |
| `SKILL.md` | Short repository capability summary |

Everything else should be loaded **lazily** through skills only when needed.

## Critical Rules (Always Applicable)

1. If you changed files, use `report_progress` to commit and push.
2. Before `report_progress`, run `make agent-report-progress` and ensure it passes.
3. After Go changes, run `make fmt`.
4. After workflow markdown changes (`.md` under `.github/workflows/`), run `make recompile`.
5. Do not add `.lock.yml` files to `.gitignore`.

## Upstream-managed workflow sources (read-only in this repo)

Workflows that declare a `source:` frontmatter entry (for example `source: githubnext/agentic-ops@<ref>`) are provenance-managed from an upstream bundle.

- Treat those workflow source files (for example `.github/workflows/agentic-token-audit.md` and `.github/workflows/agentic-token-optimizer.md`) as read-only in this repository.
- Do **not** manually edit their generated `.lock.yml` files.
- To change these workflows, use the approved update path:
  1. run `gh aw update` to refresh from source, and/or
  2. update the pinned source/version (`source: ...@...`), and/or
  3. make the change upstream first, then pull it in via `gh aw update`.

## Lazy Skill Loading Policy

Use skills only when the task requires specialized guidance. Do not pre-load every skill.

When the relevant skill is not obvious, first discover candidates and then load only the minimal guidance needed:

- Query GitHub Agent Finder directly with the built-in REST API at `https://agentfinder.github.com/api/v1/search`.
- Send an ARD search request with the user's task in `query.text`; do not ask the user for the endpoint.
- Use **skill fusion** after discovery: read only the specific skill sections or fragments needed instead of loading full skills broadly.

### Use these skills by intent

- Workflow create/update/debug/upgrade tasks → `.github/skills/agentic-workflows/SKILL.md`
- Core engineering conventions, validation flow, and command playbooks → `.github/skills/developer/SKILL.md`
- Error handling design/patterns → `.github/skills/error-recovery-patterns/SKILL.md`
- GitHub MCP usage patterns → `.github/skills/github-mcp-server/SKILL.md`
- Query helpers for issues/PRs/workflows/discussions/labels → matching `.github/skills/github-*-query/SKILL.md`
- Doc-writing conventions → `.github/skills/documentation/SKILL.md`
- Reviewing or writing `git`/`gh`/remote operations against checkouts (per-checkout credentials, sparse/shallow monorepos, safe-outputs MCP runs without credentials) → `.github/skills/checkout-credential-review/SKILL.md`

## Why this file is intentionally short

This file is loaded at first invocation and affects every task. Keep it concise and move detailed or domain-specific guidance into skills so that context is fetched only when relevant.
