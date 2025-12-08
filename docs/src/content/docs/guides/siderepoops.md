---
title: SideRepoOps
description: Use a separate repository to run agentic workflows that target your main codebase, minimizing noise and keeping workflows private
sidebar:
  badge: { text: 'Advanced', variant: 'caution' }
---

SideRepoOps is a development pattern where you run agentic workflows from a separate "side" repository that targets your main codebase. This keeps AI-generated issues, comments, and workflow runs isolated from your main repository, providing cleaner separation between automation infrastructure and your production code.

## When to Use SideRepoOps

- **Reduce noise in main repo** - Keep AI-generated issues and PRs separate from organic development
- **Private workflow storage** - Store sensitive workflow configurations in a private repo
- **Workflow experimentation** - Test agentic automation without cluttering your main repository
- **Cross-organization workflows** - Run workflows that operate on repositories you don't directly control
- **Centralized automation hub** - Manage all automation workflows in a dedicated repository

## How It Differs from MultiRepoOps

While [MultiRepoOps](/gh-aw/guides/multirepoops/) runs workflows **from** your main repository that create resources **in** other repositories, SideRepoOps inverts this pattern:

| Pattern | Workflow Location | Target Repository | Use Case |
|---------|------------------|-------------------|----------|
| **MultiRepoOps** | Main repository | Other repositories | Coordinate work across related projects |
| **SideRepoOps** | Separate side repo | Main repository | Isolate automation infrastructure from main codebase |

**Example Architecture:**

```
┌─────────────────┐          ┌──────────────────┐
│  Side Repo      │          │  Main Repo       │
│  (workflows)    │ ────────>│  (target code)   │
│                 │   Uses   │                  │
│  - automation/  │   PAT    │  - src/          │
│  - .github/     │          │  - tests/        │
│    workflows/   │          │  - docs/         │
└─────────────────┘          └──────────────────┘
```

In SideRepoOps, workflows run in GitHub Actions **on the side repository** but perform operations (create issues, PRs, comments) **on the main repository** using cross-repository authentication.

## Setup Requirements

### 1. Create the Side Repository

Create a new repository to host your agentic workflows. This repository can be:

- **Public** - If workflows don't contain sensitive information
- **Private** - For proprietary automation or sensitive configurations
- **Empty** - No code required; just workflows

```bash
# Create new repository via GitHub CLI
gh repo create my-org/my-project-automation --private

# Clone it locally
gh repo clone my-org/my-project-automation
cd my-project-automation
```

### 2. Configure Personal Access Token (PAT)

Create a PAT with access to your main repository:

**For Fine-Grained PAT:**

1. Go to [Settings → Developer settings → Personal access tokens → Fine-grained tokens](https://github.com/settings/personal-access-tokens/new)
2. Set **Repository access** to include your main repository
3. Grant permissions:
   - **Contents**: Read (to access code)
   - **Issues**: Read+Write (to create/update issues)
   - **Pull requests**: Read+Write (to create/update PRs)
   - **Metadata**: Read (required for repository access)

**For Classic PAT:**

1. Go to [Settings → Developer settings → Personal access tokens → Tokens (classic)](https://github.com/settings/tokens/new)
2. Select scopes:
   - `repo` (Full control of private repositories)

**Store the PAT as a secret in your side repository:**

```bash
cd my-project-automation
gh secret set MAIN_REPO_PAT -a actions --body "YOUR_PAT_HERE"
```

:::tip[Security Best Practice]
Use fine-grained PATs scoped to specific repositories rather than classic PATs with broad access. Set expiration dates and rotate tokens regularly.
:::

### 3. Enable GitHub Actions

Ensure GitHub Actions is enabled in your side repository:

```bash
# Check if Actions is enabled
gh api repos/:owner/:repo --jq '.has_issues,.has_projects,.has_wiki'

# Enable Actions in repository settings if needed
```

## Workflow Configuration

### Basic SideRepoOps Workflow

Create a workflow in your side repository that targets the main repository:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      task_description:
        description: "What should the agent work on?"
        required: true

engine: copilot

permissions:
  contents: read

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  create-issue:
    target-repo: "my-org/main-repo"
    title-prefix: "[automation] "
    labels: [automation, ai-generated]
  create-pull-request:
    target-repo: "my-org/main-repo"
    base-branch: main

tools:
  github:
    mode: remote
    toolsets: [repos, issues, pull_requests]
---

# Side Repository Automation

You are running from a separate automation repository and need to work on the main codebase.

**Target Repository**: my-org/main-repo

**Task**: {{inputs.task_description}}

**Instructions**:

1. Use GitHub tools to explore the main repository:
   - Search for relevant code and files
   - Review recent issues and pull requests
   - Check existing documentation

2. Complete the requested task:
   - Create issues with clear descriptions
   - Generate pull requests with well-documented changes
   - Add appropriate labels and assignments

3. All created resources should:
   - Include "[automation]" prefix in titles
   - Link back to relevant context
   - Have clear acceptance criteria
   - Include labels: automation, ai-generated

Remember: You are working across repositories. The workflow runs here in the automation repo, but all outputs go to the main repo.
```

### Scheduled Monitoring from Side Repo

Run scheduled checks on your main repository:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Weekly Monday 9am UTC

engine: copilot

permissions:
  contents: read

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  create-issue:
    target-repo: "my-org/main-repo"
    max: 5
    labels: [weekly-check, automation]

tools:
  github:
    mode: remote
    toolsets: [repos, issues, pull_requests, actions]
---

# Weekly Repository Health Check

Analyze the main repository and create issues for any concerns.

**Target Repository**: my-org/main-repo

Check the following:

1. **Stale Pull Requests**: PRs open for >30 days without activity
2. **Failed CI Runs**: Recent workflow failures on main branch
3. **Dependency Updates**: Outdated dependencies with security advisories
4. **Documentation Gaps**: Missing or outdated documentation
5. **Code Quality**: High-complexity code that needs refactoring

Create issues for significant findings with:
- Clear problem description
- Links to relevant code/PRs
- Suggested next steps
- Appropriate priority labels
```

## GitHub Tools Configuration

When workflows run in a side repository, you must enable GitHub tools with `mode: remote` to access the main repository:

```yaml wrap
tools:
  github:
    mode: remote                    # Required for cross-repo access
    toolsets: [repos, issues, pull_requests]
```

**Available toolsets for cross-repository operations:**

- **repos**: Read files, search code, list commits, get releases from main repo
- **issues**: List, search, read issues from main repo
- **pull_requests**: List, search, read PRs from main repo  
- **actions**: Access workflow runs and artifacts from main repo
- **context**: Repository metadata and settings

:::caution
GitHub tools with `mode: local` (Docker-based) can only access the repository where the workflow is running. Always use `mode: remote` for SideRepoOps.
:::

## Private Repository Access

When your main repository is private, ensure proper authentication:

### PAT Requirements

Your PAT must have:
- Repository access explicitly granted to the private main repository
- Appropriate permission scopes (contents, issues, pull_requests)

### Workflow Configuration

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}  # Required for private repos
  create-issue:
    target-repo: "my-org/private-main-repo"

tools:
  github:
    mode: remote
    github-token: ${{ secrets.MAIN_REPO_PAT }}  # Explicit token for tools
    toolsets: [repos, issues, pull_requests]
```

:::warning
The default `GITHUB_TOKEN` provided by GitHub Actions cannot access other repositories. You **must** use a PAT with appropriate permissions for SideRepoOps.
:::

## Managing Generated Issues

### Issue Transfer Workflow

Generated issues in the main repository can be transferred manually when appropriate:

1. **Review automation-generated issues** in main repository:
   ```bash
   cd main-repo
   gh issue list --label automation --state open
   ```

2. **Transfer valuable issues** to project boards or milestones:
   ```bash
   # Add to project
   gh issue edit 123 --add-project "Main Project Board"
   
   # Add to milestone
   gh issue edit 123 --milestone "v2.0"
   
   # Update labels (remove automation marker)
   gh issue edit 123 --remove-label automation --add-label feature
   ```

3. **Close false positives** from side repo workflows:
   ```bash
   gh issue close 124 --reason "not planned" \
     --comment "Issue resolved or not applicable"
   ```

### Issue Organization Strategy

Use labels to distinguish between automation stages:

| Label | Purpose |
|-------|---------|
| `automation` | Initially applied to all AI-generated issues |
| `needs-review` | Awaiting human review |
| `approved` | Human-validated and ready for work |
| `false-positive` | Not applicable; will be closed |
| `transferred` | Moved to proper project/milestone |

```aw wrap
---
safe-outputs:
  create-issue:
    target-repo: "my-org/main-repo"
    labels: [automation, needs-review]  # Start with review stage
---

All created issues should start with automation + needs-review labels.
```

## Common Patterns

### Triage from Side Repository

Run triage workflows on main repository issues:

```aw wrap
---
on:
  schedule:
    - cron: "0 */6 * * *"  # Every 6 hours

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  add-labels:
    target-repo: "my-org/main-repo"
  add-comment:
    target-repo: "my-org/main-repo"

tools:
  github:
    mode: remote
    toolsets: [issues]
---

# Triage Main Repository Issues

Find unlabeled issues in my-org/main-repo and add appropriate labels.
```

### Code Quality Monitoring

Monitor main repository for quality issues:

```aw wrap
---
on:
  schedule:
    - cron: "0 2 * * 1"  # Weekly Monday 2am UTC

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  create-issue:
    target-repo: "my-org/main-repo"
    labels: [code-quality, automation]
    max: 10

tools:
  github:
    mode: remote
    toolsets: [repos, pull_requests]
---

# Weekly Code Quality Review

Analyze recent commits and PRs in my-org/main-repo for:
- Code complexity issues
- Missing test coverage
- Outdated dependencies
- Security vulnerabilities

Create issues for significant findings.
```

### Documentation Sync

Keep documentation synchronized from side repository:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      docs_path:
        description: "Path to documentation folder"
        default: "docs/"

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  create-pull-request:
    target-repo: "my-org/main-repo"
    base-branch: main
    title-prefix: "[docs] "

tools:
  github:
    mode: remote
    toolsets: [repos]
---

# Documentation Synchronization

Review documentation in {{inputs.docs_path}} of my-org/main-repo.
Create a PR with suggested improvements.
```

## Best Practices

### Security

- **Rotate tokens regularly** - Set expiration dates and rotate PATs quarterly
- **Minimize token scope** - Grant only required permissions on main repository
- **Use GitHub Apps** - For enhanced security with automatic token revocation:
  ```yaml wrap
  safe-outputs:
    app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
      owner: "my-org"
      repositories: ["main-repo"]
    create-issue:
      target-repo: "my-org/main-repo"
  ```
- **Audit workflow runs** - Regularly review Actions runs in side repository
- **Restrict workflow access** - Limit who can trigger workflows in side repository

### Workflow Design

- **Clear labeling** - Always mark AI-generated items with distinctive labels
- **Title prefixes** - Use `[automation]` or similar prefix for visibility
- **Rate limiting** - Set appropriate `max:` values on safe outputs
- **Error handling** - Gracefully handle permission failures
- **Documentation** - Comment workflows explaining cross-repo relationships

### Issue Management

- **Regular review** - Schedule time to review automation-generated issues
- **Clear workflows** - Document the review → approve → transfer process
- **Feedback loops** - Track false positive rates and refine workflows
- **Deduplication** - Check for existing issues before creating new ones
- **Context preservation** - Always link back to triggering events or analysis

### Repository Organization

```
side-repo/
├── .github/
│   └── workflows/
│       ├── triage-main.md           # Scheduled triage
│       ├── quality-check.md         # Code quality monitoring
│       ├── doc-sync.md              # Documentation sync
│       └── manual-task.md           # Manual dispatch tasks
├── docs/
│   ├── README.md                    # Side repo documentation
│   └── workflow-guide.md            # How to use workflows
└── .env.example                     # Required secrets documentation
```

## Troubleshooting

### Authentication Failures

**Problem**: Workflow fails with "Resource not accessible by integration" error.

**Solution**: 
1. Verify PAT has access to target repository
2. Check PAT permissions include required scopes
3. Ensure PAT hasn't expired
4. Confirm `github-token` is properly configured:
   ```yaml wrap
   safe-outputs:
     github-token: ${{ secrets.MAIN_REPO_PAT }}  # Not GITHUB_TOKEN
   ```

### GitHub Tools Not Working

**Problem**: Agent cannot read files from main repository.

**Solution**:
1. Use `mode: remote` for GitHub tools:
   ```yaml wrap
   tools:
     github:
       mode: remote
       toolsets: [repos]
   ```
2. Provide explicit token if main repo is private:
   ```yaml wrap
   tools:
     github:
       mode: remote
       github-token: ${{ secrets.MAIN_REPO_PAT }}
   ```

### Issues Created in Wrong Repository

**Problem**: Issues appear in side repository instead of main repository.

**Solution**: Always specify `target-repo` in safe outputs:
```yaml wrap
safe-outputs:
  create-issue:
    target-repo: "my-org/main-repo"  # Required for cross-repo
```

Without `target-repo`, safe outputs default to the repository where the workflow runs (the side repository).

## Advanced Topics

### Multi-Target SideRepoOps

Manage multiple main repositories from one side repository:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      target_repo:
        description: "Target repository (owner/repo)"
        required: true
      task:
        description: "Task description"
        required: true

safe-outputs:
  github-token: ${{ secrets.MULTI_REPO_PAT }}
  create-issue:
    target-repo: ${{ inputs.target_repo }}  # Dynamic target
---

# Multi-Repository Automation

Work on user-specified repository: {{inputs.target_repo}}
```

### Using GitHub Apps

For enhanced security, use GitHub Apps instead of PATs:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.AUTOMATION_APP_ID }}
    private-key: ${{ secrets.AUTOMATION_APP_KEY }}
    owner: "my-org"
    repositories: ["main-repo", "secondary-repo"]
  create-issue:
    target-repo: "my-org/main-repo"
```

GitHub App benefits:
- Tokens automatically expire after job completion
- More granular permission control
- Better audit trail
- Can be installed at organization level

### Bidirectional Sync

Create workflows in main repository that report back to side repository:

**In main-repo:**
```yaml wrap
on:
  issues:
    types: [opened, labeled]

safe-outputs:
  github-token: ${{ secrets.SIDE_REPO_PAT }}
  add-comment:
    target-repo: "my-org/automation-repo"
```

This creates a feedback loop where the side repository tracks automation effectiveness.

## Example Use Cases

### Experimentation Repository

**Scenario**: Testing agentic workflows without affecting main development.

**Setup**:
- Side repo: `my-org/workflow-experiments` (private)
- Main repo: `my-org/production-app` (private)
- Workflows create issues with `experiment` label
- Review weekly, promote successful experiments

### Third-Party Repository Monitoring

**Scenario**: Monitor open-source dependencies for security issues.

**Setup**:
- Side repo: `my-company/security-monitoring` (private)
- Target repos: Various open-source projects (public)
- Scheduled workflows check for vulnerabilities
- Create internal tracking issues

### Cross-Organization Automation

**Scenario**: Consulting team managing multiple client repositories.

**Setup**:
- Side repo: `consulting-firm/client-automation` (private)
- Target repos: Multiple client organizations
- PAT with access to client repositories
- Workflows parameterized by client/repo

## Related Patterns

- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Coordinate work across multiple repositories from main repo
- **[Campaigns](/gh-aw/guides/campaigns/)** - Orchestrate multi-issue initiatives
- **[IssueOps](/gh-aw/examples/issue-pr-events/issueops/)** - Issue-driven automation patterns

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [GitHub Tools](/gh-aw/reference/tools/#github-tools-github) - GitHub API toolsets
- [GitHub Tokens](/gh-aw/reference/tokens/) - Token types and precedence
- [Security Best Practices](/gh-aw/guides/security/) - Authentication and security
- [Packaging & Distribution](/gh-aw/guides/packaging-imports/) - Sharing workflows
