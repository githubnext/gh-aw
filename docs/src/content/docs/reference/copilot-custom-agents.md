---
title: Importing Copilot Agent Files
description: Import and reuse Copilot agent files with GitHub Agentic Workflows
sidebar:
  order: 650
---

"Custom agents" is a term used in GitHub Copilot for specialized prompts for behaviors for specific tasks. They are markdown files stored in the `.github/agents/` directory and imported via the `imports` field. Copilot supports agent files natively, while other engines (Claude, Codex) inject the markdown body as a prompt.

A typical custom agent file looks like this:

```markdown title=".github/agents/my-agent.md"
---
name: My Copilot Agent
description: Specialized prompt for code review tasks
---

# Agent Instructions

You are a specialized code review agent. Focus on:
- Code quality and best practices
- Security vulnerabilities
- Performance optimization
```

## Using Copilot Agent Files from Agentic Workflows

Import Copilot agent files in your workflow using the `imports` field. Agent files can be imported from local `.github/agents/` directories or from external repositories.

### Local Agent File Import

Import an agent from your repository:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - .github/agents/my-agent.md
---

Review the pull request and provide feedback.
```

### Remote Agent File Import

Import an agent file from an external repository using the `owner/repo/path@ref` format:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - acme-org/shared-agents/.github/agents/code-reviewer.md@v1.0.0
---

Perform comprehensive code review using shared agent instructions.
```

The agent instructions are merged with the workflow prompt, customizing the AI engine's behavior for specific tasks.

## Agent File Requirements

- **Location**: Must be in a `.github/agents/` directory (local or remote repository)
- **Format**: Markdown with YAML frontmatter
- **Frontmatter**: Can include `name`, `description`, `tools`, and `mcp-servers`
- **One per workflow**: Only one agent file can be imported per workflow
- **Caching**: Remote agent files are cached by commit SHA in `.github/aw/imports/`

## Copilot Agent File Collections

Organizations can create libraries of specialized custom agent files:

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

Teams import agent files based on workflow needs:

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

## Combining Copilot Agent Files with Other Imports

You can mix custom agent file imports with tool configurations and shared components:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  # Import specialized custom agent file
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

## Related Documentation

- [Imports Reference](/gh-aw/reference/imports/) - Complete import system documentation
- [Reusing Workflows](/gh-aw/guides/packaging-imports/) - Managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference