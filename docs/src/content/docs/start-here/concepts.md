---
title: Concepts
description: Learn the core concepts and capabilities of GitHub Agentic Workflows, combining AI agents with GitHub's collaboration platform for Continuous AI.
sidebar:
  order: 300
---

GitHub Agentic Workflows represent a new paradigm where AI agents can perform complex, multi-step tasks in conjunction with your team automatically. They combine the power of AI with GitHub's collaboration platform to enable [Continuous AI](https://githubnext.com/projects/continuous-ai) — the systematic, automated application of AI to software collaboration.

GitHub Agentic Workflows are both revolutionary and yet familiar: they build on top of GitHub Actions, and use familiar AI engines such as Claude Code, GitHub Copilot, and Codex to interpret natural language instructions.

## What makes a workflow "agentic"?

Traditional GitHub Actions follow pre-programmed steps. Agentic workflows use AI to:

- **Understand context** — Read and analyze repository content, issues, PRs, and discussions
- **Make decisions** — Determine what actions to take based on the current situation  
- **Use tools** — Interact with GitHub APIs, external services, and repository files
- **Generate content** — Create meaningful comments, documentation, and code changes
- **Learn and adapt** — Adjust behavior based on past action, feedback and outcomes
- **Use ambiguity productively** — Interpret natural language instructions flexibly and productively

One crucial difference from regular agentic prompting is that GitHub Agentic Workflows can contain **both** traditional GitHub Actions steps and agentic natural language instructions. This allows the best of both worlds: traditional steps for deterministic actions, and agentic steps for flexible, context-aware AI-driven actions.

## What makes a workflow "safe"?

Agentic workflows are designed with security and reliability in mind:
- **Minimal permissions** — Workflows run with the least privilege necessary, reducing risk, including no write permissions by default, and usually no write permissions to GitHub at all during the "agentic" steps
* **Tool allowlists** — Explicitly specify which tools the AI can use, preventing unexpected actions
- **Safe outputs** — Outputs are processed through a safety layer to ensure they meet criteria before being applied, for example declaring and checking that only one issue is created, ensuring minimal safe additions to your GitHub repository
- **Human oversight** — Issues, pull requests, and other critical actions can require human approval before proceeding

## The anatomy of an agentic workflow

Every agentic workflow has two main parts:

1. **Frontmatter (YAML)** — Configuration that defines triggers, permissions, and available tools
2. **Instructions (Markdown)** — Natural language description of what the AI should do

```aw warp
---
on: ...
permissions: ...
tools: ...
steps: ...
---

# Natural Language Instructions
Analyze this issue and provide helpful triage comments...
```

One crucial difference from traditional agentic prompting is that GitHub Agentic Workflows can contain triggers, permissions and other declarative elements. This works towards more reliable and more secure agentic programming, setting the AI up to contribute to success, in a partially sandboxed way, at the right time in your team's work.

See [Workflow Structure](/gh-aw/reference/workflow-structure/) and [Frontmatter Options](/gh-aw/reference/frontmatter/) for details of file layout and configuration options.

## Understanding AI Engines

Agentic workflows are powered by different agentic AI engines:

- **GitHub Copilot CLI** (default) — GitHub's conversational AI engine with repository integration
- **Claude Code** — Anthropic's AI engine, excellent for reasoning and code analysis
- **Codex** (experimental) — OpenAI's code-focused engine

The engine interprets your natural language instructions and executes them using the tools and permissions you've configured.

## Continuous AI Patterns

GitHub Agentic Workflows enable [Continuous AI](https://githubnext.com/projects/continuous-ai) — the systematic application of AI to software collaboration:

- **Continuous Documentation** — Keep docs current and comprehensive
- **Continuous Code Improvement** — Incrementally enhance code quality
- **Continuous Triage** — Intelligent issue and PR management
- **Continuous Research** — Stay current with industry developments
- **Continuous Quality** — Automated code review and standards enforcement

## Lock Files and Compilation

When you modify a `.md` workflow file, you need to compile it:

```bash
gh aw compile
```

This generates a `.lock.yml` file containing the actual GitHub Actions workflow. Both files should be committed to your repository.

The `.lock.yml` file contains the full configuration of the workflow, including the frontmatter and the compiled agentic steps, added security hardening and job orchestration. The `.md` file is the source of truth for authoring and editing.

## Security and Permissions

Agentic workflows require careful security consideration:

- **Minimal permissions** — Grant only what the workflow needs
- **Tool allowlists** — Explicitly specify which tools the AI can use  
- **Human oversight** — Critical actions can require human approval

See [Security Guide](/gh-aw/guides/security/) for comprehensive guidelines.

## Tools and MCPs

Workflows can use various tools through the Model Context Protocol (MCP):

- **GitHub tools** — Repository management, issue/PR operations
- **External APIs** — Integration with third-party services
- **File operations** — Read, write, and analyze repository files
- **Custom MCPs** — Build your own tool integrations

Learn more in [Tools Configuration](/gh-aw/reference/tools/) and [MCPs](/gh-aw/guides/mcps/).

## Best Practices

1. **Start simple** — Begin with basic workflows and add complexity gradually
2. **Be specific** — Clear, detailed instructions produce better results
3. **Test iteratively** — Use `gh aw compile --watch` and `gh aw run` during development
4. **Monitor costs** — Use `gh aw logs` to track AI usage and optimize
5. **Review outputs** — Always verify AI-generated content before merging
6. **Use safe outputs** — Leverage [Safe Output Processing](/gh-aw/reference/safe-outputs/) to automatically create issues, comments, and PRs from agentic workflow output

## Common Patterns

- **Event-driven** — Respond to issues, PRs, pushes, etc.
- **Scheduled** — Regular maintenance and reporting tasks
- **Alias-triggered** — Activated by @mentions in comments
- **Secure** — User minimal permissions and protect against untrusted content, see [Security Guide](/gh-aw/guides/security/)

## Next Steps

Ready to build more sophisticated workflows? Explore:

- **[Packaging and Imports](/gh-aw/guides/packaging-imports/)** — Complete guide to adding, updating, and importing workflows
- **[Workflow Structure](/gh-aw/reference/workflow-structure/)** — Detailed file organization and security
- **[Frontmatter Options](/gh-aw/reference/frontmatter/)** — Complete configuration reference
- **[Tools Configuration](/gh-aw/reference/tools/)** — Available tools and permissions
- **[Security Guide](/gh-aw/guides/security/)** — Important security considerations
- **[VS Code Integration](/gh-aw/tools/vscode/)** — Enhanced authoring experience

The power of agentic workflows lies in their ability to understand context, make intelligent decisions, and take meaningful actions — all while maintaining the reliability you expect from GitHub Actions.
