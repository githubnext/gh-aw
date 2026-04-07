---
title: Sandbox Configuration
description: Configure sandbox environments for AI engines including AWF agent container, mounted tools, runtime environments, and MCP Gateway
sidebar:
  order: 1350
disable-agentic-editing: true
---

The `sandbox` field configures sandbox environments for AI engines (coding agents), providing two main capabilities:

1. **Coding Agent Sandbox** - Controls the agent runtime security using AWF (Agent Workflow Firewall)
2. **Model Context Protocol (MCP) Gateway** - Routes MCP server calls through a unified HTTP gateway

## Configuration

### Coding Agent Sandbox

Configure the coding agent sandbox type to control how the AI engine is isolated:

```yaml wrap
# Use AWF (Agent Workflow Firewall) - default
sandbox:
  agent: awf

# Disable coding agent sandbox (firewall only) - use with caution
sandbox:
  agent: false

# Or omit sandbox entirely to use the default (awf)
```

**Default Behavior**

If `sandbox` is not specified in your workflow, it defaults to `sandbox.agent: awf`. The coding agent sandbox is recommended for all workflows.

**Disabling Coding Agent Sandbox**

Setting `sandbox.agent: false` disables only the agent firewall while keeping the MCP gateway enabled. This reduces security isolation and should only be used when necessary. The MCP gateway cannot be disabled and remains active in all workflows.

### MCP Gateway (Experimental)

Route MCP server calls through a unified HTTP gateway:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  mcp:
    port: 8080
    api-key: "${{ secrets.MCP_GATEWAY_API_KEY }}"
```

### Combined Configuration

Use both coding agent sandbox and MCP gateway together:

```yaml wrap
features:
  mcp-gateway: true

sandbox:
  agent: awf
  mcp:
    port: 8080
```

## Coding Agent Sandbox Types

### AWF (Agent Workflow Firewall)

AWF is the default coding agent sandbox that provides network egress control through domain-based access controls. Network permissions are configured through the top-level [`network`](/gh-aw/reference/network/) field.

```yaml wrap
sandbox:
  agent: awf

network:
  firewall: true
  allowed:
    - defaults
    - python
    - "api.example.com"
```

#### Filesystem Access

AWF makes the host filesystem visible inside the container with appropriate permissions:

| Path Type | Mode | Examples |
|-----------|------|----------|
| User paths | Read-write | `$HOME`, `$GITHUB_WORKSPACE`, `/tmp` |
| System paths | Read-only | `/usr`, `/opt`, `/bin`, `/lib` |
| Docker socket | Hidden | `/var/run/docker.sock` (security) |

#### Host Binaries

All host binaries are available without explicit mounts: system utilities, `gh`, language runtimes, build tools, and anything installed via `apt-get` or setup actions. Verify with `which <tool>`.

> [!WARNING]
> Docker socket is hidden for security. Agents cannot spawn containers.

#### Environment Variables

AWF passes all environment variables via `--env-all`. The host `PATH` is captured as `AWF_HOST_PATH` and restored inside the container, preserving setup action tool paths.

> [!NOTE]
> Go's "trimmed" binaries require `GOROOT` - AWF automatically captures it after `actions/setup-go`.

#### Runtime Tools

Setup actions work transparently. Runtimes update `PATH`, which AWF captures and restores inside the container.

```yaml wrap
---
jobs:
  setup:
    steps:
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
---

Use `go build` or `python3` - both are available.
```

## MCP Gateway

The MCP Gateway routes all MCP server calls through a unified HTTP gateway, enabling centralized management, logging, and authentication for MCP tools.

## Feature Flags

Some sandbox features require feature flags:

| Feature | Flag | Description |
|---------|------|-------------|
| MCP Gateway | `mcp-gateway` | Enable MCP gateway routing |

Enable feature flags in your workflow:

```yaml wrap
features:
  mcp-gateway: true
```

## Long Build Times

Repositories with lengthy build or test cycles — C++ codebases, large monorepos, or complex integration suites — can exhaust the default 20-minute job timeout or hit individual tool-call time limits. This section describes how to tune those limits.

### Setting the Job Timeout (`timeout-minutes`)

The `timeout-minutes` frontmatter field sets the maximum wall-clock time for the entire agent job. The default is 20 minutes. For repositories where a full build or test run takes 10 minutes or more, increase this value:

```yaml wrap
---
on: issues

timeout-minutes: 60   # 60-minute budget for the agent job
---

Fix the failing test in the C++ core library.
```

**Recommended values by repository type:**

| Repository type | Typical build time | Suggested `timeout-minutes` |
|-----------------|-------------------|------------------------------|
| Small (scripts, docs) | < 2 min | 20 (default) |
| Medium (Go, Python, Node) | 2–10 min | 30–60 |
| Large (C++, Rust, Java monorepo) | 10–30 min | 60–120 |
| Very large (distributed, full CI) | > 30 min | 120–360 |

GitHub Actions enforces a hard upper limit of 360 minutes (6 hours) for a single job.

`timeout-minutes` also accepts a GitHub Actions expression, making it easy to parameterize in `workflow_call` reusable workflows:

```yaml wrap
on:
  workflow_call:
    inputs:
      job-timeout:
        type: number
        default: 60

---

timeout-minutes: ${{ inputs.job-timeout }}
```

### Concrete Example: 30-Minute Timeout for a C++ Repository

```yaml wrap
---
on:
  issues:
    types: [opened, labeled]

engine: copilot

runs-on: [self-hosted, linux, x64, large]   # fast self-hosted runner
timeout-minutes: 30                          # 30-minute agent budget

tools:
  bash: [":*"]
  timeout: 300                               # 5-minute per-tool-call budget

network:
  allowed:
    - defaults
    - go
    - node
---

Reproduce the bug described in this issue, add a regression test, and fix it.
Build with `cmake --build build -j$(nproc)` and verify with `ctest --output-on-failure`.
```

### Splitting Build and Test into Separate Steps

Instead of relying on a single large timeout, break long workflows into a custom `jobs:` setup step that caches build outputs, then runs the agent on the pre-built workspace:

```yaml wrap
---
on: issues

timeout-minutes: 45

jobs:
  setup:
    steps:
      - name: Restore build cache
        uses: actions/cache@v4
        with:
          path: build/
          key: cpp-build-${{ hashFiles('CMakeLists.txt', 'src/**') }}
          restore-keys: cpp-build-
      - name: Build (if cache miss)
        run: |
          cmake -B build -DCMAKE_BUILD_TYPE=Release
          cmake --build build -j$(nproc)
      - name: Save build cache
        uses: actions/cache/save@v4
        with:
          path: build/
          key: cpp-build-${{ hashFiles('CMakeLists.txt', 'src/**') }}
---

The build artifacts are already in `build/`. Run the failing tests with
`ctest --test-dir build --output-on-failure -R <pattern>` and fix any failures.
```

Pre-building in a setup job ensures the agent's `timeout-minutes` budget is spent on analysis and code changes, not waiting for compilation.

### Per-Tool-Call Timeout (`tools.timeout`)

`tools.timeout` controls the maximum time for any single tool invocation (e.g., a `bash` command or MCP server call), in seconds. Increase this when individual commands — such as a full build or a slow test suite — routinely take longer than the engine default:

```yaml wrap
tools:
  timeout: 600   # 10 minutes per tool call (seconds)
```

Default values vary by engine: Claude uses 60 s, Codex uses 120 s. See [Tool Timeout Configuration](/gh-aw/reference/tools/#tool-timeout-configuration) for details.

### Self-Hosted Runners for Fast Hardware

For repositories where build time exceeds 10 minutes on standard GitHub-hosted runners, self-hosted runners with more CPU cores, faster storage, and pre-warmed dependency caches can dramatically reduce wall-clock time:

```yaml wrap
---
on: issues

runs-on: [self-hosted, linux, x64, large]   # 32-core self-hosted runner
timeout-minutes: 30
---

Run the full test suite and fix any failures.
```

See [Self-Hosted Runners](/gh-aw/guides/self-hosted-runners/) for setup instructions, including Docker and `sudo` requirements.

### Caching Build Artifacts Between Runs

Use `actions/cache` in a custom `jobs.setup` block to persist build artifacts across agentic runs. This avoids redundant compilation and keeps the agent job within tighter time budgets:

```yaml wrap
---
on: issues

timeout-minutes: 30

jobs:
  setup:
    steps:
      - uses: actions/cache@v4
        with:
          path: |
            ~/.gradle/caches
            build/
          key: gradle-${{ hashFiles('**/*.gradle*') }}
          restore-keys: gradle-
      - run: ./gradlew build -x test --parallel
---

Review the failing tests and apply a fix. Build artifacts are pre-cached.
```

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) - Configure network access controls
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration
- [Tools](/gh-aw/reference/tools/) - Configure MCP tools and servers
- [Self-Hosted Runners](/gh-aw/guides/self-hosted-runners/) - Use custom hardware for long-running jobs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/#run-configuration-run-name-runs-on-runs-on-slim-timeout-minutes) - `timeout-minutes` syntax
