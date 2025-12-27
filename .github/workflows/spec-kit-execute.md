---
name: Spec-Kit Execute
description: Execute pending spec-kit specifications
on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: spec-kit-execute
engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[spec-kit] "
    labels: [spec-kit, automation]
    reviewers: copilot
    draft: false

tools:
  cache-memory: true
  repo-memory: true
  github:
    mode: remote
    toolsets: [default]
  edit:
  bash:
    - "find .specify/specs -type f -name '*.md'"
    - "find .specify/specs/ -maxdepth 1 -ls"
    - "cat .specify/specs/*/spec.md"
    - "cat .specify/specs/*/plan.md"
    - "cat .specify/specs/*/tasks.md"
    - "cat .specify/memory/constitution.md"
    - "git status"
    - "git diff"
    - "git branch"
    - "make fmt"
    - "make lint"
    - "make build"
    - "make test-unit"
    - "make test"

timeout-minutes: 60

---

# Execute Spec-Kit Specifications

Your task is to find and execute pending specifications in the `.specify/specs/` directory.

## Process Overview

1. Check `.specify/specs/` for feature directories
2. For each feature directory:
   - Check if `spec.md` exists
   - Check if `plan.md` exists
   - Check if `tasks.md` exists
   - Check if implementation is complete (look for completion markers)
3. For features with complete spec/plan/tasks but incomplete implementation:
   - Read the constitution from `.specify/memory/constitution.md`
   - Read the specification from `spec.md`
   - Read the implementation plan from `plan.md`
   - Read the task breakdown from `tasks.md`
   - Execute tasks in order, respecting dependencies
   - Mark parallel tasks with [P] for concurrent execution where possible
   - Create implementation files according to the plan
   - Run tests and validation after each user story
4. Report on what was implemented
5. Create a pull request with the implementation

## Step-by-Step Instructions

### Step 1: Load the Constitution

First, read the project constitution to understand the development principles:

```bash
cat .specify/memory/constitution.md
```

This constitution defines how all development should be conducted in this repository. You **MUST** follow these principles strictly throughout the implementation.

### Step 2: Scan for Feature Specifications

Check for feature specifications in the `.specify/specs/` directory:

```bash
find .specify/specs/ -maxdepth 1 -ls
```

List all feature specifications and their files:

```bash
find .specify/specs -type f -name 'spec.md' -o -name 'plan.md' -o -name 'tasks.md'
```

### Step 3: Analyze Feature Status

For each feature found in the `.specify/specs/` directory:

1. Check if the feature has all required files:
   - `spec.md` - Requirements and user stories (**REQUIRED**)
   - `plan.md` - Technical implementation plan (**REQUIRED**)
   - `tasks.md` - Task breakdown (**REQUIRED**)

2. Read the `tasks.md` file and analyze task completion status:
   - Count total tasks (lines with `- [ ]` or `- [x]`)
   - Count completed tasks (lines with `- [x]` or `- [X]`)
   - Count pending tasks (lines with `- [ ]`)

3. Create a status summary table:

```text
| Feature | Spec | Plan | Tasks | Total | Done | Pending | Status |
|---------|------|------|-------|-------|------|---------|--------|
| 001-feature-name | ✅ | ✅ | ✅ | 12 | 8 | 4 | 🔨 IN PROGRESS |
| 002-other-feature | ✅ | ✅ | ✅ | 10 | 10 | 0 | ✅ COMPLETE |
| 003-new-feature | ✅ | ✅ | ✅ | 15 | 0 | 15 | 📋 NOT STARTED |
| 004-incomplete | ✅ | ❌ | ❌ | - | - | - | ⚠️ INCOMPLETE SPEC |
```

### Step 4: Select Feature to Implement

Choose the feature to work on based on priority:

1. **First Priority**: Features that are "IN PROGRESS" (have some completed tasks)
   - Continue from where the previous implementation left off
   - This ensures incremental progress on partially completed work

2. **Second Priority**: Features that are "NOT STARTED" (no completed tasks yet)
   - Start from the first task in the task list
   - Choose the feature with the lowest feature number (e.g., 001 before 002)

3. **Skip**: Features that are "COMPLETE" (all tasks done) or "INCOMPLETE SPEC" (missing spec/plan/tasks)

**Important**: Work on only ONE feature per workflow run to keep PRs focused and reviewable.

### Step 5: Load Implementation Context

For the selected feature, load all relevant documentation:

```bash
# Read the feature specification
cat .specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]/spec.md

# Read the implementation plan
cat .specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]/plan.md

# Read the task breakdown
cat .specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]/tasks.md

# Read additional context if available
cat .specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]/data-model.md 2>/dev/null || true
cat .specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]/research.md 2>/dev/null || true
```

### Step 6: Execute Implementation

Follow the spec-kit implementation methodology:

#### 6.1 Parse Task Structure

Tasks in `tasks.md` are organized into phases. Common phases include:

- **Setup Phase**: Initialize structure, dependencies, configuration files
- **Tests Phase**: Write tests before implementation (Test-Driven Development)
- **Core Phase**: Implement models, services, core business logic
- **Integration Phase**: Connect components, add logging, error handling
- **Polish Phase**: Optimization, documentation, code cleanup

Tasks may have markers:
- `[P]` - Parallel task (can be executed concurrently with other [P] tasks in the same phase)
- `[S]` - Sequential task (must wait for previous tasks to complete)
- `[D: TaskX]` - Dependency marker (must wait for TaskX to complete)

#### 6.2 Execute Tasks by Phase

For each phase:

1. **Read all tasks in the phase** - Understand what needs to be done
2. **Identify parallel vs sequential tasks** - Look for [P] and [S] markers
3. **Respect dependencies** - Don't start a task until its dependencies are complete
4. **Execute tasks systematically**:
   - For sequential tasks: Complete one fully before moving to the next
   - For parallel tasks: You can work on multiple [P] tasks together if efficient
5. **Mark completed tasks** - Update `tasks.md` to mark each task as `[x]` when done

#### 6.3 Follow Test-Driven Development

**NON-NEGOTIABLE**: The constitution requires TDD for all new functionality.

For each feature or component:
1. **Write tests first** - Create test files before implementation
2. **Run tests** - Verify they fail initially (red)
3. **Implement code** - Write minimal code to make tests pass (green)
4. **Refactor** - Improve code quality while keeping tests passing
5. **Validate** - Run full test suite to ensure no regressions

Example workflow for a new function:
```bash
# 1. Create test file
# Use edit tool to create: pkg/feature/feature_test.go

# 2. Run tests (should fail)
make test-unit

# 3. Implement feature
# Use edit tool to create/modify: pkg/feature/feature.go

# 4. Run tests again (should pass)
make test-unit

# 5. Format and lint
make fmt
make lint
```

#### 6.4 Use Proper Tools

**Always use the appropriate tools for each task:**

- **Edit tool** - For creating and modifying files
- **Bash tool** - For running commands (make, git, find, cat, etc.)
- **GitHub tools** - For searching code, viewing files, checking references

**Console formatting**: When you need to add CLI output, use the console package:
```go
import "github.com/githubnext/gh-aw/pkg/console"

fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Success!"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### 6.5 Validate After Each Phase

After completing each phase, run validation:

```bash
# Format code (required before linting)
make fmt

# Lint code
make lint

# Build the project
make build

# Run unit tests (fast feedback)
make test-unit
```

If any step fails:
- **Fix the issues immediately** - Don't proceed to the next phase
- **Re-run validation** - Ensure all checks pass
- **Update tasks.md** - Mark the validation task as complete

Only run the full test suite (`make test`) after all phases are complete or at major milestones.

### Step 7: Update Task Status

As you complete each task, update the `tasks.md` file:

```bash
# Use the edit tool to change:
# - [ ] Task description
# to:
# - [x] Task description
```

This provides clear progress tracking and ensures the next workflow run knows where to continue.

### Step 8: Create Pull Request

Once implementation reaches a significant milestone (completed phase, user story, or all tasks):

1. **Prepare a comprehensive summary**:
   - List all completed tasks with checkmarks
   - Describe the changes made (files created/modified)
   - Include test results (unit tests, integration tests, linting, build)
   - Note any issues encountered and how they were resolved

2. **Use safe-outputs to create the PR** - The workflow will automatically create a pull request with your changes

3. **PR Description Format**:

```markdown
## Spec-Kit Implementation: [FEATURE-NUMBER]-[FEATURE-NAME]

This PR implements tasks from feature `.specify/specs/[FEATURE-NUMBER]-[FEATURE-NAME]` following the spec-driven development methodology and project constitution.

### Completed Tasks

**Phase 1: Setup** ✅
- [x] Task 1.1: Description
- [x] Task 1.2: Description

**Phase 2: Tests** ✅
- [x] Task 2.1: Write unit tests for X
- [x] Task 2.2: Write integration tests for Y

**Phase 3: Core** 🔨 (In Progress)
- [x] Task 3.1: Implement model X
- [x] Task 3.2: Implement service Y
- [ ] Task 3.3: Implement handler Z (pending)

### Changes Made

**Created Files:**
- `pkg/feature/feature.go` - Core implementation
- `pkg/feature/feature_test.go` - Unit tests
- `cmd/gh-aw/feature_command.go` - CLI command

**Modified Files:**
- `pkg/cli/root.go` - Added feature command registration
- `README.md` - Updated with feature documentation

### Validation Results

- ✅ **Unit Tests**: All 15 tests passing
- ✅ **Integration Tests**: All 5 tests passing
- ✅ **Linting**: No issues found
- ✅ **Build**: Successful
- ✅ **Format**: All files formatted correctly

### Test Coverage

```
pkg/feature/feature.go:         95.2% coverage
pkg/feature/handler.go:         88.7% coverage
```

### Notes

- Followed TDD approach: tests written before implementation
- All code follows console formatting standards
- Constitution principles strictly adhered to
- Minimal changes philosophy applied

### Next Steps

- [ ] Task 3.3: Implement handler Z
- [ ] Task 4.1: Add integration with existing commands
- [ ] Phase 5: Polish and documentation
```

### Step 9: Handle Edge Cases

**No Pending Work**: If no features have pending tasks or incomplete specs:
- Exit gracefully with a message: "No pending spec-kit work found. All features are complete or lack required specification files."
- Do not create a PR

**Build/Test Failures**: If validation fails:
- Include the error details in the PR description
- Mark the PR as draft
- Clearly indicate which tests failed and include relevant error messages
- The human reviewer can decide how to proceed

**Complex Decisions**: If a task requires human judgment or architectural decisions:
- Document the decision point in the PR description
- Mark the PR as draft
- Provide context and ask for guidance
- Complete as much as possible before blocking

**Incomplete Specifications**: If a feature lacks spec.md, plan.md, or tasks.md:
- Skip that feature
- Note it in the workflow output
- Look for the next valid feature to implement

## Guidelines

Follow these principles throughout the implementation:

1. **Constitution First** - Strictly adhere to all constitutional principles
2. **Minimal Changes** - Make the smallest possible changes to achieve task goals
3. **Test-Driven Development** - Always write tests before implementation code
4. **Incremental Progress** - Complete tasks one phase at a time
5. **Clear Documentation** - Document all changes and decisions
6. **Proper Tools** - Use make commands, edit tool, and GitHub tools appropriately
7. **Console Formatting** - Use the console package for all CLI output
8. **Security First** - Validate changes don't introduce vulnerabilities
9. **One Feature at a Time** - Focus on a single feature per workflow run
10. **Mark Progress** - Update tasks.md as you complete each task

## Important Reminders

✅ **DO**:
- Read and follow the constitution
- Write tests before implementation
- Use edit tool to modify files
- Run validation after each phase
- Update tasks.md to mark progress
- Create focused, reviewable PRs
- Use console formatting for CLI output
- Respect task dependencies and phases

❌ **DON'T**:
- Skip tests or validation
- Make unnecessary changes
- Work on multiple features at once
- Use plain fmt.* for CLI output
- Remove working code unless necessary
- Proceed with failing tests
- Create PRs without validation results

## Success Criteria

A successful implementation run includes:

1. ✅ Constitution principles followed
2. ✅ Tasks executed in correct order with dependencies respected
3. ✅ Tests written before implementation (TDD)
4. ✅ All validation checks passing (fmt, lint, build, test)
5. ✅ tasks.md updated with completed task markers
6. ✅ PR created with comprehensive description
7. ✅ Code follows existing patterns and conventions
8. ✅ No security vulnerabilities introduced
9. ✅ Minimal, surgical changes made
10. ✅ Clear documentation of changes and rationale

Now begin by scanning for pending specifications and implementing the highest priority feature!
