# ✨ GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them safely in GitHub Actions. From [GitHub Next](https://githubnext.com/) and [Microsoft Research](https://www.microsoft.com/en-us/research/group/research-software-engineering-rise/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

<!--
> [!NOTE]
> **For AI Agents**: To learn about GitHub Agentic Workflows syntax, file formats, tools, and best practices, please read the comprehensive instructions at: [.github/instructions/github-agentic-workflows.instructions.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/instructions/github-agentic-workflows.instructions.md)
>
> **Custom Agent**: Use the custom agent at `.github/agents/create-agentic-workflow.md` to interactively create agentic workflows. The custom agent is available at: [.github/agents/create-agentic-workflow.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/agents/create-agentic-workflow.md)
-->

## 🚀 Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/) to install the extension, add a sample workflow, and see it in action.

**Quick Setup:**
```bash
# Install the extension
gh extension install githubnext/gh-aw

# Initialize your repository
gh aw init

# Add a sample workflow
gh aw add githubnext/agentics/daily-team-status --pr

# Test safely without side effects
gh aw trial githubnext/agentics/daily-team-status
```

## 📖 Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://githubnext.github.io/gh-aw/introduction/how-it-works/).

## 🔧 How It Works

GitHub Agentic Workflows transforms natural language markdown files into GitHub Actions that are executed by AI agents. Here's a simple example:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  contents: read
  issues: read

safe-outputs:
  add-comment:
---

# Issue Clarifier

Analyze the current issue and ask for additional details if the issue is unclear.
```

The `gh aw compile` command converts this into a GitHub Actions workflow (`.lock.yml`) that runs an AI agent in a sandboxed environment whenever an issue is opened. The AI agent reads your repository context, understands the issue content, and takes appropriate actions - all defined in natural language.

**Key Features:**
- **Multiple AI Engines**: Choose from GitHub Copilot (default), Claude, or Codex
- **Security First**: Read-only permissions by default, write operations through sanitized [`safe-outputs`](https://githubnext.github.io/gh-aw/reference/safe-outputs/)
- **Safe Testing**: Use `gh aw trial` to test workflows in temporary repositories without side effects
- **MCP Integration**: Expose workflow tools via [Model Context Protocol](https://githubnext.github.io/gh-aw/setup/mcp-server/) for AI agents
- **Debugging Built-in**: Analyze failures with `gh aw audit` and view metrics with `gh aw logs`

## 🛠️ Common Commands

| Command | Description |
|---------|-------------|
| `gh aw init` | Initialize repository with workflow configuration |
| `gh aw new <name>` | Create a new workflow from template |
| `gh aw add <source>` | Add workflow from remote repository |
| `gh aw compile` | Compile markdown workflows to GitHub Actions YAML |
| `gh aw trial <workflow>` | Test workflow safely in temporary repository |
| `gh aw run <workflow>` | Execute workflow immediately |
| `gh aw status` | Show status of all workflows |
| `gh aw logs <workflow>` | Download and analyze execution logs with metrics |
| `gh aw audit <run-id>` | Investigate failed workflow runs |
| `gh aw mcp-server` | Start MCP server for AI agent integration |

For detailed command documentation, see the [CLI Reference](https://githubnext.github.io/gh-aw/setup/cli/).

## 📖 Documentation

For complete documentation, examples, and guides, see the [Documentation](https://githubnext.github.io/gh-aw/).

**Key Resources:**
- **[Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/)** - Get up and running in minutes
- **[Workflow Examples](https://githubnext.github.io/gh-aw/examples/issue-pr-events/issueops/)** - IssueOps, ChatOps, DailyOps patterns
- **[AI Engines](https://githubnext.github.io/gh-aw/reference/engines/)** - Configure Copilot, Claude, or Codex
- **[Safe Outputs](https://githubnext.github.io/gh-aw/reference/safe-outputs/)** - Secure GitHub API operations
- **[Security Guide](https://githubnext.github.io/gh-aw/guides/security/)** - Best practices and threat model
- **[Troubleshooting](https://githubnext.github.io/gh-aw/troubleshooting/common-issues/)** - Common issues and solutions

## 🤝 Contributing

We welcome contributions to GitHub Agentic Workflows! Here's how you can help:

- **🐛 Report bugs and request features** by filing issues in this repository
- **📖 Improve documentation** by contributing to our docs
- **🔧 Contribute code** by following our [Development Guide](DEVGUIDE.md)
- **💡 Share ideas** in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## 💬 Share Feedback

We welcome your feedback on GitHub Agentic Workflows! Please file bugs and feature requests as issues in this repository,
and share your thoughts in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord).

## 🧪 Labs

See the [Labs](https://githubnext.github.io/gh-aw/labs/) page for experimental agentic workflows used by the team to learn, build, and use agentic workflows.
