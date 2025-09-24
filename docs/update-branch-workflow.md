# Update Branch Workflow

The Update Branch workflow optimistically tries to merge changes from the pull request base branch using JavaScript and `actions/github-script`. This workflow helps keep feature branches up to date with their target branch.

## Features

- **Automatic PR Detection**: Resolves the pull request from the current branch
- **Base Branch Resolution**: Gets the base branch from pull request information
- **Git Configuration**: Sets GitHub Actions as the git user automatically
- **Fast-Forward Only Merges**: Configures git to use `ff only` for merge strategy
- **Conflict Detection**: Attempts git merge and fails gracefully on conflicts
- **Conditional Push**: Only pushes changes if merge was successful
- **Comprehensive Logging**: Detailed status reporting and error handling

## Usage

### Basic Workflow

Create a workflow file (`.github/workflows/update-branch.yml`):

```yaml
name: Update Branch

on:
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: read

jobs:
  update-branch:
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Setup GitHub CLI
        run: |
          echo "${{ secrets.GITHUB_TOKEN }}" | gh auth login --with-token

      - name: Update branch with base branch changes
        uses: actions/github-script@v8
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const fs = require('fs');
            const path = require('path');
            
            const scriptPath = path.join(process.env.GITHUB_WORKSPACE, 'pkg/workflow/js/update_branch.cjs');
            const scriptContent = fs.readFileSync(scriptPath, 'utf8');
            
            const modifiedScript = scriptContent.replace(
              'const exec = require("@actions/exec");', 
              '// exec is available from github-script context'
            );
            
            await eval(`(async () => { ${modifiedScript} })()`)
```

### Reusable Workflow

You can also use it as a reusable workflow:

```yaml
name: Update My Branch

on:
  workflow_dispatch:

jobs:
  update:
    uses: ./.github/workflows/update-branch-reusable.yml
    with:
      branch: feature-branch  # Optional: specify branch name
```

## How It Works

1. **Branch Detection**: Gets the current branch name using `git branch --show-current`
2. **PR Resolution**: Uses GitHub CLI to get pull request information for the branch
3. **Git Setup**: Configures git user as `github-actions[bot]` and enables fast-forward only merges
4. **Fetch Updates**: Fetches latest changes from origin
5. **Merge Attempt**: Tries to merge the base branch using `git merge origin/<base-branch>`
6. **Conflict Handling**: Detects merge conflicts and fails with clear error message
7. **Change Detection**: Checks if there are changes to push after merge
8. **Push Changes**: Pushes updated branch to origin if changes exist

## Outputs

The workflow provides the following outputs:

- `updated`: Boolean indicating whether the branch was updated
- `branch`: The name of the branch that was processed
- `base_branch`: The base branch used for merging
- `pr_number`: The pull request number

## Error Handling

The workflow handles several error scenarios:

- **No PR Found**: Fails if the current branch doesn't have an associated pull request
- **Merge Conflicts**: Detects conflicts and fails with helpful error message
- **Push Failures**: Handles cases where pushing changes fails
- **Git Command Failures**: Properly handles and reports git command failures

## Requirements

- **Permissions**: Requires `contents: write` and `pull-requests: read` permissions
- **Trigger**: Designed for `workflow_dispatch` (manual trigger) only
- **Dependencies**: Requires GitHub CLI (`gh`) and git to be available
- **Context**: Must be run from a branch that has an associated pull request

## Example Scenarios

### Successful Update
```
✅ Branch Update Successful
- Branch: feature-branch
- Base Branch: main
- Pull Request: #123

Successfully merged latest changes from main and pushed to feature-branch.
```

### Already Up to Date
```
ℹ️ Branch Already Up to Date
- Branch: feature-branch  
- Base Branch: main
- Pull Request: #123

No changes needed - branch is already up to date with main.
```

### Merge Conflict
```
❌ Merge conflict detected when merging main into feature-branch. Manual resolution required.
```

## Integration with GitHub Actions

This workflow can be integrated into larger automation flows:

```yaml
jobs:
  update-branch:
    uses: ./.github/workflows/update-branch-reusable.yml
    
  run-tests:
    needs: update-branch
    if: needs.update-branch.outputs.updated == 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Run tests after branch update
        run: echo "Running tests on updated branch"
```

## Security Considerations

- Uses `github-actions[bot]` as the git user identity
- Requires appropriate permissions for the repository
- Only works with pull requests in the same repository
- Fast-forward only merge strategy prevents complex merge scenarios