---
title: Agent File Imports
description: Learn how to import and reuse specialized AI agent files from external repositories to enhance your workflows with expert-crafted instructions and behavior.
sidebar:
  order: 7
---

GitHub Agentic Workflows supports importing "agent" files from external repositories, usually related to other AI systems, enabling you to reuse expert-crafted AI instructions and specialized behavior across teams and projects.

Importing agent files from external repositories enables reuse, consistency, and modularity.

## Basic Example

### Importing a Code Review Agent

Import a specialized code review agent file from an external repository:

```yaml wrap title=".github/workflows/pr-review.md"
---
on: pull_request
engine: copilot
imports:
  - acme-org/shared-agents/.github/agents/code-reviewer.md@v1.0.0
permissions:
  contents: read
  pull-requests: write
---

# Automated Code Review

Review this pull request for:
- Code quality and best practices
- Security vulnerabilities
- Performance issues
```

The agent file in `acme-org/shared-agents/.github/agents/code-reviewer.md` contains specialized instructions:

```markdown title="acme-org/shared-agents/.github/agents/code-reviewer.md"
---
name: Expert Code Reviewer
description: Specialized agent file for comprehensive code review
tools:
  github:
    toolsets: [pull_requests, repos]
---

# Code Review Instructions

You are an expert code reviewer with deep knowledge of:
- Security best practices (OWASP Top 10)
- Performance optimization patterns
- Code maintainability and readability

When reviewing code:
1. Identify security vulnerabilities first
2. Check for performance issues
3. Ensure code follows team conventions
4. Suggest specific improvements with examples
```

## Versioning Agents

Use semantic versioning to control agent file updates:

```yaml wrap
imports:
  # Production - pin to specific version
  - acme-org/ai-agents/.github/agents/security-auditor.md@v2.0.0
  
  # Development - use latest
  - acme-org/ai-agents/.github/agents/performance.md@main
  
  # Immutable - pin to commit SHA
  - acme-org/ai-agents/.github/agents/custom.md@abc123def
```

## Agent File Collections

Organizations can create libraries of specialized agent files:

```text
acme-org/ai-agents/
└── .github/
    └── agents/
        ├── code-reviewer.md         # General code review
        ├── security-auditor.md      # Security-focused analysis
        ├── performance-analyst.md   # Performance optimization
        ├── accessibility-checker.md # WCAG compliance
        └── documentation-writer.md  # Technical documentation
```

Teams import agents files based on workflow needs:

```yaml wrap title="Security-focused PR review"
---
on: pull_request
engine: copilot
imports:
  - acme-org/ai-agents/.github/agents/security-auditor.md@v2.0.0
  - acme-org/ai-agents/.github/agents/code-reviewer.md@v1.5.0
---

# Security Review

Perform comprehensive security review of this pull request.
```

```yaml wrap title="Accessibility-focused PR review"
---
on: pull_request
engine: copilot
imports:
  - acme-org/ai-agents/.github/agents/accessibility-checker.md@v1.0.0
---

# Accessibility Review

Check this pull request for WCAG 2.1 compliance issues.
```

## Combining Agents Files with Other Imports

You can mix agent file imports with tool configurations and shared components:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  # Import specialized agent files
  - acme-org/ai-agents/.github/agents/security-auditor.md@v2.0.0
  
  # Import tool configurations
  - acme-org/workflow-library/shared/tools/github-standard.md@v1.0.0
  
  # Import MCP servers
  - acme-org/workflow-library/shared/mcp/database.md@v1.0.0
  
  # Import security policies
  - acme-org/workflow-library/shared/config/security-policies.md@v1.0.0
permissions:
  contents: read
  pull-requests: write
safe-outputs:
  create-pull-request-review-comment:
    max: 10
---

# Comprehensive Security Review

Perform detailed security analysis using specialized agent files and tools.
```

## Constraints

- **One specialized agent file per workflow**: Only one agent file can be imported per workflow
- **Agent path detection**: Files in `.github/agents/` are automatically recognized
- **Local or remote**: Can import from local `.github/agents/` or remote repositories

## Related Documentation

- [Custom Agents Reference](/gh-aw/reference/copilot-custom-agents/) - Agent file format and requirements
- [Imports Reference](/gh-aw/reference/imports/) - Complete import system documentation
- [Packaging & Distribution](/gh-aw/guides/packaging-imports/) - Managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
