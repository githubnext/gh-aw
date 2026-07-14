---
description: GitHub Agentic Workflows
applyTo: ".github/workflows/*.md,.github/workflows/**/*.md"
---

# GitHub Agentic Workflows

## Persona-to-Pattern Quick Matrix

| Persona | Preferred trigger and scope | Typical read tools | Typical write path | Explicit `noop` rule |
|---|---|---|---|---|
| Backend Engineer | `pull_request` with `paths:` scoped to migrations, schema, and API contracts | `github` (`gh-proxy`) | `add-comment` for PR-local findings; `create-issue` only for cross-cutting incidents | `noop` when no backend contract files changed |
| Frontend Developer | `pull_request` with `paths:` scoped to UI, design-token, and asset files | `github` (`gh-proxy`), optional `playwright`, optional `cache-memory` for baselines | `add-comment` | `noop` when no UI/token files changed or no actionable visual/token issues were found |
| DevOps Engineer | `workflow_run` for GitHub Actions failures, `deployment_status` for external deployment failures | `github` (`gh-proxy`) with `actions: read` or `deployments: read` | `create-issue` with stable dedup key | `noop` when status is non-terminal, self-recovered, or an open incident already exists for the same dedup key |

## File Format

Agentic workflows are markdown files with YAML frontmatter.

```markdown
---
emoji: 🧠
name: My Workflow
description: Short description
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
strict: true
network:
  allowed: [defaults, github]
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  add-comment:
---

# Workflow Title

Natural language instructions for the AI agent.
```

## Recompilation Rule

- Edit the **frontmatter** → run `gh aw compile <workflow-id>`.
- Edit the **markdown body** only → no recompilation required.

See also: [workflow-editing.md](workflow-editing.md)

## Core Rules

- Keep the main agent job read-only.
- Use `safe-outputs:` for GitHub writes.
- Prefer `tools.github.mode: gh-proxy` and use `gh` for GitHub reads.
- For non-GitHub MCP servers, prefer `tools.cli-proxy: true` and use mounted `mcp-clis` commands.
- Use `${{ steps.sanitized.outputs.text }}` for untrusted user content.
- Set `strict: true` for production workflows.
- Limit network and bash access to what the workflow actually needs.
- For visual regression workflows, explicitly name the baseline source (for example `cache-memory` key, artifact, or branch path). See [visual-regression.md](visual-regression.md).

## Repository-Specific Instructions

Use `@.github/aw/instructions.md` as the canonical repository-local overlay for workflow authoring standards.

- This file is optional and repository-owned.
- Installed gh-aw agents should load and apply it automatically when present.
- Precedence: apply upstream defaults first, then apply repository overlay rules; when they conflict, repository overlay rules win.

## Trigger Selection Quick Reference

Use the smallest trigger that matches the requested automation.

| Need | Trigger | Notes |
|---|---|---|
| Review pull request changes or UI diffs | `pull_request` | Use for PR-scoped analysis, comments, and optional `playwright`-based visual regression. |
| React to the result of another GitHub Actions workflow | `workflow_run` | Scope `workflows:` explicitly, use `types: [completed]`, and gate conclusions before creating incidents. |
| Publish recurring reports or stakeholder digests | `schedule` | Define the exact reporting window and default to `create-issue`; add `workflow_dispatch` when manual reruns are useful. |
| Run the workflow on demand | `workflow_dispatch` | Use for manual tests, backfills, and operator-invoked runs; often pair with `schedule` or `workflow_run`. |

See also: [workflow-constraints.md](workflow-constraints.md)

## Ad Hoc Scenario Evaluation

Installed gh-aw agents should support scenario evaluation requests that do not create workflow files.

- Treat prompts such as `agentic-workflows evaluate this scenario without creating files` as ad hoc evaluation mode.
- Return a compact design recommendation covering trigger, scope, tools, permissions, safe outputs, `noop` behavior, and any report window / grouping / deduplication requirements.
- Offer to turn the recommendation into `.github/workflows/<workflow-id>.md` only if the user asks to proceed.

### Non-technical persona example (Program Management)

When the request is framed as a PM or stakeholder workflow (for example "weekly product health digest"):

- prefer `schedule: weekly` (or `daily on weekdays` for operational digests) plus `workflow_dispatch` for preview/backfill runs
- read with `github` (`gh-proxy`) and default to `create-issue` for the digest destination
- require an explicit report window, grouping dimensions, and a stable dedup key before creating output
- use `close-older-issues: true` for recurring issue-style digests and call `noop` when the selected window has no qualifying updates

## PR Checks with Linked References

When a PR analysis requires verifying or attaching a linked artifact (design doc, policy link, architecture decision record, or approval), follow this compact pattern:

1. **Read the linked reference** from the PR body or comments (for example, a URL, a markdown link, or an ADR reference token like `ADR-NN`) using `gh pr view`.
2. **Validate the link** — confirm the document exists and is accessible before assessing compliance.
3. **Classify the result**:
   - Link present and satisfies requirement → `add-comment` with a ✅ summary
   - Link present but does not satisfy requirement → `add-comment` flagging the specific gap
   - Link missing → `add-comment` requesting it, or `create-issue` if policy requires a blocking escalation
4. **Call `noop`** when the PR is not in scope (for example `paths:` guard excludes all changed files).

Permissions: `pull-requests: read` only; all writes route through `add-comment` safe output.

## Reference Files

| Topic | File |
|---|---|
| Editing and recompilation rules | [workflow-editing.md](workflow-editing.md) |
| Architectural and security constraints | [workflow-constraints.md](workflow-constraints.md) |
| Common design patterns | [workflow-patterns.md](workflow-patterns.md) |
| Frontmatter schema index | [syntax.md](syntax.md) |
| Safe outputs index | [safe-outputs.md](safe-outputs.md) |
| Trigger patterns | [triggers.md](triggers.md) |
| Context expressions and `{{#if}}` templates | [context.md](context.md) |
| Declarative engine configuration | [configure-agentic-engine.md](configure-agentic-engine.md) |
| CLI commands and MCP equivalents | [cli-commands.md](cli-commands.md) |
| Network configuration | [network.md](network.md) |
| Memory and persistence | [memory.md](memory.md) |
| Imports and shared components | [reuse.md](reuse.md) |
| Sub-agents | [subagents.md](subagents.md) |
| Skills | [skills.md](skills.md) |
| Token cost optimization | [token-optimization.md](token-optimization.md) |
| GitHub MCP server configuration | [github-mcp-server.md](github-mcp-server.md) |
| Campaign and KPI patterns | [campaign.md](campaign.md) |
| Experiments and A/B testing | [experiments.md](experiments.md) |
| Charts and Python data visualization | [charts.md](charts.md) |
| LLM API endpoint discovery | [llms.md](llms.md) |

## Compile Commands

```bash
gh aw compile
gh aw compile <workflow-id>
gh aw compile --purge
gh aw compile --strict
```
