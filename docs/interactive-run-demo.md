# Interactive Mode Demonstration

This document shows example output from the interactive mode.

## Step 1: Launching Interactive Mode

```bash
$ gh aw run
```

## Step 2: Workflow Selection (Interactive List)

```
Select a workflow to run:

  > ai-moderator
    1 required input(s)
  
    archie
    No inputs required
  
    test-interactive
    1 required input(s), 2 optional input(s)
  
  /_ Type to filter...
```

User navigates with arrow keys and presses Enter to select `test-interactive`.

## Step 3: Workflow Information Display

```
Workflow: test-interactive

Workflow Inputs:
  • task_description (required) - Description of the task to perform
  • priority - Priority level [default: medium]
  • dry_run - Run in dry-run mode (no actual changes) [default: false]

```

## Step 4: Input Collection (Form)

```
┃ Enter value for 'task_description'
┃ Description of the task to perform
┃ 
┃ > Fix the security vulnerability in auth module_
```

```
┃ Enter value for 'priority'
┃ Priority level
┃ 
┃ > high_
```

```
┃ Enter value for 'dry_run'
┃ Run in dry-run mode (no actual changes)
┃ 
┃ > false_
```

## Step 5: Execution Confirmation

```
┃ Run workflow 'test-interactive' with 3 input(s)?
┃ 
┃ > Yes, run it
┃   No, cancel
```

## Step 6: Running Workflow

```
ℹ Running workflow...

⚙ Equivalent command: gh aw run test-interactive -F task_description="Fix the security vulnerability in auth module" -F priority="high" -F dry_run="false"

✓ Workflow run created successfully
  Run ID: 123456789
  Run URL: https://github.com/owner/repo/actions/runs/123456789

✓ Workflow dispatched successfully!

ℹ To run this workflow again, use:
⚙ gh aw run test-interactive -F task_description="Fix the security vulnerability in auth module" -F priority="high" -F dry_run="false"
```

## Error Handling Examples

### No Workflows Available

```bash
$ gh aw run
✗ no runnable workflows found. Workflows must have 'workflow_dispatch' trigger
```

### CI Environment Detection

```bash
$ CI=true gh aw run
✗ interactive mode cannot be used in CI environments. Please provide a workflow name
```

### User Cancellation

```bash
$ gh aw run
# Select workflow
# Fill inputs
# Choose "No, cancel" at confirmation

⚠ Workflow execution cancelled
```

### Invalid Flags in Interactive Mode

```bash
$ gh aw run --repeat 3
✗ --repeat flag is not supported in interactive mode
```

```bash
$ gh aw run -F name=value
✗ workflow inputs cannot be specified in interactive mode (they will be collected interactively)
```

## Features Demonstrated

1. **Filterable List**: Type `/` to search workflows
2. **Rich Information**: See input requirements before selection
3. **Validation**: Required inputs are enforced
4. **Default Values**: Optional inputs show defaults
5. **Command Output**: Copy-paste command for future runs
6. **Error Handling**: Clear messages for various error cases
