# Setup Guide for GitHub Agentic Workflows

This guide is designed for **GitHub Copilot agents** helping users configure their repository to use GitHub Agentic Workflows (gh-aw).

## What is GitHub Agentic Workflows?

GitHub Agentic Workflows lets you write AI-powered automation in natural language markdown files that run as GitHub Actions. Instead of complex YAML, write instructions in plain English that AI agents (like GitHub Copilot) execute.

**Example workflow:**

```markdown
---
on:
  issues:
    types: [opened]
permissions: read-all
safe-outputs:
  add-comment:
---

# Issue Clarifier

Analyze the current issue and ask for additional details if the issue is unclear.
```

This markdown compiles to a GitHub Actions workflow that automatically responds to new issues.

## Quick Setup

**For a GitHub Copilot agent helping a user**, follow this streamlined process:

### Prerequisites

- ✅ GitHub CLI (`gh`) installed and authenticated
- ✅ Repository cloned locally
- ✅ User has write access to the repository

**Note:** Do NOT attempt to run `gh auth login` for the user. If authentication is needed, guide them to follow [GitHub CLI authentication documentation](https://cli.github.com/manual/gh_auth_login).

### 1. Check if gh-aw is installed

```bash
gh aw version
```

**If not installed**, run the installation script:

```bash
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

Verify installation with `gh aw version`.

### 2. Run `gh aw init`

This is the **primary command** that configures the repository:

```bash
gh aw init
```

**What `gh aw init` does:**

When run without flags, `gh aw init` enters **interactive mode** and will:

1. **Prompt for AI engine selection** (Copilot, Claude, or Codex)
2. **Configure repository structure**:
   - `.gitattributes` - marks `.lock.yml` files as generated
   - `.github/aw/github-agentic-workflows.md` - comprehensive documentation
   - `.github/agents/agentic-workflows.agent.md` - AI assistant for workflows
   - `.github/aw/*.md` - specialized prompts for creating, updating, debugging workflows
   - `.vscode/settings.json` - VSCode configuration
   - `.vscode/mcp.json` - MCP server configuration (if Copilot selected)
   - `.github/workflows/copilot-setup-steps.yml` - setup instructions for Copilot agents

3. **Detect and validate secrets** from your environment
4. **Configure repository secrets** automatically if environment variables are detected
5. **Guide you through any missing setup** interactively

**Alternative: Non-Interactive Mode**

For automated setups or when you already know the configuration:

```bash
gh aw init --tokens --engine copilot
```

This will:
- Configure the repository for Copilot engine
- Check which secrets are configured
- Show commands to set up missing secrets
- Skip interactive prompts

**Common flags:**
- `--engine copilot` - Configure for GitHub Copilot (also: `claude`, `codex`)
- `--tokens` - Validate and configure required secrets
- `--no-mcp` - Skip MCP server configuration
- `--push` - Automatically commit and push changes
- `--create-pull-request` - Create a PR with initialization changes

### 3. Configure Missing Secrets (If Needed)

If `gh aw init` reports missing secrets, configure them:

#### For GitHub Copilot Engine

**Required:** `COPILOT_GITHUB_TOKEN` - A fine-grained personal access token with "Copilot Requests" permission

**Create the token:**
1. Visit <https://github.com/settings/personal-access-tokens/new>
2. Select **Resource owner**: Your user account
3. Select **Repository access**: "Public repositories" (enables Copilot Requests permission)
4. Under **Account permissions**: Set "Copilot Requests" to "Read-only"
5. Generate and copy the token

**Add the secret:**

```bash
gh aw secret set COPILOT_GITHUB_TOKEN --owner <owner> --repo <repo>
```

You'll be prompted to enter the token securely.

**Verify:**

```bash
gh aw tokens bootstrap --engine copilot
```

### 4. Add Your First Workflow

After initialization, you have several options:

**Option A: Browse and add from the catalog**

```bash
gh aw add githubnext/agentics
```

This shows available workflows. Add one:

```bash
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

**Option B: Use the AI agent to create workflows**

Use the unified workflow agent:

```bash
activate .github/agents/agentic-workflows.agent.md
```

Then describe what you want:
- "create a workflow that triages issues"
- "add a PR reviewer workflow"
- "design a weekly research automation"

**Option C: Create manually**

```bash
gh aw new my-workflow
```

This creates `.github/workflows/my-workflow.md` which you can edit and compile.

### 5. Test the Workflow

Run the workflow:

```bash
gh aw run <workflow-name>
```

Check status:

```bash
gh aw status
```

View logs:

```bash
gh aw logs <workflow-name>
```

## After Setup

**For users:** After `gh aw init` completes:
- ✅ Your repository is configured for agentic workflows
- ✅ Documentation is available in `.github/aw/github-agentic-workflows.md`
- ✅ AI assistants are ready in `.github/agents/`
- ✅ MCP server (if configured) is ready for Copilot Chat integration

**Next steps:**
- Add workflows from the [agentics catalog](https://github.com/githubnext/agentics)
- Create custom workflows with `/agent` → `agentic-workflows` in Copilot Chat
- Edit workflows in `.github/workflows/*.md` and recompile with `gh aw compile`

## Troubleshooting

### Installation Issues

**Problem:** `gh aw version` shows "unknown command"

**Solution:** 
- Ensure GitHub CLI is installed: `gh --version`
- Re-run installation script
- Check `~/.local/bin` is in your PATH

### Secret Configuration Issues

**Problem:** Can't find "Copilot Requests" permission

**Solution:**
- Verify active [Copilot subscription](https://github.com/settings/copilot)
- Use **user account** as Resource owner (not organization)
- Create **fine-grained token** (not classic)
- Select "Public repositories" for Repository access

### Workflow Failures

**Problem:** Workflow runs but fails with authentication error

**Solution:**
- Verify `COPILOT_GITHUB_TOKEN` is set: `gh aw tokens bootstrap --engine copilot`
- Check token has Copilot Requests permission
- Ensure token hasn't expired

**Problem:** Workflow fails with missing tool error

**Solution:**
- Check workflow frontmatter has correct `tools` configuration
- Use the debug agent: `activate .github/agents/agentic-workflows.agent.md` → "debug my workflow"
- View detailed logs: `gh aw logs <workflow-name> -v`

## Agent Guidelines

When helping users set up gh-aw:

1. **Start with `gh aw init`** - This is the primary setup command
2. **Be conversational** - Explain what's happening and why
3. **Handle errors gracefully** - Check output and guide the user through issues
4. **Use interactive mode by default** - It provides the best experience
5. **Adapt to context** - If parts are already set up, skip ahead
6. **Verify each step** - Check that commands succeed before moving on
7. **Never commit secrets** - Always use `gh aw secret set` or GitHub UI

## Additional Resources

- 📖 [Official Documentation](https://githubnext.github.io/gh-aw/)
- 🔍 [Agentics Catalog](https://github.com/githubnext/agentics) - Ready-to-use workflows
- 💬 [GitHub Next Discord](https://gh.io/next-discord) - #continuous-ai channel
- 🎯 [Quick Start Guide](https://githubnext.github.io/gh-aw/setup/quick-start/)

## Security Best Practices

- ⚠️ Always use `permissions: read-all` by default
- ⚠️ Use `safe-outputs` instead of write permissions when possible
- ⚠️ Review AI-generated outputs before they're published
- ⚠️ Never commit secrets to your repository
- ⚠️ Use fine-grained tokens with minimal required permissions
