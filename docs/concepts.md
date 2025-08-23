# 📖 Introduction to GitHub Agentic Workflows

Now that you've [got your first workflow running](quick-start.md), let's dive deeper into the concepts and capabilities of GitHub Agentic Workflows.

GitHub Agentic Workflows represent a new paradigm where AI agents can perform complex, multi-step tasks in conjunction with your team automatically. They combine the power of AI with GitHub's collaboration platform to enable [Continuous AI](https://githubnext.com/projects/continuous-ai) — the systematic, automated application of AI to software collaboration.

GitHub Agentic Workflows are both revolutionary and yet familiar: they build on top of GitHub Actions, and use familiar AI engines such as Claude Code and Codex to interpret natural language instructions.

## Core Concepts

### What makes a workflow "agentic"?

Traditional GitHub Actions follow pre-programmed steps. Agentic workflows use AI to:

- **Understand context** — Read and analyze repository content, issues, PRs, and discussions
- **Make decisions** — Determine what actions to take based on the current situation  
- **Use tools** — Interact with GitHub APIs, external services, and repository files
- **Generate content** — Create meaningful comments, documentation, and code changes
- **Learn and adapt** — Adjust behavior based on past action, feedback and outcomes
- **Use ambiguity productively** — Interpret natural language instructions flexibly and productively

One crucial difference from regular agentic prompting is that GitHub Agentic Workflows can contain **both** traditional GitHub Actions steps and agentic natural language instructions. This allows the best of both worlds: traditional steps for deterministic actions, and agentic steps for flexible, context-aware AI-driven actions.

### The anatomy of an agentic workflow

Every agentic workflow has two main parts:

1. **Frontmatter (YAML)** — Configuration that defines triggers, permissions, and available tools
2. **Instructions (Markdown)** — Natural language description of what the AI should do

```markdown
---
on: ...
permissions: ...
tools: ...
steps: ...
---

# Natural Language Instructions
Analyze this issue and provide helpful triage comments...
```

Another crucial difference from traditional agentic prompting is that GitHub Agentic Workflows can contain triggers, permissions and other declarative elements. This works towards more reliable and more secure agentic programming, setting the AI up to contribute to success, in a partially sandboxed way, at the right time in your team's work.

See [Workflow Structure](workflow-structure.md) and [Frontmatter Options](frontmatter.md) for details of file layout and configuration options.

## Understanding AI Engines

Agentic workflows are powered by different agentic AI engines:

- **Claude Code** (default) — Anthropic's AI engine, excellent for reasoning and code analysis
- **Codex** (experimental) — OpenAI's code-focused engine

The engine interprets your natural language instructions and executes them using the tools and permissions you've configured.

### Continuous AI Patterns

GitHub Agentic Workflows enable [Continuous AI](https://githubnext.com/projects/continuous-ai) — the systematic application of AI to software collaboration:

- **Continuous Documentation** — Keep docs current and comprehensive
- **Continuous Code Improvement** — Incrementally enhance code quality
- **Continuous Triage** — Intelligent issue and PR management
- **Continuous Research** — Stay current with industry developments
- **Continuous Quality** — Automated code review and standards enforcement

**Demonstrator Research & Planning Workflows**
- [📚 Weekly Research](https://github.com/githubnext/agentics?tab=readme-ov-file#-weekly-research) - Collect research updates and industry trends
- [👥 Daily Team Status](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-team-status) - Assess repository activity and create status reports
- [📋 Daily Plan](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-plan) - Update planning issues for team coordination
- [🏷️ Issue Triage](https://github.com/githubnext/agentics?tab=readme-ov-file#️-issue-triage) - Triage issues and pull requests

**Demonstrator Coding & Development Workflows**
- [📦 Daily Dependency Updater](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-dependency-updater) - Update dependencies and create pull requests
- [📖 Regular Documentation Update](https://github.com/githubnext/agentics?tab=readme-ov-file#-regular-documentation-update) - Update documentation automatically
- [🔍 Daily QA](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-qa) - Perform quality assurance tasks
- [🧪 Daily Test Coverage Improver](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-test-coverage-improver) - Improve test coverage by adding meaningful tests to under-tested areas
- [⚡ Daily Performance Improver](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-performance-improver) - Analyze and improve code performance through benchmarking and optimization
- [🔍 Daily Accessibility Review](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-accessibility-review) - Review application accessibility

## Advanced Concepts

### Lock Files and Compilation

When you modify a `.md` workflow file, you need to compile it:

```bash
gh aw compile
```

This generates a `.lock.yml` file containing the actual GitHub Actions workflow. Both files should be committed to your repository.

The `.lock.yml` file contains the full configuration of the workflow, including the frontmatter and the compiled agentic steps, added security hardening and job orchestration. The `.md` file is the source of truth for authoring and editing.

### Security and Permissions

Agentic workflows require careful security consideration:

- **Minimal permissions** — Grant only what the workflow needs
- **Tool allowlists** — Explicitly specify which tools the AI can use  
- **Input validation** — All inputs are automatically sanitized
- **Human oversight** — Critical actions can require human approval

See [Security Notes](security-notes.md) for comprehensive guidelines.

### Tools and MCPs

Workflows can use various tools through the Model Context Protocol (MCP):

- **GitHub tools** — Repository management, issue/PR operations
- **External APIs** — Integration with third-party services
- **File operations** — Read, write, and analyze repository files
- **Custom MCPs** — Build your own tool integrations

Learn more in [Tools Configuration](tools.md) and [MCPs](mcps.md).

## Building Effective Workflows

### Best Practices

1. **Start simple** — Begin with basic workflows and add complexity gradually
2. **Be specific** — Clear, detailed instructions produce better results
3. **Test iteratively** — Use `gh aw compile --watch` and `gh aw run` during development
4. **Monitor costs** — Use `gh aw logs` to track AI usage and optimize
5. **Review outputs** — Always verify AI-generated content before merging
6. **Use safe outputs** — Leverage [Safe Output Processing](safe-outputs.md) to automatically create issues, comments, and PRs from agent output

### Common Patterns

- **Event-driven** — Respond to issues, PRs, pushes, etc.
- **Scheduled** — Regular maintenance and reporting tasks
- **Alias-triggered** — Activated by @mentions in comments
- **Secure** — User minimal permissions and protect against untrusted content, see [Security Notes](security-notes.md)

## Next Steps

Ready to build more sophisticated workflows? Explore:

- **[Workflow Structure](workflow-structure.md)** — Detailed file organization and security
- **[Frontmatter Options](frontmatter.md)** — Complete configuration reference
- **[Tools Configuration](tools.md)** — Available tools and permissions
- **[Security Notes](security-notes.md)** — Important security considerations
- **[VS Code Integration](vscode.md)** — Enhanced authoring experience

The power of agentic workflows lies in their ability to understand context, make intelligent decisions, and take meaningful actions — all while maintaining the reliability you expect from GitHub Actions.
