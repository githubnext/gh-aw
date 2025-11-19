# ✨ GitHub Agentic Workflows

[![Documentation](https://img.shields.io/badge/docs-githubnext.github.io%2Fgh--aw-blue)](https://githubnext.github.io/gh-aw/)
[![Version](https://img.shields.io/badge/version-v0.30.0-green)](https://github.com/githubnext/gh-aw/releases)

Write agentic workflows in natural language markdown, and run them safely in GitHub Actions. AI-powered automation that understands context, makes decisions, and takes meaningful actions across your repositories. From [GitHub Next](https://githubnext.com/) and [Microsoft Research](https://www.microsoft.com/en-us/research/group/research-software-engineering-rise/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

<!--
> [!NOTE]
> **For AI Agents**: To learn about GitHub Agentic Workflows syntax, file formats, tools, and best practices, please read the comprehensive instructions at: [.github/instructions/github-agentic-workflows.instructions.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/instructions/github-agentic-workflows.instructions.md)
>
> **Custom Agent**: Use the custom agent at `.github/agents/create-agentic-workflow.md` to interactively create agentic workflows. The custom agent is available at: [.github/agents/create-agentic-workflow.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/agents/create-agentic-workflow.md)
-->

## 🚀 Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/) to:

1. Install the `gh aw` CLI extension
2. Add a sample workflow from the [agentics collection](https://github.com/githubnext/agentics)
3. Configure AI engine secrets (GitHub Copilot, Claude, or Codex)
4. Run your first AI-powered automation

```bash
# Install the extension
gh extension install githubnext/gh-aw

# Add a workflow
gh aw add githubnext/agentics/daily-team-status --pr

# Run it
gh aw run daily-team-status
```

## 🔧 How It Works

GitHub Agentic Workflows transforms natural language markdown files into GitHub Actions that are executed by AI agents. Here's a simple example:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  contents: read

safe-outputs:
  add-comment:
---

# Issue Clarifier

Analyze the current issue and ask for additional details if the issue is unclear.
```

The `gh aw compile` command converts this into a GitHub Actions Workflow (`.lock.yml`) that runs an AI agent in a secure, sandboxed environment whenever a new issue is opened.

The AI agent reads your repository context, understands the issue content, and takes appropriate actions—all defined in natural language rather than complex code.

### Security-First Design

- **Read-only by default**: Workflows use minimal permissions with read-only access
- **Safe outputs**: Write operations (creating issues, comments, PRs) are handled through sanitized `safe-outputs` that operate in separate jobs with controlled permissions
- **Team gating**: Access can be restricted to specific repository roles (admin, maintainer, write)
- **Sandboxed execution**: AI agents run in isolated containers with network controls
- **Strict mode**: Enhanced validation for production workflows with security guardrails

Learn more in the [Security Guide](https://githubnext.github.io/gh-aw/guides/security/).

## ✨ Key Features

### Natural Language Workflows
Write automation in plain markdown instead of complex YAML syntax. Describe what you want to happen, and AI agents interpret and execute your intentions.

### Multiple AI Engines
- **GitHub Copilot** (default) - Integrated with GitHub CLI
- **Claude** - Anthropic's advanced AI model
- **Codex** - OpenAI's code-specialized model
- **Custom** - Bring your own AI processor

See [AI Engines documentation](https://githubnext.github.io/gh-aw/reference/engines/).

### Safe Outputs
Create issues, comments, discussions, and pull requests without giving AI agents write permissions. Safe outputs process AI-generated content in isolated jobs with minimal permissions.

**Available safe outputs:**
- Create issues, discussions, and agent tasks
- Add comments to issues/PRs
- Create pull requests with code changes
- Update issues and project boards
- Add labels and manage releases
- Generate security alerts (SARIF)

See [Safe Outputs documentation](https://githubnext.github.io/gh-aw/reference/safe-outputs/).

### GitHub Integration
Deep integration with GitHub APIs through the [GitHub MCP Server](https://github.com/github/github-mcp-server):
- Repository management and file operations
- Issues, pull requests, and discussions
- Code search and workflow analysis
- Actions logs and security scanning

### Network Control
Configure network access with ecosystem-based allowlists and optional firewall enforcement:
- Ecosystem identifiers (python, node, containers, etc.)
- Domain-based access control with wildcards
- AWF (Agent Workflow Firewall) for Copilot engine
- Network activity logging and audit trails

See [Network Permissions documentation](https://githubnext.github.io/gh-aw/reference/network/).

### Persistent Memory
Store and retrieve files across workflow runs using cache-memory:
- Persistent file storage backed by GitHub Actions cache
- Multiple named caches per workflow
- Simple file system operations
- Automatic cache restoration with fallback keys

See [Memory documentation](https://githubnext.github.io/gh-aw/reference/cache-memory/).

### MCP Server Support
Extend workflows with [Model Context Protocol](https://modelcontextprotocol.io/) servers:
- Built-in GitHub, Playwright, and tool servers
- Custom MCP servers (local commands or HTTP endpoints)
- Tool inspection and discovery with `gh aw mcp inspect`
- Container-based execution for isolation

See [MCP documentation](https://githubnext.github.io/gh-aw/guides/mcps/).

## 🎯 What You Can Build

### ChatOps - Interactive Automation
Trigger workflows with slash commands like `/review`, `/deploy`, or custom commands. AI agents respond to natural language requests in issue and PR comments.

**Example:** `/analyze performance` triggers a workflow that profiles your code and suggests optimizations.

[Learn more about ChatOps →](https://githubnext.github.io/gh-aw/examples/comment-triggered/chatops/)

### IssueOps - Intelligent Issue Management
Automatically triage, label, and respond to issues based on their content. AI agents analyze issues and take appropriate actions without manual intervention.

**Example:** Auto-categorize bug reports, request clarification on vague issues, or route to the right team.

[Learn more about IssueOps →](https://githubnext.github.io/gh-aw/examples/issue-pr-events/issueops/)

### DailyOps - Continuous Improvements
Schedule daily workflows that make small, compound improvements over time. Research best practices, update documentation, or optimize configurations.

**Example:** Daily code quality checks, dependency updates, or technical debt reduction.

[Learn more about DailyOps →](https://githubnext.github.io/gh-aw/examples/scheduled/dailyops/)

### Code Review & Testing
Automated PR analysis, test coverage improvements, and code quality checks. AI agents review changes and provide actionable feedback.

**Example:** Identify missing tests, suggest performance optimizations, or flag security concerns.

[Learn more about Quality & Testing →](https://githubnext.github.io/gh-aw/examples/issue-pr-events/quality-testing/)

### Research & Planning
Weekly research reports, competitive analysis, and automated status updates. AI agents synthesize information and generate insights.

**Example:** Weekly tech stack updates, industry trend reports, or team progress summaries.

[Learn more about Research & Planning →](https://githubnext.github.io/gh-aw/examples/scheduled/research-planning/)

Explore 80+ workflow examples in the [Agentics collection](https://github.com/githubnext/agentics) and [Labs](https://githubnext.github.io/gh-aw/labs/).

## 🛠️ CLI Commands

The `gh aw` CLI provides comprehensive workflow management:

### Setup & Development
```bash
gh aw init                    # Initialize repository
gh aw new my-workflow         # Create new workflow
gh aw add owner/repo/path     # Add from repository
gh aw compile                 # Compile all workflows
gh aw compile --strict        # Compile with security validation
```

### Execution & Monitoring
```bash
gh aw run workflow-name       # Run workflow on GitHub Actions
gh aw status                  # Show workflow status
gh aw logs workflow-name      # Download and analyze logs
gh aw audit <run-id>          # Debug failed runs
```

### MCP & Tools
```bash
gh aw mcp inspect             # List MCP servers
gh aw mcp list-tools github   # Show available tools
gh aw mcp-server              # Run as MCP server
```

### Management
```bash
gh aw enable                  # Enable workflows
gh aw disable                 # Disable and cancel runs
gh aw update                  # Update workflows from sources
gh aw remove workflow-prefix  # Remove workflows
```

See [CLI documentation](https://githubnext.github.io/gh-aw/setup/cli/) for complete command reference.

## 📖 Documentation

For complete documentation, examples, and guides, visit the [Documentation](https://githubnext.github.io/gh-aw/).

## 🤝 Contributing

We welcome contributions to GitHub Agentic Workflows! Here's how you can help:

- **🐛 Report bugs and request features** by filing issues in this repository
- **📖 Improve documentation** by contributing to our docs
- **🔧 Contribute code** by following our [Development Guide](DEVGUIDE.md)
- **💡 Share ideas** in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## 💬 Share Feedback

We welcome your feedback on GitHub Agentic Workflows! Please file bugs and feature requests as issues in this repository, and share your thoughts in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord).

## 🧪 Labs

See the [Labs](https://githubnext.github.io/gh-aw/labs/) page for experimental agentic workflows used by the team to learn, build, and use agentic workflows.

## 📚 Additional Resources

- [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/) - Get started in minutes
- [How It Works](https://githubnext.github.io/gh-aw/introduction/how-it-works/) - Understanding the architecture
- [Security Guide](https://githubnext.github.io/gh-aw/guides/security/) - Best practices for secure workflows
- [Examples](https://githubnext.github.io/gh-aw/examples/) - Browse workflow patterns and use cases
- [Agentics Collection](https://github.com/githubnext/agentics) - 80+ ready-to-use workflows
- [CLI Reference](https://githubnext.github.io/gh-aw/setup/cli/) - Complete command documentation
