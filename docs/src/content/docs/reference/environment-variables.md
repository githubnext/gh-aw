---
title: Environment Variables
description: Complete guide to environment variable precedence and merge behavior across all workflow scopes
sidebar:
  order: 650
---

Environment variables in GitHub Agentic Workflows can be defined at multiple scopes, each serving a specific purpose in the workflow lifecycle. This guide documents the precedence order and merge behavior across all contexts.

## Overview

Environment variables flow through the workflow compilation process with a clear precedence order. Variables defined at more specific scopes override those at more general scopes, following GitHub Actions conventions while adding AWF-specific contexts.

## Environment Variable Scopes

GitHub Agentic Workflows supports environment variables in 13 distinct contexts:

### 1. Workflow-Level (Top-Level `env:`)

Defines environment variables available to all jobs in the workflow.

```yaml wrap
---
env:
  NODE_ENV: production
  API_ENDPOINT: https://api.example.com
---
```

**GitHub Actions equivalent**: `env:` at workflow root
**Scope**: All jobs inherit these variables
**Use case**: Shared configuration across all jobs

### 2. Job-Level (Custom Jobs `env:`)

Environment variables specific to a custom GitHub Actions job.

```yaml wrap
---
jobs:
  validation:
    runs-on: ubuntu-latest
    env:
      VALIDATION_MODE: strict
    steps:
      - run: echo "Running validation"
---
```

**GitHub Actions equivalent**: `jobs.<job_id>.env`
**Scope**: All steps within the specific job
**Precedence**: Overrides workflow-level `env:`

### 3. Step-Level (GitHub Actions Steps `env:`)

Environment variables for individual steps within custom jobs.

```yaml wrap
---
jobs:
  build:
    steps:
      - name: Build
        run: npm run build
        env:
          BUILD_ENV: production
---
```

**GitHub Actions equivalent**: `jobs.<job_id>.steps[*].env`
**Scope**: Single step only
**Precedence**: Overrides job-level and workflow-level `env:`

### 4. Engine Configuration (`engine.env:`)

Custom environment variables passed to the AI engine, including secret overrides.

```yaml wrap
---
engine:
  id: copilot
  env:
    OPENAI_API_KEY: ${{ secrets.CUSTOM_KEY }}
    MODEL_TIMEOUT: "300"
---
```

**AWF-specific**: AI engine configuration
**Scope**: Engine execution context
**Use case**: Engine-specific secrets and configuration

### 5. Container Configuration (`container.env:`)

Environment variables for the top-level container running the workflow.

```yaml wrap
---
container:
  image: node:20
  env:
    DEBIAN_FRONTEND: noninteractive
    LANG: en_US.UTF-8
---
```

**GitHub Actions equivalent**: `jobs.<job_id>.container.env`
**Scope**: Container environment
**Use case**: Container-specific settings

### 6. Services Configuration (`services.<service_id>.env:`)

Environment variables for service containers.

```yaml wrap
---
services:
  postgres:
    image: postgres:15
    env:
      POSTGRES_PASSWORD: ${{ secrets.DB_PASSWORD }}
      POSTGRES_DB: testdb
---
```

**GitHub Actions equivalent**: `jobs.<job_id>.services.<service_id>.env`
**Scope**: Individual service container
**Use case**: Database credentials, service configuration

### 7. Sandbox Agent Configuration (`sandbox.agent.env:`)

Environment variables for the sandbox agent execution step (AWF or SRT).

```yaml wrap
---
sandbox:
  agent:
    env:
      SANDBOX_TIMEOUT: "600"
      DEBUG_MODE: "true"
---
```

**AWF-specific**: Sandbox runtime environment
**Scope**: Agent execution within sandbox
**Use case**: Sandbox-specific configuration

### 8. Sandbox MCP Gateway (`sandbox.mcp.env:`)

Environment variables for the MCP gateway in sandbox mode.

```yaml wrap
---
sandbox:
  mcp:
    env:
      MCP_TIMEOUT: "120"
      MCP_LOG_LEVEL: debug
---
```

**AWF-specific**: MCP gateway configuration
**Scope**: MCP gateway process
**Use case**: MCP debugging and configuration

### 9. MCP Tool Configuration (`tools.<tool_name>.env:`)

Environment variables for custom MCP servers.

```yaml wrap
---
tools:
  custom-api:
    transport: stdio
    command: npx
    args: ["-y", "custom-mcp-server"]
    env:
      API_KEY: ${{ secrets.API_KEY }}
      API_ENDPOINT: https://api.example.com
---
```

**AWF-specific**: MCP server configuration
**Scope**: Individual MCP server process
**Use case**: MCP server secrets and configuration

### 10. Safe Inputs (`safe-inputs.<tool_name>.env:`)

Environment variables for safe-input tools (custom inline tools).

```yaml wrap
---
safe-inputs:
  search-issues:
    prompt: Search for issues
    script: |
      gh issue list --search "$INPUT_QUERY"
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---
```

**AWF-specific**: Safe-input tool execution
**Scope**: Individual safe-input tool execution
**Use case**: Tool-specific secrets (typically GitHub tokens)

### 11. Safe Outputs Global (`safe-outputs.env:`)

Environment variables passed to all safe-output jobs.

```yaml wrap
---
safe-outputs:
  create-issue:
  add-comment:
  env:
    GITHUB_TOKEN: ${{ secrets.CUSTOM_PAT }}
    DEBUG_MODE: "true"
---
```

**AWF-specific**: All safe-output operations
**Scope**: All safe-output jobs
**Use case**: Shared safe-output configuration

### 12. Safe Outputs Job-Specific (`safe-outputs.jobs.<job_name>.env:`)

Environment variables for specific safe-output jobs in custom job configurations.

```yaml wrap
---
safe-outputs:
  jobs:
    custom-notification:
      steps:
        - name: Send notification
          run: ./notify.sh
      env:
        SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
---
```

**AWF-specific**: Individual safe-output job
**Scope**: Specific safe-output job
**Precedence**: Overrides `safe-outputs.env:`

### 13. GitHub Actions Step (`githubActionsStep.env:`)

Environment variables for pre-defined GitHub Actions steps in the schema.

```yaml wrap
---
steps:
  - name: Custom step
    uses: actions/github-script@v7
    env:
      CUSTOM_VAR: value
---
```

**GitHub Actions equivalent**: Step-level env in custom steps
**Scope**: Individual step
**Use case**: Step-specific configuration

## Precedence Rules

Environment variables follow a **most-specific-wins** precedence model, consistent with GitHub Actions:

### General Precedence Order (Highest to Lowest)

1. **Step-level** (`steps[*].env`, `githubActionsStep.env`) - Most specific
2. **Job-level** (`jobs.<job_id>.env`) - Job scope
3. **Workflow-level** (`env:`) - Global scope

### Safe Outputs Precedence

For safe-output operations:

1. **Safe-outputs job-specific** (`safe-outputs.jobs.<job_name>.env`) - Highest
2. **Safe-outputs global** (`safe-outputs.env:`) - Shared across safe-outputs
3. **Workflow-level** (`env:`) - Fallback

### Context-Specific Scopes

These scopes are independent and don't override each other since they operate in different contexts:

- **Engine configuration** (`engine.env`) - AI engine process
- **Container** (`container.env`) - Container environment
- **Services** (`services.<id>.env`) - Service containers
- **Sandbox** (`sandbox.agent.env`, `sandbox.mcp.env`) - Sandbox runtime
- **MCP tools** (`tools.<tool>.env`) - MCP server processes
- **Safe inputs** (`safe-inputs.<tool>.env`) - Safe-input tool execution

## Merge Behavior

### Override Strategy

AWF uses an **override** (replace) strategy, not a merge strategy:

- Variables at a more specific scope completely override variables with the same name at a less specific scope
- No deep merging or concatenation occurs
- Each scope maintains its own independent set of variables

### Example: Override Behavior

```yaml wrap
---
env:
  API_KEY: default-key
  DEBUG: "false"

jobs:
  test:
    env:
      API_KEY: test-key    # Overrides workflow-level API_KEY
      EXTRA: "value"       # New variable
    steps:
      - run: |
          # API_KEY = "test-key" (from job level)
          # DEBUG = "false" (inherited from workflow level)
          # EXTRA = "value" (from job level)
---
```

### Example: Safe Outputs Override

```yaml wrap
---
safe-outputs:
  create-issue:
  env:
    GITHUB_TOKEN: ${{ secrets.GLOBAL_PAT }}
    TIMEOUT: "300"
  jobs:
    custom-task:
      env:
        GITHUB_TOKEN: ${{ secrets.TASK_PAT }}  # Overrides global
        RETRY: "3"                              # New variable
      steps:
        - run: |
            # GITHUB_TOKEN = ${{ secrets.TASK_PAT }} (job-specific)
            # TIMEOUT = "300" (inherited from safe-outputs.env)
            # RETRY = "3" (job-specific)
---
```

## Common Patterns

### Pattern 1: Shared Configuration with Job-Specific Overrides

```yaml wrap
---
env:
  NODE_ENV: production
  LOG_LEVEL: info

jobs:
  test:
    env:
      NODE_ENV: test        # Override for testing
      TEST_TIMEOUT: "5000"  # Test-specific
---
```

### Pattern 2: Secret Management in Safe Outputs

```yaml wrap
---
safe-outputs:
  create-issue:
  create-pull-request:
  env:
    GITHUB_TOKEN: ${{ secrets.CUSTOM_PAT }}  # Shared PAT for all operations
---
```

### Pattern 3: Engine-Specific Configuration

```yaml wrap
---
engine:
  id: copilot
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_KEY }}
    MODEL_TIMEOUT: "600"
---
```

### Pattern 4: MCP Server with Secrets

```yaml wrap
---
tools:
  database:
    transport: stdio
    command: npx
    args: ["-y", "mcp-server-postgres"]
    env:
      DATABASE_URL: ${{ secrets.DATABASE_URL }}
---
```

## Best Practices

### 1. Use Secrets for Sensitive Data

Always use GitHub Actions secrets for sensitive values:

```yaml wrap
# ✅ Correct
env:
  API_KEY: ${{ secrets.API_KEY }}

# ❌ Incorrect - never hardcode secrets
env:
  API_KEY: "sk-1234567890abcdef"
```

### 2. Minimize Scope

Define variables at the narrowest scope needed:

```yaml wrap
# ✅ Good - job-specific variable
jobs:
  build:
    env:
      BUILD_MODE: production

# ❌ Less optimal - unnecessarily global
env:
  BUILD_MODE: production
```

### 3. Document Environment Requirements

Comment complex environment setups:

```yaml wrap
---
# Engine requires custom OpenAI key for extended context
engine:
  env:
    OPENAI_API_KEY: ${{ secrets.EXTENDED_CONTEXT_KEY }}
---
```

### 4. Use Consistent Naming

Follow conventions for environment variable names:

- `SCREAMING_SNAKE_CASE` for all environment variables
- Descriptive names: `API_KEY` not `KEY`
- Prefix with service name: `POSTGRES_PASSWORD`, `REDIS_PORT`

## GitHub Actions Integration

### How AWF Compiles Environment Variables

During compilation, AWF:

1. **Extracts** environment variables from frontmatter
2. **Preserves** GitHub Actions expressions (`${{ ... }}`)
3. **Renders** to the appropriate scope in `.lock.yml` files
4. **Validates** secret syntax (must use `${{ secrets.NAME }}`)

### Generated Lock File Structure

```yaml
# Workflow-level (if defined in frontmatter)
env:
  SHARED_VAR: value

jobs:
  agent:
    # Job-level (AWF-managed)
    env:
      GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl
      CUSTOM_VAR: ${{ secrets.CUSTOM_SECRET }}
    
    steps:
      - name: Execute
        # Step-level (if defined)
        env:
          STEP_VAR: value
```

## Debugging Environment Variables

### View Available Variables

In custom jobs or steps:

```yaml wrap
jobs:
  debug:
    steps:
      - name: Show environment
        run: env | sort
```

### Check Variable Precedence

Test precedence by defining the same variable at multiple scopes:

```yaml wrap
---
env:
  TEST_VAR: workflow

jobs:
  test:
    env:
      TEST_VAR: job
    steps:
      - run: echo "TEST_VAR is $TEST_VAR"  # Outputs: "job"
---
```

## Related Documentation

- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output environment configuration
- [Sandbox](/gh-aw/reference/sandbox/) - Sandbox environment variables
- [Tools](/gh-aw/reference/tools/) - MCP tool configuration
- [Safe Inputs](/gh-aw/reference/safe-inputs/) - Safe input tool configuration
- [GitHub Actions Environment Variables](https://docs.github.com/en/actions/learn-github-actions/variables) - GitHub Actions documentation
