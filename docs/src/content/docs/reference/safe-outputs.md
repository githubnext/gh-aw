---
title: Safe Outputs
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 800
---

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

## Overview

The `safe-outputs:` element of your workflow's frontmatter declares that your agentic workflow should conclude with optional automated actions based on the agentic workflow's output. This enables your workflow to write content that is then automatically processed to create GitHub issues, comments, pull requests, or add labels—all without giving the agentic portion of the workflow any write permissions.

**How It Works:**
1. The agentic part of your workflow runs with minimal read-only permissions. It is given additional prompting to write its output to the special known files
2. The compiler automatically generates additional jobs that read this output and perform the requested actions
3. Only these generated jobs receive the necessary write permissions

For example:

```yaml
safe-outputs:
  create-issue:
```

This declares that the workflow should create at most one new issue.

## Available Safe Output Types

| Output Type | Configuration Key | Description | Default Max | Cross-Repository |
|-------------|------------------|-------------|-------------|------------------|
| **Create Issue** | `create-issue:` | Create GitHub issues based on workflow output | 1 | ✅ |
| **Add Comments** | `add-comment:` | Post comments on issues, pull requests, or discussions | 1 | ✅ |
| **Update Issues** | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| **Add Issue Label** | `add-labels:` | Add labels to issues or pull requests | 3 | ✅ |
| **Create Pull Request** | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| **Pull Request Review Comments** | `create-pull-request-review-comment:` | Create review comments on specific lines of code | 1 | ✅ |
| **Create Discussions** | `create-discussion:` | Create GitHub discussions based on workflow output | 1 | ✅ |
| **Push to Pull Request Branch** | `push-to-pull-request-branch:` | Push changes directly to a branch | 1 | ❌ |
| **Create Code Scanning Alerts** | `create-code-scanning-alert:` | Generate SARIF repository security advisories and upload to GitHub Code Scanning | unlimited | ❌ |
| **Missing Tool Reporting** | `missing-tool:` | Report missing tools or functionality (enabled by default when safe-outputs is configured) | unlimited | ❌ |

Custom safe output types can be defined through [Custom Safe Output Jobs](/gh-aw/guides/custom-safe-outputs/).

### New Issue Creation (`create-issue:`)

Adding issue creation to the `safe-outputs:` section declares that the workflow should conclude with the creation of GitHub issues based on the workflow's output.

**Basic Configuration:**
```yaml
safe-outputs:
  create-issue:
```

**With Configuration:**
```yaml
safe-outputs:
  create-issue:
    title-prefix: "[ai] "            # Optional: prefix for issue titles
    labels: [automation, agentic]    # Optional: labels to attach to issues
    assignees: [user1, user2, copilot] # Optional: users/bots to assign the issue to
    max: 5                           # Optional: maximum number of issues (default: 1)
    target-repo: "owner/target-repo" # Optional: create issues in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should describe the issue(s) it wants created.

**Configuration Options:**
- **`assignees:`** - GitHub username(s) to automatically assign to created issues. Accepts either a single string (`assignees: user1`) or an array of strings (`assignees: [user1, user2]`). The workflow automatically adds steps that call `gh issue edit --add-assignee` for each assignee after the issue is created. Only runs if the issue was successfully created.
  - **Special value**: Use `copilot` to assign to the Copilot bot
  - Uses the configured GitHub token (respects `github-token` precedence: create-issue config > safe-outputs config > top-level config > default)

:::note
To assign issues to bots (including `copilot`), you must use a Personal Access Token (PAT) with appropriate permissions. The default `GITHUB_TOKEN` does not have permission to assign issues to bots. Configure a PAT using the `github-token` field at the workflow, safe-outputs, or create-issue level.
:::

**Example markdown to generate the output:**

```yaml
# Code Analysis Agent

Analyze the latest commit and provide insights.
Create new issues with your findings. For each issue, provide a title starting with "AI Code Analysis" and detailed description of the analysis findings.
```

The compiled workflow will have additional prompting describing that, to create issues, it should write the issue details to a file.

### Comment Creation (`add-comment:`)

Adding comment creation to the `safe-outputs:` section declares that the workflow should conclude with posting comments based on the workflow's output. By default, comments are posted on the triggering issue or pull request, but this can be configured to target discussions or specific issues/PRs using the `target` and `discussion` options.

**Basic Configuration:**
```yaml
safe-outputs:
  add-comment:
```

**With Configuration:**
```yaml
safe-outputs:
  add-comment:
    max: 3                          # Optional: maximum number of comments (default: 1)
    target: "*"                     # Optional: target for comments
                                    # "triggering" (default) - only comment on triggering issue/PR
                                    # "*" - allow comments on any issue (requires target number in agent output)
                                    # explicit number - comment on specific issue number
    discussion: true                # Optional: target discussions instead of issues/PRs (must be true if present)
    target-repo: "owner/target-repo" # Optional: create comments in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should describe the comment(s) it wants posted.

### Automatic Cross-Referencing Between Safe Outputs

When `add-comment` is used together with `create-issue`, `create-discussion`, or `create-pull-request` in the same workflow, the comment automatically includes a "Related Items" section with links to the created items.

**Example natural language to generate the output:**

```aw wrap
---
on:
  issues:
    types: [opened, edited]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  add-comment:
    max: 3
---

# Issue/PR Analysis Agent

Analyze the issue or pull request and provide feedback.
Create issue comments on the triggering issue or PR with your analysis findings. Each comment should provide specific insights about different aspects of the issue.
```

The compiled workflow will have additional prompting describing that, to create comments, it should write the comment content to a special file.

### Add Issue Label (`add-labels:`)

Adding `add-labels:` to the `safe-outputs:` section of your workflow declares that the workflow should conclude with adding labels to issues or pull requests based on the coding agent's analysis. By default, labels are added to the triggering issue, pull request or discussion, but this can be configured using the `target` option.

**Basic Configuration:**
```yaml
safe-outputs:
  add-labels:
```

**With Configuration:**
```yaml
safe-outputs:
  add-labels:
    allowed: [triage, bug, enhancement] # Optional: allowed labels for addition
    max: 3                              # Optional: maximum number of labels to add (default: 3)
    target: "*"                         # Optional: target for labels
                                        # "triggering" (default) - only add labels to triggering issue/PR
                                        # "*" - allow labels on any issue (requires issue_number in agent output)
                                        # Explicit number - add labels to specific issue/PR (e.g., "123")
    target-repo: "owner/target-repo"    # Optional: add labels to issues/PRs in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should analyze the issue content and determine appropriate labels. 

**Example of natural language to generate the output:**

```aw wrap
# Issue Labeling Agent

Analyze the issue content and add appropriate labels to the issue.
```

The agentic part of your workflow will have implicit additional prompting saying that, to add labels to a GitHub issue, you must write labels to a special file, one label per line.

### Issue Updates (`update-issue:`)

Adding `update-issue:` to the `safe-outputs:` section declares that the workflow should conclude with updating GitHub issues based on the coding agent's analysis. By default, updates are applied to the triggering issue, but this can be configured using the `target` option. You can also configure which fields are allowed to be updated.

**Basic Configuration:**
```yaml
safe-outputs:
  update-issue:
```

**With Configuration:**
```yaml
safe-outputs:
  update-issue:
    status:                             # Optional: presence indicates status can be updated (open/closed)
    target: "*"                         # Optional: target for updates
                                        # "triggering" (default) - only update triggering issue
                                        # "*" - allow updates to any issue (requires issue_number in agent output)
                                        # explicit number - update specific issue number
    title:                              # Optional: presence indicates title can be updated
    body:                               # Optional: presence indicates body can be updated
    max: 3                              # Optional: maximum number of issues to update (default: 1)
    target-repo: "owner/target-repo"    # Optional: update issues in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should analyze the issue and determine what updates to make.

**Example natural language to generate the output:**

```aw wrap
---
on:
  issues:
    types: [opened, edited]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  update-issue:
    status: true
    title: true
    body: true
---

# Issue Update Agent

Analyze the issue and update its status, title, or body as needed.
Update the issue based on your analysis. You can change the title, body content, or status (open/closed).
```

**Safety Features:**

- Only explicitly enabled fields (`status`, `title`, `body`) can be updated
- Status values are validated (must be "open" or "closed")
- Empty or invalid field values are rejected
- Target configuration controls which issues can be updated for security
- Update count is limited by `max` setting (default: 1)
- Only GitHub's `issues.update` API endpoint is used

### Pull Request Creation (`create-pull-request:`)

Adding pull request creation to the `safe-outputs:` section declares that the workflow should conclude with the creation of a pull request containing code changes generated by the workflow. If pull request creation fails (e.g., due to organization settings that block PR creation), the system will automatically fall back to creating an issue instead.

> [!NOTE]
> Pull request creation may fail in organizations where the setting "Allow GitHub Actions to create and approve pull requests" is disabled. This setting can be found in your organization's Settings → Actions → General → Workflow permissions. Organization administrators can enable this to allow agentic workflows to create pull requests directly.

```yaml
safe-outputs:
  create-pull-request:
```

**With Configuration:**
```yaml
safe-outputs:
  create-pull-request:               # Creates exactly one pull request
    title-prefix: "[ai] "            # Optional: prefix for PR titles
    labels: [automation, agentic]    # Optional: labels to attach to PRs
    reviewers: [user1, user2, copilot] # Optional: users/bots to assign as reviewers to the pull request
    draft: true                      # Optional: create as draft PR (defaults to true)
    if-no-changes: "warn"            # Optional: behavior when no changes to commit (defaults to "warn")
    target-repo: "owner/target-repo" # Optional: create PR in a different repository (requires github-token with appropriate permissions)
```

**Configuration Options:**
- **`reviewers:`** - GitHub username(s) to automatically assign as reviewers to the created pull request. Accepts either a single string (`reviewers: user1`) or an array of strings (`reviewers: [user1, user2]`). The workflow automatically adds steps that call `gh pr edit --add-reviewer` for each reviewer after the pull request is created. Only runs if the pull request was successfully created.
  - **Special value**: Use `copilot` to assign the Copilot bot as a reviewer
  - Uses the configured GitHub token (respects `github-token` precedence: create-pull-request config > safe-outputs config > top-level config > default)

:::note
To add bots as reviewers (including `copilot`), you must use a Personal Access Token (PAT) with appropriate permissions. The default `GITHUB_TOKEN` does not have permission to add bots as reviewers. Configure a PAT using the `github-token` field at the workflow, safe-outputs, or create-pull-request level.
:::

**Fallback Behavior:**

When pull request creation fails (common in organizations where "Allow GitHub Actions to create and approve pull requests" is disabled), the system automatically:

1. **Creates an issue instead** with the same title, description, and labels as the intended PR
2. **Adds branch information** to the issue body with a link to the created branch
3. **Includes the original error** in the issue for debugging
4. **Sets fallback outputs** including `fallback_used: "true"` and issue-related outputs

**Available Outputs:**

- **On successful PR creation:**
  - `pull_request_number`: The PR number
  - `pull_request_url`: The PR URL
  - `branch_name`: The created branch name

- **On fallback to issue:**
  - `issue_number`: The fallback issue number
  - `issue_url`: The fallback issue URL  
  - `branch_name`: The created branch name
  - `fallback_used`: Set to `"true"` when fallback was used

**`if-no-changes` Configuration Options:**
- **`"warn"` (default)**: Logs a warning message but the workflow succeeds
- **`"error"`**: Fails the workflow with an error message if no changes are detected
- **`"ignore"`**: Silent success with no console output when no changes are detected

**Examples:**
```yaml
# Default behavior - warn but succeed when no changes
safe-outputs:
  create-pull-request:
    if-no-changes: "warn"
```

```yaml
# Strict mode - fail if no changes to commit
safe-outputs:
  create-pull-request:
    if-no-changes: "error"
```

```yaml
# Silent mode - no output on empty changesets
safe-outputs:
  create-pull-request:
    if-no-changes: "ignore"
```

At most one pull request is currently supported.

The agentic part of your workflow should instruct to:
1. **Make code changes**: Make code changes and commit them to a branch 
2. **Create pull request**: Describe the pull request title and body content you want

**Example natural language to generate the output:**

```aw wrap
---
on:
  push:
    branches: [main]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "
    labels: [automation, code-improvement]
    draft: true
---

# Code Improvement Agent

Analyze the latest commit and suggest improvements.

1. Make any file changes directly in the working directory
2. Create a pull request for your improvements, with a descriptive title and detailed description of the changes made
```

**Troubleshooting Pull Request Creation:**

If your workflow consistently falls back to creating issues instead of pull requests, check:

1. **Organization Settings** (requires admin privileges): Navigate to Organization Settings → Actions → General, then scroll to the 'Workflow permissions' section
2. **Enable PR Creation**: Ensure "Allow GitHub Actions to create and approve pull requests" is enabled
3. **Repository Settings**: Repository admins can check Repository Settings → Actions → General → Workflow permissions. Note that repository settings can only be more restrictive than organization defaults, not more permissive
4. **Branch Protection**: Some branch protection rules may prevent automated PR creation

When fallback occurs, the created issue will contain:
- The intended PR title and description
- A link to the branch with your changes
- The specific error message for debugging
- Instructions for manually creating the PR if needed

### Pull Request Review Comment Creation (`create-pull-request-review-comment:`)

Adding `create-pull-request-review-comment:` to the `safe-outputs:` section declares that the workflow should conclude with creating review comments on specific lines of code in the current pull request based on the workflow's output.

**Basic Configuration:**
```yaml
safe-outputs:
  create-pull-request-review-comment:
```

**With Configuration:**
```yaml
safe-outputs:
  create-pull-request-review-comment:
    max: 3                          # Optional: maximum number of review comments (default: 1)
    side: "RIGHT"                   # Optional: side of the diff ("LEFT" or "RIGHT", default: "RIGHT")
    target: "*"                     # Optional: target for review comments
                                    # "triggering" (default) - only comment on triggering PR
                                    # "*" - allow comments on any PR (requires pull_request_number in agent output)
                                    # explicit number - comment on specific PR number
    target-repo: "owner/target-repo" # Optional: create review comments in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should describe the review comment(s) it wants created with specific file paths and line numbers.

**Example natural language to generate the output:**

```aw wrap
---
on:
  pull_request:
    types: [opened, edited, synchronize]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-pull-request-review-comment:
    max: 3
    side: "RIGHT"
---

# Code Review Agent

Analyze the pull request changes and provide line-specific feedback.
Create review comments on the pull request with your analysis findings. For each comment, specify:
- The file path
- The line number (required)
- The start line number (optional, for multi-line comments)
- The comment body with specific feedback

Review comments can target single lines or ranges of lines in the diff.
```

The compiled workflow will have additional prompting describing that, to create review comments, it should write the comment details to a special file with the following structure:
- `path`: The file path relative to the repository root
- `line`: The line number where the comment should be placed
- `start_line`: (Optional) The starting line number for multi-line comments
- `side`: (Optional) The side of the diff ("LEFT" for old version, "RIGHT" for new version)
- `pull_request_number`: (Optional) The PR number when using `target: "*"` to comment on any PR
- `body`: The comment content

**Key Features:**
- Only works in pull request contexts by default (when `target` is not specified)
- With `target: "*"`, can comment on any PR by including `pull_request_number` in the output
- With explicit `target` number, comments on that specific PR regardless of triggering event
- Supports both single-line and multi-line code comments
- Comments are automatically positioned on the correct side of the diff
- Maximum comment limits prevent spam

### Code Scanning Alert Creation (`create-code-scanning-alert:`)

Adding `create-code-scanning-alert:` to the `safe-outputs:` section declares that the workflow should conclude with creating repository security advisories in SARIF format based on the workflow's security analysis findings. The SARIF file is uploaded as an artifact and submitted to GitHub Code Scanning.

**Basic Configuration:**
```yaml
safe-outputs:
  create-code-scanning-alert:
```

**With Configuration:**
```yaml
safe-outputs:
  create-code-scanning-alert:
    max: 50                         # Optional: maximum number of security findings (default: unlimited)
```

The agentic part of your workflow should describe the security findings it wants reported with specific file paths, line numbers, severity levels, and descriptions.

**Example natural language to generate the output:**

```aw wrap
# Security Analysis Agent

Analyze the codebase for security vulnerabilities and create repository security advisories.
Create repository security advisories with your analysis findings. For each security finding, specify:
- The file path relative to the repository root
- The line number where the issue occurs
- The severity level (error, warning, info, or note)
- A detailed description of the security issue

Security findings will be formatted as SARIF and uploaded to GitHub Code Scanning.
```

The compiled workflow will have additional prompting describing that, to create repository security advisories, it should write the security findings to a special file with the following structure:
- `file`: The file path relative to the repository root
- `line`: The line number where the security issue occurs
- `column`: Optional column number where the security issue occurs (defaults to 1)
- `severity`: The severity level ("error", "warning", "info", or "note")
- `message`: The detailed description of the security issue
- `ruleIdSuffix`: Optional custom suffix for the SARIF rule ID (must contain only alphanumeric characters, hyphens, and underscores)

**Key Features:**
- Generates SARIF (Static Analysis Results Interchange Format) reports
- Automatically uploads reports as GitHub Actions artifacts
- Integrates with GitHub Code Scanning for security dashboard visibility
- Supports standard severity levels (error, warning, info, note)
- Works in any workflow context (not limited to pull requests)
- Maximum findings limit prevents overwhelming reports
- Validates all required fields before generating SARIF
- Supports optional column specification for precise location
- Customizable rule IDs via optional ruleIdSuffix field
- Rule IDs default to `{workflow-filename}-security-finding-{index}` format when no custom suffix is provided

### Push to Pull Request Branch (`push-to-pull-request-branch:`)

Adding `push-to-pull-request-branch:` to the `safe-outputs:` section declares that the workflow should conclude with pushing additional changes to the branch associated with a pull request. This is useful for applying code changes directly to a designated branch within pull requests.

**Basic Configuration:**
```yaml
safe-outputs:
  push-to-pull-request-branch:
```

**With Configuration:**
```yaml
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                          # Optional: target for push operations
                                         # "triggering" (default) - only push in triggering PR context
                                         # "*" - allow pushes to any pull request (requires pull_request_number in agent output)
                                         # explicit number - push for specific pull request number
    title-prefix: "[bot] "               # Optional: required title prefix for pull request validation
                                         # Only pull requests with this prefix will be accepted
    labels: [automated, enhancement]     # Optional: required labels for pull request validation
                                         # Only pull requests with all these labels will be accepted
    if-no-changes: "warn"                # Optional: behavior when no changes to push
                                         # "warn" (default) - log warning but succeed
                                         # "error" - fail the action
                                         # "ignore" - silent success
```

**Pull Request Validation:**

When `title-prefix` or `labels` are specified, the workflow will validate that the target pull request meets these requirements before pushing changes:

- **Title Prefix Validation**: Checks that the PR title starts with the specified prefix
- **Labels Validation**: Ensures the PR contains all required labels (additional labels are allowed)
- **Validation Failure**: If validation fails, the workflow stops with a clear error message showing current vs expected values

The agentic part of your workflow should describe the changes to be pushed and optionally provide a commit message.

**Example natural language to generate the output:**

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  push-to-pull-request-branch:
    target: "triggering"
    if-no-changes: "warn"
---

# Code Update Agent

Analyze the pull request and make necessary code improvements.

1. Make any file changes directly in the working directory  
2. Push changes to the feature branch with a descriptive commit message
```

**Examples with different error level configurations:**

```yaml
# Always succeed, warn when no changes (default behavior)
safe-outputs:
  push-to-pull-request-branch:
    branch: feature-branch
    if-no-changes: "warn"
```

```yaml
# Fail when no changes are made (strict mode)
safe-outputs:
  push-to-pull-request-branch:
    branch: feature-branch
    if-no-changes: "error"
```

```yaml
# Silent success, no output when no changes
safe-outputs:
  push-to-pull-request-branch:
    branch: feature-branch
    if-no-changes: "ignore"
```

**Safety Features:**

- Changes are applied via git patches generated from the workflow's modifications
- Only the specified branch can be modified
- Target configuration controls which pull requests can trigger pushes for security
- **Pull Request Validation**: Optional `title-prefix` and `labels` validation ensures only approved PRs receive changes
- **Fail-Fast Validation**: Validation occurs before any changes are applied, preventing partial modifications
- Push operations are limited to one per workflow execution
- Configurable error handling for empty changesets via `if-no-changes` option

**Error Level Configuration:**

Similar to GitHub's `actions/upload-artifact` action, you can configure how the action behaves when there are no changes to push:

- **`warn` (default)**: Logs a warning message but the workflow succeeds. This is the recommended setting for most use cases.
- **`error`**: Fails the workflow with an error message when no changes are detected. Useful when you always expect changes to be made.
- **`ignore`**: Silent success with no console output. The workflow completes successfully but quietly.

**Safety Features:**

- Empty lines in coding agent output are ignored
- Lines starting with `-` are rejected (no removal operations allowed)
- Duplicate labels are automatically removed
- If `allowed` is provided, all requested labels must be in the `allowed` list or the job fails with a clear error message. If `allowed` is not provided then any labels are allowed (including creating new labels).
- Label count is limited by `max` setting (default: 3) - exceeding this limit causes job failure
- Only GitHub's `issues.addLabels` API endpoint is used (no removal endpoints)

When `create-pull-request` or `push-to-pull-request-branch` are enabled in the `safe-outputs` configuration, the system automatically adds the following additional Claude tools to enable file editing and pull request creation:

### Missing Tool Reporting (`missing-tool:`)

**Note:** Missing tool reporting is **enabled by default** whenever `safe-outputs:` is configured. This helps identify tools that weren't available or lacked proper permissions during workflow execution.

**Basic Configuration (enabled by default):**
```yaml
safe-outputs:
  create-issue:    # Any safe-output configuration enables missing-tool by default
```

**Explicitly Disable:**
```yaml
safe-outputs:
  create-issue:
  missing-tool: false    # Explicitly disable missing-tool reporting
```

**With Configuration:**
```yaml
safe-outputs:
  missing-tool:
    max: 10                             # Optional: maximum number of missing tool reports (default: unlimited)
```

The agentic part of your workflow can report missing tools or functionality that prevents it from completing its task. Additionally, the system **automatically detects and reports tools that failed due to permission errors**, ensuring visibility into authorization-related issues.

**Example natural language to generate the output:**

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  create-issue:    # missing-tool is enabled by default
---

# Development Task Agent

Analyze the repository and implement the requested feature. If you encounter missing tools, capabilities, or permissions that prevent completion, they will be automatically reported so the user can address these limitations.
```

The compiled workflow will have additional prompting describing that, to report missing tools, it should write the tool information to a special file.

**Automatic Permission Error Detection:**

The workflow engine automatically scans execution logs for permission-related errors and creates missing-tool entries for:
- Tools that were attempted but lacked required permissions
- API calls that failed due to insufficient authorization
- Operations blocked by repository access controls

This helps identify configuration issues without requiring manual reporting.

**Safety Features:**

- No write permissions required - only logs missing functionality
- Optional configuration to help users understand workflow limitations when enabled  
- Reports are structured with tool name, reason, and optional alternatives
- Maximum count can be configured to prevent excessive reporting
- All missing tool data is captured in workflow artifacts for review

### New Discussion Creation (`create-discussion:`)

Adding discussion creation to the `safe-outputs:` section declares that the workflow should conclude with the creation of GitHub discussions based on the workflow's output.

**Basic Configuration:**
```yaml
safe-outputs:
  create-discussion:
```

**With Configuration:**
```yaml
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "            # Optional: prefix for discussion titles
    category: "general"               # Optional: discussion category slug, name, or ID (e.g., "General" or "DIC_kwDOGFsHUM4BsUn3"). Use repository discussions settings to find valid categories.
    max: 3                           # Optional: maximum number of discussions (default: 1)
    target-repo: "owner/target-repo" # Optional: create discussions in a different repository (requires github-token with appropriate permissions)
```

The agentic part of your workflow should describe the discussion(s) it wants created.

**Example markdown to generate the output:**

```yaml
# Research Discussion Agent

Research the latest developments in AI and create discussions to share findings.
Create new discussions with your research findings. For each discussion, provide a title starting with "AI Research Update" and detailed summary of the findings.
```

The compiled workflow will have additional prompting describing that, to create discussions, it should write the discussion details to a file.

**Note:** The `category` field accepts a category slug (e.g., `"general"`), category name (e.g., `"General"`), or category ID (e.g., `"DIC_kwDOGFsHUM4BsUn3"`). The workflow will first try to match against category IDs, then against category names, and finally against category slugs. If no `category` is specified, the workflow will use the first available discussion category in the repository. To find valid category values, visit your repository's discussions settings page, open an existing discussion to see the category in the URL, or query via the GitHub GraphQL API.

## Cross-Repository Operations

Many safe output types support the `target-repo` configuration for cross-repository operations. This enables workflows to create issues, comments, pull requests, and other outputs in repositories other than where the workflow is running.

### Authentication Requirements

Cross-repository operations require proper authentication:

1. **Default Token Limitations**: The standard `GITHUB_TOKEN` only has permissions for the repository where the workflow runs
2. **Personal Access Token Required**: Use a Personal Access Token (PAT) or GitHub App token with access to target repositories
3. **Token Configuration**: Configure the token using the `github-token` field or `GH_AW_GITHUB_TOKEN` environment variable

### Example: Multi-Repository Issue Management

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: claude
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}  # PAT with access to target repositories
  create-issue:
    target-repo: "organization/tracking-repo"
    title-prefix: "[Cross-Repo] "
    labels: [automation, cross-repo]
  add-comment:
    target-repo: "organization/notifications-repo" 
    target: "123"  # Comment on issue #123
  add-labels:
    target-repo: "organization/metrics-repo"
    allowed: [processed, analyzed]
---

# Multi-Repository Issue Processor

When an issue is opened in this repository:
1. Create a tracking issue in the organization's tracking repository
2. Add a notification comment to issue #123 in the notifications repository  
3. Add processed labels to related issues in the metrics repository

Analyze the issue content and determine appropriate actions for each target repository.
```

### Security Considerations

- **Repository Access**: Ensure the authentication token has appropriate permissions for target repositories
- **Explicit Targets**: Use specific repository names - wildcard patterns are not supported for security
- **Least Privilege**: Grant tokens only the minimum permissions needed for the intended operations
- **Token Scope**: Personal Access Tokens should be scoped to specific repositories when possible

### Error Handling

If cross-repository operations fail due to authentication or permission issues:
- The workflow job will fail with a clear error message
- Error details include the target repository and specific permission requirements
- In staged mode, errors are shown as preview issues instead of failing the workflow

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, these Claude tools are automatically added:

- **Edit**: Allows editing existing files
- **MultiEdit**: Allows making multiple edits to files in a single operation
- **Write**: Allows creating new files or overwriting existing files
- **NotebookEdit**: Allows editing Jupyter notebook files

Along with the file editing tools, these Git commands are also automatically allowed:

- `git checkout:*`
- `git branch:*`
- `git switch:*`
- `git add:*`
- `git rm:*`
- `git commit:*`
- `git merge:*`

## Security and Sanitization

All coding agent output is automatically sanitized for security before being processed:

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

**Configuration:**

```yaml
safe-outputs:
  allowed-domains:                    # Optional: domains allowed in coding agent output URIs
    - github.com                      # Default GitHub domains are always included
    - api.github.com                  # Additional trusted domains can be specified
    - trusted-domain.com              # URIs from unlisted domains are replaced with "(redacted)"
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

GitHub Agentic Workflows uses a token precedence system for authentication:

1. **`GH_AW_GITHUB_TOKEN`** - Override token (highest priority)
2. **`GITHUB_TOKEN`** - Standard GitHub Actions token (fallback)

By default, safe output jobs automatically use this precedence: `${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`. You can override this by specifying a custom GitHub token for all safe output jobs:

```yaml
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # Use custom PAT instead of default precedence
  create-issue:
  add-comment:
```

The token precedence system is useful when:
- **Trial mode execution**: `GH_AW_GITHUB_TOKEN` can be set to test workflows safely
- **Enhanced permissions**: Override with Personal Access Tokens that have broader scope
- **Cross-repository operations**: Use tokens with access to multiple repositories via `target-repo` configuration
- **Custom authentication flows**: Implement specialized token management strategies

**Cross-Repository Example:**
```yaml
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}  # PAT with multi-repo access
  create-issue:
    target-repo: "owner/project-tracker"
  add-comment:
    target-repo: "owner/notifications"
```

This is useful when:
- You need additional permissions beyond what `GITHUB_TOKEN` provides
- You want to perform actions across multiple repositories
- You need to bypass GitHub Actions token restrictions

The custom `github-token` can also be appleid to a specific output. This is most useful if GitHub organization or repository policy prevents GitHub Actions workflows from creating pull requests, and you only want to use the PAT for that specific case.

```yaml
safe-outputs:
  create-issue:
  add-comment:
  create-pull-request:
    github-token: ${{ secrets.CUSTOM_PAT }}  # Use custom PAT instead of GITHUB_TOKEN
```

### Maximum Patch Size (`max-patch-size:`)

When using `create-pull-request` or `push-to-pull-request-branch`, you can configure the maximum allowed size for git patches to prevent workflow failures from excessively large patches:

```yaml
safe-outputs:
  max-patch-size: 512                   # Optional: maximum patch size in KB (default: 1024)
  create-pull-request:
  push-to-pull-request-branch:
```

**Configuration Options:**
- **Range**: 1 KB to 10,240 KB (10 MB)
- **Default**: 1024 KB (1 MB) when not specified
- **Behavior**: If a patch exceeds the configured size, the job fails with a clear error message

**Example with different size limits:**

```yaml
# Small patches only (256 KB limit)
safe-outputs:
  max-patch-size: 256
  create-pull-request:
    title-prefix: "[SMALL] "

# Large patches allowed (5 MB limit)  
safe-outputs:
  max-patch-size: 5120
  push-to-pull-request-branch:
    if-no-changes: "error"
```

**Error Handling:**
When a patch exceeds the limit, you'll see an error like:
```
Patch size (2048 KB) exceeds maximum allowed size (512 KB)
```

In staged mode, this shows as a preview error rather than failing the workflow.

**Use Cases:**
- Prevent repository bloat from large automated changes
- Avoid GitHub API limits and timeouts
- Ensure manageable code review sizes
- Control CI/CD resource usage

## Assigning Issues and Pull Requests to Copilot

Both `create-issue` and `create-pull-request` safe outputs support assigning the created issue or adding reviewers to the pull request using the special value `copilot`. This provides automated code review and issue analysis.

### Assigning Issues to Copilot

Use the `assignees` field in `create-issue` configuration to automatically assign created issues to the Copilot bot:

```yaml
safe-outputs:
  create-issue:
    assignees: copilot  # Assigns to the Copilot bot
    # Or with multiple assignees:
    # assignees: [user1, copilot, user2]
```

### Adding Copilot as Reviewer to Pull Requests

Use the `reviewers` field in `create-pull-request` configuration to automatically add the Copilot bot as a reviewer:

```yaml
safe-outputs:
  create-pull-request:
    reviewers: copilot  # Adds the Copilot bot as reviewer
    # Or with multiple reviewers:
    # reviewers: [user1, copilot, user2]
```

### Authentication Requirements

:::caution
To assign issues to bots or add bots as reviewers (including `copilot`), you must use a **Personal Access Token (PAT)** with appropriate permissions. The default `GITHUB_TOKEN` does not have permission to assign issues to bots or add bots as reviewers.
:::

Configure a PAT using the `github-token` field at any of these levels (in order of precedence):
1. Specific safe output level (`create-issue` or `create-pull-request`)
2. Safe outputs section level
3. Top-level workflow configuration

**Example with custom token:**

```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  github-token: ${{ secrets.COPILOT_PAT }}  # PAT with permissions to add bot assignees/reviewers
  create-issue:
    assignees: copilot
  create-pull-request:
    reviewers: copilot
---

# AI Issue and PR Handler

Analyze issues and create follow-up items with Copilot assigned for automated assistance.
```

## Custom runner image

You can specify the `runs-on` field in the `safe-outputs:` section to use a custom GitHub Actions runner image for all safe output jobs.

```yaml
safe-outputs:
  runs-on: ubuntu-22.04                # Optional: custom runner image for all safe output jobs
  create-issue:
  add-comment:
```

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
