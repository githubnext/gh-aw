---
title: Spec-Kit Integration with gh-aw
description: Comprehensive guide to using spec-kit for spec-driven development in GitHub Agentic Workflows
---

# Spec-Kit Integration with gh-aw

Spec-kit enables **spec-driven development** for GitHub Agentic Workflows. This guide shows you how to define features in natural language specifications and implement them systematically using automated workflows.

## What is Spec-Driven Development?

Spec-driven development is a structured approach where you write specifications before implementation. Instead of jumping straight to code, you:

1. **Define what** you want to build (specification)
2. **Plan how** you'll build it (implementation plan)
3. **Break down** the work into tasks (task breakdown)
4. **Implement** following the plan and tasks

This approach ensures clear requirements, thoughtful technical design, and predictable outcomes.

### Why Use Spec-Kit?

- **Clarity**: Requirements are documented before implementation
- **Consistency**: All features follow the same development workflow
- **Automation**: Specifications can be executed automatically
- **Quality**: TDD approach ensures comprehensive test coverage
- **Governance**: Constitution ensures alignment with project principles

## How Spec-Kit Works with gh-aw

Spec-kit provides a structured workflow that integrates with gh-aw's development process:

```
Constitution → Specification → Plan → Tasks → Implementation
```

1. **Constitution**: Project principles guide all decisions (`.specify/memory/constitution.md`)
2. **Specification**: Define requirements and user stories (`spec.md`)
3. **Plan**: Create technical implementation approach (`plan.md`)
4. **Tasks**: Break down work into ordered tasks (`tasks.md`)
5. **Implementation**: Execute tasks to build the feature

All of this can be done manually with AI agents or automatically using the spec-kit-executor workflow.

## Directory Structure

The `.specify/` directory organizes all spec-kit related files:

```
.specify/
├── README.md                  # Spec-kit documentation
├── QUICKSTART.md             # Quick start guide
├── memory/
│   └── constitution.md       # Project governing principles
├── specs/
│   └── NNN-feature-name/     # Individual feature specifications
│       ├── spec.md           # Requirements and user stories
│       ├── plan.md           # Technical implementation plan
│       └── tasks.md          # Task breakdown
├── commands/                 # Spec-kit command definitions
│   ├── specify.md
│   ├── plan.md
│   ├── tasks.md
│   └── implement.md
├── scripts/
│   └── bash/                 # Workflow automation scripts
└── templates/                # Specification templates
```

:::note
The constitution at `.specify/memory/constitution.md` defines core principles that govern all development. Review it before creating specifications.
:::

## Creating a Specification

Specifications define **what** you want to build, not **how** to build it. Focus on user stories, requirements, and success metrics.

### Step 1: Review the Constitution

Before creating a specification, review the project constitution:

```bash
cat .specify/memory/constitution.md
```

The constitution defines critical principles:
- Go-first architecture
- Minimal changes philosophy
- Test-driven development requirements
- Console output standards
- Security and quality standards

All specifications must align with these principles.

### Step 2: Create the Specification

Use the `/speckit.specify` command with your AI agent:

```
/speckit.specify Build a feature that validates agentic workflow 
configuration files against security best practices. The validator 
should check for overly broad permissions, unvalidated inputs, 
missing safe-output configurations, and other common security issues.
```

This creates:
- A new feature branch (e.g., `001-workflow-security-validator`)
- A specification file at `.specify/specs/001-workflow-security-validator/spec.md`

### Specification Structure

A complete specification includes:

#### Overview
Brief description of the feature and its purpose.

#### User Stories
Written in the format:
```
**As a** [user type]
**I want** [goal]
**So that** [benefit]

**Acceptance Criteria:**
- Criterion 1
- Criterion 2
```

#### Requirements

**Functional Requirements** (FR-N):
- What the feature must do
- Observable behaviors
- Input/output specifications

**Non-Functional Requirements** (NFR-N):
- Performance targets
- Security requirements
- Usability constraints
- Compatibility needs

#### Success Metrics
Measurable criteria that indicate the feature is successful.

:::tip
Focus on **what** and **why**, not **how**. Technical decisions belong in the implementation plan.
:::

## Creating an Implementation Plan

Implementation plans define **how** you'll build the feature. This includes technical decisions, architecture, and testing strategy.

### Generate the Plan

Use the `/speckit.plan` command:

```
/speckit.plan Use Go for validation logic in pkg/workflow/. Add a 
validate command to CLI in cmd/gh-aw/. Follow existing validation 
patterns. Use table-driven tests. Integrate with workflow compilation 
pipeline.
```

This creates `.specify/specs/001-workflow-security-validator/plan.md`.

### Plan Structure

A complete implementation plan includes:

#### Technical Approach
High-level description of the implementation strategy.

#### Technology Stack
- **Language**: Primary language (Go for gh-aw)
- **Frameworks**: Any frameworks or libraries
- **Testing**: Testing approach and tools
- **Location**: Where code will live in the repository

#### Architecture

**Component Design**:
```
pkg/workflow/
├── security_validator.go       # Core validation logic
├── security_validator_test.go  # Unit tests
└── validation_rules.go         # Security rule definitions

cmd/gh-aw/
└── validate_command.go         # CLI command implementation
```

**Key Components**:
- Component name and responsibility
- Interfaces and contracts
- Dependencies

#### Implementation Phases

Break implementation into logical phases:
1. **Setup**: Directory structure, files, dependencies
2. **Tests**: Write failing tests (TDD)
3. **Core**: Implement core functionality
4. **Integration**: Connect components
5. **Polish**: Documentation, examples, refinements

#### Dependencies
- Internal dependencies (other packages)
- External dependencies (third-party libraries)
- Version requirements

#### Testing Strategy
- Unit test coverage targets (80%+)
- Integration test requirements
- Manual testing scenarios
- Performance testing if applicable

#### Documentation Requirements
- Code comments
- README updates
- User-facing documentation
- API documentation

:::caution
Plans must follow the constitution. For gh-aw, this means:
- Go-first architecture
- Minimal changes
- TDD with unit tests
- Console formatting for output
- Integration with existing patterns
:::

## Task Breakdown

Task breakdown converts the implementation plan into ordered, actionable tasks. Each task should be small enough to complete in one focused session.

### Generate Tasks

Use the `/speckit.tasks` command:

```
/speckit.tasks
```

This creates `.specify/specs/001-workflow-security-validator/tasks.md`.

### Task Structure

Tasks are organized by phase:

```markdown
## Phase 1: Setup

- [ ] 1.1: Create pkg/workflow/security_validator.go
  - **Dependencies**: None
  - **Acceptance**: File exists with package declaration
  
- [ ] 1.2: Create pkg/workflow/security_validator_test.go
  - **Dependencies**: 1.1
  - **Acceptance**: Test file exists with package declaration

## Phase 2: Tests (TDD)

- [ ] 2.1: Write test for overly broad permissions check
  - **Dependencies**: 1.2
  - **Acceptance**: Test fails expectedly (not implemented yet)
  
- [ ] 2.2: Write test for unvalidated input detection
  - **Dependencies**: 1.2
  - **Acceptance**: Test fails expectedly

## Phase 3: Core Implementation

- [ ] 3.1: Implement overly broad permissions validator
  - **Dependencies**: 2.1
  - **Acceptance**: Tests from 2.1 pass
  
- [ ] 3.2: Implement unvalidated input detector
  - **Dependencies**: 2.2
  - **Acceptance**: Tests from 2.2 pass
```

### Task Guidelines

Each task should have:
- **Clear objective**: What to accomplish
- **Dependencies**: Which tasks must complete first
- **Acceptance criteria**: How to know it's done
- **Phase marker**: Which phase it belongs to

### Phase Progression

Follow phases in order:
1. **Setup**: Prepare structure and files
2. **Tests**: Write failing tests (TDD approach)
3. **Core**: Implement to make tests pass
4. **Integration**: Connect components
5. **Polish**: Documentation and refinements

:::note
The TDD approach requires writing tests **before** implementation. Tests should fail initially, then pass once the code is written.
:::

## Automated Execution

The `spec-kit-executor` workflow automates implementation of features with pending tasks.

### How the Executor Works

The workflow (`.github/workflows/spec-kit-executor.md`):

1. **Scans** for feature specifications in `.specify/specs/`
2. **Identifies** features with pending (unchecked) tasks
3. **Prioritizes**:
   - First: In-progress features (some tasks done)
   - Second: Not-started features (no tasks done)
   - Skip: Complete features (all tasks done)
4. **Loads** the constitution, spec, plan, and tasks
5. **Executes** tasks phase-by-phase
6. **Validates** after each phase (`make fmt`, `make lint`, `make test-unit`)
7. **Creates** a pull request with completed work

### Schedule

The executor runs:
- **Daily** at 8am UTC
- **Manually** via workflow dispatch

### Execution Flow

```aw
---
on:
  schedule:
    - cron: "0 8 * * *"
  workflow_dispatch:

engine: copilot
---

# Spec-Kit Executor

Load constitution and scan for pending specifications.
Implement features following TDD and validation discipline.
Create PRs with completed implementations.
```

### Monitoring Execution

Check workflow runs in GitHub Actions to monitor:
- Which features are being implemented
- Task completion progress
- Validation results
- Pull request creation

:::tip
The executor handles routine implementation automatically. You can focus on reviewing and merging the resulting pull requests.
:::

## Best Practices

### Specification Best Practices

1. **Focus on User Value**: Write user stories that articulate clear benefits
2. **Be Specific**: Define concrete acceptance criteria
3. **Avoid Implementation Details**: Leave technical decisions for the plan
4. **Define Success Metrics**: Include measurable outcomes
5. **Consider Edge Cases**: Think through error conditions and boundaries

### Planning Best Practices

1. **Follow the Constitution**: Align with project principles
2. **Use Existing Patterns**: Build on established code patterns
3. **Plan for Testing**: Define test strategy upfront
4. **Document Architecture**: Clearly explain component relationships
5. **Identify Dependencies**: Note what must exist before implementation
6. **Keep It Simple**: Choose the simplest approach that meets requirements

### Task Breakdown Best Practices

1. **Small Tasks**: Each task should take 15-30 minutes
2. **Clear Dependencies**: Mark what must complete first
3. **Testable Outcomes**: Define how to verify completion
4. **Follow TDD**: Write tests before implementation
5. **Phase by Phase**: Complete each phase before moving on
6. **Validation Points**: Run `make fmt`, `make lint`, `make test-unit` after phases

### Implementation Best Practices

1. **Read the Constitution**: Review principles before starting
2. **Follow the Plan**: Don't deviate without updating the plan
3. **Write Tests First**: TDD is non-negotiable
4. **Validate Frequently**: Run tests after each task
5. **Commit Incrementally**: Use `make recompile` and validation
6. **Document Changes**: Update relevant documentation
7. **Review PRs Carefully**: All implementations need human review

:::caution
Never skip the constitution review. It contains critical requirements that must be followed.
:::

## Troubleshooting

### "Prerequisites not met" Error

**Problem**: Scripts require a feature branch but you're on main.

**Solution**: Check your current branch:
```bash
git branch
```

Should show a pattern like `001-feature-name`. If not, create a specification using `/speckit.specify`.

### "tasks.md not found" Error

**Problem**: Trying to implement without a task breakdown.

**Solution**: Generate tasks first:
```
/speckit.tasks
/speckit.implement
```

### Tests Failing During Implementation

**Problem**: Tests fail unexpectedly during implementation.

**Solution**: Follow TDD:
1. Write tests that fail (expected)
2. Implement code to make tests pass
3. Refactor if needed

Run tests frequently:
```bash
make test-unit  # Fast feedback during development
make test       # Complete validation
```

### Linter Errors

**Problem**: Linter reports formatting or style issues.

**Solution**: Format before linting:
```bash
make fmt   # Format code
make lint  # Run linter
```

### Workflow Compilation Errors

**Problem**: Workflow markdown files don't compile.

**Solution**: Recompile workflows:
```bash
make recompile  # Recompile all workflow lock files
```

If schema changes were made:
```bash
make build      # Rebuild binary first
make recompile  # Then recompile workflows
```

### Constitution Conflicts

**Problem**: Plan or implementation conflicts with constitution principles.

**Solution**:
1. Review `.specify/memory/constitution.md`
2. Align your approach with principles
3. Update plan or tasks to comply
4. Ask for clarification if principles conflict

### Executor Not Running

**Problem**: Spec-kit-executor workflow doesn't pick up your feature.

**Solution**: Verify:
1. Specification is in `.specify/specs/NNN-feature-name/`
2. Files exist: `spec.md`, `plan.md`, `tasks.md`
3. Tasks have unchecked items: `- [ ]` not `- [x]`
4. Branch is pushed to GitHub
5. Wait for scheduled run (8am UTC) or trigger manually

## Examples

### Example Specification

See `.specify/specs/example-feature/` for a complete example showing:
- Well-structured specification with user stories
- Detailed implementation plan with architecture
- Task breakdown following TDD principles
- Documentation explaining the example

This example demonstrates best practices for:
- Writing clear requirements
- Planning technical approach
- Breaking down work into phases
- Following the constitution

### Example Workflow

The spec-kit-executor workflow (`.github/workflows/spec-kit-executor.md`) shows:
- How to scan for pending specifications
- How to load constitution, spec, plan, and tasks
- How to execute tasks phase-by-phase
- How to validate after each phase
- How to create pull requests

### Reference Documentation

- [.specify/README.md](/.specify/README.md) - Complete spec-kit documentation
- [.specify/QUICKSTART.md](/.specify/QUICKSTART.md) - Step-by-step quick start
- [Spec-Kit Repository](https://github.com/github/spec-kit) - Official spec-kit project
- [Project Constitution](/.specify/memory/constitution.md) - Governing principles

## Spec-Kit Commands Reference

When using spec-kit with AI agents, these commands are available:

### `/speckit.constitution`
View or update the project constitution.

### `/speckit.specify`
Create a new feature specification. Focus on what and why.

### `/speckit.plan`
Create an implementation plan. Define technical approach and architecture.

### `/speckit.tasks`
Generate task breakdown from the plan. Creates ordered, actionable tasks.

### `/speckit.implement`
Execute tasks to implement the feature. Follows TDD and validation discipline.

### `/speckit.clarify`
Clarify underspecified areas before planning. Identifies missing requirements.

### `/speckit.analyze`
Analyze consistency across spec, plan, and tasks. Verifies alignment.

### `/speckit.checklist`
Generate custom quality checklists for validation.

## Next Steps

Ready to create your first specification? Follow these steps:

1. **Read the Constitution**: Review `.specify/memory/constitution.md`
2. **Read the Quick Start**: See `.specify/QUICKSTART.md` for a walkthrough
3. **Study the Example**: Explore `.specify/specs/example-feature/`
4. **Create a Specification**: Use `/speckit.specify` to define a feature
5. **Let Automation Work**: The spec-kit-executor will handle implementation
6. **Review and Merge**: Review the resulting PR and merge when ready

For questions or issues, file an issue in the [gh-aw repository](https://github.com/githubnext/gh-aw/issues).
