---
title: Compiler Enterprise Environment Controls
description: Enterprise environment variables injected and managed by the compiler for default guardrails and model overrides
sidebar:
  order: 655
---

Use these variables to set organization- or repository-wide defaults without editing individual workflow frontmatter files.

## Enterprise Control Variables

| Variable | Source | Purpose | Applies when |
| --- | --- | --- | --- |
| `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS` | Compiler process environment | Default AWF `apiProxy.maxEffectiveTokens` budget | `max-effective-tokens` is not set in frontmatter |
| `GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS` | Compiler process environment | Default `max-daily-ai-credits` guardrail | `max-daily-ai-credits` is not set in frontmatter |
| `GH_AW_DEFAULT_MAX_TURNS` | Compiler process environment | Default top-level `max-turns` | `max-turns` is not set in frontmatter and the selected engine supports max-turns |
| `GH_AW_DEFAULT_TIMEOUT_MINUTES` | Compiler process environment | Default top-level `timeout-minutes` | `timeout-minutes` is not set in frontmatter |
| `GH_AW_DEFAULT_DETECTION_MODEL` | Compiler process environment | Default threat-detection model | `safe-outputs.threat-detection.engine.model` is not set |
| `GH_AW_DEFAULT_UTC` | Compiler process environment | Default project home UTC offset for rendered CLI timestamps | `utc` is not set in `.github/workflows/aw.json` |
| `GH_AW_DEFAULT_MODEL_COPILOT` | GitHub Actions `vars.*` at runtime | Default fallback model for Copilot | `GH_AW_MODEL_AGENT_COPILOT` / `GH_AW_MODEL_DETECTION_COPILOT` is unset |
| `GH_AW_DEFAULT_MODEL_CLAUDE` | GitHub Actions `vars.*` at runtime | Default fallback model for Claude | `GH_AW_MODEL_AGENT_CLAUDE` / `GH_AW_MODEL_DETECTION_CLAUDE` is unset |
| `GH_AW_DEFAULT_MODEL_CODEX` | GitHub Actions `vars.*` at runtime | Default fallback model for Codex | `GH_AW_MODEL_AGENT_CODEX` / `GH_AW_MODEL_DETECTION_CODEX` is unset |

Use `gh aw env get` and `gh aw env update` to manage these
variables in batch at repo, org, or enterprise scope. The defaults file uses
`default_`-prefixed keys such as `default_max_effective_tokens`, `default_max_daily_ai_credits`, `default_timeout_minutes`,
`default_model_copilot`, and `default_utc`.

## Project Timezone

By default, the CLI renders timestamps (table output, expiration footers, and the closing messages on expired issues, pull requests, and discussions) using the runner's local clock. Set a project home UTC offset so these times render consistently regardless of where the CLI runs.

Configure the offset per repository with the `utc` field in `.github/workflows/aw.json`:

```json
{
  "utc": "-08:00"
}
```

The value must be a numeric UTC offset in `+HH:MM` or `-HH:MM` form (for example `+00:00`, `+05:30`, or `-08:00`), within the range `-14:00` to `+14:00`. Named timezones and abbreviations are not accepted.

To set an organization- or enterprise-wide default, use the `GH_AW_DEFAULT_UTC` environment variable (or the `default_utc` key managed by `gh aw env`). The repository `aw.json` value takes precedence over this enterprise default.

When neither is configured, timestamp formatting is left unchanged and uses the runner's local time.

## Precedence

For model selection, precedence is:

1. `engine.model` in workflow frontmatter
2. `GH_AW_MODEL_AGENT_*` or `GH_AW_MODEL_DETECTION_*`
3. `GH_AW_DEFAULT_MODEL_*`
4. Built-in compiler fallback

For max effective tokens, precedence is:

1. `max-effective-tokens` in workflow frontmatter
2. `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS`
3. Built-in compiler default

A negative `GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS` disables AWF token steering and
omits the budget limit when frontmatter does not set `max-effective-tokens`.
Positive values also accept `K`/`M` suffixes such as `100M`.

For daily effective-token workflow guardrails, precedence is:

1. `max-daily-ai-credits` in workflow frontmatter
2. `GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS`

When both are unset, the daily guardrail stays disabled. A value of `-1`
explicitly disables the guardrail.
Positive values also accept `K`/`M` suffixes such as `100M`.

For default timeout-minutes, precedence is:

1. `timeout-minutes` in workflow frontmatter
2. `GH_AW_DEFAULT_TIMEOUT_MINUTES`
3. Built-in compiler default

For detection engine selection, precedence is:

1. `safe-outputs.threat-detection.engine` in workflow frontmatter
2. Main workflow engine (`engine`)
3. Built-in compiler default

For detection model selection, precedence is:

1. `safe-outputs.threat-detection.engine.model` in workflow frontmatter
2. `GH_AW_DEFAULT_DETECTION_MODEL`
3. Engine-specific detection defaults

For project timezone (rendered CLI timestamps), precedence is:

1. `utc` in `.github/workflows/aw.json`
2. `GH_AW_DEFAULT_UTC`
3. The runner's local clock (formatting left unchanged)

## Example

Set an org-wide Codex model fallback:

```bash
gh variable set GH_AW_DEFAULT_MODEL_CODEX --org my-org --body "gpt-5.5"
```

Set an org-wide default max-effective-tokens guardrail:

```bash
gh variable set GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS --org my-org --body "15M"
```

```bash
gh variable set GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS --org my-org --body "100M"
```

Set an org-wide default daily workflow ET guardrail:

```bash
gh variable set GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS --org my-org --body "15M"
```

Set compiler process defaults for timeout and max-turns:

```bash
export GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS=15M
export GH_AW_DEFAULT_TIMEOUT_MINUTES=30
export GH_AW_DEFAULT_MAX_TURNS=12
export GH_AW_DEFAULT_DETECTION_MODEL=gpt-5.5-mini
```

Set an org-wide default project timezone (Pacific Standard Time):

```bash
gh variable set GH_AW_DEFAULT_UTC --org my-org --body "-08:00"
```
