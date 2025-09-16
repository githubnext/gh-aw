# ✨ GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them safely in GitHub Actions. From [GitHub Next](https://githubnext.com/) and [Microsoft Research](https://www.microsoft.com/en-us/research/group/research-software-engineering-rise/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## 🚀 Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://githubnext.github.io/gh-aw/start-here/quick-start/) to install the extension, add a sample workflow, and see it in action.

## 📖 Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [Concepts](https://githubnext.github.io/gh-aw/start-here/concepts/).

## 🔧 How It Works

GitHub Agentic Workflows transforms natural language markdown files into GitHub Actions that are executed by AI agents. Here's a simple example:

```markdown
---
on:
  issues:
    types: [opened]
permissions: read-all 
safe-outputs:
  add-issue-comment:
---
# Issue Clarifier

Analyze the current issue and ask for additional details if the issue is unclear.
```

The `gh aw` cli compiles this into a GitHub Actions Workflow (.yml) that runs an AI agent (Claude, Codex, ...) in a containerized environment whenever a new issue is opened in the repository.

The AI agent reads your repository context, understands the issue content, and takes appropriate actions - all defined in natural language rather than complex code.

**Security Benefits:** Workflows use read-only permissions by default, with write operations only allowed through sanitized `safe-outputs`. Access can be gated to team members only, ensuring AI agents operate within controlled boundaries.

## 🎨 VSCode Extension

The repository includes a minimalistic VSCode extension that provides syntax highlighting and schema validation for agentic workflow files:

- **Syntax Highlighting**: Rich highlighting for YAML frontmatter and markdown content
- **Schema Validation**: IntelliSense and validation for workflow configuration
- **Auto-completion**: Smart completions for properties and values
- **Hover Documentation**: Helpful tooltips explaining workflow features

The extension automatically detects `.md` files in `.github/workflows/` and provides enhanced editing support. To build the extension:

```bash
make vscode-compile
```

The extension source is located in [`vscode/gh-aw/`](vscode/gh-aw/) with full TypeScript support and development configurations.

## 📖 Documentation

For complete documentation, examples, and guides, see the [Documentation](https://githubnext.github.io/gh-aw/).

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
