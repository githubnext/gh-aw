---
title: AI Engines (aka Coding Agents)
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Copilot, Claude, Codex, and Gemini with their specific configuration options.
sidebar:
  order: 600
---

GitHub Agentic Workflows use [AI Engines](/gh-aw/reference/glossary/#engine) (normally a coding agent) to interpret and execute natural language instructions. Each coding agent has unique capabilities and configuration options.

## Available Coding Agents

- [**Copilot CLI**](#using-copilot-cli)
- [**Claude by Anthropic (Claude Code)**](#using-claude-by-anthropic-claude-code)
- [**OpenAI Codex**](#using-openai-codex)
- [**Google Gemini CLI**](#using-google-gemini-cli)

## Using Copilot CLI

[GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) is the default AI engine (coding agent).

To use Copilot CLI with GitHub Agentic Workflows:

1. Copilot CLI is the default AI engine (coding agent). You can optionally request the use of the Copilot CLI in your workflow frontmatter:

   ```yaml wrap
   engine: copilot
   ```

2. Configure the `COPILOT_GITHUB_TOKEN` secret. See [Authentication: COPILOT_GITHUB_TOKEN](/gh-aw/reference/auth/#copilot_github_token) for setup instructions.

## Using Claude by Anthropic (Claude Code)

To use [Claude by Anthropic](https://www.anthropic.com/index/claude) (aka Claude Code):

1. Request the use of the Claude by Anthropic engine in your workflow frontmatter:

   ```yaml wrap
   engine: claude
   ```

2. Configure the `ANTHROPIC_API_KEY` secret. See [Authentication: ANTHROPIC_API_KEY](/gh-aw/reference/auth/#anthropic_api_key) for setup instructions.

## Using OpenAI Codex

To use [OpenAI Codex](https://openai.com/blog/openai-codex):

1. Request the use of the Codex engine in your workflow frontmatter:

   ```yaml wrap
   engine: codex
   ```

2. Configure the `OPENAI_API_KEY` secret. See [Authentication: OPENAI_API_KEY](/gh-aw/reference/auth/#openai_api_key) for setup instructions.

## Using Google Gemini CLI

To use [Google Gemini CLI](https://github.com/google-gemini/gemini-cli):

1. Request the use of the Gemini engine in your workflow frontmatter:

   ```yaml wrap
   engine: gemini
   ```

2. Configure the `GEMINI_API_KEY` secret. See [Authentication: GEMINI_API_KEY](/gh-aw/reference/auth/#gemini_api_key) for setup instructions.

## Extended Coding Agent Configuration

Workflows can specify extended configuration for the coding agent:

```yaml wrap
engine:
  id: copilot
  version: latest                       # defaults to latest
  model: gpt-5                          # defaults to claude-sonnet-4
  command: /usr/local/bin/copilot       # custom executable path
  args: ["--add-dir", "/workspace"]     # custom CLI arguments
  agent: agent-id                       # custom agent file identifier
```

### Copilot Custom Configuration

For the Copilot engine, you can specify a specialized prompt to be used whenever the coding agent is invoked. This is called a "custom agent" in Copilot vocabulary. You specify this using the `agent` field. This references a file located in the `.github/agents/` directory:

```yaml wrap
engine:
  id: copilot
  agent: technical-doc-writer
```

The `agent` field value should match the agent file name without the `.agent.md` extension. For example, `agent: technical-doc-writer` references `.github/agents/technical-doc-writer.agent.md`.

See [Copilot Agent Files](/gh-aw/reference/copilot-custom-agents/) for details on creating and configuring custom agents.

### Engine Environment Variables

All engines support custom environment variables through the `env` field:

```yaml wrap
engine:
  id: copilot
  env:
    DEBUG_MODE: "true"
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
```

Environment variables can also be defined at workflow, job, step, and other scopes. See [Environment Variables](/gh-aw/reference/environment-variables/) for complete documentation on precedence and all 13 env scopes.

### Engine Command-Line Arguments

All engines support custom command-line arguments through the `args` field, injected before the prompt:

```yaml wrap
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
```

Arguments are added in order and placed before the `--prompt` flag. Common uses include adding directories (`--add-dir`), enabling verbose logging (`--verbose`, `--debug`), and passing engine-specific flags. Consult the specific engine's CLI documentation for available flags.

### Custom Engine Command

Override the default engine executable using the `command` field. Useful for testing pre-release versions, custom builds, or non-standard installations. Installation steps are automatically skipped.

```yaml wrap
engine:
  id: copilot
  command: /usr/local/bin/copilot-dev  # absolute path
  args: ["--verbose"]
```

The command supports absolute paths (`/usr/local/bin/copilot`), relative paths (`./bin/claude`), environment variables (`$HOME/.local/bin/codex`), or commands in PATH.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/introduction/architecture/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
