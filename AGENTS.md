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

## Lazy Skill Loading Policy

Use skills only when the task requires specialized guidance. Do not pre-load every skill.

### Use these skills by intent

- Workflow create/update/debug/upgrade tasks → `.github/skills/agentic-workflows/SKILL.md`
- Core engineering conventions, validation flow, and command playbooks → `.github/skills/developer/SKILL.md`
- Error handling design/patterns → `.github/skills/error-recovery-patterns/SKILL.md`
- GitHub MCP usage patterns → `.github/skills/github-mcp-server/SKILL.md`
- Query helpers for issues/PRs/workflows/discussions/labels → matching `.github/skills/github-*-query/SKILL.md`
- Doc-writing conventions → `.github/skills/documentation/SKILL.md`

## Why this file is intentionally short

This file is loaded at first invocation and affects every task. Keep it concise and move detailed or domain-specific guidance into skills so that context is fetched only when relevant.
