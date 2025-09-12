# ⚙️ Frontmatter Options for GitHub Agentic Workflows

This guide covers all available frontmatter configuration options for GitHub Agentic Workflows.

## Overview

The YAML frontmatter supports standard GitHub Actions properties plus additional agentic-specific options:

**Standard GitHub Actions Properties:**
- `on`: Trigger events for the workflow
- `permissions`: Required permissions for the workflow
- `run-name`: Name of the workflow run
- `runs-on`: Runner environment for the workflow
- `timeout_minutes`: Workflow timeout
- `concurrency`: Concurrency settings for the workflow
- `env`: Environment variables for the workflow
- `if`: Conditional execution of the workflow
- `steps`: Custom steps for the job

**Properties specific to GitHub Agentic Workflows:**
- `engine`: AI engine configuration (claude/codex) with optional max-turns setting
- `network`: Network access control for AI engines
- `tools`: Available tools and MCP servers for the AI engine  
- `cache`: Cache configuration for workflow dependencies
- `safe-outputs`: [Safe Output Processing](safe-outputs.md) for automatic issue creation and comment posting.

## Trigger Events (`on:`)

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. Here are some common examples:

```yaml
on:
  issues:
    types: [opened]
```

### Stop After Configuration (`stop-after:`)

You can add a `stop-after:` option within the `on:` section as a cost-control measure to automatically disable workflow triggering after a deadline:

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+25h"  # 25 hours from compilation time
```

**Relative time delta (calculated from compilation time):**
```yaml
on:
  issues:
    types: [opened]
  stop-after: "+25h"      # 25 hours from now
```

**Supported absolute date formats:**
- Standard: `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD`
- US format: `MM/DD/YYYY HH:MM:SS`, `MM/DD/YYYY`  
- European: `DD/MM/YYYY HH:MM:SS`, `DD/MM/YYYY`
- Readable: `January 2, 2006`, `2 January 2006`, `Jan 2, 2006`
- Ordinals: `1st June 2025`, `June 1st 2025`, `23rd December 2025`
- ISO 8601: `2006-01-02T15:04:05Z`

**Supported delta units:**
- `d` - days
- `h` - hours
- `m` - minutes

Note that if you specify a relative time, it is calculated at the time of workflow compilation, not when the workflow runs. If you re-compile your workflow, e.g. after a change, the effective stop time will be reset.

### Visual Feedback (`reaction:`)

You can add a `reaction:` option within the `on:` section to enable emoji reactions on the triggering GitHub item (issue, PR, comment, discussion) to provide visual feedback about the workflow status:

```yaml
on:
  issues:
    types: [opened]
  reaction: "eyes"
```

**Available reactions:**
- `+1` (👍)
- `-1` (👎)
- `laugh` (😄)
- `confused` (😕)
- `heart` (❤️)
- `hooray` (🎉)
- `rocket` (🚀)
- `eyes` (👀)

**Enhanced functionality**: When using the `reaction:` feature with command workflows, the system will also automatically edit the triggering comment to include a link to the workflow run. This provides users with immediate feedback and easy access to view the workflow execution. For non-command workflows, only the reaction is added without comment editing.

**Note**: This feature uses inline JavaScript code with `actions/github-script@v7` to add reactions and edit comments, so no additional action files are created in the repository.

**Default behavior** (if no `on:` specified):
```yaml
on:
  # Semi-active agent - triggers frequently and on repository activity
  schedule:
    - cron: "0/10 * * * *"
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches:
      - main
  workflow_dispatch:
```

An additional kind of trigger called `command:` is supported, see [Command Triggers](command-triggers.md) for special `/mention` triggers and context text functionality.

## Permissions (`permissions:`)

The `permissions:` section uses standard GitHub Actions permissions syntax to specify the permissions relevant to the agentic (natural language) part of the execution of the workflow. See [GitHub Actions permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions).

```yaml
# Specific permissions
permissions:
  issues: write
  contents: read
  pull-requests: write

# All permissions
permissions: write-all
permissions: read-all

# No permissions
permissions: {}
```

If you specify any permission, unspecified ones are set to `none`.

## AI Engine (`engine:`)

The `engine:` section specifies which AI engine to use to interpret the markdown section of the workflow, and controls options about how this execution proceeds. Defaults to `claude`.

```yaml
engine: claude  # Default: Claude Code
engine: codex   # Experimental: OpenAI Codex CLI with MCP support
engine: custom  # Custom: Execute user-defined GitHub Actions steps
```

**Engine Override**:
You can override the engine specified in frontmatter using CLI flags:
```bash
gh aw add weekly-research --engine codex
gh aw compile --engine claude
```

Simple format:

```yaml
engine: claude  # or codex or custom
```

Extended format:

```yaml
engine:
  id: claude                        # Required: engine identifier
  version: beta                     # Optional: version of the action
  model: claude-3-5-sonnet-20241022 # Optional: specific LLM model
  max-turns: 5                      # Optional: maximum chat iterations per run
  env:                              # Optional: custom environment variables
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
    DEBUG_MODE: "true"
```

**Fields:**
- **`id`** (required): Engine identifier (`claude`, `codex`)
- **`version`** (optional): Action version (`beta`, `stable`)
- **`model`** (optional): Specific LLM model to use
- **`max-turns`** (optional): Maximum number of chat iterations per run (cost-control option)
- **`env`** (optional): Custom environment variables to pass to the agentic engine as key-value pairs

**Model Defaults:**
- **Claude**: Uses the default model from the claude-code-base-action (typically latest Claude model)
- **Codex**: Defaults to `o4-mini` when no model is specified

## AI Engine (`engine:`)

**Max-turns Cost Control:**

The `max-turns` option is now configured within the engine configuration to limit the number of chat iterations within a single agentic run:

```yaml
engine:
  id: claude
  max-turns: 5
```

**Behavior:**
1. Passes the limit to the AI engine (e.g., Claude Code action)
2. Engine stops iterating when the turn limit is reached
3. Helps prevent runaway chat loops and control costs
4. Only applies to engines that support turn limiting (currently Claude)

**Custom Environment Variables (`env`):**

The `env` option allows you to pass custom environment variables to the agentic engine:

```yaml
engine:
  id: claude
  env:
    - "AWS_REGION=us-west-2"
    - "CUSTOM_API_ENDPOINT: https://api.example.com"  
    - "DEBUG_MODE: true"
```

**Format Options:**
- `KEY=value` - Standard environment variable format
- `KEY: value` - YAML-style format

**Behavior:**
1. Custom environment variables are added to the built-in engine variables
2. For Claude: Variables are passed via the `claude_env` input and GitHub Actions `env` section
3. For Codex: Variables are added to the command-based execution environment
4. Supports secrets and GitHub context variables: `"API_KEY: ${{ secrets.MY_SECRET }}"`
5. Useful for custom configurations like Claude on Amazon Vertex AI

**Use Cases:**
- Configure cloud provider regions: `AWS_REGION=us-west-2`
- Set custom API endpoints: `API_ENDPOINT: https://vertex-ai.googleapis.com`
- Pass authentication tokens: `API_TOKEN: ${{ secrets.CUSTOM_TOKEN }}`
- Enable debug modes: `DEBUG_MODE: true`

## Network Permissions (`network:`)

> This is only supported by the claude engine today.

Control network access for AI engines using the top-level `network` field. If no `network:` permission is specified, it defaults to `network: defaults` which uses a curated allow-list of common development and package manager domains.

### Supported Formats

```yaml
# Default allow-list (basic infrastructure only)
engine:
  id: claude

network: defaults

# Or use ecosystem identifiers + custom domains
engine:
  id: claude

network:
  allowed:
    - defaults              # Basic infrastructure (certs, JSON schema, Ubuntu, etc.)
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Custom domain

# Or allow specific domains only (no ecosystems)
engine:
  id: claude

network:
  allowed:
    - "api.example.com"      # Exact domain match
    - "*.trusted.com"        # Wildcard matches any subdomain (including nested subdomains)

# Or combine defaults with additional domains
engine:
  id: claude

network:
  allowed:
    - "defaults"             # Expands to the full default whitelist
    - "good.com"             # Add custom domain
    - "api.example.org"      # Add another custom domain

# Or deny all network access (empty object)
engine:
  id: claude

network: {}
```

### Security Model

- **Default Allow List**: When no network permissions are specified or `network: defaults` is used, access is restricted to basic infrastructure domains only (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources)
- **Ecosystem Access**: Use ecosystem identifiers like `python`, `node`, `containers` to enable access to specific development ecosystems
- **Selective Access**: When `network: { allowed: [...] }` is specified, only listed domains/ecosystems are accessible
- **No Access**: When `network: {}` is specified, all network access is denied
- **Domain Validation**: Supports exact matches and wildcard patterns (`*` matches any characters including dots, allowing nested subdomains)

### Examples

```yaml
# Default infrastructure only (basic certificates, JSON schema, Ubuntu, etc.)
engine:
  id: claude

network: defaults

# Python development environment
engine:
  id: claude

network:
  allowed:
    - defaults             # Basic infrastructure
    - python              # Python/PyPI ecosystem
    - github              # GitHub domains

# Full-stack development with multiple ecosystems
engine:
  id: claude

network:
  allowed:
    - defaults
    - python
    - node
    - containers
    - dotnet
    - "api.custom.com"    # Custom domain

# Allow all subdomains of a trusted domain
# Note: "*.github.com" matches api.github.com, subdomain.github.com, and even nested.api.github.com
engine:
  id: claude

network:
  allowed:
    - "*.company-internal.com"
    - "public-api.service.com"

# Specific ecosystems only (no basic infrastructure)
engine:
  id: claude

network:
  allowed:
    - "defaults"                    # Expands to full default whitelist
    - java
    - rust
    - "api.mycompany.com"           # Add custom API
    - "*.internal.mycompany.com"    # Add internal services

# Deny all network access (empty object)
engine:
  id: claude

network: {}
```

### Available Ecosystem Identifiers

The `network: { allowed: [...] }` format supports these ecosystem identifiers:

- **`defaults`**: Basic infrastructure (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources)
- **`containers`**: Container registries (Docker Hub, GitHub Container Registry, Quay, etc.)
- **`dotnet`**: .NET and NuGet ecosystem
- **`dart`**: Dart and Flutter ecosystem  
- **`github`**: GitHub domains (api.github.com, github.com, etc.)
- **`go`**: Go ecosystem (golang.org, proxy.golang.org, etc.)
- **`terraform`**: HashiCorp and Terraform ecosystem
- **`haskell`**: Haskell ecosystem (hackage.haskell.org, etc.)
- **`java`**: Java ecosystem (Maven Central, Gradle, etc.)
- **`linux-distros`**: Linux distribution package repositories (Debian, Alpine, etc.)
- **`node`**: Node.js and NPM ecosystem (npmjs.org, nodejs.org, etc.)
- **`perl`**: Perl and CPAN ecosystem
- **`php`**: PHP and Composer ecosystem
- **`playwright`**: Playwright testing framework domains
- **`python`**: Python ecosystem (PyPI, Conda, etc.)
- **`ruby`**: Ruby and RubyGems ecosystem
- **`rust`**: Rust and Cargo ecosystem (crates.io, etc.)
- **`swift`**: Swift and CocoaPods ecosystem

You can mix ecosystem identifiers with specific domain names for fine-grained control:

```yaml
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python ecosystem
    - "api.custom.com"     # Custom domain
    - "*.internal.corp"    # Wildcard domain
```

### Permission Modes

1. **Default allow-list**: Curated list of development domains (default when no `network:` field specified)
   ```yaml
   engine:
     id: claude
     # No network block - defaults to curated allow-list
   ```

2. **Explicit default allow-list**: Curated list of development domains (explicit)
   ```yaml
   engine:
     id: claude

   network: defaults  # Curated allow-list of development domains
   ```

3. **No network access**: Complete network access denial
   ```yaml
   engine:
     id: claude

   network: {}  # Deny all network access
   ```

4. **Specific domains**: Granular access control to listed domains only
   ```yaml
   engine:
     id: claude

   network:
     allowed:
       - "trusted-api.com"
       - "*.safe-domain.org"
   ```

## Safe Outputs Configuration (`safe-outputs:`)

See [Safe Outputs Processing](safe-outputs.md) for automatic issue creation, comment posting and other safe outputs.

## Run Configuration (`run-name:`, `runs-on:`, `timeout_minutes:`)

Standard GitHub Actions properties:
```yaml
run-name: "Custom workflow run name"  # Defaults to workflow name
runs-on: ubuntu-latest               # Defaults to ubuntu-latest
timeout_minutes: 30                  # Defaults to 15 minutes
```

## Concurrency Control (`concurrency:`)

GitHub Agentic Workflows automatically generates enhanced concurrency policies based on workflow trigger types to provide better isolation and resource management. For example, most workflows produce this:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

Different workflow types receive different concurrency groups and cancellation behavior:

| Trigger Type | Concurrency Group | Cancellation | Description |
|--------------|-------------------|--------------|-------------|
| `issues` | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}` | ❌ | Issue workflows include issue number for isolation |
| `pull_request` | `gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number \|\| github.ref }}` | ✅ | PR workflows include PR number with cancellation |
| `discussion` | `gh-aw-${{ github.workflow }}-${{ github.event.discussion.number }}` | ❌ | Discussion workflows include discussion number |
| Mixed issue/PR | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number \|\| github.event.pull_request.number }}` | ✅ | Mixed workflows handle both contexts with cancellation |
| Alias workflows | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number \|\| github.event.pull_request.number }}` | ❌ | Alias workflows handle both contexts without cancellation |
| Other triggers | `gh-aw-${{ github.workflow }}` | ❌ | Default behavior for schedule, push, etc. |

**Benefits:**
- **Better Isolation**: Workflows operating on different issues/PRs can run concurrently
- **Conflict Prevention**: No interference between unrelated workflow executions  
- **Resource Management**: Pull request workflows can cancel previous runs when updated
- **Predictable Behavior**: Consistent concurrency rules based on trigger type

If you need custom concurrency behavior, you can override the automatic generation by specifying your own `concurrency` section in the frontmatter.

## Environment Variables (`env:`)

GitHub Actions standard `env:` syntax:

```yaml
env:
  CUSTOM_VAR: "value"
  SECRET_VAR: ${{ secrets.MY_SECRET }}
```

## Conditional Execution (`if:`)

Standard GitHub Actions `if:` syntax:

```yaml
if: github.event_name == 'push'
```

## Custom Steps (`steps:`)

Add custom steps before the agentic execution step using GitHub Actions standard `steps:` syntax:

```yaml
steps:
  - name: Custom setup
    run: echo "Custom step before agentic execution"
  - uses: actions/setup-node@v4
    with:
      node-version: '18'
```

If no custom steps are specified, a default step to checkout the repository is added automatically.

## Cache Configuration (`cache:`)

Cache configuration using standard GitHub Actions `actions/cache` syntax:

Single cache:
```yaml
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

Multiple caches:
```yaml
cache:
  - key: node-modules-${{ hashFiles('package-lock.json') }}
    path: node_modules
    restore-keys: |
      node-modules-
  - key: build-cache-${{ github.sha }}
    path: 
      - dist
      - .cache
    restore-keys:
      - build-cache-
    fail-on-cache-miss: false
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Alias Triggers](alias-triggers.md) - Special @mention triggers and context text
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
