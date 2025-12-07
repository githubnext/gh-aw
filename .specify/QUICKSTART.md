# Spec-Kit Quick Start Guide

This guide shows you how to use spec-kit to create and implement features in the gh-aw repository.

## What is Spec-Kit?

Spec-kit enables **spec-driven development** where you write specifications in natural language, and they guide implementation. Instead of jumping straight to code, you define:

1. **What** you want to build (specification)
2. **How** you'll build it (implementation plan)
3. **Steps** to take (task breakdown)
4. Then implement following those steps

## Prerequisites

- An AI agent that supports spec-kit commands (GitHub Copilot, Claude Code, Cursor, etc.)
- Access to this repository
- Familiarity with the project's constitution (see `.specify/memory/constitution.md`)

## Quick Start: Create a New Feature

### Step 1: Review the Constitution

Before starting any work, review the project's development principles:

```bash
cat .specify/memory/constitution.md
```

This defines:
- Go-first architecture
- Minimal changes philosophy
- Test-driven development requirements
- Console output standards
- Security and quality requirements

### Step 2: Create a Specification

Use your AI agent's spec-kit command to define what you want to build:

```
/speckit.specify Build a feature that validates agentic workflow configuration against security best practices. The validator should check for common security issues like overly broad permissions, unvalidated inputs, and missing safe-output configurations.
```

This will:
- Create a new feature branch (e.g., `001-workflow-security-validator`)
- Generate a specification in `specs/001-workflow-security-validator/spec.md`
- Define user stories and functional requirements

### Step 3: Create an Implementation Plan

Define the technical approach:

```
/speckit.plan Use Go for the core validation logic in pkg/workflow/. Add a new command to the CLI in cmd/gh-aw/. Follow existing patterns for validation (see pkg/workflow/validation.go). Use table-driven tests. Integration with existing workflow compilation pipeline.
```

This creates `specs/001-workflow-security-validator/plan.md` with:
- Technology choices
- Architecture decisions
- File structure
- Dependencies
- Testing approach

### Step 4: Generate Task Breakdown

Break the plan into actionable tasks:

```
/speckit.tasks
```

This creates `specs/001-workflow-security-validator/tasks.md` with:
- Ordered list of tasks
- Task phases (Setup, Tests, Core, Integration, Polish)
- Dependencies and parallelization markers
- Acceptance criteria for each task

### Step 5: Implement (Manual or Automated)

**Option A: Manual Implementation**

Execute the tasks yourself using your AI agent:

```
/speckit.implement
```

The agent will:
- Load the specification, plan, and tasks
- Execute tasks phase-by-phase
- Write tests before code (TDD)
- Validate with `make fmt`, `make lint`, `make build`, `make test`
- Mark completed tasks in `tasks.md`

**Option B: Automated Implementation**

Let the spec-kit-executor workflow handle it:

1. Commit your spec, plan, and tasks to a branch
2. The workflow runs daily at 8am UTC
3. It will detect your pending tasks and implement them
4. A PR will be created with the implementation

### Step 6: Review and Merge

Whether implemented manually or automatically:

1. Review the generated PR
2. Check that tests pass
3. Verify code follows the constitution
4. Request human review if needed
5. Merge when ready

## Example Workflow

Here's a complete example of creating a small feature:

```bash
# 1. Start your AI agent (e.g., GitHub Copilot in VS Code)

# 2. Review constitution
/speckit.constitution

# 3. Define the feature
/speckit.specify Add a --version flag to the gh aw CLI that displays the version and build information

# 4. Create implementation plan
/speckit.plan Add a version flag to cmd/gh-aw/main.go. Version is injected at build time via -ldflags. Display version, commit hash, and build date. Follow existing CLI flag patterns.

# 5. Generate tasks
/speckit.tasks

# 6. Implement
/speckit.implement

# The agent will:
# - Add the --version flag
# - Write tests for version display
# - Update documentation
# - Validate with make commands
# - Create a PR
```

## Additional Commands

### Clarify Underspecified Areas

Before planning, clarify ambiguous requirements:

```
/speckit.clarify
```

This helps identify:
- Missing requirements
- Ambiguous specifications
- Edge cases
- User expectations

### Analyze Consistency

Check cross-artifact consistency:

```
/speckit.analyze
```

Verifies:
- Spec matches plan
- Plan matches tasks
- Tasks cover all requirements
- No contradictions

### Generate Quality Checklists

Create custom validation checklists:

```
/speckit.checklist
```

Generates checklists for:
- Security review
- Performance validation
- UX consistency
- Documentation completeness

## Best Practices

1. **Start Small**: Begin with small features to learn the workflow
2. **Spec-First**: Always write the spec before coding
3. **Plan Thoroughly**: Take time to think through the technical approach
4. **TDD Always**: Write tests before implementation
5. **Incremental**: Complete one phase before moving to the next
6. **Review Constitution**: Check alignment with project principles
7. **Use Automation**: Let the executor workflow handle routine work
8. **Human Review**: Always review AI-generated implementations

## Troubleshooting

### "Prerequisites not met"

The scripts require a feature branch. Check that you're on the right branch:

```bash
git branch
```

Should show something like `001-feature-name`.

### "Tasks.md not found"

You need to run `/speckit.tasks` before `/speckit.implement`:

```
/speckit.tasks
/speckit.implement
```

### "Tests failing"

Follow TDD - write tests that fail first, then implement:

```bash
make test-unit  # Run specific tests
make test       # Run all tests
```

### "Linter errors"

Format code before linting:

```bash
make fmt
make lint
```

## Tips

- **Read Examples**: Check existing workflows in `.github/workflows/` for patterns
- **Check Specs**: Look at `specs/` for design specifications and guidelines
- **Use Skills**: Reference skills in `skills/` directory for specialized knowledge
- **Ask Questions**: Use `/speckit.clarify` when unsure
- **Iterate**: Refine your spec/plan/tasks before implementing
- **Small PRs**: Keep changes focused and reviewable

## Resources

- [Spec-Kit Documentation](https://github.com/github/spec-kit)
- [gh-aw Documentation](../../docs/)
- [Project Constitution](memory/constitution.md)
- [Development Guide](../../DEVGUIDE.md)
- [Contributing Guidelines](../../CONTRIBUTING.md)

## Next Steps

1. Read the constitution: `.specify/memory/constitution.md`
2. Try creating a small feature using `/speckit.specify`
3. Review existing specs in the `specs/` directory
4. Check the spec-kit-executor workflow: `.github/workflows/spec-kit-executor.md`

Happy spec-driven development! 🚀
