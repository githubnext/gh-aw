---
title: Concepts
description: Learn the core concepts and capabilities of GitHub Agentic Workflows, combining AI agents with GitHub's collaboration platform for Continuous AI.
sidebar:
  order: 300
---

GitHub Agentic Workflows enable AI agents to perform complex, multi-step tasks automatically with your team. Built on GitHub Actions, they use Claude Code, GitHub Copilot, or Codex to interpret natural language instructions and enable [Continuous AI](https://githubnext.com/projects/continuous-ai) — systematic, automated application of AI to software collaboration.

## Agentic vs Traditional Workflows

Traditional GitHub Actions execute pre-programmed steps. Agentic workflows use AI to understand context, make decisions, use tools, and generate content by interpreting natural language instructions flexibly. Unlike traditional workflows, agentic workflows combine both deterministic GitHub Actions steps and flexible AI-driven instructions for context-aware automation.

## Security Design

Agentic workflows run with minimal permissions (no write access by default), use tool allowlists to prevent unexpected actions, and process outputs through a safety layer before applying changes. Critical actions can require human approval. See [Security Guide](/gh-aw/guides/security/) for details.

## Workflow Structure

Each workflow contains YAML frontmatter (triggers, permissions, tools) and markdown instructions (natural language tasks). This declarative structure enables reliable, secure agentic programming by sandboxing AI capabilities and triggering at the right moments.

```aw warp
---
on: ...
permissions: ...
tools: ...
---
# Natural Language Instructions
Analyze this issue and provide helpful triage comments...
```

See [Workflow Structure](/gh-aw/reference/workflow-structure/) and [Frontmatter](/gh-aw/reference/frontmatter/) for configuration details.

## AI Engines

Workflows support GitHub Copilot CLI (default), Claude Code, and Codex (experimental). Each engine interprets natural language instructions and executes them using configured tools and permissions. See [AI Engines](/gh-aw/reference/engines/) for details.

## Continuous AI Patterns

Enable [Continuous AI](https://githubnext.com/projects/continuous-ai) patterns: keep documentation current, incrementally improve code quality, intelligently triage issues and PRs, stay current with research, and automate code review and standards enforcement.

## Compilation

Compile workflow `.md` files with `gh aw compile` to generate `.lock.yml` files containing the actual GitHub Actions workflow with security hardening and job orchestration. Commit both files — `.md` is the source of truth for editing, `.lock.yml` is the compiled output.


## Tools and MCPs

Workflows use tools through the Model Context Protocol (MCP) for GitHub operations, external APIs, file operations, and custom integrations. Learn more in [Tools](/gh-aw/reference/tools/) and [MCPs](/gh-aw/guides/mcps/).

## Best Practices

Start simple and iterate. Write clear, specific instructions. Test with `gh aw compile --watch` and `gh aw run`. Monitor costs with `gh aw logs`. Review AI-generated content before merging. Use [Safe Output Processing](/gh-aw/reference/safe-outputs/) for controlled creation of issues, comments, and PRs.


## Next Steps

Explore [Workflow Structure](/gh-aw/reference/workflow-structure/), [Frontmatter](/gh-aw/reference/frontmatter/), [Tools](/gh-aw/reference/tools/), [Security Guide](/gh-aw/guides/security/), and [VS Code Integration](/gh-aw/tools/vscode/) to build sophisticated workflows that understand context, make intelligent decisions, and take meaningful actions while maintaining GitHub Actions reliability.
