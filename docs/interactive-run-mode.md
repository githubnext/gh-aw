# Interactive Mode for `gh aw run`

## Overview

The `gh aw run` command now supports an interactive mode that activates when called without arguments. This provides a guided experience for selecting and running workflows.

## Usage

Simply run the command without arguments:

```bash
gh aw run
```

## Features

### 1. Workflow Selection
- Displays a filterable list of workflows that support `workflow_dispatch`
- Shows workflow descriptions with input counts (required/optional)
- Uses arrow keys for navigation, Enter to select
- Press `/` to filter workflows by name

### 2. Workflow Information
After selecting a workflow, displays:
- Workflow name
- List of inputs (required and optional)
- Default values for optional inputs
- Input descriptions

### 3. Input Collection
- Prompts for each workflow input
- Shows descriptions and default values
- Validates required inputs
- Allows empty values for optional inputs with defaults

### 4. Execution Confirmation
- Confirms workflow execution with input summary
- Options to proceed or cancel

### 5. Command Display
After execution, shows the equivalent CLI command:
```bash
gh aw run workflow-name -F input1=value1 -F input2=value2
```

This allows you to easily re-run the workflow with the same inputs.

## Supported Flags

All standard `run` command flags work in interactive mode:
- `--repo owner/repo` - Target a different repository
- `--ref branch` - Run on a specific branch
- `--engine copilot` - Override AI engine
- `--auto-merge-prs` - Auto-merge created PRs
- `--use-local-secrets` - Use local secrets
- `--push` - Push changes before running

## Limitations

The following flags are NOT supported in interactive mode:
- `--repeat` - Use the displayed command for repeated runs
- `--enable-if-needed` - Enable workflows manually first
- `-F` / `--raw-field` - Inputs are collected interactively

## CI Detection

Interactive mode automatically disables in CI environments:
```bash
CI=true gh aw run  # Returns error: interactive mode cannot be used in CI
```

## Examples

### Basic Interactive Run
```bash
$ gh aw run
# Displays workflow list
# Select workflow with arrow keys
# Fill in inputs
# Confirm and run
```

### Interactive Run with Repo Override
```bash
$ gh aw run --repo owner/repo
# Same interactive flow but targets different repo
```

### Using the Displayed Command
After interactive run completes:
```bash
✓ Workflow dispatched successfully!

To run this workflow again, use:
gh aw run test-workflow -F task_description="Fix bug" -F priority="high"
```

Copy and paste this command for future runs.

## Technical Details

### Workflow Filtering
Only workflows with `workflow_dispatch` or `schedule` triggers are shown. This is determined by checking the `on:` section in workflow frontmatter.

### Input Types
Supports all workflow_dispatch input types:
- `string` - Text input
- `boolean` - Use string type with values 'true'/'false' (GitHub Actions requirement)
- `choice` - Not yet implemented in UI (use CLI)
- `number` - Treated as string input

**Note**: GitHub Actions workflow_dispatch inputs must use string defaults, even for boolean types. Use `type: string` with `default: 'false'` instead of `type: boolean` with `default: false`.

### TTY Detection
Falls back to numbered text list in non-TTY environments (e.g., piped output).

## Troubleshooting

### No Workflows Found
If you see "no runnable workflows found":
1. Ensure workflows have `workflow_dispatch:` in their `on:` section
2. Check that workflows are in `.github/workflows/` directory
3. Verify workflows have `.md` extension

### Input Validation Errors
If input validation fails:
- Check that all required inputs are provided
- Verify input names match workflow definitions
- Use the displayed command to see expected format

### Interactive Mode Not Starting
If interactive mode doesn't activate:
- Ensure no workflow arguments are provided
- Check that you're not in a CI environment
- Verify the binary is up to date
