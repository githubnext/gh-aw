---
title: Error Reference
description: Comprehensive reference of error messages in GitHub Agentic Workflows, including schema validation, compilation, and runtime errors with solutions.
sidebar:
  order: 100
---

This reference documents common error messages encountered when working with GitHub Agentic Workflows, organized by when they occur during the workflow lifecycle.

## Schema Validation Errors

Schema validation errors occur when the workflow frontmatter does not conform to the expected JSON schema. These errors are detected during the compilation process.

### Frontmatter Not Properly Closed

**Error Message:**
```
frontmatter not properly closed
```

**Cause:** The YAML frontmatter section lacks a closing `---` delimiter, or the delimiters are malformed.

**Solution:** Ensure the frontmatter is enclosed between two `---` lines:

```aw
---
on: push
permissions:
  contents: read
---

# Workflow content
```

**Related:** Frontmatter must start with `---` on the first line and end with `---` before the markdown content begins.

### Failed to Parse Frontmatter

**Error Message:**
```
failed to parse frontmatter: [yaml error details]
```

**Cause:** The YAML syntax in the frontmatter is invalid. Common issues include incorrect indentation, missing colons, or invalid characters.

**Solution:** Validate the YAML syntax. Common fixes include:

- Check indentation (use spaces, not tabs)
- Ensure colons are followed by spaces
- Quote strings containing special characters
- Verify array and object syntax

```yaml
# Incorrect
on:
issues:
  types:[opened]

# Correct
on:
  issues:
    types: [opened]
```

### Invalid Field Type

**Error Message:**
```
timeout_minutes must be an integer
```

**Cause:** A field received a value of the wrong type according to the schema.

**Solution:** Provide the correct type as specified in the [frontmatter reference](/gh-aw/reference/frontmatter/):

```yaml
# Incorrect
timeout_minutes: "10"

# Correct
timeout_minutes: 10
```

### Imports Field Must Be Array

**Error Message:**
```
imports field must be an array of strings
```

**Cause:** The `imports:` field was provided but is not an array of string paths.

**Solution:** Provide an array of import paths:

```yaml
# Incorrect
imports: shared/tools.md

# Correct
imports:
  - shared/tools.md
  - shared/security.md
```

### Multiple Agent Files in Imports

**Error Message:**
```
multiple agent files found in imports: 'file1.md' and 'file2.md'. Only one agent file is allowed per workflow
```

**Cause:** More than one file under `.github/agents/` was included in the imports list.

**Solution:** Import only one agent file per workflow:

```yaml
# Incorrect
imports:
  - .github/agents/agent1.md
  - .github/agents/agent2.md

# Correct
imports:
  - .github/agents/agent1.md
```

## Compilation Errors

Compilation errors occur when the workflow file is being converted to a GitHub Actions YAML workflow (`.lock.yml` file).

### Workflow File Not Found

**Error Message:**
```
workflow file not found: [path]
```

**Cause:** The specified workflow file does not exist at the given path.

**Solution:** Verify the file exists in `.github/workflows/` and the filename is correct. Use `gh aw compile` without arguments to compile all workflows in the directory.

### Failed to Resolve Import

**Error Message:**
```
failed to resolve import 'path': [details]
```

**Cause:** An imported file specified in the `imports:` field could not be found or accessed.

**Solution:** Verify the import path:

- Check the file exists at the specified path
- Ensure the path is relative to the repository root
- Verify file permissions allow reading

```yaml
# Imports are relative to repository root
imports:
  - .github/workflows/shared/tools.md
```

### Invalid Workflow Specification

**Error Message:**
```
invalid workflowspec: must be owner/repo/path[@ref]
```

**Cause:** When using remote imports, the specification format is incorrect.

**Solution:** Use the correct format: `owner/repo/path[@ref]`

```yaml
imports:
  - githubnext/gh-aw/.github/workflows/shared/example.md@main
```

### Section Not Found

**Error Message:**
```
section 'name' not found
```

**Cause:** An attempt to extract a specific section from the frontmatter failed because the section doesn't exist.

**Solution:** Verify the referenced section exists in the frontmatter. This typically occurs during internal processing and may indicate a bug.

## Runtime Errors

Runtime errors occur when the compiled workflow executes in GitHub Actions.

### Time Delta Errors

**Error Message:**
```
invalid time delta format: +[value]. Expected format like +25h, +3d, +1w, +1mo, +1d12h30m
```

**Cause:** The `stop-after` field in the `on:` section contains an invalid time delta format.

**Solution:** Use the correct time delta syntax:

```yaml
on:
  issues:
    types: [opened]
  stop-after: +24h  # Valid: hours, days, weeks, months
```

**Supported units:**
- `h` - hours (minimum unit for stop-after)
- `d` - days
- `w` - weeks
- `mo` - months

**Error Message:**
```
minute unit 'm' is not allowed for stop-after. Minimum unit is hours 'h'. Use +[hours]h instead of +[minutes]m
```

**Cause:** The `stop-after` field uses minutes (`m`), but the minimum allowed unit is hours.

**Solution:** Convert to hours:

```yaml
# Incorrect
stop-after: +90m

# Correct
stop-after: +2h
```

### Time Delta Too Large

**Error Message:**
```
time delta too large: [value] [unit] exceeds maximum of [max]
```

**Cause:** The time delta exceeds the maximum allowed value for the specified unit.

**Solution:** Reduce the time delta or use a larger unit:

- Maximum: 12 months, 52 weeks, 365 days, 8760 hours

```yaml
# Incorrect
stop-after: +400d

# Correct
stop-after: +12mo
```

### Duplicate Time Unit

**Error Message:**
```
duplicate unit '[unit]' in time delta: +[value]
```

**Cause:** The same time unit appears multiple times in a time delta.

**Solution:** Combine values for the same unit:

```yaml
# Incorrect
stop-after: +1d2d

# Correct
stop-after: +3d
```

### Unable to Parse Date-Time

**Error Message:**
```
unable to parse date-time: [value]. Supported formats include: YYYY-MM-DD HH:MM:SS, MM/DD/YYYY, January 2 2006, 1st June 2025, etc
```

**Cause:** The `stop-after` field contains an absolute timestamp that couldn't be parsed.

**Solution:** Use one of the supported date formats:

```yaml
stop-after: "2025-12-31 23:59:59"
# or
stop-after: "December 31, 2025"
# or
stop-after: "12/31/2025"
```

### JQ Not Found

**Error Message:**
```
jq not found in PATH
```

**Cause:** The `jq` command-line tool is required but not available in the environment.

**Solution:** Install `jq` on the system:

```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq
```

### Authentication Errors

**Error Message:**
```
authentication required
```

**Cause:** GitHub CLI authentication is required but not configured.

**Solution:** Authenticate with GitHub CLI:

```bash
gh auth login
```

For GitHub Actions, ensure `GITHUB_TOKEN` or the appropriate token is available:

```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Engine-Specific Errors

### Manual Approval Invalid Format

**Error Message:**
```
manual-approval value must be a string
```

**Cause:** The `manual-approval:` field in the `on:` section has an incorrect type.

**Solution:** Provide a string value:

```yaml
# Incorrect
on:
  manual-approval: true

# Correct
on:
  manual-approval: "Approve deployment to production"
```

### Invalid On Section Format

**Error Message:**
```
invalid on: section format
```

**Cause:** The `on:` trigger configuration is malformed or contains unsupported syntax.

**Solution:** Verify the trigger configuration follows [GitHub Actions syntax](/gh-aw/reference/triggers/):

```yaml
# Valid formats
on: push

# or
on:
  push:
    branches: [main]

# or
on:
  issues:
    types: [opened, edited]
```

## File Processing Errors

### Failed to Read File

**Error Message:**
```
failed to read file [path]: [details]
```

**Cause:** The file cannot be read due to permissions, missing file, or I/O error.

**Solution:** Verify:
- File exists at the specified path
- File permissions allow reading
- Disk is not full or experiencing errors

### Failed to Create Directory

**Error Message:**
```
failed to create .github/workflows directory: [details]
```

**Cause:** The required directory structure cannot be created.

**Solution:** Check file system permissions and available disk space.

### Workflow File Already Exists

**Error Message:**
```
workflow file '[path]' already exists. Use --force to overwrite
```

**Cause:** Attempting to create a workflow that already exists.

**Solution:** Use the `--force` flag to overwrite:

```bash
gh aw init my-workflow --force
```

## Safe Output Errors

### Failed to Parse Existing MCP Config

**Error Message:**
```
failed to parse existing mcp.json: [details]
```

**Cause:** The existing `.vscode/mcp.json` file contains invalid JSON.

**Solution:** Fix the JSON syntax or delete the file to regenerate:

```bash
# Validate JSON
cat .vscode/mcp.json | jq .

# Or remove and regenerate
rm .vscode/mcp.json
```

### Failed to Marshal MCP Config

**Error Message:**
```
failed to marshal mcp.json: [details]
```

**Cause:** Internal error when generating the MCP configuration.

**Solution:** This typically indicates a bug. Report the issue with reproduction steps.

## Top User-Facing Errors

This section documents the most common errors you may encounter when working with GitHub Agentic Workflows.

### Cannot Use Command with Event Trigger

**Error Message:**
```
cannot use 'command' with 'issues' in the same workflow
```

**Cause:** The workflow specifies both a `command:` trigger and a conflicting event like `issues`, `issue_comment`, `pull_request`, or `pull_request_review_comment`. Command triggers automatically handle these events internally.

**Solution:** Remove the conflicting event trigger. The `command:` configuration already includes support for these events:

```yaml
# Incorrect - command conflicts with issues
on:
  command:
    name: bot-helper
  issues:
    types: [opened]

# Correct - command handles issues automatically
on:
  command:
    name: bot-helper
```

**Note:** Command triggers can be restricted to specific events using the `events:` field:

```yaml
on:
  command:
    name: bot-helper
    events: [issues, issue_comment]  # Only active on these events
```

### Strict Mode Network Configuration Required

**Error Message:**
```
strict mode: 'network' configuration is required
```

**Cause:** The workflow is compiled with `--strict` flag but does not include network configuration. Strict mode requires explicit network permissions for security.

**Solution:** Add network configuration to the workflow:

```yaml
# Option 1: Use defaults (recommended for most workflows)
network: defaults

# Option 2: Specify allowed domains explicitly
network:
  allowed:
    - "api.github.com"
    - "*.example.com"

# Option 3: Deny all network access
network: {}
```

**Example:** Complete workflow with network configuration:

```aw
---
on: issues
permissions:
  contents: read
network: defaults
tools:
  github:
    allowed: [list_issues]
---

# Issue Handler

Process issues with network access restricted to defaults.
```

### Strict Mode Write Permission Not Allowed

**Error Message:**
```
strict mode: write permission 'contents: write' is not allowed
```

**Cause:** The workflow is compiled with `--strict` flag but requests write permissions on `contents`, `issues`, or `pull-requests`. Strict mode enforces read-only operations.

**Solution:** Use `safe-outputs` instead of write permissions:

```yaml
# Incorrect - write permissions in strict mode
permissions:
  contents: write
  issues: write

# Correct - use safe-outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    labels: [automation]
  create-pull-request:
    draft: true
```

**Example:** Complete workflow with safe outputs:

```aw
---
on: push
permissions:
  contents: read
  actions: read
network: defaults
safe-outputs:
  create-issue:
    title-prefix: "[analysis] "
    labels: [automated-review]
---

# Code Analysis

Analyze changes and create an issue with findings.
```

### Strict Mode Network Wildcard Not Allowed

**Error Message:**
```
strict mode: wildcard '*' is not allowed in network.allowed domains
```

**Cause:** The workflow uses `*` wildcard in network.allowed domains when compiled with `--strict` flag. Strict mode requires specific domain patterns.

**Solution:** Replace wildcard with specific domains or patterns:

```yaml
# Incorrect
network:
  allowed:
    - "*"

# Correct - use specific domains
network:
  allowed:
    - "api.github.com"
    - "*.githubusercontent.com"
    - "example.com"

# Or use defaults
network: defaults
```

### HTTP MCP Tool Missing Required URL Field

**Error Message:**
```
http MCP tool 'my-tool' missing required 'url' field
```

**Cause:** An HTTP-based MCP server configuration is missing the required `url:` field.

**Solution:** Add the `url:` field to the HTTP MCP server configuration:

```yaml
# Incorrect
mcp-servers:
  my-api:
    type: http
    headers:
      Authorization: "Bearer token"

# Correct
mcp-servers:
  my-api:
    type: http
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer token"
```

**Example:** Complete HTTP MCP configuration:

```aw
---
on: workflow_dispatch
mcp-servers:
  custom-api:
    type: http
    url: "https://api.example.com/v1/mcp"
    headers:
      X-API-Key: "${{ secrets.API_KEY }}"
    allowed:
      - search_data
      - analyze_results
---

# API Integration

Use custom MCP server to process data.
```

### Job Name Cannot Be Empty

**Error Message:**
```
job name cannot be empty
```

**Cause:** A job definition in the workflow has an empty or missing name field.

**Solution:** This is typically an internal error. If you encounter it, report it with your workflow file. The workflow compiler should generate valid job names automatically.

**Workaround:** If using custom jobs in `steps:` configuration, ensure they have valid names:

```yaml
# Incorrect - empty job name would be generated internally
steps:
  "":
    uses: some-action@v1

# Jobs are normally auto-generated; if customizing, ensure valid names
```

### Invalid Time Delta Format

**Error Message:**
```
invalid time delta format: +[value]. Expected format like +25h, +3d, +1w, +1mo, +1d12h30m
```

**Cause:** The `stop-after:` field contains an invalid time delta format.

**Solution:** Use the correct time delta syntax with supported units:

```yaml
# Incorrect formats
stop-after: "24h"     # Missing + prefix
stop-after: "+24"     # Missing unit
stop-after: "+1y"     # Unsupported unit

# Correct formats
stop-after: "+24h"    # 24 hours
stop-after: "+3d"     # 3 days
stop-after: "+2w"     # 2 weeks
stop-after: "+1mo"    # 1 month
stop-after: "+1d12h"  # 1 day and 12 hours
```

**Supported units:**
- `h` - hours
- `d` - days
- `w` - weeks
- `mo` - months

**Example:** Multiple time delta formats:

```aw
---
on:
  workflow_dispatch:
  stop-after: "+2w3d"  # 2 weeks and 3 days
---

# Long Running Task

Task will automatically stop after configured time.
```

### Minute Unit Not Allowed for Stop-After

**Error Message:**
```
minute unit 'm' is not allowed for stop-after. Minimum unit is hours 'h'. Use +2h instead of +90m
```

**Cause:** The `stop-after:` field uses minutes (`m`), but the minimum allowed unit is hours.

**Solution:** Convert minutes to hours:

```yaml
# Incorrect
stop-after: "+90m"

# Correct - convert to hours (round up if needed)
stop-after: "+2h"
```

**Conversion examples:**
- 90 minutes → `+2h` (rounds up)
- 120 minutes → `+2h`
- 30 minutes → `+1h` (rounds up)

### Time Delta Too Large

**Error Message:**
```
time delta too large: 400 days exceeds maximum of 365 days
```

**Cause:** The time delta exceeds the maximum allowed value for the specified unit.

**Solution:** Use a smaller value or larger unit:

**Maximums:**
- Hours: 8,760 (1 year)
- Days: 365 (1 year)
- Weeks: 52 (1 year)
- Months: 12

```yaml
# Incorrect - exceeds maximum
stop-after: "+400d"
stop-after: "+60w"
stop-after: "+15mo"

# Correct - within limits
stop-after: "+365d"
stop-after: "+52w"
stop-after: "+12mo"
```

### Duplicate Time Unit in Time Delta

**Error Message:**
```
duplicate unit 'd' in time delta: +1d2d
```

**Cause:** The same time unit appears multiple times in a time delta expression.

**Solution:** Combine values for the same unit:

```yaml
# Incorrect
stop-after: "+1d2d"
stop-after: "+3h5h"

# Correct
stop-after: "+3d"
stop-after: "+8h"
```

### Unable to Determine MCP Type

**Error Message:**
```
unable to determine MCP type for tool 'my-tool': missing type, url, command, or container
```

**Cause:** An MCP server configuration is missing the required fields to determine its type.

**Solution:** Specify at least one of: `type`, `url`, `command`, or `container`:

```yaml
# Incorrect - missing required fields
mcp-servers:
  my-tool:
    allowed:
      - some_function

# Correct - using type and command
mcp-servers:
  my-tool:
    type: stdio
    command: "node"
    args: ["server.js"]

# Or using container
mcp-servers:
  my-tool:
    container: "myorg/mcp-server:latest"

# Or using HTTP
mcp-servers:
  my-tool:
    type: http
    url: "https://api.example.com/mcp"
```

### Tool MCP Configuration Cannot Specify Both Container and Command

**Error Message:**
```
tool 'my-tool' mcp configuration cannot specify both 'container' and 'command'
```

**Cause:** An MCP server configuration includes both `container:` and `command:` fields, which are mutually exclusive.

**Solution:** Use either `container:` OR `command:`, not both:

```yaml
# Incorrect - both container and command
mcp-servers:
  my-tool:
    container: "myorg/server:latest"
    command: "node"
    args: ["server.js"]

# Correct - use container only
mcp-servers:
  my-tool:
    container: "myorg/server:latest"
    args: ["--port", "8080"]

# Or use command only
mcp-servers:
  my-tool:
    command: "node"
    args: ["server.js"]
```

### HTTP MCP Configuration Cannot Use Container

**Error Message:**
```
tool 'my-tool' mcp configuration with type 'http' cannot use 'container' field
```

**Cause:** An HTTP MCP server configuration includes the `container:` field, which is only valid for stdio-based servers.

**Solution:** Remove the `container:` field from HTTP configurations:

```yaml
# Incorrect - container with HTTP
mcp-servers:
  my-api:
    type: http
    url: "https://api.example.com/mcp"
    container: "myorg/server:latest"

# Correct - HTTP without container
mcp-servers:
  my-api:
    type: http
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
```

### Strict Mode Bash Wildcard Not Allowed

**Error Message:**
```
strict mode: bash wildcard '*' is not allowed - use specific commands instead
```

**Cause:** The workflow uses bash wildcard `*` or `:*` when compiled with `--strict` flag.

**Solution:** Replace wildcards with specific command allowlists:

```yaml
# Incorrect
tools:
  bash:
    - "*"

# Correct - specify exact commands
tools:
  bash:
    - "git status"
    - "git diff"
    - "npm test"
    - "ls -la"
```

**Example:** Complete workflow with specific bash commands:

```aw
---
on: push
permissions:
  contents: read
network: defaults
tools:
  bash:
    - "git --no-pager status"
    - "git --no-pager diff"
    - "npm run lint"
---

# Code Check

Run specific bash commands for validation.
```

### Strict Mode Custom MCP Server Requires Network Configuration

**Error Message:**
```
strict mode: custom MCP server 'my-server' with container must have network configuration
```

**Cause:** A containerized MCP server lacks network configuration when workflow is compiled with `--strict` flag.

**Solution:** Add network configuration to the MCP server:

```yaml
# Incorrect - container without network in strict mode
mcp-servers:
  my-server:
    container: "myorg/server:latest"

# Correct - add network configuration
mcp-servers:
  my-server:
    container: "myorg/server:latest"
    network:
      allowed:
        - "api.example.com"
        - "*.safe-domain.com"
```

### HTTP Transport Not Supported by Engine

**Error Message:**
```
tool 'my-tool' uses HTTP transport which is not supported by engine 'codex' (only stdio transport is supported)
```

**Cause:** The workflow uses an HTTP MCP server with an engine that only supports stdio transport.

**Solution:** Either switch to a stdio-based MCP server or use a different engine that supports HTTP transport:

```yaml
# Option 1: Switch to stdio transport
mcp-servers:
  my-tool:
    type: stdio
    command: "node"
    args: ["server.js"]

# Option 2: Use engine that supports HTTP (e.g., copilot)
engine: copilot
mcp-servers:
  my-tool:
    type: http
    url: "https://api.example.com/mcp"
```

**Engines and HTTP support:**
- ✅ Copilot: Supports HTTP
- ❌ Claude: stdio only
- ❌ Codex: stdio only

### Repository Features Not Enabled for Safe Outputs

**Error Message:**
```
workflow uses safe-outputs.create-issue but repository owner/repo does not have issues enabled
```

**Cause:** The workflow uses `safe-outputs.create-issue` but the target repository has issues disabled.

**Solution:** Enable the required repository feature or remove the safe-outputs configuration:

```yaml
# Option 1: Enable issues in repository settings
# Go to Settings → General → Features → Issues (check the box)

# Option 2: Use a different safe output
safe-outputs:
  create-discussion:  # Use discussions instead
    category: "General"

# Option 3: Remove safe-outputs if not needed
# (remove the safe-outputs section entirely)
```

**Similar errors:**
- `create-discussion` requires discussions enabled
- `add-comment` with `discussion: true` requires discussions enabled

### Engine Does Not Support Firewall

**Error Message:**
```
strict mode: engine 'claude' does not support firewall
```

**Cause:** The workflow specifies network restrictions but uses an engine that doesn't support network firewalling, and strict mode is enabled.

**Solution:** Either use an engine with firewall support or remove network restrictions:

```yaml
# Option 1: Use engine with firewall support
engine: copilot  # Supports firewall
network:
  allowed:
    - "api.github.com"

# Option 2: Remove strict mode (not recommended for security)
# Compile without --strict flag

# Option 3: Use network: defaults with no specific restrictions
network: defaults
```

**Firewall support by engine:**
- ✅ Copilot: Full firewall support
- ❌ Claude: No firewall support (warnings only in non-strict mode)
- ❌ Codex: No firewall support (warnings only in non-strict mode)

## Troubleshooting Tips

1. **Enable verbose output:** Use `--verbose` flag with CLI commands for detailed error information
2. **Validate YAML syntax:** Use online YAML validators or editor extensions
3. **Check file paths:** Ensure all paths are correct and files exist
4. **Review frontmatter schema:** Consult the [frontmatter reference](/gh-aw/reference/frontmatter-full/) for all available options
5. **Compile early:** Run `gh aw compile` frequently to catch errors early
6. **Check logs:** Review GitHub Actions workflow logs for runtime errors
7. **Use strict mode:** Compile with `--strict` flag to catch security issues early
8. **Test incrementally:** Add one feature at a time and compile after each change

## Getting Help

If you encounter an error not documented here:

1. **Search this page:** Use Ctrl+F / Cmd+F to search for keywords from your error message
2. **Check examples:** Review workflow examples in [Research & Planning](/gh-aw/samples/research-planning/), [Triage & Analysis](/gh-aw/samples/triage-analysis/), [Coding & Development](/gh-aw/samples/coding-development/), or [Quality & Testing](/gh-aw/samples/quality-testing/)
3. **Enable verbose mode:** Run `gh aw compile --verbose` for detailed error context
4. **Review validation timing:** See [Validation Timing](/gh-aw/troubleshooting/validation-timing/) to understand when errors occur
5. **Report issues:** If you believe you've found a bug, [report it on GitHub](https://github.com/githubnext/gh-aw/issues)

For additional help, see [Common Issues](/gh-aw/troubleshooting/common-issues/) and [Validation Timing](/gh-aw/troubleshooting/validation-timing/).
