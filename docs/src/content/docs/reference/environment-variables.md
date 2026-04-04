---
title: Environment Variables
description: Reference for all environment variables in GitHub Agentic Workflows — CLI configuration, model overrides, guard policy fallbacks, and workflow-level scope precedence
sidebar:
  order: 650
---

Environment variables in GitHub Agentic Workflows can be defined at multiple scopes, each serving a specific purpose in the workflow lifecycle. Variables defined at more specific scopes override those at more general scopes, following GitHub Actions conventions while adding AWF-specific contexts.

## Environment Variable Scopes

GitHub Agentic Workflows supports environment variables in 13 distinct contexts:

| Scope | Syntax | Context | Typical Use |
|-------|--------|---------|-------------|
| **Workflow-level** | `env:` | All jobs | Shared configuration |
| **Job-level** | `jobs.<job_id>.env` | All steps in job | Job-specific config |
| **Step-level** | `steps[*].env` | Single step | Step-specific config |
| **Engine** | `engine.env` | AI engine | Engine secrets, timeouts |
| **Container** | `container.env` | Container runtime | Container settings |
| **Services** | `services.<id>.env` | Service containers | Database credentials |
| **Sandbox Agent** | `sandbox.agent.env` | Sandbox runtime | Sandbox configuration |
| **Sandbox MCP** | `sandbox.mcp.env` | Model Context Protocol (MCP) gateway | MCP debugging |
| **MCP Tools** | `tools.<name>.env` | MCP server process | MCP server secrets |
| **MCP Scripts** | `mcp-scripts.<name>.env` | MCP script execution | Tool-specific tokens |
| **Safe Outputs Global** | `safe-outputs.env` | All safe-output jobs | Shared safe-output config |
| **Safe Outputs Job** | `safe-outputs.jobs.<name>.env` | Specific safe-output job | Job-specific config |
| **GitHub Actions Step** | `githubActionsStep.env` | Pre-defined steps | Step configuration |

### Example Configurations

**Workflow-level shared configuration:**

```yaml wrap
---
env:
  NODE_ENV: production
  API_ENDPOINT: https://api.example.com
---
```

**Job-specific overrides:**

```yaml wrap
---
jobs:
  validation:
    env:
      VALIDATION_MODE: strict
    steps:
      - run: npm run build
        env:
          BUILD_ENV: production  # Overrides job and workflow levels
---
```

**AWF-specific contexts:**

```yaml wrap
---
# Engine configuration
engine:
  id: copilot
  env:
    OPENAI_API_KEY: ${{ secrets.CUSTOM_KEY }}

# MCP server with secrets
tools:
  database:
    command: npx
    args: ["-y", "mcp-server-postgres"]
    env:
      DATABASE_URL: ${{ secrets.DATABASE_URL }}

# Safe outputs with custom PAT
safe-outputs:
  create-issue:
  env:
    GITHUB_TOKEN: ${{ secrets.CUSTOM_PAT }}
---
```

## Agent Step Summary (`GITHUB_STEP_SUMMARY`)

Agents can write markdown content to the `$GITHUB_STEP_SUMMARY` environment variable to publish a formatted summary visible in the GitHub Actions run view.

Inside the AWF sandbox, `$GITHUB_STEP_SUMMARY` is redirected to a file at `/tmp/gh-aw/agent-step-summary.md`. After agent execution completes, the framework automatically appends the contents of that file to the real GitHub step summary. Secret redaction runs before the content is published.

> [!NOTE]
> The first 2000 characters of the summary are appended. If the content is longer, a `[truncated: ...]` notice is included. Write your most important content first.

Example: an agent writing a brief analysis result to the step summary:

```bash
echo "## Analysis complete" >> "$GITHUB_STEP_SUMMARY"
echo "Found 3 issues across 12 files." >> "$GITHUB_STEP_SUMMARY"
```

The output appears in the **Summary** tab of the GitHub Actions workflow run.

## System-Injected Runtime Variables

GitHub Agentic Workflows automatically injects the following environment variables into every agentic engine execution step (both the main agent run and the threat detection run). These variables are read-only from the agent's perspective and are useful for writing workflows or agents that need to detect their execution context.

| Variable | Value | Description |
|----------|-------|-------------|
| `GITHUB_AW` | `"true"` | Present in every gh-aw engine execution step. Agents can check for this variable to confirm they are running inside a GitHub Agentic Workflow. |
| `GH_AW_PHASE` | `"agent"` or `"detection"` | Identifies which execution phase is active. `"agent"` for the main run; `"detection"` for the threat-detection safety check run that precedes the main run. |
| `GH_AW_VERSION` | e.g. `"0.40.1"` | The gh-aw compiler version that generated the workflow. Useful for conditional logic that depends on a minimum feature version. |

These variables appear alongside other `GH_AW_*` context variables in the compiled workflow:

```yaml
env:
  GITHUB_AW: "true"
  GH_AW_PHASE: agent        # or "detection"
  GH_AW_VERSION: "0.40.1"
  GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt
```

> [!NOTE]
> These variables are injected by the compiler and cannot be overridden by user-defined `env:` blocks in the workflow frontmatter.

## CLI Configuration Variables

These variables configure the `gh aw` CLI tool. Set them in your local shell environment or as repository/organization variables in GitHub Actions.

| Variable | Default | Description |
|---|---|---|
| `DEBUG` | disabled | npm-style namespace debug logging. `DEBUG=*` enables all output; `DEBUG=cli:*,workflow:*` selects specific namespaces. Exclusions are supported: `DEBUG=*,-workflow:test`. Also activated when `ACTIONS_RUNNER_DEBUG=true`. |
| `DEBUG_COLORS` | `1` (enabled) | Set to `0` to disable ANSI colors in debug output. Colors are automatically disabled when output is not a TTY. |
| `ACCESSIBLE` | empty | Any non-empty value enables accessibility mode, which disables spinners and animations. Also enabled when `TERM=dumb` or `NO_COLOR` is set. |
| `NO_COLOR` | empty | Any non-empty value disables colored output and enables accessibility mode. Follows the [no-color.org](https://no-color.org/) standard. |
| `GH_AW_ACTION_MODE` | auto-detected | Overrides how JavaScript is embedded in compiled workflows. Valid values: `dev`, `release`, `script`, `action`. When unset, the CLI auto-detects the appropriate mode. |
| `GH_AW_FEATURES` | empty | Comma-separated list of experimental feature flags to enable globally. Values in workflow `features:` frontmatter take precedence over this variable. |
| `GH_AW_MAX_CONCURRENT_DOWNLOADS` | `10` | Maximum number of parallel log and artifact downloads for `gh aw logs`. Valid range: `1`–`100`. |
| `GH_AW_MCP_SERVER` | unset | When set, disables the automatic update check. Set automatically when `gh aw` runs as an MCP server subprocess — no manual configuration needed. |

**Enabling debug logging:**

```bash
# All namespaces
DEBUG=* gh aw compile

# Specific namespaces
DEBUG=cli:*,workflow:* gh aw compile

# Without colors
DEBUG_COLORS=0 DEBUG=* gh aw compile
```

---

## Model Override Variables

These variables override the default AI model used for agent runs and threat detection. Set them as GitHub Actions repository or organization variables to apply org-wide defaults without modifying workflow frontmatter.

> [!NOTE]
> The `engine.model:` field in workflow frontmatter takes precedence over these variables.

### Agent runs

| Variable | Engine |
|---|---|
| `GH_AW_MODEL_AGENT_COPILOT` | GitHub Copilot |
| `GH_AW_MODEL_AGENT_CLAUDE` | Anthropic Claude |
| `GH_AW_MODEL_AGENT_CODEX` | OpenAI Codex |
| `GH_AW_MODEL_AGENT_GEMINI` | Google Gemini |
| `GH_AW_MODEL_AGENT_CUSTOM` | Custom engine |

### Detection runs

| Variable | Engine |
|---|---|
| `GH_AW_MODEL_DETECTION_COPILOT` | GitHub Copilot |
| `GH_AW_MODEL_DETECTION_CLAUDE` | Anthropic Claude |
| `GH_AW_MODEL_DETECTION_CODEX` | OpenAI Codex |
| `GH_AW_MODEL_DETECTION_GEMINI` | Google Gemini |

Set a model override as an organization variable:

```bash
gh variable set GH_AW_MODEL_AGENT_COPILOT --org my-org --body "gpt-5"
```

See [Engines](/gh-aw/reference/engines/) for available engine identifiers and model configuration options.

---

## Guard Policy Fallback Variables

These variables provide fallback values for guard policy fields when the corresponding `tools.github.*` configuration is absent from workflow frontmatter. Set them as GitHub Actions organization or repository variables to enforce a consistent policy across all workflows.

> [!NOTE]
> Explicit `tools.github.*` values in workflow frontmatter always take precedence over these variables.

| Variable | Frontmatter field | Format | Description |
|---|---|---|---|
| `GH_AW_GITHUB_BLOCKED_USERS` | `tools.github.blocked-users` | Comma- or newline-separated usernames | GitHub usernames blocked from triggering agent runs |
| `GH_AW_GITHUB_APPROVAL_LABELS` | `tools.github.approval-labels` | Comma- or newline-separated label names | Labels that promote content to "approved" integrity for guard checks |
| `GH_AW_GITHUB_TRUSTED_USERS` | `tools.github.trusted-users` | Comma- or newline-separated usernames | GitHub usernames elevated to "approved" integrity, bypassing guard checks |

Set an org-wide blocked user list:

```bash
gh variable set GH_AW_GITHUB_BLOCKED_USERS --org my-org --body "bot-account1,bot-account2"
```

See [Tools Reference](/gh-aw/reference/tools/) for complete guard policy documentation.

---

## Precedence Rules

Environment variables follow a **most-specific-wins** model, consistent with GitHub Actions. Variables at more specific scopes completely override variables with the same name at less specific scopes.

### General Precedence (Highest to Lowest)

1. **Step-level** (`steps[*].env`, `githubActionsStep.env`)
2. **Job-level** (`jobs.<job_id>.env`)
3. **Workflow-level** (`env:`)

### Safe Outputs Precedence

1. **Job-specific** (`safe-outputs.jobs.<job_name>.env`)
2. **Global** (`safe-outputs.env`)
3. **Workflow-level** (`env:`)

### Context-Specific Scopes

These scopes are independent and operate in different contexts: `engine.env`, `container.env`, `services.<id>.env`, `sandbox.agent.env`, `sandbox.mcp.env`, `tools.<tool>.env`, `mcp-scripts.<tool>.env`.

### Override Example

```yaml wrap
---
env:
  API_KEY: default-key
  DEBUG: "false"

jobs:
  test:
    env:
      API_KEY: test-key    # Overrides workflow-level
      EXTRA: "value"
    steps:
      - run: |
          # API_KEY = "test-key" (job-level override)
          # DEBUG = "false" (workflow-level inherited)
          # EXTRA = "value" (job-level)
---
```

## Related Documentation

- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output environment configuration
- [Sandbox](/gh-aw/reference/sandbox/) - Sandbox environment variables
- [Tools](/gh-aw/reference/tools/) - MCP tool configuration and guard policies
- [MCP Scripts](/gh-aw/reference/mcp-scripts/) - MCP script tool configuration
- [Engines](/gh-aw/reference/engines/) - AI engine configuration and model selection
- [Tokens](/gh-aw/reference/tokens/) - Engine secrets and GitHub token reference
- [GitHub Actions Environment Variables](https://docs.github.com/en/actions/learn-github-actions/variables) - GitHub Actions documentation
