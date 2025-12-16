---
marp: true
theme: default
paginate: true
---

# GitHub Agentic Workflows
## Write AI Automation in Natural Language
### Research Preview

https://github.com/githubnext/gh-aw

---

# Overview

GitHub Agentic Workflows (gh-aw) is a CLI tool and GitHub extension that enables developers to create AI-powered automation workflows using natural language.

**Key Features:**
- Natural language workflow definitions
- Multiple AI engine support (Copilot, Claude, Codex)
- Built-in security and safety controls
- Containerized execution environment
- Model Context Protocol (MCP) integration

---

# Getting Started

Install the GitHub CLI extension:

```bash
gh extension install githubnext/gh-aw
gh aw init
```

Create your first workflow:

```bash
gh aw compile
```

---

# Workflow Format

Agentic workflows use markdown with YAML frontmatter:

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  add-comment:
---
Summarize this issue and respond in a comment.
```

---

# Security by Default

- **Read-only permissions** - Default to minimal access
- **Safe outputs** - Validated write operations
- **Network firewall** - Control external access
- **Container isolation** - Sandboxed execution
- **MCP proxy** - Secure tool access

---

# Tools & Integrations

Built-in tools:
- `bash` - Shell command execution
- `edit` - File editing capabilities
- `web-fetch` / `web-search` - Web access
- `github` - GitHub API operations
- `playwright` - Browser automation
- `agentic-workflows` - Workflow introspection
- `cache-memory` / `repo-memory` - Persistent storage

---

# MCP Servers

Extend workflows with Model Context Protocol:

```yaml
mcp-servers:
  custom-analyzer:
    command: "node"
    args: ["path/to/server.js"]
    allowed: ["analyze", "report"]
```

---

# AI Engines

Multiple engine options:
- **Copilot** - GitHub's AI pair programmer
- **Claude** - Anthropic's Claude models
- **Codex** - OpenAI's code model
- **Custom** - Bring your own AI

---

# Safe Outputs

Controlled write operations by category:

**Issues & PRs:**
- `create-issue`, `update-issue`, `close-issue`
- `create-pull-request`, `update-pull-request`
- `add-comment`, `create-pull-request-review-comment`

**Discussions & Organization:**
- `create-discussion`, `close-discussion`
- `add-labels`, `assign-milestone`, `add-reviewer`
- `assign-to-user`, `assign-to-agent`

**Other:** `update-project`, `update-release`, `create-code-scanning-alert`

---

# Cache & Memory

Speed up workflows with persistent memory:

```yaml
cache-memory: true
```

Benefits:
- Faster execution
- Context retention across runs
- Reduced token usage

---

# Network Control

Fine-grained network access:

```yaml
network:
  allowed:
    - defaults  # Core infrastructure
    - node      # NPM ecosystem
    - "*.github.com"
```

---

# Monitoring & Logs

Track workflow performance:

```bash
# View recent runs
gh aw logs

# Filter by workflow
gh aw logs accessibility-review

# Analyze specific run
gh aw audit 123456
```

---

# Documentation

Complete documentation available at:

https://githubnext.github.io/gh-aw/

Topics covered:
- Setup and installation
- Workflow creation
- Security best practices
- Tool configuration
- API reference

---

# Community & Support

- **GitHub Repository**: githubnext/gh-aw
- **Documentation**: githubnext.github.io/gh-aw
- **Issues**: Report bugs and request features
- **Discussions**: Community support

---

# Thank You!

Questions?

Visit: https://github.com/githubnext/gh-aw
