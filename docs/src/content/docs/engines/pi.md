---
title: Using Pi with GitHub Agentic Workflows
description: Select and authenticate the Pi AI engine for GitHub Agentic Workflows (gh-aw), configure its required proxies, understand its limitations, and start from an example.
---

GitHub Agentic Workflows (`gh-aw`) includes Pi as a provider-agnostic AI engine. GitHub Actions runs Pi from the same Markdown workflow format as the stable engines, but Pi has additional tool requirements and selects authentication from the provider prefix in `model:`.

## Selection and authentication

Set `engine: pi`. A model without a provider prefix uses the Copilot backend; an explicit `provider/model` value selects Copilot, Anthropic, or OpenAI/Codex authentication.

| `model:` prefix | Authentication |
| --- | --- |
| `copilot/` or `github-copilot/` | [`copilot-requests: write`](/gh-aw/reference/auth/#copilot-requests-write-permission) or [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) |
| `anthropic/` | [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key) |
| `openai/` or `codex/` | `CODEX_API_KEY` or [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key) |

Pi does not provide native MCP server integration, so the compiler automatically derives CLI GitHub access (`tools.github.mode: cli`) and CLI MCP exposure (`tools.mcp-mode: cli`). Do not select `mcp-local` or `mcp-remote` for Pi, and do not add either derived field unless another workflow requirement makes it necessary.

## Example: scheduled repository report

```aw wrap title=".github/workflows/pi-status.md"
---
on:
  schedule: weekly

permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write

engine:
  id: pi
  model: copilot/gpt-5.4

safe-outputs:
  create-issue:
    title-prefix: "[status] "
    labels: [report]
---

# Weekly Repository Status

Analyze open issues, recent pull requests, and current blockers.
Create one concise status issue with prioritized follow-up actions.
```

For a production example, see the [`unbloat-docs` Pi workflow](https://github.com/github/gh-aw/blob/main/.github/workflows/unbloat-docs.md).

## Capabilities and limitations

Pi supports top-level `max-turns`, provider-prefixed models, and `engine.extensions`. Pi already runs in bare mode, so `engine.bare: true` is accepted but has no effect. Pi does not provide native MCP server integration, native `tools.web-search`, per-command bash allowlisting, `max-continuations`, native `engine.agent` selection, or custom `engine.harness` scripts. MCP-backed tools are automatically exposed through CLI wrappers. See [Security Profile Selection](/gh-aw/reference/security-profiles/#github-access-profiles).

See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison) and [Pi extensions reference](/gh-aw/reference/engines/#pi-extensions-extensions).

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [Authentication](/gh-aw/reference/auth/)
- [Security architecture](/gh-aw/introduction/architecture/)
- [Examples by task](/gh-aw/examples/)
