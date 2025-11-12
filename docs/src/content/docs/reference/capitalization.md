---
title: Capitalization Guidelines
description: Understanding capitalization conventions for "agentic workflows" vs "Agentic Workflows" in command descriptions.
sidebar:
  order: 900
---

The gh-aw CLI follows context-based capitalization to distinguish between the product name and generic workflow references. This approach provides clarity and maintains professional consistency across all user-facing text.

## Capitalization Rules

The CLI uses different capitalization based on context:

| Context | Format | Example |
|---------|--------|---------|
| Product name | **Capitalized** | "GitHub Agentic Workflows CLI from GitHub Next" |
| Generic workflows | **Lowercase** | "Enable agentic workflows" |
| Technical terms | **Capitalized** | "Compile Markdown to YAML workflows" |

## Design Rationale

This approach balances several considerations:

### Clarity

Using different capitalization distinguishes between referring to the product (GitHub Agentic Workflows) and the concept (agentic workflows). Users can immediately understand whether documentation discusses the tool itself or the workflows it manages.

### Industry Standards

The pattern follows established conventions where product names use title case while generic references use sentence case. This matches how other CLI tools handle similar distinctions (e.g., "GitHub Actions" as a product vs. "actions" as workflow components).

### Consistency

Applying the same rules across all commands creates a predictable user experience. The pattern extends to:

- Command short descriptions
- Command long descriptions  
- Help text
- Error messages
- Documentation

### Maintenance

The approach simplifies decision-making for contributors. The rule is simple: capitalize when referring to the product by name, use lowercase for generic workflow references.

## Examples

### Product References (Capitalized)

When referring to the gh-aw tool or GitHub Agentic Workflows as a product:

```
GitHub Agentic Workflows CLI from GitHub Next
GitHub Agentic Workflows from GitHub Next
```

### Generic Workflow References (Lowercase)

When describing actions performed on workflows:

```
Enable agentic workflows
Disable agentic workflows and cancel any in-progress runs
Show status of agentic workflows
Initialize repository for agentic workflows
Download and analyze agentic workflow logs with aggregated metrics
Add an MCP tool to an agentic workflow
List MCP servers defined in agentic workflows
Run one or more agentic workflows on GitHub Actions
Trial one or more agentic workflows as if they were running in a repository
```

### Technical Terms (Capitalized)

Proper nouns and acronyms maintain their standard capitalization:

```
Compile Markdown to YAML workflows
MCP helpers
```

## Implementation

The capitalization rules are enforced through automated tests in `cmd/gh-aw/capitalization_test.go`. These tests verify:

- Product name uses "GitHub Agentic Workflows" (capitalized)
- Generic workflow references use "agentic workflows" (lowercase)
- Technical terms maintain proper capitalization

The tests run as part of the standard test suite and prevent inconsistencies from being introduced.

## Historical Context

This pattern was established to resolve potential confusion between the product and the concept. Early analysis confirmed the codebase already followed this approach consistently, and the formalization through tests and documentation prevents future drift.
