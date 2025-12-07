# Spec-Kit Integration for gh-aw

This directory contains the spec-kit configuration for the GitHub Agentic Workflows (gh-aw) repository. Spec-kit enables spec-driven development where specifications become executable and guide implementation.

## What is Spec-Kit?

[Spec-kit](https://github.com/github/spec-kit) is an open-source toolkit that allows you to focus on product scenarios and predictable outcomes instead of vibe coding. It implements a spec-driven development workflow where:

1. **Constitution** defines project principles and development guidelines
2. **Specifications** define what you want to build (requirements and user stories)
3. **Plans** create technical implementation approaches with chosen tech stack
4. **Tasks** break down plans into actionable, ordered task lists
5. **Implementation** executes tasks to build features according to the plan

## Directory Structure

```
.specify/
├── README.md              # This file
├── memory/
│   └── constitution.md    # Project governing principles and development guidelines
├── scripts/
│   └── bash/              # Shell scripts for spec-kit workflow support
│       ├── check-prerequisites.sh    # Validate feature prerequisites
│       ├── common.sh                 # Shared utility functions
│       ├── create-new-feature.sh     # Create new feature branches
│       ├── setup-plan.sh             # Initialize planning phase
│       └── update-agent-context.sh   # Update agent context files
└── commands/
    ├── constitution.md    # /speckit.constitution command
    ├── specify.md         # /speckit.specify command
    ├── plan.md            # /speckit.plan command
    ├── tasks.md           # /speckit.tasks command
    ├── implement.md       # /speckit.implement command
    ├── analyze.md         # /speckit.analyze command
    ├── clarify.md         # /speckit.clarify command
    └── checklist.md       # /speckit.checklist command
```

## Automated Execution

The repository includes an agentic workflow that automatically executes pending spec-kit work:

**Workflow**: `.github/workflows/spec-kit-executor.md`
- **Schedule**: Runs daily at 8am UTC
- **Purpose**: Scans for feature specifications with pending tasks and implements them
- **Output**: Creates pull requests with completed implementations

### How the Executor Works

1. Loads the project constitution from `.specify/memory/constitution.md`
2. Scans the `specs/` directory for feature specifications
3. Identifies features with pending tasks in their `tasks.md` files
4. Prioritizes:
   - First: Features that are IN PROGRESS (partially completed)
   - Second: Features that are NOT STARTED (no completed tasks)
   - Skip: Features that are COMPLETE (all tasks done)
5. Executes implementation following the spec-kit workflow:
   - Loads specification, plan, and tasks
   - Executes tasks phase-by-phase (Setup → Tests → Core → Integration → Polish)
   - Follows TDD approach (tests before code)
   - Runs validation (fmt, lint, build, test) after each phase
6. Creates a pull request with the completed work

## Using Spec-Kit Commands

While the executor automates implementation, you can manually use spec-kit commands when working with AI agents like GitHub Copilot, Claude Code, or Cursor:

### 1. Establish Project Principles

```
/speckit.constitution Create principles focused on code quality, testing standards, and development practices
```

This updates `.specify/memory/constitution.md` with your project's governing principles.

### 2. Create a Specification

```
/speckit.specify Build a feature that allows users to [describe the feature]. Focus on what and why, not the tech stack.
```

Creates a new feature branch and specification in `specs/NNN-feature-name/spec.md`.

### 3. Create an Implementation Plan

```
/speckit.plan Use Go as the primary language. Follow existing code patterns in pkg/. Integrate with the CLI in cmd/gh-aw/.
```

Creates `specs/NNN-feature-name/plan.md` with technical approach and architecture.

### 4. Break Down into Tasks

```
/speckit.tasks
```

Creates `specs/NNN-feature-name/tasks.md` with ordered, actionable task list.

### 5. Implement Features

```
/speckit.implement
```

Executes all tasks following the implementation plan. Can also be done automatically by the spec-kit-executor workflow.

### 6. Additional Commands

- `/speckit.clarify` - Clarify underspecified areas before planning
- `/speckit.analyze` - Cross-artifact consistency and coverage analysis
- `/speckit.checklist` - Generate custom quality checklists

## Constitution

The project constitution in `.specify/memory/constitution.md` defines:

- **Core Principles**: Go-first architecture, minimal changes, TDD, console standards, workflow compilation, build discipline, security
- **GitHub Actions Integration**: JavaScript standards, workflow security
- **Development Workflow**: Repository tools, git workflow, code organization
- **Governance**: How principles guide all development decisions

All development must follow these constitutional principles.

## Feature Specifications

When using spec-kit to create new features, feature specifications will be stored with this structure:

```
specs/
└── NNN-feature-name/
    ├── spec.md          # Requirements and user stories
    ├── plan.md          # Technical implementation plan
    ├── tasks.md         # Ordered task breakdown
    ├── data-model.md    # (Optional) Entities and relationships
    ├── contracts/       # (Optional) API specifications
    ├── research.md      # (Optional) Technical decisions
    └── checklists/      # (Optional) Quality validation checklists
```

**Note**: The existing `specs/` directory contains design specifications and architecture documentation for the repository. Spec-kit feature specifications created with `/speckit.specify` will follow the naming pattern `NNN-feature-name/` where NNN is a sequential number.

## Integration with gh-aw

Spec-kit complements the gh-aw development workflow:

1. **Manual Development**: Use spec-kit commands in your AI agent to create specifications and implementations
2. **Automated Development**: The spec-kit-executor workflow handles pending work automatically
3. **Code Review**: All implementations follow the constitution and go through standard PR review
4. **Testing**: TDD approach ensures all features have comprehensive test coverage
5. **Documentation**: Implementations include documentation updates as part of the task breakdown

## Best Practices

1. **Start with Constitution**: Always review `.specify/memory/constitution.md` before development
2. **Spec-First**: Create specifications before implementation
3. **Plan Thoroughly**: Technical plans should be detailed and validated
4. **Task Breakdown**: Break complex features into small, manageable tasks
5. **TDD Always**: Write tests before implementation code
6. **Incremental Delivery**: Complete and validate each phase before moving to the next
7. **Use Automation**: Let the spec-kit-executor handle routine implementation
8. **Review Changes**: All automated implementations create PRs for human review

## Resources

- [Spec-Kit Repository](https://github.com/github/spec-kit)
- [Spec-Driven Development Guide](https://github.com/github/spec-kit/blob/main/spec-driven.md)
- [gh-aw Repository](https://github.com/githubnext/gh-aw)
- [gh-aw Documentation](../../docs/)

## Support

For issues or questions:
- Spec-kit: https://github.com/github/spec-kit/issues
- gh-aw: https://github.com/githubnext/gh-aw/issues
