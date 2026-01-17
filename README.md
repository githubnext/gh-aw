# 🤖 GitHub Agentic Workflows

> **Transform natural language into powerful AI-driven automation for your repositories**

Write agentic workflows in simple markdown and run them as GitHub Actions—no complex code required. Let AI agents handle the heavy lifting: analyzing issues, generating reports, reviewing PRs, and automating repository tasks.

[![Documentation](https://img.shields.io/badge/docs-githubnext.github.io-blue)](https://githubnext.github.io/gh-aw/)
[![GitHub Next](https://img.shields.io/badge/GitHub-Next-purple)](https://githubnext.com/)
[![Discord](https://img.shields.io/badge/Discord-continuous--ai-5865F2)](https://gh.io/next-discord)

---

## 🚀 Quick Example

Transform this simple markdown:

```markdown
---
on:
  schedule: daily
permissions:
  contents: read
  issues: read
safe-outputs:
  create-issue:
    title-prefix: "[team-status] "
    labels: [report, daily-status]
---

## Daily Issues Report

Create an upbeat daily status report for the team as a GitHub issue.
```

Into a fully automated AI workflow that:
- 🔍 Analyzes your repository's issues
- 📊 Generates insights and visualizations
- ✍️ Creates well-formatted reports
- 🎯 Posts them as GitHub issues—all hands-free!

## ✨ Why GitHub Agentic Workflows?

<table>
<tr>
<td width="33%" valign="top">

### 🎯 **Simple to Use**

Write workflows in plain English markdown—no complex code or APIs to learn. If you can write a GitHub issue, you can create an agentic workflow.

</td>
<td width="33%" valign="top">

### 🔒 **Security First**

Built with multiple layers of protection: sandboxed execution, read-only by default, validated safe-outputs, and supply chain security with SHA-pinned dependencies.

</td>
<td width="33%" valign="top">

### ⚡ **Powerful & Flexible**

Choose from multiple AI engines (Copilot, Claude, Codex), integrate with tools and APIs, and extend with custom actions—all within GitHub's familiar environment.

</td>
</tr>
</table>

## 🎬 Get Started in 3 Steps

### 1️⃣ Install the CLI Extension

```bash
gh extension install githubnext/gh-aw
```

### 2️⃣ Create Your First Workflow

```bash
gh aw new my-first-workflow.md
```

### 3️⃣ Compile and Run

```bash
gh aw compile my-first-workflow.md
git add . && git commit -m "Add agentic workflow" && git push
```

🎉 **That's it!** Your AI agent is now running on GitHub Actions.

👉 **[Full Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/)**

## 📚 What Can You Build?

<details>
<summary><b>📊 Automated Status Reports</b></summary>

Generate daily, weekly, or on-demand reports about repository activity, pull request status, issue trends, and team productivity.

</details>

<details>
<summary><b>🔍 PR Review Assistant</b></summary>

Automatically review pull requests for code quality, security issues, documentation completeness, and adherence to coding standards.

</details>

<details>
<summary><b>🐛 Bug Triage &amp; Analysis</b></summary>

Analyze incoming issues, categorize bugs, detect duplicates, suggest labels, and route to appropriate team members.

</details>

<details>
<summary><b>📝 Documentation Generator</b></summary>

Keep documentation up-to-date by analyzing code changes and automatically updating relevant docs, READMEs, and API references.

</details>

<details>
<summary><b>🎯 Project Management</b></summary>

Update project boards, track milestones, generate burndown charts, and send status updates based on repository activity.

</details>

<details>
<summary><b>🔐 Security Scanning</b></summary>

Scan for security vulnerabilities, check dependencies, validate configurations, and create security advisories automatically.

</details>

**🌟 See more examples:** [Peli's Agent Factory](https://githubnext.github.io/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)

## 🛡️ Safe Agentic Workflows

Security isn't an afterthought—it's foundational:

- 🔐 **Read-only by default** - Agents never get write permissions
- ✅ **Validated safe-outputs** - All write operations sanitized and validated
- 📦 **Sandboxed execution** - Isolated containerized environment
- 🔒 **Network isolation** - Control exactly what agents can access
- 📌 **Supply chain security** - SHA-pinned dependencies
- 🛠️ **Tool allow-listing** - Explicit tool permissions
- ✔️ **Compile-time validation** - Catch issues before deployment

> [!WARNING]
> Using agentic workflows requires careful attention to security considerations and human supervision. Use with caution and at your own risk.

**📖 [Security Guide](https://githubnext.github.io/gh-aw/guides/security/)**

## 🎨 Key Features

| Feature | Description |
|---------|-------------|
| **🤖 Multiple AI Engines** | Choose from GitHub Copilot, Claude, Codex, or custom engines |
| **🔧 Extensible Tools** | Integrate with GitHub API, MCP servers, and custom tools |
| **🌐 Network Control** | Domain-restricted access with firewall integration |
| **🎭 Browser Automation** | Built-in Playwright for web scraping and testing |
| **📝 Safe Outputs** | Create issues, PRs, comments without write permissions |
| **⚡ Event-Driven** | Trigger on push, issues, PRs, schedules, or manual dispatch |
| **📊 Rich Context** | Access repository data, files, issues, and PRs |
| **🔄 Custom Actions** | Build reusable workflow components |

## 📖 Documentation

- **[Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/)** - Get up and running in minutes
- **[How It Works](https://githubnext.github.io/gh-aw/introduction/how-it-works/)** - Understand the concepts
- **[Safe Outputs Reference](https://githubnext.github.io/gh-aw/reference/safe-outputs/)** - Learn about validated operations
- **[Security Guide](https://githubnext.github.io/gh-aw/guides/security/)** - Security best practices
- **[Complete Documentation](https://githubnext.github.io/gh-aw/)** - Full reference and guides

## 🤝 Contributing

We welcome contributions! Here's how you can help:

- 🐛 **Report bugs** - File issues in this repository
- 💡 **Request features** - Share your ideas
- 📝 **Improve docs** - Help others learn
- 💻 **Contribute code** - See our [Development Guide](DEVGUIDE.md)
- 💬 **Share ideas** - Join `#continuous-ai` in the [GitHub Next Discord](https://gh.io/next-discord)

**Development Setup:** [CONTRIBUTING.md](CONTRIBUTING.md)

## 💬 Share Feedback

We'd love to hear from you!

- 📝 **File issues** - Report bugs or request features
- 💬 **Join Discord** - Chat in `#continuous-ai` at [GitHub Next Discord](https://gh.io/next-discord)
- 🌟 **Star the repo** - Show your support

## 🏭 Peli's Agent Factory

Take a guided tour through creative uses of agentic workflows, from simple automations to complex multi-agent systems.

**[Visit Peli's Agent Factory →](https://githubnext.github.io/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)**

## 🔗 Related Projects

GitHub Agentic Workflows is supported by companion projects:

- **[Agent Workflow Firewall (AWF)](https://github.com/githubnext/gh-aw-firewall)** - Network egress control with domain-based access controls and activity logging
- **[MCP Gateway](https://github.com/githubnext/gh-aw-mcpg)** - Routes MCP server calls through a unified HTTP gateway for centralized management
- **[The Agentics](https://github.com/githubnext/agentics)** - Reusable agentic workflow components, tools, and templates

---

<div align="center">

**Built with ❤️ by [GitHub Next](https://githubnext.com/)**

[Documentation](https://githubnext.github.io/gh-aw/) • [Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/) • [Security](https://githubnext.github.io/gh-aw/guides/security/) • [Discord](https://gh.io/next-discord)

</div>
