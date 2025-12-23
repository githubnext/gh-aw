# Copilot CLI 0.0.370 --plain-diff Flag Testing

## Overview

This document provides test coverage and usage documentation for the `--plain-diff` flag introduced in Copilot CLI version 0.0.370.

## What is --plain-diff?

The `--plain-diff` flag disables rich diff rendering in Copilot CLI and uses the git-configured diff tool instead. This is useful for:

- **CI/CD Integration**: Parseable diff output without ANSI codes or rich formatting
- **Custom Diff Tools**: Respects `git config diff.tool` settings
- **Logging Systems**: Simplified output for log aggregation
- **Compatibility**: Works with existing git diff configurations

## Usage

### Basic Usage

Add the flag through the engine configuration in your workflow:

```yaml
---
engine:
  id: copilot
  args:
    - --plain-diff
---
```

### Combined with Other Flags

The `--plain-diff` flag can be combined with other Copilot CLI arguments:

```yaml
---
engine:
  id: copilot
  args:
    - --plain-diff
    - --verbose
---
```

## Test Coverage

### Unit Tests

**File**: `pkg/workflow/copilot_plain_diff_test.go`

1. **TestCopilotEnginePlainDiffFlag**
   - Verifies `--plain-diff` flag is included in the Copilot command
   - Tests flag ordering (before `--prompt` argument)
   - Tests combination with other arguments
   - Tests absence when not configured

2. **TestCopilotPlainDiffWithFirewall**
   - Verifies `--plain-diff` works with AWF firewall enabled
   - Ensures flag is preserved in sandboxed environments

3. **TestCopilotPlainDiffExtractEngineConfig**
   - Tests frontmatter parsing of the args field
   - Verifies extraction from YAML configuration
   - Tests with single and multiple arguments

### Integration Test Workflow

**File**: `pkg/cli/workflows/test-copilot-plain-diff.md`

A manual test workflow demonstrating:
- Configuration of the `--plain-diff` flag
- Use cases and expected behavior
- Integration with GitHub Actions environment

## Behavior

### Default (Without --plain-diff)

```bash
copilot --add-dir /tmp/gh-aw/ --log-level all --prompt "Show diff"
```

Produces rich diff output with:
- Syntax highlighting
- Color-coded additions/deletions
- Inline diff context

### With --plain-diff

```bash
copilot --add-dir /tmp/gh-aw/ --log-level all --plain-diff --prompt "Show diff"
```

Produces plain diff output:
- No syntax highlighting
- Uses git-configured diff tool
- Standard unified diff format
- Parseable by scripts and CI tools

## Compatibility

- **Minimum Version**: Copilot CLI 0.0.370
- **Current Default**: 0.0.372 (includes this feature)
- **Firewall Support**: ✅ Works with AWF
- **SRT Support**: ✅ Works with Sandbox Runtime

## Examples

### Example 1: Simple Diff Review

```yaml
---
on: pull_request
engine:
  id: copilot
  args:
    - --plain-diff
safe-outputs:
  add-comment:
    max: 1
---

Review the changes in this PR using plain diff format.
```

### Example 2: CI Integration

```yaml
---
on: push
engine:
  id: copilot
  args:
    - --plain-diff
    - --log-level
    - debug
---

Analyze code changes and report issues using plain diff format
for better CI/CD integration.
```

## Verification

To verify the flag is properly included:

1. Compile your workflow:
   ```bash
   gh aw compile your-workflow.md
   ```

2. Check the lock file for the flag:
   ```bash
   grep "plain-diff" your-workflow.lock.yml
   ```

3. Look for the Copilot command line in the `Execute GitHub Copilot CLI` step

## Related Documentation

- [Copilot CLI Release Notes](https://github.com/github/copilot-cli/releases)
- [Engine Configuration](https://githubnext.github.io/gh-aw/reference/engines/)
- [Custom Arguments](https://githubnext.github.io/gh-aw/reference/engines/#custom-arguments)

## Testing Notes

- The flag is passed through the `Args` field in `EngineConfig`
- Arguments are injected before the `--prompt` parameter
- The flag works in all execution modes (AWF, SRT, and direct)
- No special handling required; standard argument passing
