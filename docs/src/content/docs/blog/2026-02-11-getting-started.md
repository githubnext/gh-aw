---
title: "Getting Started with Agentic Workflows"
description: "Begin your journey with agentic automation"
authors:
  - dsyme
  - pelikhan
  - mnkiefer
date: 2026-02-11
draft: true
prev:
  link: /gh-aw/blog/2026-02-08-authoring-workflows/
  label: Authoring Workflows
---

[Previous Article](/gh-aw/blog/2026-02-08-authoring-workflows/)

---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

We've reached the *grand conclusion* of our Peli's Agent Factory series! You've toured the [workflows](/gh-aw/blog/2026-01-13-meet-the-workflows/), discovered [lessons](/gh-aw/blog/2026-01-21-twelve-lessons/), learned the [patterns](/gh-aw/blog/2026-01-24-design-patterns/), mastered [operations](/gh-aw/blog/2026-01-27-operational-patterns/), explored [imports](/gh-aw/blog/2026-01-30-imports-and-sharing/), secured the [vault](/gh-aw/blog/2026-02-02-security-lessons/), glimpsed the [magnificent machinery](/gh-aw/blog/2026-02-05-how-workflows-work/), and practiced [authoring](/gh-aw/blog/2026-02-08-authoring-workflows/). Now for the *golden ticket* - your practical getting started guide!

Ready to build your own agent ecosystem? Let's get you up and running!

This guide will take you from zero to your first running workflow in just a few minutes, then show you how to grow from there. We'll start simple, build confidence, and then explore what's possible. By the end, you'll have a solid foundation for agentic automation.

Let's do this! 🚀

## Quick Start: Your First Workflow in 5 Minutes

The fastest way to experience agentic workflows is to install a working example. We'll walk you through it step by step.

### Prerequisites

Before starting, make sure you have:

- ✅ **GitHub CLI** (`gh`) - [Install here](https://cli.github.com) v2.0.0+
- ✅ **GitHub account** with admin or write access to a repository
- ✅ **GitHub Actions** enabled in your repository
- ✅ **Git** installed on your machine
- ✅ **Operating System**: Linux, macOS, or Windows with WSL

**Verify your setup:**

```bash
gh --version      # Should show version 2.0.0 or higher
gh auth status    # Should show "Logged in to github.com"
git --version     # Should show git version 2.x or higher
```

Looking good? Let's keep going!

### Step 1: Install the Extension

Install the GitHub Agentic Workflows CLI extension:

```bash
gh extension install githubnext/gh-aw
```

:::note
If you're working in GitHub Codespaces and the installation fails, use the standalone installer:

```bash
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

:::

Easy, right?

### Step 2: Add a Sample Workflow

Navigate to your repository and install a sample workflow from the [Agentics Collection](https://github.com/githubnext/agentics):

```bash
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

This creates a pull request that adds:

- `.github/workflows/daily-team-status.md` (the natural language workflow)
- `.github/workflows/daily-team-status.lock.yml` (the compiled GitHub Actions workflow)

Review the PR and merge it into your repository. You're doing great!

### Step 3: Configure AI Authentication

Workflows need to authenticate with an AI service. By default, they use **GitHub Copilot**.

#### Create a Personal Access Token (PAT)

1. Visit <https://github.com/settings/personal-access-tokens/new>
2. Configure the token:
   - **Token name**: "Agentic Workflows Copilot"
   - **Expiration**: 90 days (recommended for testing)
   - **Resource owner**: Your personal account
   - **Repository access**: "Public repositories" or "All repositories"
3. Add permissions:
   - In **"Account permissions"** (not Repository permissions)
   - Find **"Copilot Requests"**
   - Set to **"Access: Read"**
4. Click **"Generate token"** and copy it immediately

:::tip
Can't find "Copilot Requests" permission? Make sure you have:

- An active [GitHub Copilot subscription](https://github.com/settings/copilot)
- A fine-grained token (not classic)
- Personal account as Resource owner
- Public or all repositories selected

:::

#### Add Token to Your Repository

1. Go to your repository → **Settings** → **Secrets and variables** → **Actions**
2. Click **"New repository secret"**
3. Set **Name** to `COPILOT_GITHUB_TOKEN`
4. Paste the token in **Secret**
5. Click **"Add secret"**

Perfect! You're almost there.

### Step 4: Verify Setup

Check that everything is configured correctly:

```bash
gh aw status
```

**Expected output:**

```text
Workflow              Engine    State     Enabled  Schedule
──────────────────────────────────────────────────────────
daily-team-status     copilot   ✓         Yes      0 9 * * 1-5
```

Looking good!

### Step 5: Run Your First Workflow

Trigger the workflow immediately (no need to wait for the schedule):

```bash
gh aw run daily-team-status
```

After a minute or two, check the results:

```bash
gh aw status
```

Once complete, check your repository's **Discussions** section for the generated team status report!

🎉 **Congratulations!** You've just run your first agentic workflow!

## Growth Path: From One to Many

Now that you have one workflow running, here's how to grow your agent ecosystem:

### Phase 1: Learn by Example (Week 1)

**Run multiple example workflows to understand patterns:**

```bash
# Add a triage agent
gh aw add githubnext/agentics/issue-triage

# Add a CI doctor
gh aw add githubnext/agentics/ci-doctor

# Add a weekly summary
gh aw add githubnext/agentics/weekly-research
```

Observe how different workflows:

- Trigger on different events
- Use different tools
- Create different outputs
- Serve different purposes

### Phase 2: Customize Examples (Week 2)

**Modify existing workflows to fit your needs:**

1. Copy a workflow you like:

   ```bash
   cp .github/workflows/issue-triage.md .github/workflows/my-triage.md
   ```

2. Edit the prompt to match your repository:
   - Change label names
   - Adjust categories
   - Add custom rules
   - Update terminology

3. Recompile:

   ```bash
   gh aw compile .github/workflows/my-triage.md
   ```

4. Test manually:

   ```bash
   gh aw run my-triage
   ```

### Phase 3: Create Original Workflows (Week 3+)

**Build workflows specific to your team's needs:**

Start with a simple Read-Only Analyst:

```markdown
---
description: Weekly dependency report
on:
  schedule: "0 9 * * 1"  # Monday mornings
permissions:
  contents: read
safe_outputs:
  create_discussion:
    title: "Dependency Report - {date}"
    category: "Reports"
imports:
  - shared/reporting.md
---

## Weekly Dependency Analysis

Analyze package.json (or requirements.txt, go.mod, etc.):

1. List all dependencies
2. Check for available updates
3. Identify security vulnerabilities
4. Prioritize updates by importance

Create a discussion with:
- Summary of dependency health
- List of available updates
- Security alerts
- Recommended actions
```

### Phase 4: Build Your Factory (Ongoing)

**Systematically address pain points:**

For each repetitive task, ask:

1. Could an agent do this?
2. What pattern fits best?
3. What's the minimum viable version?
4. How can we test it safely?

**Common starting points:**

- Issue triage and labeling
- CI failure diagnosis
- Documentation updates
- Weekly metrics reports
- Security scanning
- Code quality checks

## Essential Commands Reference

### Workflow Management

```bash
# List all workflows
gh aw list

# Show workflow status
gh aw status [workflow-name]

# Add workflow from collection
gh aw add <source> [--create-pull-request]

# Compile workflow
gh aw compile <workflow.md>

# Run workflow manually
gh aw run <workflow-name>

# Download workflow logs
gh aw logs <workflow-name>
```

### Secret Management

```bash
# Configure AI engine secrets
gh aw secrets bootstrap --engine copilot

# List required secrets
gh aw secrets list

# Validate secret configuration
gh aw secrets validate
```

### Debugging

```bash
# Validate workflow syntax
gh aw validate <workflow.md>

# Show compilation output
gh aw compile <workflow.md> --output preview.yml

# Audit workflow runs
gh aw audit <run-id>

# Inspect MCP configuration
gh aw mcp inspect <workflow-name>
```

## Best Practices for Beginners

### Start Small

✅ **Do**: Begin with read-only analyst workflows
❌ **Don't**: Start with workflows that modify code

### Test Manually First

✅ **Do**: Use `workflow_dispatch` triggers initially
❌ **Don't**: Deploy directly to automatic schedules

### Use Time Limits

✅ **Do**: Add `stop-after: "+1mo"` to experiments
❌ **Don't**: Let experimental workflows run indefinitely

### Copy Successful Patterns

✅ **Do**: Clone and modify working workflows
❌ **Don't**: Build everything from scratch

### Review Every Output

✅ **Do**: Check issues, PRs, and discussions agents create
❌ **Don't**: Assume agents always get it right

### Iterate Gradually

✅ **Do**: Make small changes, test, adjust
❌ **Don't**: Make large changes without testing

## Common First-Week Questions

### "Which AI engine should I use?"

**Start with Copilot** (default). It's integrated with GitHub and uses your Copilot subscription. Try other engines later:

- **Claude**: For longer context and detailed analysis
- **Codex**: For enterprise Azure integration
- **Custom**: For proprietary or specialized models

### "How do I handle secrets?"

Use repository secrets (Settings → Secrets → Actions):

- `COPILOT_GITHUB_TOKEN` for Copilot
- `ANTHROPIC_API_KEY` for Claude
- `AZURE_OPENAI_*` for Codex

Never put secrets in workflow files!

### "What if a workflow creates too many issues?"

Use safe output guardrails:

```yaml
safe_outputs:
  create_issue:
    max_items: 3         # Limit to 3
    close_older: true    # Close duplicates
    expire: "+7d"        # Auto-close after 7 days
```

### "How much does this cost?"

Costs depend on:

- **GitHub Actions**: Free tier covers many workflows
- **AI API calls**: Billed per request/token
- **Copilot**: Included in Copilot subscription

Start with free tier, monitor usage with `gh aw audit`.

### "Can I use this in production?"

⚠️ **GitHub Agentic Workflows is a research demonstrator** in early development. Use with caution:

- Review all agent outputs
- Use time-limited trials
- Implement human approval gates
- Monitor security alerts
- Have rollback plans

### "Where can I get help?"

Resources:

- **Documentation**: <https://githubnext.github.io/gh-aw/>
- **Examples**: <https://github.com/githubnext/agentics>
- **Discussions**: <https://github.com/githubnext/gh-aw/discussions>
- **Discord**: [GitHub Next Discord](https://gh.io/next-discord) #continuous-ai

## Your First Week Plan

### Day 1: Installation and Setup

- Install gh-aw extension
- Add first sample workflow
- Configure authentication
- Run first workflow successfully

### Day 2-3: Exploration

- Install 3-5 different workflow types
- Observe how they behave
- Review their outputs
- Identify patterns

### Day 4-5: Customization

- Pick your favorite workflow
- Modify it for your repository
- Test the changes
- Deploy to schedule

### Day 6-7: Creation

- Identify a pain point in your workflow
- Find similar example workflow
- Adapt it to your needs
- Start with manual trigger only

## Next Steps

Once you're comfortable with the basics:

1. **Study the patterns** - Review [12 Design Patterns](03-design-patterns.md)
2. **Explore advanced features** - Repo-memory, multi-phase workflows
3. **Join the community** - Share your workflows
4. **Contribute back** - Add your workflows to Agentics collection
5. **Build your factory** - Create an ecosystem of cooperating agents

## Welcome to the Factory

You're now part of a growing community exploring the frontier of automated agentic development. Start small, experiment safely, and share what you learn.

The agents you build today will help shape the future of software development.

**Ready to build your first workflow?** Head over to the [documentation](https://githubnext.github.io/gh-aw/) and start experimenting!

## What's Next?

_More articles in this series coming soon._

[Previous Article](/gh-aw/blog/2026-02-08-authoring-workflows/)
