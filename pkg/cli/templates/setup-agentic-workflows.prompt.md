---
title: "Configure Your Agentic Workflow"
description: "A guided prompt to help you set up your agentic workflows using gh-aw."
tools: ['runInTerminal', 'getTerminalOutput', 'createFile', 'createDirectory', 'editFiles', 'search', 'changes', 'githubRepo']
model: GPT-5
---

# Agentic Workflow Setup

Welcome! Let's get your agentic workflow configured. First, choose which agent you'd like to use:

- `copilot` (GitHub Copilot CLI) - **Recommended for most users**
- `claude` (Anthropic Claude Code) - Great for reasoning and code analysis
- `codex` (OpenAI Codex) - Designed for code-focused tasks

Once you choose, I'll guide you through setting up any required secrets.

## Configure Secrets for Your Chosen Agent

### For `copilot` (Recommended)

You'll need a GitHub Personal Access Token with Copilot subscription. 

**Steps:**
1. Go to [GitHub Token Settings](https://github.com/settings/tokens)
2. Create a Personal Access Token (Classic) with appropriate scopes
3. Ensure you have an active Copilot subscription

**Documentation:** [GitHub Copilot Engine Setup](https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default)

**Set the secret** in a separate terminal window (never share your secret directly with the agent):

```bash
gh secret set COPILOT_CLI_TOKEN -a actions --body "your-github-pat-here"
```

### For `claude`

You'll need an Anthropic API key.

**Steps:**
1. Sign up for Anthropic API access at [console.anthropic.com](https://console.anthropic.com/)
2. Generate an API key from your account settings

**Documentation:** [Anthropic Claude Code Engine](https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code)

**Set the secret** in a separate terminal window:

```bash
gh secret set ANTHROPIC_API_KEY -a actions --body "your-anthropic-api-key-here"
```

### For `codex`

You'll need an OpenAI API key.

**Steps:**
1. Sign up for OpenAI API access at [platform.openai.com](https://platform.openai.com/)
2. Generate an API key from your account settings

**Documentation:** [OpenAI Codex Engine](https://githubnext.github.io/gh-aw/reference/engines/#openai-codex)

**Set the secret** in a separate terminal window:

```bash
gh secret set OPENAI_API_KEY -a actions --body "your-openai-api-key-here"
```

## Build Your First Workflow

When you're ready, just type the command:

```
/create-agentic-workflow
```

This will start the configuration flow to help you create your first agentic workflow.

## Additional Resources

- **Quick Start Guide:** [Getting Started](https://githubnext.github.io/gh-aw/start-here/quick-start/)
- **Engine Reference:** [AI Engines Documentation](https://githubnext.github.io/gh-aw/reference/engines/)
- **Full Documentation:** [https://githubnext.github.io/gh-aw/](https://githubnext.github.io/gh-aw/)

---

**Important Security Note:**

🔒 **Never share your secrets directly with the agent.** Always set secrets using the `gh secret set` command in a separate terminal window. The agent should never have direct access to your API keys or tokens.
