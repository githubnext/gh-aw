---
title: Validation Timing
description: Understand when validation and errors occur during the GitHub Agentic Workflows lifecycle, from authoring to runtime execution.
sidebar:
  order: 300
---

GitHub Agentic Workflows validates workflows at three distinct stages: during authoring, at compilation time, and during runtime execution. Understanding when each type of validation occurs helps identify and fix errors more efficiently.

## Validation Stages Overview

| Stage | When | What is Validated | Tool/Process |
|-------|------|-------------------|--------------|
| **Schema Validation** | Compilation | Frontmatter structure and types | `gh aw compile` |
| **Compilation Validation** | Compilation | File resolution, imports, and workflow generation | `gh aw compile` |
| **Runtime Validation** | Execution | GitHub Actions syntax, permissions, and engine operations | GitHub Actions |

## Schema Validation (Compile Time)

Schema validation occurs when running `gh aw compile` and validates the workflow frontmatter against the defined JSON schema.

### What is Checked

**Frontmatter Structure:**
- YAML syntax is valid
- All delimiters (`---`) are present
- Fields match the expected schema

**Field Types:**
- Strings are strings (`engine: "copilot"`)
- Numbers are numbers (`timeout_minutes: 10`)
- Booleans are booleans (`strict: true`)
- Arrays are arrays (`imports: [...]`)
- Objects are objects (`tools: {...}`)

**Required Fields:**
- `on:` trigger configuration is present
- Engine-specific required fields exist

**Enum Values:**
- Fields with fixed value sets use valid values
- Examples: `engine: copilot`, `state: open`

**Field Constraints:**
- Numeric ranges (e.g., `timeout_minutes: 1-360`)
- String patterns (e.g., time delta format)

### When Schema Validation Runs

```bash
# Explicit compilation
gh aw compile

# Compile specific workflow
gh aw compile my-workflow

# Compile with verbose output
gh aw compile --verbose
```

**Timing:** Immediately when the command executes, before any file I/O or transformation.

### Example Schema Errors

**Invalid YAML Syntax:**
```aw
---
on:
issues:  # Missing indentation
  types: [opened]
---
```

**Error:** `failed to parse frontmatter: yaml: line X: mapping values are not allowed in this context`

**Wrong Field Type:**
```aw
---
on: push
timeout_minutes: "10"  # String instead of number
---
```

**Error:** `timeout_minutes must be an integer`

**Invalid Enum Value:**
```aw
---
on: push
engine: gpt4  # Not a valid engine ID
---
```

**Error:** `engine must be one of: copilot, claude, codex, custom`

## Compilation Validation (Compile Time)

Compilation validation occurs during the transformation of the `.md` file into a `.lock.yml` GitHub Actions workflow file.

### What is Checked

**File Resolution:**
- Source workflow file exists
- Import files can be found and read
- Custom agent files are accessible

**Import Processing:**
- Import paths are valid
- Imported files have valid frontmatter
- No circular import dependencies

**Workflow Generation:**
- Engine configuration is complete
- Tool configurations are valid
- Safe outputs can be processed
- MCP server configurations are correct

**Expression Validation:**
- Context expressions use allowed variables only
- Expression syntax is valid

**Cross-File References:**
- Shared components resolve correctly
- Remote workflow specifications are valid

### When Compilation Validation Runs

Compilation validation runs after schema validation passes:

```bash
gh aw compile
```

**Timing:** After schema validation, during the workflow transformation process.

### Example Compilation Errors

**Import Not Found:**
```aw
---
on: push
imports:
  - shared/missing-file.md  # File doesn't exist
---
```

**Error:** `failed to resolve import 'shared/missing-file.md': file not found`

**Multiple Agent Files:**
```aw
---
on: push
imports:
  - .github/agents/agent1.md
  - .github/agents/agent2.md
---
```

**Error:** `multiple agent files found in imports: 'agent1.md' and 'agent2.md'. Only one agent file is allowed per workflow`

**Unauthorized Expression:**
```aw
---
on: push
---

Access secret: ${{ secrets.MY_SECRET }}
```

**Error:** Compilation fails due to unauthorized expression `${{ secrets.MY_SECRET }}`

## Runtime Validation (Execution Time)

Runtime validation occurs when the compiled workflow executes in GitHub Actions.

### What is Checked

**GitHub Actions Syntax:**
- The generated `.lock.yml` is valid GitHub Actions YAML
- Job dependencies are correct
- Step configurations are valid

**Permissions:**
- Token has required permissions for operations
- Repository settings allow the operations

**Environment:**
- Required tools are available (jq, git, etc.)
- Environment variables are set
- Secrets are accessible

**Engine Operations:**
- AI engine is accessible and authenticated
- MCP servers can connect
- Network access is available (if needed)

**Dynamic Conditions:**
- Trigger conditions match event
- Expressions evaluate correctly
- File paths resolve

**Safe Outputs:**
- Output format is correct
- Target repositories are accessible
- Branch protections allow operations

### When Runtime Validation Occurs

Runtime validation happens in GitHub Actions during workflow execution:

1. **Workflow Trigger:** Event matches `on:` configuration
2. **Job Start:** Environment setup and authentication
3. **Step Execution:** Each step validates its preconditions
4. **Tool Invocation:** MCP servers and tools validate inputs
5. **Safe Output Processing:** Post-processing jobs validate outputs

**Timing:** Continuous throughout the workflow run, as each component executes.

### Example Runtime Errors

**Missing Tool:**
```bash
jq not found in PATH
```

**Cause:** The `jq` command is required but not installed in the runner environment.

**Permission Denied:**
```
Error: Resource not accessible by integration
```

**Cause:** `GITHUB_TOKEN` lacks the required permission (e.g., `issues: write`).

**Authentication Required:**
```
Error: authentication required
```

**Cause:** GitHub CLI authentication failed or token is invalid.

**Time Delta Validation:**
```
Error: invalid time delta format: +90m. Expected format like +25h, +3d
```

**Cause:** `stop-after: +90m` uses minutes, but minimum unit is hours.

**Network Connection Failure:**
```
Error: failed to connect to MCP server at https://example.com
```

**Cause:** MCP server is unreachable or network access is blocked.

## Validation Flow Diagram

```
┌─────────────────┐
│ Write Workflow  │
│   (.md file)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   gh aw compile │ ◄─── Command executed
└────────┬────────┘
         │
         ▼
┌─────────────────────┐
│ Schema Validation   │ ◄─── Frontmatter structure & types
├─────────────────────┤
│ • YAML syntax       │
│ • Field types       │
│ • Required fields   │
│ • Enum values       │
└────────┬────────────┘
         │
         ▼
    [PASS/FAIL]
         │
         ▼ (if pass)
┌──────────────────────┐
│ Compilation          │ ◄─── File resolution & transformation
│ Validation           │
├──────────────────────┤
│ • Import resolution  │
│ • Agent files        │
│ • Tool configs       │
│ • Expression checks  │
└────────┬─────────────┘
         │
         ▼
    [PASS/FAIL]
         │
         ▼ (if pass)
┌──────────────────────┐
│ Generate .lock.yml   │ ◄─── Output file created
└────────┬─────────────┘
         │
         ▼
┌──────────────────────┐
│ Commit & Push        │
└────────┬─────────────┘
         │
         ▼
┌──────────────────────┐
│ GitHub Actions       │ ◄─── Workflow triggered
│ Workflow Triggered   │
└────────┬─────────────┘
         │
         ▼
┌──────────────────────┐
│ Runtime Validation   │ ◄─── Execution-time checks
├──────────────────────┤
│ • GH Actions syntax  │
│ • Permissions        │
│ • Environment setup  │
│ • Tool availability  │
│ • Engine operations  │
│ • Safe outputs       │
└────────┬─────────────┘
         │
         ▼
    [SUCCESS/FAILURE]
```

## Best Practices for Each Stage

### Schema Validation Best Practices

1. **Validate early:** Run `gh aw compile` after each significant change
2. **Use verbose mode:** Add `--verbose` flag to see detailed validation messages
3. **Check types:** Ensure numbers are not quoted, booleans are true/false
4. **Verify enums:** Consult the [frontmatter reference](/gh-aw/reference/frontmatter-full/) for valid values
5. **Format YAML:** Use consistent indentation (2 spaces) and proper YAML syntax

### Compilation Validation Best Practices

1. **Organize imports:** Keep import files in a consistent location
2. **Test imports:** Verify imported files compile independently
3. **Limit agent files:** Use only one agent file per workflow
4. **Check expressions:** Use only [allowed context expressions](/gh-aw/reference/templating/)
5. **Review lock files:** Inspect generated `.lock.yml` to verify correctness

### Runtime Validation Best Practices

1. **Test locally:** Use `gh aw compile` to catch issues before pushing
2. **Check permissions:** Verify `permissions:` section matches operations
3. **Verify secrets:** Ensure required secrets are set in repository settings
4. **Monitor logs:** Review GitHub Actions logs for runtime errors
5. **Use safe outputs:** Prefer safe outputs over direct write permissions
6. **Set timeouts:** Configure appropriate `timeout_minutes` for task complexity

## Debugging by Stage

### Schema Errors

**Symptoms:** Compilation fails immediately with YAML or type errors.

**Debug Steps:**
1. Run `gh aw compile --verbose`
2. Check the error line number in frontmatter
3. Validate YAML syntax with online validator
4. Consult the [frontmatter schema reference](/gh-aw/reference/frontmatter-full/)

### Compilation Errors

**Symptoms:** Schema validation passes but compilation fails.

**Debug Steps:**
1. Check import file paths and verify files exist
2. Review error message for specific file or configuration issue
3. Test imported files independently
4. Validate expression usage against allowed list

### Runtime Errors

**Symptoms:** Workflow compiles but fails during execution.

**Debug Steps:**
1. Review GitHub Actions workflow logs
2. Check permissions and token access
3. Verify environment has required tools
4. Test MCP server connectivity
5. Download logs with `gh aw logs` for detailed analysis

## Error Recovery Workflow

When an error occurs:

1. **Identify the stage:** Determine if error is schema, compilation, or runtime
2. **Read the message:** Error messages indicate the specific problem
3. **Consult references:** Use this guide and [error reference](/gh-aw/troubleshooting/errors/)
4. **Fix and revalidate:** Make corrections and run `gh aw compile` again
5. **Test incrementally:** Compile after each fix to isolate remaining issues

## Related Resources

- [Error Reference](/gh-aw/troubleshooting/errors/) - Detailed error messages and solutions
- [Common Issues](/gh-aw/troubleshooting/common-issues/) - Frequently encountered problems
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Complete frontmatter options
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Workflow file format
- [Templating](/gh-aw/reference/templating/) - Allowed context expressions
