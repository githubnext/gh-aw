# 🤖 GitHub Agentic Workflows

Write agentic workflows in natural language markdown, and run them in GitHub Actions. From [GitHub Next](https://githubnext.com/) 

> **Note**: This extension is in early development and may change significantly. Use with caution.

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

Your repository will also need a ANTHROPIC_API_KEY or OPENAI_API_KEY Actions secret set up to run workflows that use AI models. You can add this using one of the following commands:

```bash
gh secret set ANTHROPIC_API_KEY -a actions --body <your-anthropic-api-key>
#gh secret set OPENAI_API_KEY -a actions --body <your-openai-api-key>
```

Once you've reviewed and merged the PR you're all set! Each week, the workflow will run automatically and create a research report issue in your repository. If you're in a hurry and would like to run the workflow immediately, you can do so using:

```bash
gh aw run weekly-research
```

You can explore other samples at [githubnext/agentics](https://github.com/githubnext/agentics). You can also copy those samples and write your own workflows. Any repository that has a "workflows" directory can be used as a source of workflows.

## 📝 Agent Workflow Example

Here's what a simple agent workflow looks like. This example automatically triages new issues:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  issues: write

tools:
  github:
    allowed: [add_issue_comment]

timeout_minutes: 5
---

# Issue Triage

Analyze issue #${{ github.event.issue.number }} and help with triage:

1. Read the issue content
2. Post a helpful comment summarizing the issue

Keep responses concise and helpful.
```

> **💡 Learn more**: For a complete list of frontmatter options and workflow configuration details, see [REFERENCE.md](REFERENCE.md).

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

### Updating after workflow edits

You are in control of the workflow files in `.github/workflows/` and can adapt them to your needs. If you modify the markdown file, you can compile it to update the lock file:

```bash
gh aw compile
```

You will see the changes reflected in the `.lock.yml` file, which is the actual workflow that will run on GitHub Actions. You should commit changes to both files to your repository.

### Generated code marking

When workflows are compiled, gh-aw automatically creates or updates a `.gitattributes` file in your repository root to mark `.lock.yml` files as generated code:

```
.github/workflows/*.lock.yml linguist-generated=true
```

This ensures that GitHub properly recognizes these files as generated code, which:
- Collapses them in pull request diffs for better readability
- Excludes them from repository language statistics  
- Marks them as machine-generated in the GitHub UI

The `.gitattributes` file is automatically managed - you don't need to create or maintain it manually.

> **📚 Workflow commands**: See [REFERENCE.md](REFERENCE.md) for complete workflow management commands including `list`, `status`, `enable`, `disable`, and more.

### Configuring the agentic processor

By default Claude Code is used as the agentic processor. You can configure the agentic processor by editing the frontmatter of the markdown workflow files.

```markdown
engine: codex  # Optional: specify AI engine (claude, codex, ai-inference, gemini)
```

You can also specify this on the command line when adding or running workflows:

```bash
gh aw add weekly-research --engine codex
```

This will override the `engine` setting in the frontmatter of the markdown file.

## Security and untrusted inputs

Security is a key consideration when using agentic workflows.

This repository is for demonstration purposes and the workflows are not intended to be run in production. The workflows are designed to be run in a controlled environment, and should not be used in production.

⚠️ You should carefully review the contents of any workflow before installing it, especially if you have not authored it, as it may contain workflows that you do not want to run in your repository.

⚠️⚠️ You should carefully review and assess the compiled workflows before adding them to your repository. You should assess the permissions required by the workflows, and ensure that they are appropriate for your repository. You can do this by reviewing the `.lock.yml` file for your workflow.

⚠️⚠️ GitHub Actions applies many organization settings, limitations and defaults to help protect repositories from unintended actions. For example, by default workflows run in a read-only mode, and do not have access to secrets, when run over pull requests from forks. These rules all apply to agentic workflows as well. You can read more about [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use) and [GitHub Actions permissions](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions) in the documentation.

⚠️⚠️ If you specify the access for any permissions, all of those that are not specified are set to none.

> **🔧 Advanced configuration**: For detailed information about permissions, tools, secrets, and all frontmatter options, see [REFERENCE.md](REFERENCE.md).

## 🔗 Related Projects

- [Continuous AI](https://githubnext.com/projects/continuous-ai/)
- [GitHub Actions](https://github.com/features/actions)
- [GitHub CLI](https://cli.github.com/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
