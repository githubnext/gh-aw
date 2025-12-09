# Custom Action References Implementation

## Overview

This implementation adds support for generating custom action references in compiled workflows instead of embedding JavaScript inline via `actions/github-script`. This enables a development mode where workflows can reference local actions (e.g., `./actions/create-issue`) that are tested and validated before being published.

## What Was Implemented

### 1. Action Mode Type (`pkg/workflow/action_mode.go`)
- Added `ActionMode` enum type with two modes:
  - `ActionModeInline`: Embeds JavaScript inline using `actions/github-script` (default, backward compatible)
  - `ActionModeCustom`: References custom actions using local paths
- Added validation methods `IsValid()` and `String()`

### 2. Compiler Support (`pkg/workflow/compiler_types.go`)
- Added `actionMode` field to `Compiler` struct
- Added `SetActionMode()` and `GetActionMode()` methods
- Default mode is `ActionModeInline` for backward compatibility
- Both `NewCompiler()` and `NewCompilerWithCustomOutput()` initialize with inline mode

### 3. Script Registry Extensions (`pkg/workflow/script_registry.go`)
- Extended `scriptEntry` to include optional `actionPath` field
- Added `RegisterWithAction()` method to register scripts with custom action paths
- Added `GetActionPath()` method to retrieve action paths
- Maintained backward compatibility with existing `Register()` and `RegisterWithMode()` methods

### 4. Custom Action Step Generation (`pkg/workflow/safe_outputs.go`)
- Added `buildCustomActionStep()` method to generate steps using custom action references
- Added token mapping helpers:
  - `addCustomActionGitHubToken()`
  - `addCustomActionCopilotGitHubToken()`
  - `addCustomActionAgentGitHubToken()`
- Updated `buildSafeOutputJob()` to choose between inline and custom modes based on compiler settings
- Falls back to inline mode if action path is not registered

### 5. Safe Output Job Configuration (`pkg/workflow/safe_outputs.go`)
- Extended `SafeOutputJobConfig` struct with `ScriptName` field
- Script name enables lookup of custom action path from registry
- Updated `create_issue.go` to pass script name ("create_issue")

### 6. Tests (`pkg/workflow/compiler_custom_actions_test.go`)
- Added comprehensive tests for:
  - `ActionMode` type validation
  - `String()` method
  - Compiler action mode default and setter
  - Script registry action path registration
  - Custom action mode compilation
  - Inline action mode compilation (default)
  - Fallback behavior when action path not found

## Current Status

### ✅ Completed
- Core infrastructure for action mode switching
- Script registry extension for action path mapping
- Custom action step generation logic
- Token input mapping for custom actions
- Backward compatibility (all existing tests pass)
- Basic unit tests for infrastructure

### ⚠️ Known Issues
- Custom action compilation tests are failing
- The `buildCustomActionStep` function is being called correctly (confirmed via debug logging)
- Action paths are being registered and found successfully
- However, generated lock files still contain `actions/github-script` references
- Issue appears to be in step generation or output formatting

### 🔄 Next Steps
1. **Debug step generation**: Investigate why custom action steps aren't appearing in lock files
2. **Complete test suite**: Fix failing compilation tests
3. **Extend to other safe outputs**: Add `ScriptName` to other safe output types (add_comment, create_pull_request, etc.)
4. **Add CLI flag**: Implement `--action-mode=custom|inline` flag in compile command
5. **Release mode support**: Add support for SHA-pinned action references (e.g., `githubnext/gh-aw/.github/actions/create-issue@SHA`)
6. **Documentation**: Update compiler documentation with action mode usage

## Usage Example (When Fully Working)

```go
// Register script with action path
workflow.DefaultScriptRegistry.RegisterWithAction(
    "create_issue",
    createIssueScript,
    workflow.RuntimeModeGitHubScript,
    "./actions/create-issue", // Development mode: local path
)

// Compile with custom action mode
compiler := workflow.NewCompiler(false, "", "1.0.0")
compiler.SetActionMode(workflow.ActionModeCustom)
compiler.CompileWorkflow("test-workflow.md")
```

### Expected Output (Development Mode)

```yaml
jobs:
  create_issue:
    runs-on: ubuntu-latest
    steps:
      - name: Create Output Issue
        id: create_issue
        uses: ./actions/create-issue
        env:
          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```

### Current Output (Default Inline Mode)

```yaml
jobs:
  create_issue:
    runs-on: ubuntu-latest
    steps:
      - name: Create Output Issue
        id: create_issue
        uses: actions/github-script@SHA
        env:
          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            // JavaScript code here
```

## Architecture Notes

### Design Decisions

1. **Registry-based approach**: Scripts are registered once with optional action paths, avoiding duplicate configuration
2. **Fallback strategy**: If action path not found, automatically falls back to inline mode
3. **Backward compatibility**: Default mode is inline, no breaking changes to existing workflows
4. **Token mapping**: Custom actions use `token` input instead of `github-token` parameter

### Future Enhancements

1. **Input parameter mapping**: Map environment variables to action inputs for better type safety
2. **Action output handling**: Support custom action outputs in addition to standard outputs
3. **Validation**: Add compile-time validation of action paths
4. **Cache support**: Cache compiled custom actions for faster subsequent compilations
