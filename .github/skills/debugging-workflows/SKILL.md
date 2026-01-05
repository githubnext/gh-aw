---
name: debugging-workflows
description: Guide for debugging GitHub Agentic Workflows - analyzing logs, auditing runs, and troubleshooting issues
---


# Debugging GitHub Agentic Workflows

This skill provides comprehensive guidance for debugging GitHub Agentic Workflows, including scripts to download and analyze workflow logs, audit specific runs, and understand how agentic workflows operate.

## Table of Contents

- [Quick Start](#quick-start)
- [Downloading Workflow Logs](#downloading-workflow-logs)
- [Auditing Specific Runs](#auditing-specific-runs)
- [How Agentic Workflows Work](#how-agentic-workflows-work)
- [Common Issues and Solutions](#common-issues-and-solutions)
- [Advanced Debugging Techniques](#advanced-debugging-techniques)
- [Reference Commands](#reference-commands)

## Quick Start

### Download Logs from Recent Runs

```bash
# Download logs from the last 24 hours
gh aw logs --start-date -1d -o /tmp/workflow-logs

# Download logs for a specific workflow
gh aw logs weekly-research --start-date -1d

# Download logs with JSON output for programmatic analysis
gh aw logs --json
```

### Audit a Specific Run

```bash
# Audit by run ID
gh aw audit 1234567890

# Audit from a GitHub Actions URL
gh aw audit https://github.com/owner/repo/actions/runs/1234567890

# Audit with JSON output
gh aw audit 1234567890 --json
```

## Downloading Workflow Logs

The `gh aw logs` command downloads workflow run artifacts and logs from GitHub Actions for analysis.

### Basic Usage

```bash
# Download logs for all workflows (last 10 runs)
gh aw logs

# Download logs for a specific workflow
gh aw logs <workflow-name>

# Download with custom output directory
gh aw logs -o ./my-logs
```

### Filter Options

```bash
# Filter by date range
gh aw logs --start-date 2024-01-01 --end-date 2024-01-31
gh aw logs --start-date -1w                    # Last week
gh aw logs --start-date -1mo                   # Last month

# Filter by AI engine
gh aw logs --engine copilot
gh aw logs --engine claude
gh aw logs --engine codex

# Filter by count
gh aw logs -c 5                                # Last 5 runs

# Filter by branch/tag
gh aw logs --ref main
gh aw logs --ref feature-xyz

# Filter by run ID range
gh aw logs --after-run-id 1000 --before-run-id 2000

# Filter firewall-enabled runs
gh aw logs --firewall                          # Only firewall-enabled
gh aw logs --no-firewall                       # Only non-firewall
```

### Output Options

```bash
# Generate JSON summary
gh aw logs --json

# Parse agent logs and generate Markdown reports
gh aw logs --parse

# Generate Mermaid tool sequence graph
gh aw logs --tool-graph

# Set download timeout
gh aw logs --timeout 300                       # 5 minute timeout
```

### Downloaded Artifacts

When you run `gh aw logs`, the following artifacts are downloaded for each run:

| File | Description |
|------|-------------|
| `aw_info.json` | Engine configuration and workflow metadata |
| `safe_output.jsonl` | Agent's final output content (when non-empty) |
| `agent_output/` | Agent logs directory |
| `agent-stdio.log` | Agent standard output/error logs |
| `aw.patch` | Git patch of changes made during execution |
| `workflow-logs/` | GitHub Actions job logs (organized by job) |
| `summary.json` | Complete metrics and run data for all runs |

### Example: Analyze Recent Failures

```bash
# Download failed runs from last week
gh aw logs --start-date -1w -o /tmp/debug-logs

# Check the summary for patterns
cat /tmp/debug-logs/summary.json | jq '.runs[] | select(.conclusion == "failure")'
```

## Auditing Specific Runs

The `gh aw audit` command investigates a single workflow run in detail, downloading artifacts, detecting errors, and generating a report.

### Basic Usage

```bash
# Audit by numeric run ID
gh aw audit 1234567890

# Audit from GitHub Actions URL
gh aw audit https://github.com/owner/repo/actions/runs/1234567890

# Audit from job URL (extracts first failing step)
gh aw audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210

# Audit from job URL with specific step
gh aw audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1
```

### Output Options

```bash
# JSON output for programmatic analysis
gh aw audit 1234567890 --json

# Custom output directory
gh aw audit 1234567890 -o ./audit-reports

# Parse agent logs and firewall logs
gh aw audit 1234567890 --parse

# Verbose output
gh aw audit 1234567890 -v
```

### Audit Report Contents

The audit command provides:

- **Error Detection**: Errors and warnings from workflow logs
- **MCP Tool Usage**: Statistics on tool calls by the AI agent
- **Missing Tools**: Tools the agent tried to use but weren't available
- **Execution Metrics**: Duration, token usage, and cost information
- **Safe Output Analysis**: What GitHub operations were attempted

### Example: Investigate a Failed Run

```bash
# Get detailed audit report
gh aw audit 1234567890 --json > audit.json

# Extract key information
cat audit.json | jq '{
  status: .status,
  conclusion: .conclusion,
  errors: .errors,
  missing_tools: .missing_tools,
  tool_usage: .tool_usage
}'
```

## How Agentic Workflows Work

Understanding the workflow architecture helps in debugging.

### Workflow Structure

Agentic workflows use a **markdown + YAML frontmatter** format:

```markdown
---
on:
  issues:
    types: [opened]
permissions:
  issues: write
timeout-minutes: 10
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    labels: [ai-generated]
---

# Workflow Title

Natural language instructions for the AI agent.

Use GitHub context like ${{ github.event.issue.number }}.
```

### Execution Flow

```
1. Trigger Event (issue opened, PR created, schedule, etc.)
     ↓
2. Activation Job
   - Validates permissions
   - Processes safe-inputs
   - Sanitizes context
     ↓
3. AI Agent Job
   - Loads MCP servers and tools
   - Executes AI agent with prompt
   - Agent makes tool calls
   - Agent produces output
     ↓
4. Safe Outputs Job
   - Processes agent output
   - Creates GitHub resources (issues, PRs, etc.)
   - Applies labels, comments
     ↓
5. Completion
   - Workflow summary generated
   - Artifacts uploaded
```

### Key Components

| Component | Purpose | Configuration |
|-----------|---------|---------------|
| **Engine** | AI model to use | `engine: copilot`, `claude`, `codex` |
| **Tools** | APIs available to agent | `tools:` section with MCP servers |
| **Safe-Inputs** | Context passed to agent | `safe-inputs:` with GitHub expressions |
| **Safe-Outputs** | Resources agent can create | `safe-outputs:` with allowed operations |
| **Permissions** | GitHub token permissions | `permissions:` block |
| **Network** | Allowed network access | `network:` with domain/ecosystem lists |

### Compilation Process

```bash
# Compile workflow to GitHub Actions YAML
gh aw compile <workflow-name>

# Result: .github/workflows/<name>.md → .github/workflows/<name>.lock.yml
```

The `.lock.yml` file is the actual GitHub Actions workflow that runs.

## Common Issues and Solutions

### Missing Tool Errors

**Symptoms**:
- Error: "Tool 'github:read_issue' not found"
- Agent cannot access GitHub APIs

**Solution**: Add GitHub MCP server configuration:

```yaml
tools:
  github:
    mode: remote
    toolsets: [default]
```

### Permission Errors

**Symptoms**:
- HTTP 403 (Forbidden) errors
- "Resource not accessible" errors

**Solution**: Add required permissions:

```yaml
permissions:
  contents: read
  issues: write
  pull-requests: write
```

### Safe-Input Errors

**Symptoms**:
- "missing tool configuration for safeinputs-gh"
- Environment variable not available

**Solution**: Configure safe-inputs:

```yaml
safe-inputs:
  issue:
    title: ${{ github.event.issue.title }}
    body: ${{ github.event.issue.body }}
```

### Safe-Output Errors

**Symptoms**:
- Agent tries to create resources but fails
- "Safe output not enabled" errors

**Solution**: Enable safe-outputs:

```yaml
safe-outputs:
  staged: false  # Set to false to actually create resources
  create-issue:
    labels: [ai-generated]
```

### Network Access Errors

**Symptoms**:
- Firewall denials
- URLs appearing as "(redacted)"

**Solution**: Configure network access:

```yaml
network:
  allowed:
    - defaults
    - python    # For PyPI
    - node      # For npm
    - "api.example.com"  # Custom domains
```

### Timeout Errors

**Symptoms**:
- Workflow exceeds time limit
- Agent loops or hangs

**Solution**: Increase timeout or optimize prompt:

```yaml
timeout-minutes: 30  # Increase from default
```

## Advanced Debugging Techniques

### Polling In-Progress Runs

When a run is still executing:

```bash
# Poll until completion
while true; do
  output=$(gh aw audit <run-id> --json 2>&1)
  if echo "$output" | grep -q '"status":.*"\(completed\|failure\|cancelled\)"'; then
    echo "$output"
    break
  fi
  echo "⏳ Run still in progress. Waiting 45 seconds..."
  sleep 45
done
```

### Inspecting MCP Configuration

```bash
# Inspect MCP servers for a workflow
gh aw mcp inspect <workflow-name>

# List all workflows with MCP servers
gh aw mcp list
```

### Checking Workflow Status

```bash
# Show status of all agentic workflows
gh aw status
```

### Downloading Specific Artifacts

```bash
# Download only the agent log artifact
GH_REPO=owner/repo gh run download <run-id> -n agent-stdio.log
```

### Inspecting Job Logs

```bash
# View specific job logs
gh run view <run-id>
gh run view --job <job-id> --log
```

### Analyzing Firewall Logs

```bash
# Parse firewall logs for network issues
gh aw logs --parse

# Check firewall-enabled runs
gh aw logs --firewall
```

### Debug Mode Compilation

```bash
# Compile with verbose output
gh aw compile --verbose

# Compile with strict security checks
gh aw compile --strict

# Run security scanners
gh aw compile --actionlint --zizmor --poutine
```

## Reference Commands

### Log Analysis Commands

| Command | Description |
|---------|-------------|
| `gh aw logs` | Download logs for all workflows |
| `gh aw logs <workflow>` | Download logs for specific workflow |
| `gh aw logs --json` | Output as JSON |
| `gh aw logs --start-date -1d` | Filter by date |
| `gh aw logs --engine copilot` | Filter by engine |
| `gh aw logs --parse` | Generate Markdown reports |

### Audit Commands

| Command | Description |
|---------|-------------|
| `gh aw audit <run-id>` | Audit specific run |
| `gh aw audit <url>` | Audit from GitHub URL |
| `gh aw audit <run-id> --json` | Output as JSON |
| `gh aw audit <run-id> --parse` | Parse logs to Markdown |

### MCP Commands

| Command | Description |
|---------|-------------|
| `gh aw mcp list` | List workflows with MCP servers |
| `gh aw mcp inspect <workflow>` | Inspect MCP configuration |

### Status Commands

| Command | Description |
|---------|-------------|
| `gh aw status` | Show all workflow status |
| `gh aw compile` | Compile all workflows |
| `gh aw compile <workflow>` | Compile specific workflow |
| `gh aw compile --strict` | Compile with security checks |

### Workflow Execution Commands

| Command | Description |
|---------|-------------|
| `gh aw run <workflow>` | Trigger workflow manually |
| `gh workflow run <name>.lock.yml` | Alternative trigger method |
| `gh run watch <run-id>` | Monitor running workflow |

## Additional Resources

- [Workflow Health Monitoring Runbook](../../aw/runbooks/workflow-health.md) - Step-by-step investigation procedures
- [Common Issues Reference](../../../docs/src/content/docs/troubleshooting/common-issues.md) - Frequently encountered issues
- [Error Reference](../../../docs/src/content/docs/troubleshooting/errors.md) - Error codes and solutions
- [GitHub MCP Server Documentation](../../../skills/github-mcp-server/SKILL.md) - Tool configuration reference
