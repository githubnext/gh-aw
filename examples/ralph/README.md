# Ralph Loop - Iterative Learning Workflow

Ralph Loop is an iterative learning workflow that persists discoveries between iterations, enabling continuous improvement and knowledge accumulation.

## Overview

Ralph is an AI agent that:
- Works on tasks through multiple iterations
- Learns from each iteration and applies those learnings to subsequent iterations
- Persists learnings in two files:
  - `progress.txt` - Append-only log of iteration-by-iteration progress
  - `AGENTS.md` - Discovered patterns, conventions, and best practices

## Key Features

### 1. Iterative Execution

Ralph performs multiple iterations on a single task, with each iteration:
- Building on previous work
- Applying accumulated learnings
- Discovering new patterns and gotchas
- Documenting progress

### 2. Learnings Persistence

**progress.txt**:
- Append-only log format
- Each entry includes:
  - Timestamp (UTC)
  - Iteration number
  - Run ID for traceability
  - Work completed
  - Key learnings
  - Gotchas encountered
  - Next steps
- Never modified or deleted - complete history preserved

**AGENTS.md**:
- Living document of patterns and conventions
- Updated with significant, reusable discoveries
- Organized by category:
  - Code patterns
  - Gotchas and edge cases
  - Testing patterns
  - Useful conventions
- Filters out task-specific or obvious patterns

### 3. Continuous Improvement

Each iteration:
1. Reads previous learnings from both files
2. Applies those learnings to current work
3. Discovers new patterns or gotchas
4. Documents findings for future iterations
5. Commits changes for persistence

## Usage

### Trigger the Workflow

```bash
gh workflow run ralph-loop-basic.yml \
  -f task="Implement feature X with tests" \
  -f iterations="3"
```

Or via GitHub UI:
1. Go to Actions → Ralph Loop Basic
2. Click "Run workflow"
3. Enter task description
4. Enter number of iterations (default: 3)
5. Click "Run workflow"

### Input Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `task` | Yes | - | Task to work on iteratively |
| `iterations` | No | `3` | Number of iterations to perform |

## Example Workflow

### Iteration 1: Discovery

**Work**: Start implementing feature X
**Learnings**:
- Discovered that tests should use table-driven approach
- Found edge case with nil inputs
- Identified useful helper function pattern

**Files Updated**:
```
progress.txt:
  + Entry for Iteration 1/3
AGENTS.md:
  + Table-driven test pattern
  + Nil input handling convention
```

### Iteration 2: Application

**Work**: Continue implementing, applying Iteration 1 learnings
**Learnings**:
- Applied table-driven tests successfully
- Discovered another edge case with empty strings
- Found better error message format

**Files Updated**:
```
progress.txt:
  + Entry for Iteration 2/3
AGENTS.md:
  + Empty string edge case
  + Error message format convention
```

### Iteration 3: Refinement

**Work**: Complete feature, applying all previous learnings
**Learnings**:
- Feature complete with comprehensive tests
- All edge cases handled
- Code follows discovered conventions

**Files Updated**:
```
progress.txt:
  + Entry for Iteration 3/3
  + Summary of completed work
AGENTS.md:
  + Final patterns discovered
```

**Result**: PR created with all changes, including comprehensive progress.txt log and updated AGENTS.md

## progress.txt Format

Each entry follows this format:

```markdown
---
2026-01-24 06:00:00 UTC - Iteration 1/3 - Run 12345678

Task: Implement feature X with tests

## Work Completed
- Implemented core feature logic
- Added initial test cases
- Fixed nil input edge case

## Key Learnings
- Table-driven tests work well for this type of feature
- Always check for nil/empty inputs early
- Helper function reduces code duplication

## Gotchas
- Forgot to handle nil inputs initially (caused panic)
- Test setup was repetitive (solved with helper function)

## Next Steps
- Add more edge case tests
- Improve error messages
- Update documentation
---
```

## AGENTS.md Updates

Ralph adds discovered patterns to the appropriate sections:

```markdown
## Ralph Loop Learnings

Patterns and conventions discovered through iterative Ralph Loop workflows.

### Code Patterns

- **Table-driven tests**: Use table-driven approach for testing multiple scenarios
  ```go
  tests := []struct{
    name string
    input string
    expected string
  }{...}
  ```

### Gotchas and Edge Cases

- **Nil input handling**: Always check for nil inputs early to avoid panics
- **Empty string edge case**: Empty strings should be treated differently from nil

### Testing Patterns

- **Helper functions**: Extract common test setup into helper functions to reduce duplication

### Useful Conventions

- **Error messages**: Use format "failed to X: Y" for consistency
```

## Benefits

### For the Current Task

- **Incremental Progress**: Break down complex tasks into manageable iterations
- **Error Recovery**: Learn from mistakes in one iteration and avoid them in the next
- **Quality Improvement**: Each iteration produces better quality code than the previous

### For Future Tasks

- **Knowledge Base**: AGENTS.md becomes a living knowledge base
- **Faster Onboarding**: New team members can learn from documented patterns
- **Consistency**: Shared conventions lead to more consistent codebase
- **Efficiency**: Avoid repeating mistakes documented in progress.txt

## Guidelines for Effective Use

### Choose Good Iteration Tasks

✅ **Good Tasks**:
- Implementing a new feature with tests
- Refactoring a complex module
- Debugging a challenging issue
- Exploring a new technology or pattern

❌ **Poor Tasks**:
- Simple one-step tasks (no need for iteration)
- Tasks that don't generate reusable learnings
- Purely mechanical tasks (formatting, renaming)

### Document Meaningful Learnings

✅ **Document**:
- Patterns that apply beyond the current task
- Non-obvious gotchas or edge cases
- Effective testing strategies
- Reusable conventions

❌ **Don't Document**:
- Task-specific implementation details
- Obvious or well-known patterns
- Temporary workarounds
- Extremely niche edge cases

### Set Appropriate Iteration Count

- **Simple tasks**: 2-3 iterations
- **Medium tasks**: 3-5 iterations
- **Complex tasks**: 5-7 iterations
- **Exploratory tasks**: 3-4 iterations (focus on learning)

## Troubleshooting

### progress.txt not updating

**Problem**: progress.txt doesn't show new entries
**Solution**: 
- Check that the append command completed successfully
- Verify git commit included progress.txt
- Ensure workflow has `contents: write` permission

### AGENTS.md updates not appearing

**Problem**: AGENTS.md updates are missing
**Solution**:
- Verify Ralph Loop Learnings section exists in AGENTS.md
- Check that edit tool was used correctly
- Ensure changes were committed before pushing

### Learnings not applied in subsequent iterations

**Problem**: Iteration 2 repeats Iteration 1 mistakes
**Solution**:
- Verify "Read Previous Learnings" step is executed
- Check that progress.txt and AGENTS.md are readable
- Ensure previous commits are visible to current iteration

## Advanced Usage

### Custom Learning Categories

You can customize AGENTS.md sections for your specific needs:

```markdown
## Ralph Loop Learnings

### Performance Patterns
- Patterns related to optimization

### Security Patterns  
- Security-related discoveries

### Integration Patterns
- Patterns for external integrations
```

### Combining with Other Workflows

Ralph Loop can be integrated with other workflows:

```yaml
# Trigger Ralph Loop after code review
on:
  pull_request_review:
    types: [submitted]
  workflow_dispatch:
    # ... use Ralph Loop to address review feedback iteratively
```

## Related Workflows

- **`dev.md`** - Single-iteration development workflow
- **`changeset.md`** - Automated changelog generation
- **`daily-doc-updater.md`** - Documentation updates with PR creation

## Contributing

To improve Ralph Loop:
1. Modify `.github/workflows/ralph-loop-basic.md`
2. Update this README with new features
3. Test with a real task
4. Submit a PR with your improvements

## License

Part of the GitHub Agentic Workflows project.
