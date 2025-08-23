# 🔒 Safe Output Processing

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

## Overview (`output:`)

The `output:` element of your workflow's frontmatter declares that your agentic workflow should conclude with optional automated actions based on the agent's output. This enables your AI agent to write content that is then automatically processed to create GitHub issues, comments, pull requests, or add labels—all without giving the agentic portion of the workflow any write permissions.

**How It Works:**
1. Your agentic workflow runs with minimal read-only permissions
2. The agent writes its output to the special `${{ env.GITHUB_AW_OUTPUT }}` environment variable
3. The compiler automatically generates additional jobs that read this output and perform the requested actions
4. Only these generated jobs receive the necessary write permissions

```yaml
output:
  allowed-domains:                    # Optional: domains allowed in agent output URIs
    - github.com                      # Default GitHub domains are always included
    - api.github.com                  # Additional trusted domains can be specified
    - trusted-domain.com              # URIs from unlisted domains are replaced with "(redacted)"
  issue:
    title-prefix: "[ai] "           # Optional: prefix for issue titles
    labels: [automation, ai-agent]  # Optional: labels to attach to issues
  issue_comment: {}                 # Create comments on issues/PRs from agent output
  pull-request:
    title-prefix: "[ai] "           # Optional: prefix for PR titles
    labels: [automation, ai-agent]  # Optional: labels to attach to PRs
    draft: true                     # Optional: create as draft PR (defaults to true)
  labels:
    allowed: [triage, bug, enhancement] # Mandatory: allowed labels for addition
    max-count: 3                        # Optional: maximum number of labels to add (default: 3)
```

## Security and Sanitization

All agent output is automatically sanitized for security before being processed:

- **XML Character Escaping**: Special characters (`<`, `>`, `&`, `"`, `'`) are escaped to prevent injection attacks
- **URI Protocol Filtering**: Only HTTPS URIs are allowed; other protocols (HTTP, FTP, file://, javascript:, etc.) are replaced with "(redacted)"
- **Domain Allowlisting**: HTTPS URIs are checked against the `allowed-domains` list. Unlisted domains are replaced with "(redacted)"
- **Default Allowed Domains**: When `allowed-domains` is not specified, safe GitHub domains are used by default:
  - `github.com`
  - `github.io`
  - `githubusercontent.com`
  - `githubassets.com`
  - `github.dev`
  - `codespaces.new`
- **Length and Line Limits**: Content is truncated if it exceeds safety limits (0.5MB or 65,000 lines)
- **Control Character Removal**: Non-printable characters and ANSI escape sequences are stripped

## Issue Creation (`output.issue`)

Adding `output.issue` to your workflow declares that the workflow should conclude with the creation of a GitHub issue based on the agent's output.

**How Your Agent Provides Output:**
Your agentic workflow writes its content to `${{ env.GITHUB_AW_OUTPUT }}`. The output should be structured as:
- **First non-empty line**: Becomes the issue title (markdown heading syntax like `# Title` is automatically stripped)
- **Remaining content**: Becomes the issue body

**What This Configuration Does:**
When you add `output.issue` to your frontmatter, the compiler automatically generates a separate `create_issue` job that:
- Runs after your main agentic job completes
- Reads the content from `${{ env.GITHUB_AW_OUTPUT }}`
- Parses it to extract title and body
- Creates a GitHub issue with optional title prefix and labels
- **Security**: Only this generated job gets `issues: write` permission—your agentic code runs with minimal permissions

**Example workflow using issue creation:**
```yaml
---
on: push
permissions:
  contents: read      # Main job only needs minimal permissions
  actions: read
engine: claude
output:
  issue:
    title-prefix: "[analysis] "
    labels: [automation, code-review]
---

# Code Analysis Agent

Analyze the latest commit and provide insights.
Write your analysis to ${{ env.GITHUB_AW_OUTPUT }} at the end.

The first line of your output will become the issue title.
The rest will become the issue body.
```

## Issue Comment Creation (`output.issue_comment`)

Adding `output.issue_comment` to your workflow declares that the workflow should conclude with posting a comment on the triggering issue or pull request based on the agent's output.

**How Your Agent Provides Output:**
Your agentic workflow writes its content to `${{ env.GITHUB_AW_OUTPUT }}`. The entire content becomes the comment body—no special formatting is required.

**What This Configuration Does:**
When you add `output.issue_comment` to your frontmatter, the compiler automatically generates a separate `create_issue_comment` job that:
- Only runs when triggered by an issue or pull request event
- Reads the content from `${{ env.GITHUB_AW_OUTPUT }}`
- Posts the entire output as a comment on the triggering issue or PR
- Automatically skips execution if not running in an issue/PR context
- **Security**: Only this generated job gets `issues: write` and `pull-requests: write` permissions

**Example workflow using comment creation:**
```yaml
---
on:
  issues:
    types: [opened, labeled]
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read      # Main job only needs minimal permissions
  actions: read
engine: claude
output:
  issue_comment: {}
---

# Issue/PR Analysis Agent

Analyze the issue or pull request and provide feedback.
Write your analysis to ${{ env.GITHUB_AW_OUTPUT }} at the end.

Your entire output will be posted as a comment on the triggering issue or PR.
```

This automatically creates GitHub issues or comments from the agent's analysis without requiring write permissions on the main job.

## Pull Request Creation (`output.pull-request`)

Adding `output.pull-request` to your workflow declares that the workflow should conclude with the creation of a pull request containing code changes generated by the agent.

**How Your Agent Provides Output:**
Your agentic workflow provides output in two ways:
1. **File changes**: Make any file changes in the working directory—these are automatically collected using `git add -A` and committed
2. **PR description**: Write to `${{ env.GITHUB_AW_OUTPUT }}` with:
   - **First non-empty line**: Becomes the PR title
   - **Remaining content**: Becomes the PR description

**What This Configuration Does:**
When you add `output.pull-request` to your frontmatter, the compiler automatically:
1. **Adds a git patch generation step** to your main job that:
   - Runs `git add -A` to stage any file changes made by your agent
   - Commits staged files with message "[agent] staged files"
   - Generates git patches using `git format-patch` and saves to `/tmp/aw.patch`
2. **Generates a separate `create_pull_request` job** that:
   - Reads the patches from `/tmp/aw.patch` and validates they exist and are valid
   - Creates a new branch with cryptographically secure random naming
   - Applies the git patches to create the code changes
   - Reads the PR description from `${{ env.GITHUB_AW_OUTPUT }}`
   - Creates a pull request with optional title prefix, labels, and draft status
   - **Security**: Only this generated job gets `contents: write`, `issues: write`, and `pull-requests: write` permissions

**Configuration:**
```yaml
output:
  pull-request:
    title-prefix: "[ai] "           # Optional: prefix for PR titles
    labels: [automation, ai-agent]  # Optional: labels to attach to PRs
    draft: true                     # Optional: create as draft PR (defaults to true)
```

**Example workflow using pull request creation:**
```markdown
---
on: push
permissions:
  actions: read       # Main job only needs minimal permissions
engine: claude
output:
  pull-request:
    title-prefix: "[bot] "
    labels: [automation, ai-generated]
---

# Code Improvement Agent

Analyze the latest commit and suggest improvements.

1. Make any file changes directly in the working directory
2. Write a PR title and description to ${{ env.GITHUB_AW_OUTPUT }}

The workflow will automatically:
- Stage your changes with `git add -A`
- Commit them as "[agent] staged files"  
- Generate patches with `git format-patch`
- Create a pull request with your changes

Example output format for ${{ env.GITHUB_AW_OUTPUT }}:
```
Fix coding style issues

- Updated variable naming conventions
- Fixed indentation in helper functions
- Added missing documentation
```

**Automatic Patch Generation:**
The workflow automatically handles patch creation—your agent simply makes file changes, and the system:
1. Stages changes with `git add -A`
2. Commits them as "[agent] staged files"
3. Generates git patches using `git format-patch`
4. Validates patch existence and content before proceeding with PR creation

## Label Addition (`output.labels`)

Adding `output.labels` to your workflow declares that the workflow should conclude with adding labels to the current issue or pull request based on the agent's analysis.

**How Your Agent Provides Output:**
Your agentic workflow writes labels to add to `${{ env.GITHUB_AW_OUTPUT }}`, one label per line:
```
triage
bug
needs-review
```

**What This Configuration Does:**
When you add `output.labels` to your frontmatter, the compiler automatically generates a separate `add_labels` job that:
- Only runs when triggered by an issue or pull request event
- Reads the labels from `${{ env.GITHUB_AW_OUTPUT }}` (one per line)
- Validates each label against the mandatory `allowed` list
- Enforces the `max-count` limit (default: 3 labels)
- Adds only valid labels to the current issue or pull request
- **Security**: Only label addition is supported—no removal operations are allowed
- **Validation**: The job fails if any requested label is not in the `allowed` list

**Configuration:**
```yaml
output:
  labels:
    allowed: [triage, bug, enhancement]  # Mandatory: list of allowed labels (must be non-empty)
    max-count: 3                         # Optional: maximum number of labels to add (default: 3)
```

**Agent Output Format:**
The agent should write labels to add, one per line, to the `${{ env.GITHUB_AW_OUTPUT }}` file:
```
triage
bug
needs-review
```

**Safety Features:**
- Empty lines in agent output are ignored
- Lines starting with `-` are rejected (no removal operations allowed)
- Duplicate labels are automatically removed
- All requested labels must be in the `allowed` list or the job fails with a clear error message
- Label count is limited by `max-count` setting (default: 3) - exceeding this limit causes job failure
- Only GitHub's `issues.addLabels` API endpoint is used (no removal endpoints)

**Example workflow using label addition:**
```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read       # Main job only needs minimal permissions
engine: claude
output:
  labels:
    allowed: [triage, bug, enhancement, documentation, needs-review]
---

# Issue Labeling Agent

Analyze the issue content and determine appropriate labels.

Write the labels you want to add (one per line) to ${{ env.GITHUB_AW_OUTPUT }}.

Only use labels from the allowed list: triage, bug, enhancement, documentation, needs-review.
```

## Related Documentation

- [Frontmatter Options](frontmatter.md) - All configuration options for workflows
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Alias Triggers](alias-triggers.md) - Special @mention triggers and context text
- [Commands](commands.md) - CLI commands for workflow management
