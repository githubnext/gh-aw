---
title: Your First Workflow
description: Create your first agentic workflow with this minimal hello world example. Learn the basics by building a simple workflow that creates a GitHub issue.
sidebar:
  order: 1
---

Welcome! This guide shows you the absolute simplest agentic workflow possible. You'll create a "hello world" workflow that automatically creates a GitHub issue. No complex concepts—just copy, paste, and run.

## Prerequisites

Before starting, ensure you have:
- **GitHub CLI (`gh`)** installed and authenticated ([Quick Start](/gh-aw/setup/quick-start/))
- **GitHub Agentic Workflows extension** installed: `gh extension install githubnext/gh-aw`
- **Repository access** where you can commit files
- **`COPILOT_GITHUB_TOKEN` secret** configured ([see setup instructions](/gh-aw/setup/quick-start/#step-3--add-an-ai-secret))

## The Simplest Workflow

Here's your first agentic workflow—just 5 lines:

```aw
---
engine: copilot
---

Create a GitHub issue with title "Hello from my first workflow" and body "This is my first automated workflow!"
```

That's it! This workflow:
- Uses the **GitHub Copilot engine** to understand your instructions
- Automatically gets **read-only permissions** (safe by default)
- **Creates a GitHub issue** based on your natural language instruction

## Step-by-Step Guide

### Step 1: Create the workflow file

In your repository, create a new file at `.github/workflows/hello.md`:

```bash
mkdir -p .github/workflows
```

Then create the file with this content:

```bash
cat > .github/workflows/hello.md << 'EOF'
---
engine: copilot
---

Create a GitHub issue with title "Hello from my first workflow" and body "This is my first automated workflow!"
EOF
```

Or use your favorite text editor to create `.github/workflows/hello.md` with the workflow content above.

### Step 2: Compile the workflow

Compile your markdown file into a GitHub Actions workflow:

```bash
gh aw compile hello.md
```

This creates `hello.lock.yml`—the compiled GitHub Actions workflow file. You'll see both files:
- `hello.md` - Your human-friendly source (edit this)
- `hello.lock.yml` - The generated GitHub Actions YAML (never edit directly)

:::tip[What's compilation?]
The `compile` command translates your natural language markdown into a secure GitHub Actions workflow with safety checks and permissions configured automatically.
:::

### Step 3: Commit and push

Commit both files to your repository:

```bash
git add .github/workflows/hello.md .github/workflows/hello.lock.yml
git commit -m "Add hello world workflow"
git push
```

:::caution[Commit both files]
Always commit **both** `.md` and `.lock.yml` files. GitHub Actions runs the `.lock.yml` file, but the `.md` file is your source of truth for editing.
:::

### Step 4: Trigger the workflow

Since this workflow has no trigger defined, you'll run it manually:

```bash
gh aw run hello
```

This triggers the workflow on GitHub Actions. After a few moments, check your repository—you'll see a new issue titled "Hello from my first workflow"!

### Step 5: View the results

Check the workflow status:

```bash
gh aw status
```

View the created issue in your repository's Issues tab, or use:

```bash
gh issue list --limit 5
```

🎉 **Congratulations!** You've created your first agentic workflow!

## Understanding What Happened

Let's break down what just happened:

### The Workflow File

Your workflow has two parts:

1. **Frontmatter** (between `---` markers):
   ```yaml
   engine: copilot
   ```
   This tells the system to use GitHub Copilot as the AI engine.

2. **Markdown Instructions**:
   ```
   Create a GitHub issue with title "Hello from my first workflow" and body "This is my first automated workflow!"
   ```
   This is the natural language instruction the AI follows.

### How It Works

1. **Compilation**: `gh aw compile` translates your markdown into a secure GitHub Actions workflow
2. **Execution**: GitHub Actions runs the workflow in a containerized environment
3. **AI Processing**: GitHub Copilot reads your instructions and understands the task
4. **Safe Output**: The AI creates the issue using a validated, secure output mechanism
5. **Result**: A new GitHub issue appears in your repository

### Why Two Files?

- **`.md` file**: Human-friendly source you write and edit
- **`.lock.yml` file**: Machine-ready GitHub Actions YAML that GitHub executes

Think of it like source code (`.md`) and compiled binary (`.lock.yml`). Always edit the `.md` file and recompile—never edit `.lock.yml` directly.

## Next Steps

Now that you've created your first workflow, try these modifications:

### Add a Trigger

Make the workflow run automatically when issues are opened:

```aw
---
engine: copilot
on:
  issues:
    types: [opened]
---

Create a welcome comment on issue #${{ github.event.issue.number }} saying "Thanks for opening this issue!"
```

Save, recompile (`gh aw compile hello.md`), commit, and push. Now the workflow runs automatically when anyone opens an issue!

### Add Permissions

If you need specific permissions, declare them explicitly:

```aw
---
engine: copilot
permissions:
  issues: write
  contents: read
---

Create a GitHub issue with title "Hello from my first workflow" and body "This is my first automated workflow!"
```

### Add Safe Outputs

For more control over what the AI can create:

```aw
---
engine: copilot
safe-outputs:
  create-issue:
---

Create a GitHub issue with title "Hello from my first workflow" and body "This is my first automated workflow!"
```

[Safe outputs](/gh-aw/reference/safe-outputs/) provide validated, sanitized ways for AI to interact with GitHub without direct write access.

## What You've Learned

✅ How to create an agentic workflow file  
✅ How to compile workflows with `gh aw compile`  
✅ How to commit both `.md` and `.lock.yml` files  
✅ How to trigger workflows manually with `gh aw run`  
✅ The basic structure: frontmatter + markdown instructions  

## Continue Learning

Ready for more? Check out these guides:

- **[Quick Start](/gh-aw/setup/quick-start/)** - Set up more complex workflows with triggers and scheduling
- **[Creating Workflows](/gh-aw/setup/agentic-authoring/)** - Learn advanced workflow authoring techniques
- **[Workflow Structure](/gh-aw/reference/workflow-structure/)** - Understand the anatomy of workflows
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** - Learn about secure AI interactions with GitHub
- **[Examples](/gh-aw/examples/issue-pr-events/)** - Explore real-world workflow patterns

---

**Questions or issues?** Visit the [Troubleshooting](/gh-aw/troubleshooting/common-issues/) section or open an issue in the [gh-aw repository](https://github.com/githubnext/gh-aw).
