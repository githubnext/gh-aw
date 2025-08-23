# ✨ GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them in GitHub Actions. From [GitHub Next](https://githubnext.com/).

> [!WARNING]
> This extension is a research demonstrator. It is in early development and may change significantly. Using agentic workflows in your repository requires careful attention to security considerations and careful human supervision, and even then things can still go wrong. Use it with caution, and at your own risk.

## ⚡ Quick Start (30 seconds)

Install the extension:

```bash
gh extension install githubnext/gh-aw
```

Now, add a weekly research report to your repo (this adds [this sample](https://github.com/githubnext/agentics/blob/main/workflows/weekly-research.md)):

```bash
gh aw add weekly-research -r githubnext/agentics --pr
```
This command will create a PR to your repo adding several files including `.github/workflows/weekly-research.md` and `.github/workflows/weekly-research.lock.yml`.

Your repository will also need an `ANTHROPIC_API_KEY` (for Anthropic Claude) or `OPENAI_API_KEY` (for OpenAI Codex) Actions secret set up to run workflows that use AI models. You can add this using one of the following commands:

```bash
# For Claude engine (default)
gh secret set ANTHROPIC_API_KEY -a actions --body <your-anthropic-api-key>

# For Codex engine (experimental, requires "--engine codex")
gh secret set OPENAI_API_KEY -a actions --body <your-openai-api-key>
```

Once you've reviewed and merged the PR you're all set! Each week, the workflow will run automatically and create a research report issue in your repository. If you're in a hurry and would like to run the workflow immediately, you can do so using:

```bash
gh aw run weekly-research
```

You can explore other samples at [githubnext/agentics](https://github.com/githubnext/agentics). You can also copy those samples and write your own workflows. Any repository that has a "workflows" directory can be used as a source of workflows.

## 📝 Agentic Workflow Example

Here's what a simple agentic workflow looks like. This example automatically triages new issues:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  contents: read      # Minimal permissions for main job
  issues: write      # Needed to add issue comments

tools:
  github:
    allowed: [add_issue_comment]

timeout_minutes: 5
---

# Issue Triage

Analyze issue #${{ github.event.issue.number }} and help with triage:

1. Read the issue content
2. Study the repository
3. Post a helpful comment summarizing the issue

Keep responses concise and helpful.
```

## 📂 Available Demonstrator Workflows from "[The Agentics](https://github.com/githubnext/agentics?tab=readme-ov-file#-the-agentics)"

### Research & Planning Workflows
- [📚 Weekly Research](https://github.com/githubnext/agentics?tab=readme-ov-file#-weekly-research) - Collect research updates and industry trends
- [👥 Daily Team Status](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-team-status) - Assess repository activity and create status reports
- [📋 Daily Plan](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-plan) - Update planning issues for team coordination
- [🏷️ Issue Triage](https://github.com/githubnext/agentics?tab=readme-ov-file#️-issue-triage) - Triage issues and pull requests

### Coding & Development Workflows
- [📦 Daily Dependency Updater](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-dependency-updater) - Update dependencies and create pull requests
- [📖 Regular Documentation Update](https://github.com/githubnext/agentics?tab=readme-ov-file#-regular-documentation-update) - Update documentation automatically
- [🔍 Daily QA](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-qa) - Perform quality assurance tasks
- [🔍 Daily Accessibility Review](https://github.com/githubnext/agentics?tab=readme-ov-file#-daily-accessibility-review) - Review application accessibility


## 📖 Deep Dive

### What's this extension for?

The extension is to support [Continuous AI](https://githubnext.com/projects/continuous-ai) workflows. Continuous AI is a label we've identified for all uses of automated AI to support software collaboration on any platform.

We've chosen the term "Continuous AI” to align with the established concept of Continuous Integration/Continuous Deployment (CI/CD). Just as CI/CD transformed software development by automating integration and deployment, Continuous AI covers the ways in which AI can be used to automate and enhance collaboration workflows.

“Continuous AI” is not a term GitHub owns, nor a technology GitHub builds: it's a term we use to focus our minds, and which we're introducing to the industry. This means Continuous AI is an open-ended set of activities, workloads, examples, recipes, technologies and capabilities; a category, rather than any single tool.

Some examples of Continuous AI are:

* **Continuous Documentation**: Continually populate and update documentation, offering suggestions for improvements.

* **Continuous Code Improvement**: Incrementally improve code comments, tests and other aspects of code e.g. ensuring code comments are up-to-date and relevant.

* **Continuous Triage**: Label, summarize, and respond to issues using natural language.

* **Continuous Summarization**: Provide up-to-date summarization of content and recent events in the software projects.

* **Continuous Fault Analysis**: Watch for failed CI runs and offer explanations of them with contextual insights.

* **Continuous Quality**: Using LLMs to automatically analyze code quality, suggest improvements, and ensure adherence to coding standards.

* **Continuous Team Motivation**: Turn PRs and other team activity into poetry, zines, podcasts; provide nudges, or celebrate team achievements.

* **Continuous Accessibility**: Automatically check and improve the accessibility of code and documentation.

* **Continuous Research**: Automatically research and summarize relevant topics, technologies, and trends to keep the team informed.

So far you've just explored the **Continuous Research** example, but you can write your own workflows to explore all the others! Further samples are available at [githubnext/agentics](https://github.com/githubnext/agentics).

### What are lock files?

Adding an agentic workflow adds two main files, for example:

- `.github/workflows/weekly-research.md`
- `.github/workflows/weekly-research.lock.yml`

Both files are stored in `.github/workflows/` - the first file is the markdown file that defines the workflow, and the second is a lock file that contains the resolved workflow configuration to an actual GitHub Actions workflow.

### You're in control

You are in control of the workflow files in `.github/workflows/` and can adapt them to your needs. If you modify the markdown file, you can compile it to update the lock file:

```bash
gh aw compile
```

You will see the changes reflected in the `.lock.yml` file, which is the actual workflow that will run on GitHub Actions. You should commit changes to both files to your repository.

### Security of Agentic Workflows

Security is a key consideration when using agentic workflows. Please see the [Security Notes](docs/security-notes.md) for guidelines related to workflow security and handling untrusted inputs.

> **💡 Learn more**: For complete workflow configuration details, see the [Documentation](docs/index.md)

> **📚 Workflow commands**: See [Commands Documentation](docs/commands.md) for complete workflow management commands including `list`, `status`, `enable`, `disable`, and more.

> **🤖 Teach AI** how write agentic workflows with [custom instructions](docs/vscode.md#copilot-instructions).

## 💬 Share Feedback

We welcome your feedback on GitHub Agentic Workflows! Please file bugs and feature requests as issues in this repository,
and share your thoughts in the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord).

## 🔗 Related Projects

- [Continuous AI](https://githubnext.com/projects/continuous-ai/)
- [GitHub Actions](https://github.com/features/actions)
- [GitHub CLI](https://cli.github.com/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
