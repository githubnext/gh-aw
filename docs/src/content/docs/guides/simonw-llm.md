---
title: Using simonw/llm CLI
description: Learn how to use simonw/llm CLI tool as a custom engine for GitHub Agentic Workflows.
---

This guide covers how to use the `simonw/llm` CLI tool as a custom agentic engine in GitHub Agentic Workflows.

## Overview

[simonw/llm](https://github.com/simonw/llm) is a powerful CLI tool for interacting with Large Language Models from the command line. It supports multiple LLM providers including OpenAI, Anthropic Claude, Google Gemini, and many others through plugins.

This repository includes a shared component (`shared/simonw-llm.md`) that makes it easy to use the llm CLI in your agentic workflows.

## Quick Start

The simplest way to use the llm CLI with GitHub Models is to import the shared component:

```aw
---
name: My LLM Workflow
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
  models: read
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

The `simonw-llm` shared component uses GitHub Models with the built-in `GITHUB_TOKEN`. No additional API key setup is required.

### GitHub Models Setup (Free Tier)

1. No additional setup required - uses the built-in `GITHUB_TOKEN`
2. The workflow will automatically use `github/gpt-4o-mini` model
3. Requires `models: read` permission in the workflow
4. Free tier available for all GitHub users
5. Over 30+ models available including GPT-4o, DeepSeek, Llama, and more

### MCP Tools Support

The workflow includes the `llm-tools-mcp` plugin which enables integration with MCP servers:
- Tools from MCP servers are automatically available
- MCP configuration can be customized via workflow configuration
- Supports stdio, SSE, and HTTP transports for MCP servers

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
  models: read
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

- **GitHub Models Integration**: Uses free GitHub Models with 30+ AI models
- **MCP Tools Support**: Integrated MCP server support via llm-tools-mcp plugin
- **Automatic Configuration**: Uses built-in GITHUB_TOKEN, no API key setup needed
- **Safe Outputs Integration**: Works seamlessly with GitHub's safe-outputs for issue comments, PRs, etc.
- **Simple Installation**: Automatically installs via pip in the workflow
- **Free Tier**: GitHub Models provides free access to GPT-4o, DeepSeek, Llama, and more

## How It Works

1. **Installation**: The workflow installs the llm CLI, llm-github-models, and llm-tools-mcp plugins via pip
2. **Configuration**: Uses the built-in GITHUB_TOKEN (no API key setup needed)
3. **MCP Integration**: The llm-tools-mcp plugin provides MCP server support for tool access
4. **Execution**: Runs your prompt using the llm CLI with GitHub Models
5. **Output Processing**: Captures output for safe-outputs processing

## Customization

To use different models or customize settings, you can create your own custom engine configuration:

```aw
---
engine:
  id: custom
  steps:
    - name: Install simonw/llm CLI
      run: |
        pip install llm llm-github-models llm-tools-mcp
        llm --version
    
    - name: Run with custom model
      run: |
        # Use a different GitHub model
        llm -m "github/gpt-4o" "$(cat $GITHUB_AW_PROMPT)"
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---
```

## Comparison with Other Engines

| Feature | Claude Engine | Copilot Engine | LLM CLI (GitHub Models) |
|---------|--------------|----------------|-------------------------|
| Setup Complexity | Low | Low | Very Low (no API keys) |
| Model Choice | Claude only | GitHub models | GitHub Models (30+) |
| MCP Support | Native | No | Via llm-tools-mcp plugin |
| Cost | Pay per use | Included with subscription | Free |
| Offline Mode | No | No | Yes (with local models) |

## Best Practices

1. **Use Safe Outputs**: Always use safe-outputs for GitHub API operations instead of direct API calls
2. **Set Timeouts**: Configure appropriate timeout_minutes for your workflows
3. **Minimal Permissions**: Only grant the permissions your workflow actually needs (`models: read` is required)
4. **Test Locally**: The llm CLI can be tested locally before deploying to workflows
5. **MCP Tools**: Leverage MCP servers for additional tool capabilities

## Troubleshooting

### GITHUB_TOKEN Not Available

If you see "No API key configured":
- Ensure the workflow has `models: read` permission
- Verify the workflow is running in a GitHub Actions environment

### Model Not Available

If the default model is not available:
- Check GitHub Models status and available models
- Try using a different model: `github/gpt-4o`, `github/DeepSeek-V3`, etc.

### Installation Issues

If pip install fails:
- The workflow runs on ubuntu-latest with Python pre-installed
- Network connectivity issues may require retries or caching

## Resources

- [simonw/llm GitHub Repository](https://github.com/simonw/llm)
- [LLM Documentation](https://llm.datasette.io/)
- [Example Workflow: issue-triage-llm.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/issue-triage-llm.md)
- [Shared Component: simonw-llm.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/shared/simonw-llm.md)
