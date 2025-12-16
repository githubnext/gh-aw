# Example Feature: Workflow Validation Report

This directory contains a complete example specification demonstrating best practices for spec-driven development with gh-aw.

## What This Example Demonstrates

This example shows how to create a comprehensive feature specification following the spec-kit workflow:

1. **Complete Specification** (`spec.md`): Well-structured requirements with user stories, acceptance criteria, and success metrics
2. **Detailed Implementation Plan** (`plan.md`): Technical approach, architecture, and testing strategy
3. **Task Breakdown** (`tasks.md`): Ordered tasks following TDD principles with phases and validation points
4. **Best Practices**: Alignment with the project constitution and development standards

## The Feature

The example feature adds a workflow validation report capability that:
- Analyzes agentic workflow markdown files
- Detects security issues and configuration errors
- Generates comprehensive validation reports
- Integrates with the CLI as a new command

This is a realistic feature that follows actual gh-aw patterns and demonstrates real-world complexity.

## How to Use This Example

### As a Template

Use this example as a template for your own specifications:

1. **Copy the structure**: Use the same sections and organization
2. **Adapt the content**: Replace with your feature's requirements
3. **Follow the patterns**: User stories, acceptance criteria, validation rules
4. **Maintain the format**: Keep the same markdown structure

### As a Learning Resource

Study this example to learn:

- **How to write user stories**: Clear "As a/I want/So that" format with acceptance criteria
- **How to organize requirements**: Functional, non-functional, security, quality categories
- **How to plan architecture**: Component design, data flow, phase breakdown
- **How to create tasks**: TDD approach with phases, dependencies, and acceptance criteria
- **How to follow TDD**: Tests phase before implementation phase

### As a Reference

Reference this example when:

- You're unsure about specification structure
- You need examples of good acceptance criteria
- You want to see how to break down complex features
- You're planning technical architecture
- You need task breakdown patterns

## Key Learning Points

### Specification Best Practices

From `spec.md`:
- **User-focused stories**: Each story articulates clear user value
- **Specific acceptance criteria**: Concrete, testable conditions
- **Multiple requirement types**: Functional, non-functional, security, quality
- **Success metrics**: Measurable outcomes for adoption, quality, performance
- **Explicit scope boundaries**: What's included and what's out of scope

### Planning Best Practices

From `plan.md`:
- **Clear technical approach**: High-level strategy before diving into details
- **Aligned with constitution**: Go-first, minimal changes, TDD, console formatting
- **Component organization**: Logical file structure with clear responsibilities
- **Phase-based implementation**: Setup → Tests → Core → Integration → Polish
- **Comprehensive testing strategy**: Unit, integration, manual, performance
- **Performance considerations**: Optimization strategies and targets

### Task Breakdown Best Practices

From `tasks.md`:
- **TDD discipline**: Tests phase before implementation phase
- **Small, focused tasks**: Each task completable in 15-30 minutes
- **Clear dependencies**: What must complete before each task
- **Acceptance criteria**: How to verify completion
- **Phase validation**: Run validation commands after each phase
- **Incremental progress**: Complete phases before moving forward

## Alignment with Constitution

This example demonstrates alignment with `.specify/memory/constitution.md`:

- ✅ **Go-First Architecture**: Pure Go implementation, no external dependencies
- ✅ **Minimal Changes Philosophy**: Focused feature, surgical additions
- ✅ **Test-Driven Development**: Tests phase before implementation phase
- ✅ **Console Output Standards**: Uses `pkg/console` for all output
- ✅ **Workflow Compilation**: N/A (no workflow changes)
- ✅ **Build & Test Discipline**: Validation after each phase
- ✅ **Security & Quality**: Security rules, 80% test coverage target

## Common Patterns to Emulate

### User Story Format
```
**As a** [user type]
**I want** [goal]
**So that** [benefit]

**Acceptance Criteria:**
- Criterion 1
- Criterion 2
```

### Requirement Numbering
- Functional: FR-1, FR-2, ...
- Non-Functional: NFR-1, NFR-2, ...
- Security: SR-1, SR-2, ...
- Quality: QR-1, QR-2, ...

### Architecture Documentation
```
pkg/feature/
├── component.go       # Description
├── component_test.go  # Description
└── types.go          # Description
```

### Task Structure
```
### X.Y: Task name
- **Dependencies**: Previous task numbers
- **File**: Specific file(s) to create/modify
- **Acceptance**: How to verify completion
```

### Phase Validation
```
**Phase N Validation**: Run `make command` - expected outcome
```

## Tips for Creating Your Own Specifications

1. **Start with user value**: Focus on who benefits and how
2. **Be specific**: Vague requirements lead to vague implementations
3. **Think about edge cases**: Include them in requirements
4. **Plan for testing**: Test strategy should be clear upfront
5. **Follow TDD**: Write tests before implementation
6. **Validate incrementally**: Don't wait until the end
7. **Align with constitution**: Check principles before planning
8. **Keep tasks small**: If a task takes more than 30 minutes, break it down

## Related Documentation

- [Spec-Kit Integration Guide](../../../docs/src/content/docs/guides/spec-kit-integration.md) - Comprehensive guide
- [.specify/README.md](../../README.md) - Spec-kit documentation
- [.specify/QUICKSTART.md](../../QUICKSTART.md) - Quick start guide
- [.specify/memory/constitution.md](../../memory/constitution.md) - Project constitution

## Questions or Feedback

If you have questions about this example or suggestions for improvement:
- File an issue: https://github.com/githubnext/gh-aw/issues
- Reference this example in your issue
- Tag with `spec-kit` label

---

**Note**: This is an example specification for demonstration purposes. It is not currently being implemented. Use it as a template and reference for creating your own specifications.
