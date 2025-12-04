# GitHub Agentic Workflows Setup Guide

A guided walkthrough to help you set up your agentic workflows using gh-aw.

## Starting the Setup

### 1. Check gh-aw Installation

First, let's verify that gh-aw is installed correctly.

Run:
```bash
gh aw version
```

✓ If the command succeeds and shows a version, you're all set! Continue to the next step.

❌ If the command fails, gh-aw is not installed. Install it using one of these methods:

**Option 1: GitHub CLI extension (recommended)**
```bash
gh extension install githubnext/gh-aw
```

**Option 2: Standalone installer (for Codespaces or if extension install fails)**
```bash
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

After installation, verify with `gh aw version` and then continue.

### 2. Choose Your AI Agent

Select which AI agent you'll use for your agentic workflows:

- **`copilot`** (GitHub Copilot CLI) - **Recommended for most users**
- **`claude`** (Anthropic Claude Code) - Great for reasoning and code analysis
- **`codex`** (OpenAI Codex) - Designed for code-focused tasks

## Configure Secrets for Your Chosen Agent

### For `copilot` (Recommended)

You'll need a GitHub Personal Access Token with Copilot subscription.

**Steps:**
1. Go to [GitHub Token Settings](https://github.com/settings/tokens)
2. Create a Personal Access Token (Classic) with appropriate scopes
3. Ensure you have an active Copilot subscription

**Documentation:** [GitHub Copilot Engine Setup](https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default)

**Set the secret** in your terminal:

```bash
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "your-github-pat-here"
```

### For `claude`

You'll need an Anthropic API key or Claude Code OAuth token.

**Steps:**
1. Sign up for Anthropic API access at [console.anthropic.com](https://console.anthropic.com/)
2. Generate an API key from your account settings

**Documentation:** [Anthropic Claude Code Engine](https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code)

**Set the secret** (choose one):

```bash
# Option 1: Using CLAUDE_CODE_OAUTH_TOKEN
gh secret set CLAUDE_CODE_OAUTH_TOKEN -a actions --body "your-claude-oauth-token-here"

# Option 2: Using ANTHROPIC_API_KEY
gh secret set ANTHROPIC_API_KEY -a actions --body "your-anthropic-api-key-here"
```

### For `codex`

You'll need an OpenAI API key.

**Steps:**
1. Sign up for OpenAI API access at [platform.openai.com](https://platform.openai.com/)
2. Generate an API key from your account settings

**Documentation:** [OpenAI Codex Engine](https://githubnext.github.io/gh-aw/reference/engines/#openai-codex)

**Set the secret**:

```bash
gh secret set OPENAI_API_KEY -a actions --body "your-openai-api-key-here"
```

## Build Your First Workflow

When you're ready, use Copilot to create your workflow:

**Using GitHub Copilot CLI:**
1. Launch Copilot: `npx @github/copilot`
2. Type `/create-agentic-workflow` in the chat

This will help you create your first agentic workflow interactively.

## Next Steps

- Explore the [documentation](https://githubnext.github.io/gh-aw/)
- Check out example workflows in the [agentics catalog](https://github.com/githubnext/agentics)
- Run `gh aw --help` to see all available commands
