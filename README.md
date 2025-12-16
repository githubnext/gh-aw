# GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them safely in GitHub Actions. From [GitHub Next](https://githubnext.com/) and [Microsoft Research](https://www.microsoft.com/en-us/research/group/research-software-engineering-rise/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

<!--
> [!NOTE]
> **For AI Agents**: To learn about GitHub Agentic Workflows syntax, file formats, tools, and best practices, please read the comprehensive instructions at: [.github/aw/github-agentic-workflows.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/aw/github-agentic-workflows.md)
>
> **Custom Agent**: Use the custom agent at `.github/agents/create-agentic-workflow.md` to interactively create agentic workflows. The custom agent is available at: [.github/agents/create-agentic-workflow.md](https://raw.githubusercontent.com/githubnext/gh-aw/main/.github/agents/create-agentic-workflow.md)
-->

## Quick Start

Ready to get your first agentic workflow running? Follow our step-by-step [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/) to install the extension, add a sample workflow, and see it in action.

## Overview

Learn about the concepts behind agentic workflows, explore available workflow types, and understand how AI can automate your repository tasks. See [How It Works](https://githubnext.github.io/gh-aw/introduction/how-it-works/).

## How It Works

GitHub Agentic Workflows transforms natural language markdown files into GitHub Actions that are executed by AI agents. Here's an example:

```markdown
---
on:
  schedule:
    - cron: "0 6 * * *"

permissions: read

safe-outputs:
  create-discussion:
---

# Daily Issues Report

Analyze repository issues and create a daily discussion 
with metrics, trends, and key insights.
```

The `gh aw` cli converts this into a GitHub Actions Workflow (.yml) that runs an AI agent (Copilot, Claude, Codex, ...) in a containerized environment on a schedule or manually.

The AI agent reads your repository context, analyzes issues, generates visualizations, and creates reports - all defined in natural language rather than complex code.

**Security Benefits:** Workflows use read-only permissions by default, with write operations only allowed through sanitized `safe-outputs`. Access can be gated to team members only, ensuring AI agents operate within controlled boundaries.

## Spec-Driven Development

GitHub Agentic Workflows supports spec-driven development through [spec-kit](https://github.com/github/spec-kit) integration. Define features in natural language specifications, create implementation plans, and execute them systematically.

The spec-kit workflow follows a structured process:

1. **Constitution** - Project principles guide all decisions
2. **Specification** - Define requirements and user stories
3. **Plan** - Create technical implementation approach
4. **Tasks** - Break down work into ordered tasks
5. **Implementation** - Execute tasks following TDD principles

An automated executor workflow scans for pending specifications and implements them daily, creating pull requests for review.

**Get Started:**
- Quick start guide: [.specify/QUICKSTART.md](.specify/QUICKSTART.md)
- Complete documentation: [.specify/README.md](.specify/README.md)
- Comprehensive guide: [Spec-Kit Integration](https://githubnext.github.io/gh-aw/guides/spec-kit-integration/)
- Example specification: [.specify/specs/example-feature/](.specify/specs/example-feature/)

## Documentation

For complete documentation, examples, and guides, see the [Documentation](https://githubnext.github.io/gh-aw/).

## Contributing

We welcome contributions to GitHub Agentic Workflows! Here's how you can help:

- **Report bugs and request features** by filing issues in this repository
- **Improve documentation** by contributing to our docs
- **Contribute code** by following our [Development Guide](DEVGUIDE.md)
- **Share ideas** in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)

For development setup and contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Share Feedback

We welcome your feedback on GitHub Agentic Workflows! Please file bugs and feature requests as issues in this repository,
and share your thoughts in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord).

## Labs

See the [Labs](https://githubnext.github.io/gh-aw/labs/) page for experimental agentic workflows used by the team to learn, build, and use agentic workflows.
