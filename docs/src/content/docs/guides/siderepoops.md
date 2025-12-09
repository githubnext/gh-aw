---
title: SideRepoOps
description: Use a separate repository to run agentic workflows that target your main codebase, minimizing noise and keeping workflows private
sidebar:
  badge: { text: 'Advanced', variant: 'caution' }
---

SideRepoOps is a development pattern where you run agentic workflows from a separate "side" repository that targets your main codebase. This keeps AI-generated issues, comments, and workflow runs isolated from your main repository, providing cleaner separation between automation infrastructure and your production code.

## When to Use SideRepoOps

Use SideRepoOps to reduce noise by keeping AI-generated issues separate from organic development, store sensitive workflows in a private repository, or manage automation for repositories you don't directly control. It's ideal for workflow experimentation and creating a centralized automation hub.

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

Create a new repository (public or private) to host your agentic workflows. No code is required—just workflows.

```bash
gh repo create my-org/my-project-automation --private
gh repo clone my-org/my-project-automation
cd my-project-automation
```

### 2. Configure Personal Access Token (PAT)

Create a [fine-grained PAT](https://github.com/settings/personal-access-tokens/new) with repository access to your main repository and grant these permissions: **Contents** (Read), **Issues** (Read+Write), **Pull requests** (Read+Write), and **Metadata** (Read).

For classic PATs, use the `repo` scope. Store the token as a secret:

```bash
gh secret set MAIN_REPO_PAT -a actions --body "YOUR_PAT_HERE"
```

:::tip[Security Best Practice]
Use fine-grained PATs scoped to specific repositories. Set expiration dates and rotate tokens regularly.
:::

### 3. Enable GitHub Actions

Ensure GitHub Actions is enabled in your side repository settings.

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

1. Use GitHub tools to explore the main repository (search code, review issues/PRs, check documentation)
2. Complete the task by creating issues or PRs with clear descriptions and appropriate labels
3. All resources should include "[automation]" prefix, link to context, and have labels: automation, ai-generated

Remember: The workflow runs in the automation repo, but all outputs go to the main repo.
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

Check for stale PRs (>30 days), failed CI runs on main, outdated dependencies with security advisories, documentation gaps, and high-complexity code needing refactoring.

Create issues for significant findings with clear problem descriptions, links to relevant code/PRs, suggested next steps, and priority labels.
```

## GitHub Tools Configuration

When workflows run in a side repository, you must enable GitHub tools with `mode: remote` to access the main repository:

```yaml wrap
tools:
  github:
    mode: remote                    # Required for cross-repo access
    toolsets: [repos, issues, pull_requests]
```

Available toolsets: **repos** (files, code, commits, releases), **issues** (list/search/read), **pull_requests** (list/search/read), **actions** (workflow runs/artifacts), and **context** (repository metadata).

:::caution
GitHub tools with `mode: local` (Docker-based) can only access the repository where the workflow is running. Always use `mode: remote` for SideRepoOps.
:::

## Private Repository Access

For private repositories, your PAT must have explicit repository access with appropriate permission scopes (contents, issues, pull_requests):

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

Review automation-generated issues with `gh issue list --label automation`, transfer valuable ones to project boards or milestones, update labels as needed, and close false positives:

```bash
# Add to project and milestone
gh issue edit 123 --add-project "Main Project Board" --milestone "v2.0"

# Update labels
gh issue edit 123 --remove-label automation --add-label feature

# Close false positives
gh issue close 124 --reason "not planned"
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

## Testing with Trial Mode

Before deploying SideRepoOps workflows to production, use the `trial` command to test them in a safe, isolated environment. Trial mode creates a temporary repository, installs your workflow, and executes it while capturing safe outputs without making actual changes to your target repositories.

### Basic Trial Usage

Test a workflow against your main repository:

```bash
gh aw trial my-automation-repo/my-workflow --logical-repo my-org/main-repo
```

This simulates running the workflow as if it were installed in `my-org/main-repo`, allowing you to verify:
- Workflow logic and agent behavior
- GitHub API calls and permissions
- Safe output generation (issues, PRs, comments)
- Cross-repository operations

:::tip[Safe Testing]
Trial mode prevents actual changes to your main repository. All safe outputs are captured as artifacts, and any PRs or issues are created in the temporary trial repository, not your target repository.
:::

### Trial Mode Configuration

**Key flags for SideRepoOps testing:**

```bash
# Simulate running against a specific repository
gh aw trial my-side-repo/triage --logical-repo my-org/main-repo

# Use a custom trial repository (keeps it for inspection)
gh aw trial my-side-repo/triage --logical-repo my-org/main-repo \
  --host-repo my-org/workflow-testing

# Test with secrets from your environment
gh aw trial my-side-repo/triage --logical-repo my-org/main-repo \
  --use-local-secrets

# Run multiple iterations to test consistency
gh aw trial my-side-repo/triage --logical-repo my-org/main-repo \
  --repeat 3

# Auto-cleanup after testing
gh aw trial my-side-repo/triage --logical-repo my-org/main-repo \
  --delete-host-repo-after
```

### Repository Modes

Trial mode supports different repository configurations:

| Flag | Purpose | Use Case |
|------|---------|----------|
| `--logical-repo` | Simulate execution against target repo | Test SideRepoOps workflows (most common) |
| `--host-repo` | Specify custom trial repository | Keep trial artifacts organized |
| `--clone-repo` | Clone target repo contents into trial | Test with actual repository structure |
| `--repo` | Run directly in specified repository | Production deployment (no simulation) |

**Example: Test triage workflow targeting main repository:**

```bash
# Create and save triage workflow locally
cat > triage-workflow.md << 'EOF'
---
on:
  workflow_dispatch:

engine: copilot

permissions:
  contents: read

safe-outputs:
  github-token: ${{ secrets.MAIN_REPO_PAT }}
  add-labels:
    target-repo: "my-org/main-repo"
    max: 10
  add-comment:
    target-repo: "my-org/main-repo"

tools:
  github:
    mode: remote
    toolsets: [issues]
---

# Triage Unlabeled Issues

Find unlabeled issues in my-org/main-repo and add appropriate labels.
Review recent issues and suggest labels based on content.
EOF

# Test the workflow in trial mode
gh aw trial ./triage-workflow.md \
  --logical-repo my-org/main-repo \
  --use-local-secrets \
  --yes
```

### Understanding Trial Results

After execution, trial mode provides:

**1. Console Output**
- Real-time workflow execution logs
- Workflow run URL for GitHub Actions logs
- Summary of artifacts collected

**2. Local Trial Results** (`trials/` directory)
```json
{
  "workflow_name": "triage-workflow",
  "run_id": "123456789",
  "safe_outputs": {
    "add_labels": [
      {
        "issue_number": 42,
        "labels": ["bug", "needs-triage"]
      }
    ]
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**3. Trial Repository Artifacts**
- Workflow run logs in GitHub Actions
- Generated artifacts (safe outputs, patches, logs)
- Trial results committed to `trials/` directory

:::note
Trial results are saved both locally and in the trial repository's `trials/` directory. This allows you to review outputs, compare multiple runs, and verify workflow behavior before production deployment.
:::

### Multi-Workflow Comparison

Compare multiple workflow variations:

```bash
gh aw trial \
  my-side-repo/triage-v1 \
  my-side-repo/triage-v2 \
  --logical-repo my-org/main-repo \
  --use-local-secrets
```

This generates:
- Individual results for each workflow
- Combined comparison results
- Side-by-side analysis of outputs

Use this to:
- A/B test workflow approaches
- Validate workflow improvements
- Compare different AI engines or prompts

### Common Trial Patterns

**Pattern 1: Pre-deployment Validation**
```bash
# Test workflow before deploying to side repo
gh aw trial ./new-automation.md \
  --logical-repo my-org/main-repo \
  --use-local-secrets \
  --yes

# Review results in trials/ directory
cat trials/new-automation-*.json | jq '.safe_outputs'

# If satisfied, deploy to side repository
gh aw install my-side-repo ./new-automation.md
```

**Pattern 2: Iterative Development**
```bash
# Keep trial repo for iterative testing
gh aw trial ./workflow.md \
  --logical-repo my-org/main-repo \
  --host-repo my-org/workflow-dev \
  --use-local-secrets

# Make changes to workflow.md, then rerun
gh aw trial ./workflow.md \
  --logical-repo my-org/main-repo \
  --host-repo my-org/workflow-dev \
  --use-local-secrets \
  --force-delete-host-repo-before
```

**Pattern 3: Integration Testing**
```bash
# Clone target repository structure
gh aw trial my-side-repo/code-review \
  --clone-repo my-org/main-repo \
  --use-local-secrets

# Workflow runs with actual codebase structure
# (but still isolated in trial repository)
```

### Trial Mode Limitations

**What trial mode tests:**
- Workflow logic and agent reasoning
- GitHub API calls and toolset usage  
- Safe output generation and formatting
- Cross-repository authentication (with proper tokens)

**What trial mode doesn't test:**
- Actual resource creation in target repositories (by design)
- Scheduled triggers (use `workflow_dispatch` for testing)
- Event-based triggers (simulate with `--trigger-context` flag)
- Long-term workflow behavior or race conditions

**Simulating Event Triggers:**
```bash
# Simulate issue-triggered workflow
gh aw trial my-side-repo/issue-responder \
  --logical-repo my-org/main-repo \
  --trigger-context "https://github.com/my-org/main-repo/issues/123" \
  --use-local-secrets
```

### Best Practices for Trial Testing

**Before Deployment:**
1. Always test workflows with `--logical-repo` matching your target
2. Use `--use-local-secrets` to test with your credentials
3. Review captured safe outputs in `trials/` directory
4. Verify GitHub API calls in workflow run logs
5. Test edge cases with different `--trigger-context` values

**During Development:**
1. Keep a dedicated trial repository (`--host-repo my-org/workflow-testing`)
2. Use `--repeat` to catch non-deterministic behavior
3. Compare multiple workflow versions side-by-side
4. Save trial results for regression testing

**Security:**
1. Trial repositories are private by default
2. Secrets pushed with `--use-local-secrets` are automatically cleaned up
3. Review permissions before running with production PATs
4. Use `--delete-host-repo-after` to remove sensitive trial data

## Best Practices

**Security**: Rotate PATs quarterly with expiration dates, minimize token scope to required permissions, use GitHub Apps for automatic token revocation, audit workflow runs regularly, and restrict who can trigger workflows.

**Workflow Design**: Mark AI-generated items with distinctive labels and `[automation]` prefixes, set appropriate rate limits with `max:` values, handle permission failures gracefully, and document cross-repo relationships.

**Issue Management**: Schedule regular reviews of automation-generated issues, document the review → approve → transfer process, track false positive rates, deduplicate before creating issues, and preserve context by linking to triggering events.

**Repository Organization**:

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

If you see "Resource not accessible by integration" errors, verify your PAT has access to the target repository, check permissions include required scopes, ensure the PAT hasn't expired, and confirm `github-token: ${{ secrets.MAIN_REPO_PAT }}` is configured (not `GITHUB_TOKEN`).

### GitHub Tools Not Working

If the agent cannot read files from the main repository, use `mode: remote` for GitHub tools and provide an explicit token if the main repo is private:

```yaml wrap
tools:
  github:
    mode: remote
    github-token: ${{ secrets.MAIN_REPO_PAT }}
    toolsets: [repos]
```

### Issues Created in Wrong Repository

Always specify `target-repo` in safe outputs. Without it, safe outputs default to the repository where the workflow runs:

```yaml wrap
safe-outputs:
  create-issue:
    target-repo: "my-org/main-repo"
```

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

For enhanced security, use GitHub Apps instead of PATs. GitHub Apps provide automatic token expiration after job completion, more granular permission control, better audit trails, and can be installed at organization level:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.AUTOMATION_APP_ID }}
    private-key: ${{ secrets.AUTOMATION_APP_KEY }}
    owner: "my-org"
    repositories: ["main-repo"]
  create-issue:
    target-repo: "my-org/main-repo"
```

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

**Experimentation Repository**: Test agentic workflows without affecting main development. Side repo creates issues with `experiment` label in main repo, reviewed weekly to promote successful experiments.

**Third-Party Repository Monitoring**: Monitor open-source dependencies for security issues using scheduled workflows that create internal tracking issues in your private side repo.

**Cross-Organization Automation**: Consulting teams can manage multiple client repositories using parameterized workflows with PATs that have access to client repositories.

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
