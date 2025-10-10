---
title: Using simonw/llm CLI
description: Learn how to use simonw/llm CLI tool as a custom engine for GitHub Agentic Workflows.
---

This guide covers how to use the `simonw/llm` CLI tool as a custom agentic engine in GitHub Agentic Workflows.

## Overview

[simonw/llm](https://github.com/simonw/llm) is a powerful CLI tool for interacting with Large Language Models from the command line. It supports multiple LLM providers including OpenAI, Anthropic Claude, Google Gemini, and many others through plugins.

This repository includes a shared component (`shared/simonw-llm.md`) that makes it easy to use the llm CLI in your agentic workflows.

## Quick Start

The simplest way to use the llm CLI is to import the shared component:

```aw
---
name: My LLM Workflow
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
imports:
  - shared/simonw-llm.md
---

# Your workflow instructions

Analyze the issue: ${{ needs.activation.outputs.text }}
```

## Configuration

The `simonw-llm` shared component supports both OpenAI and Anthropic API keys. You need to configure at least one API key as a repository secret.

### OpenAI Setup

1. Get your API key from [OpenAI Platform](https://platform.openai.com/api-keys)
2. Add as repository secret:
   ```bash
   gh secret set OPENAI_API_KEY -a actions --body "<your-api-key>"
   ```
3. The workflow will automatically use `gpt-4o-mini` model

### Anthropic Setup

1. Get your API key from [Anthropic Console](https://console.anthropic.com/)
2. Add as repository secret:
   ```bash
   gh secret set ANTHROPIC_API_KEY -a actions --body "<your-api-key>"
   ```
3. The workflow will automatically install the `llm-claude` plugin and use `claude-3-5-sonnet-20241022` model

## Example: Issue Triage Workflow

This repository includes a complete example workflow (`issue-triage-llm.md`) that demonstrates how to use llm CLI for automated issue triage:

```aw
---
name: Issue Triage (LLM)
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 1
imports:
  - shared/simonw-llm.md
---

# Issue Triage

You are an issue triage assistant. Analyze the issue and provide:

1. Issue type classification
2. Priority assessment
3. Initial analysis and recommendations

**Issue Content**: ${{ needs.activation.outputs.text }}
```

## Features

- **Multiple LLM Providers**: Supports OpenAI and Anthropic (via plugins)
- **Automatic Model Selection**: Chooses the right model based on available API keys
- **Safe Outputs Integration**: Works seamlessly with GitHub's safe-outputs for issue comments, PRs, etc.
- **Simple Installation**: Automatically installs via pip in the workflow
- **Plugin Support**: Automatically installs required plugins (e.g., llm-claude)

## How It Works

1. **Installation**: The workflow installs the llm CLI via pip
2. **API Key Configuration**: Configures the appropriate API key based on available secrets
3. **Plugin Installation**: Installs required plugins (e.g., llm-claude for Anthropic)
4. **MCP Configuration**: The GITHUB_AW_MCP_CONFIG environment variable is available for MCP server integration (future compatibility)
5. **Execution**: Runs your prompt using the llm CLI
6. **Output Processing**: Captures output for safe-outputs processing

## Customization

To customize the model or other settings, you can create your own custom engine configuration based on the shared component:

```aw
---
engine:
  id: custom
  steps:
    - name: Install simonw/llm CLI
      run: |
        pip install llm
        llm --version
    
    - name: Configure API key
      run: |
        echo "$OPENAI_API_KEY" | llm keys set openai
      env:
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    
    - name: Run custom llm command
      run: |
        # Use custom model or options
        llm -m gpt-4o "$(cat $GITHUB_AW_PROMPT)"
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
---
```

## Comparison with Other Engines

| Feature | Claude Engine | Copilot Engine | LLM CLI |
|---------|--------------|----------------|---------|
| Setup Complexity | Low | Low | Low |
| Model Choice | Claude only | GitHub models | Multiple providers |
| Plugin Support | No | No | Yes (extensive) |
| Cost | Pay per use | Included with subscription | Pay per use |
| Offline Mode | No | No | Yes (with local models) |

## Best Practices

1. **Use Safe Outputs**: Always use safe-outputs for GitHub API operations instead of direct API calls
2. **Set Timeouts**: Configure appropriate timeout_minutes for your workflows
3. **Minimal Permissions**: Only grant the permissions your workflow actually needs
4. **API Key Security**: Never hardcode API keys; always use repository secrets
5. **Test Locally**: The llm CLI can be tested locally before deploying to workflows

## Troubleshooting

### API Key Not Found

If you see "No API key configured":
- Verify the secret is named exactly `OPENAI_API_KEY` or `ANTHROPIC_API_KEY`
- Check that the secret is accessible to Actions (not just Dependabot or Codespaces)

### Model Not Available

If the default model is not available:
- For OpenAI: Check your account has access to gpt-4o-mini
- For Anthropic: Verify your API key has access to claude-3-5-sonnet

### Installation Issues

If pip install fails:
- The workflow runs on ubuntu-latest with Python pre-installed
- Network connectivity issues may require retries or caching

## Resources

- [simonw/llm GitHub Repository](https://github.com/simonw/llm)
- [LLM Documentation](https://llm.datasette.io/)
- [Example Workflow: issue-triage-llm.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/issue-triage-llm.md)
- [Shared Component: simonw-llm.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/shared/simonw-llm.md)
