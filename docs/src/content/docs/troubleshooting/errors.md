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

## Troubleshooting Tips

1. **Enable verbose output:** Use `--verbose` flag with CLI commands for detailed error information
2. **Validate YAML syntax:** Use online YAML validators or editor extensions
3. **Check file paths:** Ensure all paths are correct and files exist
4. **Review frontmatter schema:** Consult the [frontmatter reference](/gh-aw/reference/frontmatter-full/) for all available options
5. **Compile early:** Run `gh aw compile` frequently to catch errors early
6. **Check logs:** Review GitHub Actions workflow logs for runtime errors

For additional help, see [Common Issues](/gh-aw/troubleshooting/common-issues/) and [Validation Timing](/gh-aw/troubleshooting/validation-timing/).
