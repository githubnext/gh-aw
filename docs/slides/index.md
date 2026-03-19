---
marp: true
theme: gh-aw
paginate: true
---

<script src="../js/mermaid.min.js">
mermaid.initialize({ startOnLoad: true });
</script>

# GitHub Agentic Workflows

## Agentic Processes for Continuous AI

### Technical Preview

<https://github.com/github/gh-aw>

---

# Software Engineer → Agentic Engineer

| Software Engineer | Agentic Engineer |
|---|---|
| Writes code manually | Writes prompts & workflows |
| Reviews PRs | Reviews agent outputs |
| Runs CI/CD pipelines | Orchestrates AI agents |
| Debugging code | Debugging agent behavior |

> The role evolves: from coding to orchestrating intelligent agents

---

# Agentic Human Processes

## Humans and AI collaborate at every stage

- **Author** — Write natural language workflows
- **Reviewer** — Approve plans and validate outputs
- **Supervisor** — Monitor running agents and handle exceptions
- **Debugger** — Diagnose workflow behavior and improve prompts

> The human stays in the loop while AI handles execution

---

# Pull Request Process

<pre class="mermaid">
flowchart TD
    PR["Pull Request Opened"] --> Activation["Activation Job\nAuthorize & Sanitize"]
    Activation --> Agent["Agent Job\nAnalyze Changes"]
    Agent --> SafeOutput["Safe Outputs Job\nPost Comment / Review"]
    SafeOutput --> Human{"Human Review"}
    Human -->|Approve| Merge["Merge PR"]
    Human -->|"Request Changes"| Agent
</pre>

---

# Research → Plan → Act

<pre class="mermaid">
flowchart LR
    Research["Research\nGather context\nAnalyze codebase"] --> Plan["Plan\nDefine approach\nCreate checklist"]
    Plan --> Act["Act\nImplement changes\nCreate PR"]
    Act --> Human{"Human Review"}
    Human -->|Approved| Done["Done ✓"]
    Human -->|"Revise"| Research
</pre>

---

# Continuous Integration to Continuous AI

- **Accessibility review** - Automated WCAG compliance checks

- **Documentation** - Auto-generate API docs and README files

- **Code review** - AI-powered PR analysis and suggestions

- **Test improvement** - Identify missing test coverage

- **Bundle analysis** - Monitor package size and dependencies

- **Issue triage** - Automated labeling and prioritization

> <https://githubnext.com/projects/continuous-ai/>

<!--
https://github.com/github/gh-aw/issues/1920
-->

---

# Evolution: LLMs to SWE Agents

## From code completion to autonomous workflows

**2021: GitHub Copilot** - AI-powered code completion

**2022: ChatGPT** - Conversational AI assistant

**2023: LLMs & Web UI Generators** - Prompt to Web App

**2024: Agent CLIs** - Claude Code: File edit, bash

**2025: MCP, SKILLS.md** - Unified tooling

---

# CI/CD with GitHub Actions

YAML workflows as configuration stored in `.github/workflows/` that trigger on events like push, pull requests, issues.

```yaml
on:
  issues:
    types: [opened]
permissions:
  issues: write # DANGER zone
jobs:
  agent:
    steps:
      - run: copilot -p "Summarize issue and respond in a comment."
```

---

# The "Lethal Trifecta" for AI Agents

AI agents become risky when they combine **three capabilities** at once:

- **Private data access**

- **Untrusted content**

- **External communication**

> <https://simonw.substack.com/p/the-lethal-trifecta-for-ai-agents>

---

# Useful Sandboxes: The Philosophy

## Safe by design. Useful by default.

---

# From Scratch to MakeCode

## Kid dev environments got here first

- **Scratch** — Block-based coding (MIT) — can't break anything important
- **MakeCode / pxt** — Hardware + game programming for beginners
- **BASIC** — First language for a generation of developers

These environments share one superpower:

> You're protected from catastrophe — and you can still build something **real and delightful**

---

# What Makes a Sandbox "Useful"?

## The beginner runtime principles

- **Guardrails without walls** — protected, not trapped
- **Immediate feedback** — actions produce visible results
- **Progressive disclosure** — start simple, grow into complexity
- **Real outputs** — even beginners ship something that matters
- **Delight** — the sandbox feels like a superpower, not a restriction

> The best sandboxes don't limit what's possible — they shape *how* it's possible

---

# Applying These Principles to Agentic Workflows

| Beginner Runtime | Agentic Workflows |
|---|---|
| Runs in the browser | Container isolation |
| Safe defaults | Read-only permissions |
| Guardrailed actions | Safe outputs |
| Real outputs | Issues, PRs, comments |
| Delightful UX | Natural language workflows |

> Same philosophy, enterprise scale

---

# Combine GitHub Actions and SWE Agents **SAFELY**

---

# Loved by Developers

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read # read-only by default
safe-outputs:
  add-comment: # guardrails for write operations
---
Summarize issue and respond in a comment.
```

> Natural language → compiled to GitHub Actions YAML

---

# Trusted by Enterprises

## Safe by default

- **Containers**: Isolated GitHub Actions Jobs

- **Firewall**: Network Control

- **Minimal Permissions**: Read-only by default

- **MCP Gateway**: Secure tool access

- **Threat Detection**: Agentic detection of threats

- **Safe Outputs**: Deterministic, guardrailed outputs

- **Plan / Check / Act**: Human in the loop

---

# Compiled Action Yaml

```yaml
jobs:
  activation:
    run: check authorization

  agent: needs[activation] # isolated container
    permissions: contents: read # read-only!
    run: copilot -p "Analyze package.json for breaking changes..."

  safe-outputs: needs[agent] # isolated container
    run: gh issue comment add ...
    permissions: issues: write
```

> Markdown workflows compiled to GitHub Actions YAML for auditability

---

# Safe Outputs

```yaml
---
on: 
  pull_request:
    types: [opened]
permissions: 
  contents: read
safe-outputs:
  create-issue:
---
Check for breaking changes in package.json and create an issue.
```

**Security:** AI agents cannot directly write to GitHub. Safe-outputs validate AI responses and execute actions in isolated containers.

---

# Network Permissions

```yaml
---
on:
  pull_request:
network:
  allowed:
    - defaults  # Basic infrastructure
    - node      # NPM ecosystem
tools:
  web-fetch:
---
Fetch latest TypeScript docs report findings in a comment.
```

> Control external access for security

---

# Safe Outputs → Copilot Handoff

```yaml
---
on:
  issues:
    types: [opened]
safe-outputs:
  create-issue:
    assignees: ["copilot"]
---
Analyze issue and break down into implementation tasks
```

> Triage agent → Creates tasks → @copilot implements → Review

---

# AI Engines

## Multiple AI providers supported

- **GitHub Copilot** (default, recommended)
- **Claude Code**
- **Codex**
- **Gemini CLI**

```yaml
engine: copilot  # sensible defaults
```

> GitHub Copilot offers MCP support and conversational workflows

---

# MCP Servers Configuration

```yaml
# GitHub MCP (recommended: use toolsets)
tools:
  github:
    toolsets: [default]  # context, repos, issues, pull_requests

# Custom MCP servers
mcp-servers:
  bundle-analyzer:
    command: "node"
    args: ["path/to/mcp-server.js"]
    allowed: "*"
```

**MCP:** Extend AI with [Model Context Protocol](https://modelcontextprotocol.io/)

---

# Containerized, Firewalled MCPs

```yaml
mcp-servers:
  web-scraper:
    container: mcp/fetch
    network:
      allowed: ["npmjs.com", "*.jsdelivr.com"]
    allowed: ["fetch"]
```

**Defense in depth:** Container + network + permissions

---

# Monitoring & Optimization

Track workflow performance and AI agent behavior.

```sh
# View recent runs
gh aw logs

# Filter by date range
gh aw logs --start-date -1w accessibility-review

# Generate the lock file for a workflow
gh aw compile
```

> Lock files (`.lock.yml`) ensure reproducibility and auditability

---

# Cache & Persistent Memory

## Speed up workflows and maintain context

```yaml
---
on:
  pull_request:
    types: [opened]
tools:
  cache-memory:  # AI remembers across runs
---
Review this PR with context from previous reviews:
- Check for repeated issues
- Track improvement trends
- Reference past discussions
```

**Benefits:** Faster builds + contextual AI analysis

---

# Playwright + Upload Assets

## Browser automation for web app testing

```yaml
---
on:
  pull_request:
    types: [ready_for_review]
tools:
  playwright:      # Headless browser automation
safe-outputs:
  create-issue:
  upload-asset:   # Attach screenshots to artifacts
---
Test the web application:
1. Navigate to the deployed preview URL
2. Take screenshots of key pages
3. Check for visual regressions
4. Validate responsive design (mobile, tablet, desktop)
5. Create issue with findings and screenshots
```

**Use cases:** Visual regression, accessibility audits, E2E validation for SPAs

---

# Sanitized Context & Security

## Protect against prompt injection

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
---
# RECOMMENDED: Use sanitized context
Analyze this issue content (safely sanitized):
"${{ steps.sanitized.outputs.text }}"

Metadata:
- Issue #${{ github.event.issue.number }}
- Repository: ${{ github.repository }}
- Author: ${{ github.actor }}
```

**Auto-sanitization:** @mentions neutralized, bot triggers blocked, malicious URIs filtered

---

# Security Architecture

## Multi-layered defense in depth

GitHub Agentic Workflows implements a comprehensive security architecture with multiple isolation layers to protect against threats.

**Key Security Principles:**

- Container isolation for all components
- Network firewall controls at every layer
- Minimal permissions by default
- Separation of concerns

---

# Security Architecture Diagram

<pre class="mermaid">
flowchart TB
    subgraph ActionJobVM["Action Job VM"]
        subgraph Sandbox1["Sandbox"]
            Agent["Agent Process"]
        end

        Proxy1["Proxy / Firewall"]
        Gateway["Gateway&lt;br/&gt;(mcpg)"]

        Agent --> Proxy1
        Proxy1 --> Gateway

        subgraph Sandbox2["Sandbox"]
            MCP["MCP Server"]
        end

        subgraph Sandbox3["Sandbox"]
            Skill["Skill"]
        end

        Gateway --> MCP
        Gateway --> Skill

        Proxy2["Proxy / Firewall"]
        Proxy3["Proxy / Firewall"]

        MCP --> Proxy2
        Skill --> Proxy3
    end

    Service1{{"Service"}}
    Service2{{"Service"}}

    Proxy2 --> Service1
    Proxy3 --> Service2
</pre>

---

# Security Layer 1: Coding Agent Sandbox

## Isolated agent process

**Coding Agent Sandbox:**

- Agent process runs in isolated container
- Read-only permissions by default
- No direct write access to repository
- Limited system access

**Primary Proxy/Firewall:**

- Filters outbound traffic from agent
- Controls access to MCP Gateway
- Enforces network allowlists

---

# Security Layer 2: MCP Gateway

## Central routing with access controls

**MCP Gateway (mcpg):**

- Central routing component
- Manages communication between agents and services
- Validates tool invocations
- Enforces permission boundaries

**Benefits:**

- Single point of control
- Auditable tool access
- Prevents direct agent-to-service communication

---

# Security Layer 3: Tool Sandboxes

## Isolated MCP servers and skills

**MCP Server & Skill Sandboxes:**

- Each MCP server runs in own container
- Each skill runs in separate sandbox
- Non-root user IDs
- Dropped capabilities

**Secondary Proxy/Firewalls:**

- Additional proxy layers for egress traffic
- Domain-specific allowlists
- Defense against data exfiltration

---

# Security Layer 4: Service Access

## Controlled external communication

**Service Layer:**

- External services accessed through proxies
- Multiple security controls before reaching services
- Comprehensive audit trail
- Network traffic monitoring

**Defense in Depth:**
Even if one layer is compromised, multiple additional security controls remain in place.

---

# Security Features Summary

**Container Isolation:**

- GitHub Actions Jobs in VMs
- Separate sandboxes for agent, MCP servers, skills

**Network Controls:**

- Proxy/firewall at every layer
- Domain allowlisting
- Ecosystem-based controls (node, python, containers)

**Permissions:**

- Read-only by default
- Safe outputs for write operations
- Explicit permission grants

**Supply Chain Integrity:**

- Classic: GitHub Actions pinned to commit SHAs at compile time
- Agentic: Protected files guard CI/CD configuration from tampering

**GitHub Guard:**

- `min-integrity` restricts agent access by content trust level
- Secrecy & integrity metadata on safe output messages
- Automatic `min-integrity: approved` for public repositories

**Monitoring:**

- Threat detection
- Audit logs
- Workflow run analysis

---

# Best Practices: Human in the Loop

**Manual Approval Gates:**
Critical operations require human review

```yaml
---
on:
  issues:
    types: [labeled]
  manual-approval: production
safe-outputs:
  create-pull-request:
---
Analyze issue and create implementation PR
```

**Plan / Check / Act Pattern:**

- AI generates plan (read-only)
- Human reviews and approves
- Automated execution with safe outputs

---

# Learn More About Security

**Documentation:**

- Security Best Practices Guide
- Threat Detection Configuration
- Network Configuration Reference
- Safe Outputs Reference

**Visit:** <https://github.github.com/gh-aw/introduction/architecture/>

Security is foundational to GitHub Agentic Workflows. We continuously evolve our security controls and welcome community feedback.

# Getting Started (Agentically)

```sh
# Install GitHub Agentic Workflows extension
gh extension install github/gh-aw
gh aw init

# Agentic setup with Copilot CLI (optional)
npx --yes @github/copilot -i "activate https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install.md"
```

> Built with AI agents in mind from day 0

> Quick Start: <https://github.github.com/gh-aw/setup/quick-start/>

---
