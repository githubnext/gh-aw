---
title: Using Beads for Task Management
description: Learn how to use Beads - a Git-backed issue tracker for AI agents - in your agentic workflows to provide persistent memory across sessions
sidebar:
  order: 850
---

:::caution[Experimental Feature]
Beads integration is currently in development. This guide shows how to use Beads with bash commands and manual installation until native compiler support is complete.
:::

[Beads](https://github.com/steveyegge/beads) is a Git-backed issue tracker designed specifically for AI coding agents. It provides persistent, structured memory that helps agents remember tasks, track dependencies, and maintain context across multiple sessions.

## Why Use Beads?

Traditional AI agents lose their context when sessions end. Beads solves this by:

- **Persistent Memory**: All tasks are stored in `.beads/` directory and versioned with Git
- **Task Dependencies**: Graph-based relationships (blocks, parent-child, related) help agents understand task order
- **Conflict-Free**: Hash-based IDs prevent merge conflicts in multi-agent workflows
- **Agent-Optimized**: CLI-first design with JSON output for easy AI consumption
- **Automatic Context**: Agents can query ready tasks, check status, and update completion

## Key Concepts

### Task States
- **Ready**: Unblocked tasks that can be worked on now
- **Blocked**: Tasks waiting on dependencies
- **Completed**: Finished tasks (can be compacted/summarized)
- **In Progress**: Tasks currently being worked on

### Dependencies
- **Blocks**: Task A blocks Task B (B can't start until A completes)
- **Parent-Child**: Hierarchical relationship for epics and sub-tasks
- **Related**: Informational link between tasks
- **Discovered-From**: Tracks task discovery chain

### Storage
- **`.beads/` directory**: JSONL files with all task data
- **SQLite cache**: Local cache for performance
- **Git integration**: Version control for task history

## Basic Setup

### 1. Configure Your Workflow

Add Beads installation and bash commands to your workflow:

```yaml title=".github/workflows/task-manager.md"
---
description: Task management with Beads
on:
  issues:
    types: [opened]
  workflow_dispatch:

permissions:
  contents: read
  issues: read

steps:
  - name: Install Beads
    run: |
      # Install Beads CLI
      curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
      
      # Add to PATH
      echo "$HOME/.beads/bin" >> $GITHUB_PATH
      
      # Verify installation
      bd --version

tools:
  github:
    toolsets: [default]
  bash:
    - "bd init"
    - "bd ready"
    - "bd create *"
    - "bd status *"
    - "bd close *"
    - "bd list *"
    - "bd sync"
    - "git add .beads/"
    - "git commit -m *"
    - "git push"

safe-outputs:
  add-comment:
    max: 1

timeout-minutes: 10
---

# Task Manager

You are a task management assistant using Beads.

## Initialize Beads

First check if Beads is initialized:

\`\`\`bash
bd init
\`\`\`

## Check Ready Tasks

List all unblocked tasks that are ready to work on:

\`\`\`bash
bd ready
\`\`\`

## Create New Task

Create a task from the GitHub issue:

\`\`\`bash
bd create "{{ github.event.issue.title }}" --priority high
\`\`\`

## Sync with Git

After making changes, sync with Git:

\`\`\`bash
bd sync
\`\`\`

This commits and pushes changes to `.beads/` directory.
```

## Common Commands

### Initialize Repository
```bash
bd init
```

Initializes Beads in the current repository. Creates `.beads/` directory and SQLite database.

### List Ready Tasks
```bash
bd ready
```

Shows all tasks that are unblocked and ready to work on.

### Create Task
```bash
bd create "Fix authentication bug" --priority high
bd create "Update documentation" --priority low
```

Creates a new task with specified priority (high, medium, low).

### Check Status
```bash
bd status <task-id>
```

Shows detailed information about a specific task including dependencies and status.

### Close Task
```bash
bd close <task-id>
```

Marks a task as completed.

### List All Tasks
```bash
bd list
bd list --status ready     # Filter by status
bd list --priority high    # Filter by priority
```

Lists tasks with optional filters.

### Add Dependencies
```bash
bd dep add <child-task-id> <parent-task-id>
```

Creates a blocking dependency (parent must complete before child).

### Sync with Git
```bash
bd sync
```

Commits all changes to `.beads/` directory and pushes to Git.

## Example Workflow: Issue Tracker

Here's a complete example that creates Beads tasks from GitHub issues:

```yaml title=".github/workflows/beads-issue-sync.md"
---
description: Sync GitHub issues to Beads task tracker
on:
  issues:
    types: [opened, edited, closed]
  workflow_dispatch:

permissions:
  contents: read
  issues: read

steps:
  - name: Install Beads
    run: |
      curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
      echo "$HOME/.beads/bin" >> $GITHUB_PATH

tools:
  github:
    toolsets: [default]
  bash:
    - "bd init"
    - "bd ready"
    - "bd create *"
    - "bd status *"
    - "bd close *"
    - "bd list *"
    - "bd sync"
    - "git add .beads/"
    - "git commit -m *"
    - "git push"

safe-outputs:
  add-comment:
    max: 1

timeout-minutes: 10
---

# Beads Issue Sync

You are an issue synchronization assistant that keeps Beads tasks in sync with GitHub issues.

## Current Issue

- **Number**: ${{ github.event.issue.number }}
- **Title**: ${{ github.event.issue.title }}
- **State**: ${{ github.event.issue.state }}
- **Labels**: ${{ join(github.event.issue.labels.*.name, ', ') }}

## Your Task

1. **Initialize Beads** (if needed):
   \`\`\`bash
   bd init
   \`\`\`

2. **For opened issues**: Create a new Beads task
   \`\`\`bash
   bd create "${{ github.event.issue.title }}" --priority medium
   \`\`\`
   
   Store the task ID and add it to your comment.

3. **For closed issues**: Find and close corresponding Beads task
   \`\`\`bash
   bd list
   # Find task with matching title
   bd close <task-id>
   \`\`\`

4. **Sync changes**:
   \`\`\`bash
   bd sync
   \`\`\`

5. **Add comment** with Beads task ID and current ready tasks:
   \`\`\`bash
   bd ready
   \`\`\`

## Comment Format

Use the safe-outputs to add a comment like:

\`\`\`markdown
✅ Task synced with Beads

**Task ID**: \`abc123\`
**Action**: Created/Closed
**Priority**: medium

**Current Ready Tasks**:
1. \`abc123\` - Fix authentication bug (high)
2. \`def456\` - Update documentation (low)
3. \`ghi789\` - Add unit tests (medium)

---
View all tasks in the \`.beads/\` directory
\`\`\`
```

## Best Practices

### 1. Always Initialize First
Check if Beads is initialized before creating tasks:

```bash
bd init || echo "Already initialized"
```

### 2. Use Meaningful Task Titles
Make task titles clear and actionable:

```bash
# Good
bd create "Fix memory leak in authentication handler"

# Not as good
bd create "Bug fix"
```

### 3. Set Appropriate Priorities
Use priorities to guide agent focus:

- **high**: Urgent bugs, security issues, blockers
- **medium**: Important features, improvements
- **low**: Nice-to-haves, documentation, cleanup

### 4. Sync Regularly
Always sync after making changes:

```bash
bd create "New task"
bd sync  # Commit and push changes
```

### 5. Use Dependencies
Link related tasks to show relationships:

```bash
# Parent task
parent_id=$(bd create "Implement user authentication")

# Child tasks
bd create "Add login form" --parent $parent_id
bd create "Add session management" --parent $parent_id
```

### 6. Query Before Acting
Check existing tasks before creating duplicates:

```bash
bd list | grep "authentication"
```

## Limitations

### Current Limitations
- **Manual Installation**: Beads must be installed via custom steps
- **Bash Commands**: Must use bash tool with explicit command allowlist
- **Git Operations**: Requires write permissions for syncing

### Coming Soon
- **Native Tool Support**: Automatic Beads installation when `tools: beads:` is configured
- **Simplified Configuration**: No need for manual installation steps
- **Better Integration**: Direct Beads tool calls without bash wrapper

## Troubleshooting

### Beads Not Found
If you see "bd: command not found":

1. Check PATH setup in installation step
2. Verify Beads installation completed successfully
3. Use absolute path: `$HOME/.beads/bin/bd`

### Permission Denied
If you see permission errors when syncing:

1. Ensure workflow has write permissions to repository
2. Check Git configuration in the workflow
3. Verify `.beads/` directory is writable

### Merge Conflicts
Beads uses hash-based IDs to avoid conflicts, but if you encounter them:

1. Pull latest changes before syncing
2. Resolve conflicts in `.beads/` files
3. Run `bd sync` again

## Learn More

- **Beads Repository**: [github.com/steveyegge/beads](https://github.com/steveyegge/beads)
- **Beads Documentation**: See README in the repository
- **Example Workflows**: `.github/workflows/example-beads.md` in gh-aw repository
- **Tool Configuration**: [Tools Reference](/gh-aw/reference/tools/#beads-tool-beads)

## Next Steps

1. Try the example workflow: `.github/workflows/example-beads.md`
2. Customize task priorities and dependencies for your use case
3. Integrate Beads with your existing issue tracking workflow
4. Share feedback on Beads integration in the gh-aw repository

---

**Note**: Beads integration is actively being developed. Native tool support will eliminate the need for manual installation and simplify configuration. Follow the [gh-aw repository](https://github.com/githubnext/gh-aw) for updates.
