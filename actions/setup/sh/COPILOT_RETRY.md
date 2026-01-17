# Copilot CLI Retry Wrapper

## Overview

The `run_copilot_with_retry.sh` script provides automatic retry logic for the GitHub Copilot CLI when it encounters the transient "missing finish_reason" API error. This error occurs when the Copilot backend returns an incomplete response without the required `finish_reason` field.

**Key Features:**
- 🔄 Automatic retry on transient errors
- 📺 Real-time stdout/stderr streaming - feels like Copilot is running directly
- 🔒 Preserves all output and exit codes - errors are never swallowed
- 🎯 Smart detection - only retries specific error conditions

## Problem

The Copilot CLI occasionally fails with:
```
Execution failed: Error: missing finish_reason for choice 0
```

This is a transient backend API error that typically resolves on retry. The script detects this specific error when it occurs within 5 seconds of execution and automatically retries the command.

## Usage

### Basic Usage

```bash
./run_copilot_with_retry.sh copilot --prompt "Your prompt here"
```

### In GitHub Actions Workflows

Replace direct Copilot CLI invocations with the wrapper script:

**Before:**
```yaml
- name: Execute GitHub Copilot CLI
  run: |
    copilot --add-dir /tmp --prompt "$(cat prompt.txt)" 2>&1 | tee output.log
```

**After:**
```yaml
- name: Execute GitHub Copilot CLI
  run: |
    /opt/gh-aw/run_copilot_with_retry.sh copilot --add-dir /tmp --prompt "$(cat prompt.txt)" 2>&1 | tee output.log
```

### With AWF Firewall

When using the AWF (Agentic Workflows Firewall), wrap the entire command:

```bash
./run_copilot_with_retry.sh \
  sudo -E awf --env-all \
    -- copilot --add-dir /tmp --prompt "$(cat prompt.txt)"
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `COPILOT_RETRY_MAX_ATTEMPTS` | 3 | Maximum number of retry attempts |
| `COPILOT_RETRY_DELAY` | 2 | Delay in seconds between retries |

### Example with Custom Configuration

```bash
export COPILOT_RETRY_MAX_ATTEMPTS=5
export COPILOT_RETRY_DELAY=3
./run_copilot_with_retry.sh copilot --prompt "Your prompt"
```

## Behavior

### Output Streaming

The script **streams all output in real-time**:
- ✅ **stdout** is piped through in real-time - you see all Copilot output as it happens
- ✅ **stderr** is piped through in real-time - you see all errors as they occur
- ✅ Both streams are also captured internally for error detection
- ✅ Exit codes are preserved - failed commands still report failure

This means the wrapper is **transparent** - it feels like you're running Copilot directly, with automatic retry on transient errors.

### Retry Logic

The script will retry **only** if ALL of the following conditions are met:

1. ✅ Command fails (non-zero exit code)
2. ✅ Failure occurs within **5 seconds** of execution start
3. ✅ Error output contains the text `"missing finish_reason"`
4. ✅ Maximum retry attempts not yet reached

### No Retry Cases

The script will **NOT** retry if:

- ❌ Command succeeds (exit code 0)
- ❌ Command runs longer than 5 seconds before failing (indicates different issue)
- ❌ Error message doesn't contain "missing finish_reason" (different error)
- ❌ Maximum retry attempts already reached

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Command succeeded |
| 1 | Command failed after all retry attempts |
| 2 | Invalid arguments (no command provided) |

## Testing

Run the included test suite:

```bash
./run_copilot_with_retry_test.sh
```

The test suite validates:
- ✅ Successful command execution
- ✅ Retry on quick finish_reason failures
- ✅ No retry on slow failures
- ✅ No retry on other error types
- ✅ Success after retry
- ✅ Failure after max retries
- ✅ **Both stdout and stderr are properly piped in real-time**
- ✅ Success after retry
- ✅ Failure after max retries

## Integration with Workflow Compilation

To automatically use this script in all Copilot engine workflows, the gh-aw compiler should be updated to:

1. Copy the script to `/opt/gh-aw/` during the setup phase
2. Wrap Copilot CLI execution commands with the retry script

This would be implemented in:
- `pkg/workflow/copilot_engine_execution.go` - Modify Copilot execution step generation
- `actions/setup/` - Ensure script is copied to the execution environment

## Examples

### Example 1: Simple Prompt

```bash
./run_copilot_with_retry.sh copilot --prompt "List files in current directory"
```

Output:
```
[Attempt 1/3] Executing: copilot --prompt List files in current directory
✓ Command succeeded on attempt 1
```

### Example 2: Retry on Failure

```bash
./run_copilot_with_retry.sh copilot --prompt "Complex task"
```

Output (if first attempt fails):
```
[Attempt 1/3] Executing: copilot --prompt Complex task
Execution failed: Error: missing finish_reason for choice 0
⚠ Command failed quickly (2s) - checking for finish_reason error
✗ Detected 'missing finish_reason' error
⟳ Retrying in 2s...
[Attempt 2/3] Executing: copilot --prompt Complex task
✓ Command succeeded on attempt 2
```

## Related

- Issue #4397: Similar fix in Codex CLI for finish_reason handling
- Copilot CLI v0.0.384: Includes "Fixed bug causing model call failures" but transient API errors can still occur
- GitHub Actions retry: This script provides immediate retry without waiting for full workflow retry

## License

Part of the GitHub Agentic Workflows project.
