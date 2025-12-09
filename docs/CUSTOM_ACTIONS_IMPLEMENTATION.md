# Custom Action References Implementation

> Documentation for the custom action mode feature in the workflow compiler
> Last updated: 2025-12-09
> See also: [Custom GitHub Actions Build System](../specs/actions.md) for details on the actions build infrastructure

## Overview

This implementation adds support for generating custom action references in compiled workflows instead of embedding JavaScript inline via `actions/github-script`. This enables a development mode where workflows can reference local actions (e.g., `./actions/create-issue`) that are tested and validated before being published.

The custom action mode complements the existing [Custom GitHub Actions Build System](../specs/actions.md), which provides the infrastructure for creating, building, and managing custom GitHub Actions in the `actions/` directory.

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
6. **Create custom actions**: Build actual actions in `actions/` directory to match script names (see [Actions Build System](../specs/actions.md))
7. **Documentation**: Update compiler documentation with action mode usage

## Related Documentation

- **[Custom GitHub Actions Build System](../specs/actions.md)**: Complete specification for the actions build infrastructure
  - Directory structure and conventions
  - Build system implementation details
  - CI integration and validation
  - Developer guide for creating new actions

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

### Relationship to Actions Build System

The custom action mode feature integrates with the [Custom GitHub Actions Build System](../specs/actions.md):

- **Build System** (`specs/actions.md`): Manages the `actions/` directory, builds custom actions, and handles dependency bundling
- **Action Mode** (this document): Enables the compiler to generate `uses:` references to those custom actions instead of inline JavaScript

Together, these systems enable a complete workflow:
1. Create custom actions in `actions/` directory
2. Build them using `make actions-build`
3. Compile workflows with `ActionModeCustom` to reference those actions

### Design Decisions

1. **Registry-based approach**: Scripts are registered once with optional action paths, avoiding duplicate configuration
2. **Fallback strategy**: If action path not found, automatically falls back to inline mode
3. **Backward compatibility**: Default mode is inline, no breaking changes to existing workflows
4. **Token mapping**: Custom actions use `token` input instead of `github-token` parameter
5. **Reuse existing infrastructure**: Leverages the same script registry and bundler used for inline mode

### Integration Points

**With Build System**:
- Action paths registered in script registry match directories in `actions/`
- Example: `RegisterWithAction("create_issue", script, mode, "./actions/create-issue")`
- The action must exist and be built using `make actions-build`

**With Compiler**:
- `SetActionMode(ActionModeCustom)` switches from inline to custom action references
- `buildSafeOutputJob()` checks mode and calls appropriate step builder
- Falls back gracefully if action path not registered

**With Safe Outputs**:
- `ScriptName` field in `SafeOutputJobConfig` enables action path lookup
- Each safe output type can specify its corresponding action name
- Token parameters are mapped to action inputs automatically

### Future Enhancements

1. **Input parameter mapping**: Map environment variables to action inputs for better type safety
2. **Action output handling**: Support custom action outputs in addition to standard outputs
3. **Validation**: Add compile-time validation of action paths (check if action exists in `actions/` directory)
4. **Cache support**: Cache compiled custom actions for faster subsequent compilations
5. **Automatic action creation**: Generate action scaffold from script registry entries
6. **Release mode**: Support versioned action references like `githubnext/gh-aw/.github/actions/create-issue@v1.0.0`

## Complete Workflow Example

### 1. Create a Custom Action

First, create the action using the build system (see [Actions Build System](../specs/actions.md)):

```bash
# Create action directory
mkdir -p actions/create-issue/src

# Create action.yml
cat > actions/create-issue/action.yml << 'EOF'
name: 'Create Issue'
description: 'Creates a GitHub issue from agent output'
inputs:
  token:
    description: 'GitHub token for API access'
    required: true
  agent-output:
    description: 'Path to agent output JSON file'
    required: true
runs:
  using: 'node20'
  main: 'index.js'
EOF

# Create source file (see specs/actions.md for complete example)
# Build the action
make actions-build
```

### 2. Register Script with Action Path

```go
// In your code or during initialization
workflow.DefaultScriptRegistry.RegisterWithAction(
    "create_issue",
    createIssueScriptSource,
    workflow.RuntimeModeGitHubScript,
    "./actions/create-issue",
)
```

### 3. Compile Workflow with Custom Mode

```go
compiler := workflow.NewCompiler(false, "", "1.0.0")
compiler.SetActionMode(workflow.ActionModeCustom)
compiler.CompileWorkflow("workflow.md")
```

### 4. Generated Workflow Uses Custom Action

The compiled workflow will reference your custom action:

```yaml
jobs:
  create_issue:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/create-issue
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          agent-output: /tmp/agent-output.json
```
