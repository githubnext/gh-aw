# Interactive Mode Demonstration

Quick walkthrough of the interactive mode for `gh aw run`.

## Usage

```bash
$ gh aw run
```

## Workflow Selection

```
Select a workflow to run:

  > test-interactive
    1 required input(s), 2 optional input(s)
  
  /_ Type to filter...
```

## Input Collection

```
┃ Enter value for 'task_description'
┃ > Fix the security vulnerability in auth module_

┃ Enter value for 'priority'
┃ > high_

┃ Enter value for 'dry_run'
┃ > false_
```

## Execution

```
✓ Workflow dispatched successfully!

To run this workflow again, use:
⚙ gh aw run test-interactive -F task_description="Fix bug" -F priority="high" -F dry_run="false"
```

## Error Examples

**CI environment:**
```bash
$ CI=true gh aw run
✗ interactive mode cannot be used in CI environments
```

**Invalid flag:**
```bash
$ gh aw run --repeat 3
✗ --repeat flag is not supported in interactive mode
```

