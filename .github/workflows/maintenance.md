---
on:
  workflow_dispatch:

permissions:
  contents: write

engine: claude

tools:
  bash:
    - "make deps"
    - "make fmt"
    - "make recompile"
    - "git add ."
    - "git commit"
    - "git config"
    - "git status"
    - "git diff"

timeout_minutes: 15
---

# Repository Maintenance

Perform routine maintenance tasks including dependency installation, code formatting, recompilation, and committing any changes.

## Tasks

1. **Install Dependencies**: Run `make deps` to ensure all project dependencies are up to date
2. **Format Code**: Run `make fmt` to format all code according to project standards  
3. **Recompile Workflows**: Run `make recompile` to recompile all workflow files and ensure they are current
4. **Check for Changes**: Verify if any files were modified during the maintenance process
5. **Commit Changes**: If changes exist, configure git user and commit them with an appropriate message

## Instructions

Configure the GitHub Actions bot user for commits:
```bash
git config --global user.name "github-actions[bot]"
git config --global user.email "github-actions[bot]@users.noreply.github.com"
```

Execute the maintenance tasks in sequence and commit any resulting changes. Keep the commit message simple and descriptive, indicating this was an automated maintenance run.

Only commit if there are actual changes to avoid empty commits.