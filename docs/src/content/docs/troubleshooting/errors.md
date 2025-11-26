---
title: Error Reference
description: Comprehensive reference of error messages in GitHub Agentic Workflows, including schema validation, compilation, and runtime errors with solutions.
sidebar:
  order: 100
---

This reference documents common error messages encountered when working with GitHub Agentic Workflows, organized by when they occur during the workflow lifecycle.

## Schema Validation Errors

Schema validation errors occur when the workflow frontmatter does not conform to the expected JSON schema. These errors are detected during the compilation process.

:::tip[Typo Detection]
When you make a typo in frontmatter field names, the compiler automatically suggests correct field names using fuzzy matching. Look for "Did you mean" suggestions in error messages to quickly identify and fix common typos like `permisions` → `permissions` or `engnie` → `engine`.
:::

### Frontmatter Not Properly Closed

**Error Message:**

```
frontmatter not properly closed
```

**Cause:** The YAML frontmatter section lacks a closing `---` delimiter, or the delimiters are malformed.

**Solution:** Ensure the frontmatter is enclosed between two `---` lines:

```aw wrap
---
on: push
permissions:
  contents: read
---

# Workflow content
```

### Failed to Parse Frontmatter

**Error Message:**

```
failed to parse frontmatter: [yaml error details]
```

**Cause:** The YAML syntax in the frontmatter is invalid. Common issues include incorrect indentation, missing colons, or invalid characters.

**Solution:** Validate the YAML syntax. Check indentation (use spaces, not tabs), ensure colons are followed by spaces, quote strings with special characters, and verify array/object syntax.

```yaml wrap
# Correct indentation and spacing
on:
  issues:
    types: [opened]
```

### Invalid Field Type

**Error Message:**

```
timeout-minutes must be an integer
```

**Cause:** A field received a value of the wrong type according to the schema.

**Solution:** Use the correct type as specified in the [frontmatter reference](/gh-aw/reference/frontmatter/). For example, use `timeout-minutes: 10` (integer) not `"10"` (string).

### Unknown Property

**Error Message:**

```
Unknown property: permisions. Did you mean 'permissions'?
```

**Cause:** A field name in the frontmatter is not recognized by the schema validator, often due to a typo.

**Solution:** Use the suggested field name from the error message. The compiler uses fuzzy matching to suggest corrections for common typos like `permisions` → `permissions`, `engnie` → `engine`, `toolz` → `tools`, `timeout_minute` → `timeout-minutes`, or `runs_on` → `runs-on`.

### Imports Field Must Be Array

**Error Message:**

```
imports field must be an array of strings
```

**Cause:** The `imports:` field was provided but is not an array of string paths.

**Solution:** Use array syntax for imports:

```yaml wrap
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

**Solution:** Import only one agent file per workflow.

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

**Solution:** Ensure the file exists at the specified path (relative to repository root) and has read permissions.

### Invalid Workflow Specification

**Error Message:**

```
invalid workflowspec: must be owner/repo/path[@ref]
```

**Cause:** When using remote imports, the specification format is incorrect.

**Solution:** Use the correct format: `owner/repo/path[@ref]`, for example `githubnext/gh-aw/.github/workflows/shared/example.md@main`.

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

**Solution:** Use the correct time delta syntax with supported units: `h` (hours, minimum for stop-after), `d` (days), `w` (weeks), `mo` (months). Example: `stop-after: +24h`.

**Error Message:**

```
minute unit 'm' is not allowed for stop-after. Minimum unit is hours 'h'. Use +[hours]h instead of +[minutes]m
```

**Cause:** The `stop-after` field uses minutes (`m`), but the minimum allowed unit is hours.

**Solution:** Convert minutes to hours (round up as needed). For example, use `+2h` instead of `+90m`.

### Time Delta Too Large

**Error Message:**

```
time delta too large: [value] [unit] exceeds maximum of [max]
```

**Cause:** The time delta exceeds the maximum allowed value for the specified unit.

**Solution:** Reduce the time delta or use a larger unit. Maximum values: 12 months, 52 weeks, 365 days, 8760 hours.

### Duplicate Time Unit

**Error Message:**

```
duplicate unit '[unit]' in time delta: +[value]
```

**Cause:** The same time unit appears multiple times in a time delta.

**Solution:** Combine values for the same unit (e.g., `+3d` instead of `+1d2d`).

### Unable to Parse Date-Time

**Error Message:**

```
unable to parse date-time: [value]. Supported formats include: YYYY-MM-DD HH:MM:SS, MM/DD/YYYY, January 2 2006, 1st June 2025, etc
```

**Cause:** The `stop-after` field contains an absolute timestamp that couldn't be parsed.

**Solution:** Use a supported date format like `"2025-12-31 23:59:59"`, `"December 31, 2025"`, or `"12/31/2025"`.

### JQ Not Found

**Error Message:**

```
jq not found in PATH
```

**Cause:** The `jq` command-line tool is required but not available in the environment.

**Solution:** Install `jq` (Ubuntu/Debian: `sudo apt-get install jq`, macOS: `brew install jq`).

### Authentication Errors

**Error Message:**

```
authentication required
```

**Cause:** GitHub CLI authentication is required but not configured.

**Solution:** Authenticate with GitHub CLI (`gh auth login`) or ensure `GITHUB_TOKEN` is available in GitHub Actions environment.

## Engine-Specific Errors

### Manual Approval Invalid Format

**Error Message:**

```
manual-approval value must be a string
```

**Cause:** The `manual-approval:` field in the `on:` section has an incorrect type.

**Solution:** Use a string value, e.g. `manual-approval: "Approve deployment to production"`.

### Invalid On Section Format

**Error Message:**

```
invalid on: section format
```

**Cause:** The `on:` trigger configuration is malformed or contains unsupported syntax.

**Solution:** Verify the trigger configuration follows [GitHub Actions syntax](/gh-aw/reference/triggers/). Valid formats include `on: push`, `on: push: branches: [main]`, or `on: issues: types: [opened, edited]`.

## File Processing Errors

### Failed to Read File

**Error Message:**

```
failed to read file [path]: [details]
```

**Cause:** The file cannot be read due to permissions, missing file, or I/O error.

**Solution:** Verify the file exists, has read permissions, and the disk is not full.

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

**Solution:** Use `gh aw init my-workflow --force` to overwrite.

## Safe Output Errors

### Failed to Parse Existing MCP Config

**Error Message:**

```
failed to parse existing mcp.json: [details]
```

**Cause:** The existing `.vscode/mcp.json` file contains invalid JSON.

**Solution:** Fix the JSON syntax (validate with `cat .vscode/mcp.json | jq .`) or delete the file to regenerate.

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

**Solution:** Remove the conflicting event trigger. The `command:` configuration automatically handles these events. To restrict to specific events, use the `events:` field within the command configuration.

### Strict Mode Network Configuration Required

**Error Message:**

```
strict mode: 'network' configuration is required
```

**Cause:** The workflow is compiled with `--strict` flag but does not include network configuration. Strict mode requires explicit network permissions for security.

**Solution:** Add network configuration: use `network: defaults` (recommended), specify allowed domains explicitly, or deny all network access with `network: {}`.

### Strict Mode Write Permission Not Allowed

**Error Message:**

```
strict mode: write permission 'contents: write' is not allowed
```

**Cause:** The workflow is compiled with `--strict` flag but requests write permissions on `contents`, `issues`, or `pull-requests`. Strict mode enforces read-only operations.

**Solution:** Use `safe-outputs` instead of write permissions. Configure safe outputs like `create-issue` or `create-pull-request` with appropriate options.

### Strict Mode Network Wildcard Not Allowed

**Error Message:**

```
strict mode: wildcard '*' is not allowed in network.allowed domains
```

**Cause:** The workflow uses `*` wildcard in network.allowed domains when compiled with `--strict` flag. Strict mode requires specific domain patterns.

**Solution:** Replace wildcard with specific domains (e.g., `githubusercontent.com` which automatically includes all subdomains) or use `network: defaults`.

### HTTP MCP Tool Missing Required URL Field

**Error Message:**

```
http MCP tool 'my-tool' missing required 'url' field
```

**Cause:** An HTTP-based MCP server configuration is missing the required `url:` field.

**Solution:** Add the required `url:` field to the HTTP MCP server configuration.

### Job Name Cannot Be Empty

**Error Message:**

```
job name cannot be empty
```

**Cause:** A job definition in the workflow has an empty or missing name field.

**Solution:** This is typically an internal error. If you encounter it, report it with your workflow file. The workflow compiler should generate valid job names automatically.

### Unable to Determine MCP Type

**Error Message:**

```
unable to determine MCP type for tool 'my-tool': missing type, url, command, or container
```

**Cause:** An MCP server configuration is missing the required fields to determine its type.

**Solution:** Specify at least one of: `type`, `url`, `command`, or `container`.

### Tool MCP Configuration Cannot Specify Both Container and Command

**Error Message:**

```
tool 'my-tool' mcp configuration cannot specify both 'container' and 'command'
```

**Cause:** An MCP server configuration includes both `container:` and `command:` fields, which are mutually exclusive.

**Solution:** Use either `container:` OR `command:`, not both.

### HTTP MCP Configuration Cannot Use Container

**Error Message:**

```
tool 'my-tool' mcp configuration with type 'http' cannot use 'container' field
```

**Cause:** An HTTP MCP server configuration includes the `container:` field, which is only valid for stdio-based servers.

**Solution:** Remove the `container:` field from HTTP MCP server configurations.

### Strict Mode Custom MCP Server Requires Network Configuration

**Error Message:**

```
strict mode: custom MCP server 'my-server' with container must have network configuration
```

**Cause:** A containerized MCP server lacks network configuration when workflow is compiled with `--strict` flag.

**Solution:** Add network configuration with allowed domains to containerized MCP servers in strict mode.

### Repository Features Not Enabled for Safe Outputs

**Error Message:**

```
workflow uses safe-outputs.create-issue but repository owner/repo does not have issues enabled
```

**Cause:** The workflow uses `safe-outputs.create-issue` but the target repository has issues disabled.

**Solution:** Enable the required repository feature (Settings → General → Features) or use a different safe output type. Note: `create-discussion` requires discussions enabled, `create-issue` requires issues enabled.

### Engine Does Not Support Firewall

**Error Message:**

```
strict mode: engine does not support firewall
```

**Cause:** The workflow specifies network restrictions but uses an engine that doesn't support network firewalling, and strict mode is enabled.

**Solution:** Use an engine with firewall support (e.g., `copilot`), compile without `--strict` flag, or use `network: defaults`.

## Troubleshooting Tips

- Use `--verbose` flag for detailed error information
- Validate YAML syntax and check file paths
- Consult the [frontmatter reference](/gh-aw/reference/frontmatter-full/)
- Run `gh aw compile` frequently to catch errors early
- Use `--strict` flag to catch security issues early
- Test incrementally: add one feature at a time

## Getting Help

If you encounter an error not documented here, search this page (Ctrl+F / Cmd+F) for keywords, review workflow examples in the documentation, enable verbose mode with `gh aw compile --verbose`, or [report issues on GitHub](https://github.com/githubnext/gh-aw/issues). See [Common Issues](/gh-aw/troubleshooting/common-issues/) for additional help.
