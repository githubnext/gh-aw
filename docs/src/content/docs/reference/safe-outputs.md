---
title: Safe Output Processing
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 5
---

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

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

| Output Type | Configuration Key | Description | Default Max |
|-------------|------------------|-------------|-------------|
| **Create Issue** | `create-issue:` | Create GitHub issues based on workflow output | 1 |
| **Add Issue Comments** | `add-comment:` | Post comments on issues or pull requests | 1 |
| **Update Issues** | `update-issue:` | Update issue status, title, or body | 1 |
| **Add Issue Label** | `add-labels:` | Add labels to issues or pull requests | 3 |
| **Create Pull Request** | `create-pull-request:` | Create pull requests with code changes | 1 |
| **Pull Request Review Comments** | `create-pull-request-review-comment:` | Create review comments on specific lines of code | 1 |
| **Push to Pull Request Branch** | `push-to-pr-branch:` | Push changes directly to a branch | 1 |
| **Create Code Scanning Alerts** | `create-code-scanning-alert:` | Generate SARIF repository security advisories and upload to GitHub Code Scanning | unlimited |
| **Missing Tool Reporting** | `missing-tool:` | Report missing tools or functionality needed to complete tasks | unlimited |

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
    max: 5                           # Optional: maximum number of issues (default: 1)
```

The agentic part of your workflow should describe the issue(s) it wants created.

**Example markdown to generate the output:**

```yaml
# Code Analysis Agent

Analyze the latest commit and provide insights.
Create new issues with your findings. For each issue, provide a title starting with "AI Code Analysis" and detailed description of the analysis findings.
```

The compiled workflow will have additional prompting describing that, to create issues, it should write the issue details to a file.

### Issue Comment Creation (`add-comment:`)

Adding comment creation to the `safe-outputs:` section declares that the workflow should conclude with posting comments based on the workflow's output. By default, comments are posted on the triggering issue or pull request, but this can be configured using the `target` option.

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
                                    # "*" - allow comments on any issue (requires issue_number in agent output)
                                    # explicit number - comment on specific issue number
```

The agentic part of your workflow should describe the comment(s) it wants posted.

**Example natural language to generate the output:**

```markdown
# Issue/PR Analysis Agent

Analyze the issue or pull request and provide feedback.
Create issue comments on the triggering issue or PR with your analysis findings. Each comment should provide specific insights about different aspects of the issue.
```

The compiled workflow will have additional prompting describing that, to create comments, it should write the comment content to a special file.

### Add Issue Label (`add-labels:`)

Adding `add-labels:` to the `safe-outputs:` section of your workflow declares that the workflow should conclude with adding labels to the current issue or pull request based on the coding agent's analysis.

```yaml
safe-outputs:
  add-labels:
```

or with further configuration:

```yaml
safe-outputs:
  add-labels:
    allowed: [triage, bug, enhancement] # Optional: allowed labels for addition.
    max: 3                              # Optional: maximum number of labels to add (default: 3)
```

The agentic part of your workflow should analyze the issue content and determine appropriate labels. 

**Example of natural language to generate the output:**

```markdown
# Issue Labeling Agent

Analyze the issue content and add appropriate labels to the issue.
```

The agentic part of your workflow will have implicit additional prompting saying that, to add labels to a GitHub issue, you must write labels to a special file, one label per line.

### Issue Updates (`update-issue:`)

Adding `update-issue:` to the `safe-outputs:` section declares that the workflow should conclude with updating GitHub issues based on the coding agent's analysis. You can configure which fields are allowed to be updated.

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
```

The agentic part of your workflow should analyze the issue and determine what updates to make.

**Example natural language to generate the output:**

```markdown
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

Adding pull request creation to the `safe-outputs:` section declares that the workflow should conclude with the creation of a pull request containing code changes generated by the workflow.

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
    draft: true                      # Optional: create as draft PR (defaults to true)
    if-no-changes: "warn"            # Optional: behavior when no changes to commit (defaults to "warn")
```

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

```markdown
# Code Improvement Agent

Analyze the latest commit and suggest improvements.

1. Make any file changes directly in the working directory
2. Create a pull request for your improvements, with a descriptive title and detailed description of the changes made
```

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
```

The agentic part of your workflow should describe the review comment(s) it wants created with specific file paths and line numbers.

**Example natural language to generate the output:**

```markdown
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
- `body`: The comment content

**Key Features:**
- Only works in pull request contexts for security
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

```markdown
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

### Push to Pull Request Branch (`push-to-pr-branch:`)

Adding `push-to-pr-branch:` to the `safe-outputs:` section declares that the workflow should conclude with pushing additional changes to the branch associated with a pull request. This is useful for applying code changes directly to a designated branch within pull requests.

**Basic Configuration:**
```yaml
safe-outputs:
  push-to-pr-branch:
```

**With Configuration:**
```yaml
safe-outputs:
  push-to-pr-branch:
    target: "*"                          # Optional: target for push operations
                                         # "triggering" (default) - only push in triggering PR context
                                         # "*" - allow pushes to any pull request (requires pull_request_number in agent output)
                                         # explicit number - push for specific pull request number
    if-no-changes: "warn"                # Optional: behavior when no changes to push
                                         # "warn" (default) - log warning but succeed
                                         # "error" - fail the action
                                         # "ignore" - silent success
```

The agentic part of your workflow should describe the changes to be pushed and optionally provide a commit message.

**Example natural language to generate the output:**

```markdown
# Code Update Agent

Analyze the pull request and make necessary code improvements.

1. Make any file changes directly in the working directory  
2. Push changes to the feature branch with a descriptive commit message
```

**Examples with different error level configurations:**

```yaml
# Always succeed, warn when no changes (default behavior)
safe-outputs:
  push-to-pr-branch:
    branch: feature-branch
    if-no-changes: "warn"
```

```yaml
# Fail when no changes are made (strict mode)
safe-outputs:
  push-to-pr-branch:
    branch: feature-branch
    if-no-changes: "error"
```

```yaml
# Silent success, no output when no changes
safe-outputs:
  push-to-pr-branch:
    branch: feature-branch
    if-no-changes: "ignore"
```

**Safety Features:**

- Changes are applied via git patches generated from the workflow's modifications
- Only the specified branch can be modified
- Target configuration controls which pull requests can trigger pushes for security
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

When `create-pull-request` or `push-to-pr-branch` are enabled in the `safe-outputs` configuration, the system automatically adds the following additional Claude tools to enable file editing and pull request creation:

### Missing Tool Reporting (`missing-tool:`)

**Note:** Missing tool reporting is optional and must be explicitly configured in the `safe-outputs:` section if you want workflows to report when they encounter limitations or need tools that aren't available.

**Basic Configuration:**
```yaml
safe-outputs:
  missing-tool:                           # Enable missing-tool reporting
```

**With Configuration:**
```yaml
safe-outputs:
  missing-tool:
    max: 10                             # Optional: maximum number of missing tool reports (default: unlimited)
```

The agentic part of your workflow can report missing tools or functionality that prevents it from completing its task.

**Example natural language to generate the output:**

```markdown
# Development Task Agent

Analyze the repository and implement the requested feature. If you encounter missing tools, capabilities, or permissions that prevent completion, report them so the user can address these limitations.
```

The compiled workflow will have additional prompting describing that, to report missing tools, it should write the tool information to a special file.

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
    category-id: "DIC_kwDOGFsHUM4BsUn3"  # Optional: specific discussion category ID
    max: 3                           # Optional: maximum number of discussions (default: 1)
```

The agentic part of your workflow should describe the discussion(s) it wants created.

**Example markdown to generate the output:**

```yaml
# Research Discussion Agent

Research the latest developments in AI and create discussions to share findings.
Create new discussions with your research findings. For each discussion, provide a title starting with "AI Research Update" and detailed summary of the findings.
```

The compiled workflow will have additional prompting describing that, to create discussions, it should write the discussion details to a file.

**Note:** If no `category-id` is specified, the workflow will use the first available discussion category in the repository.

## Automatically Added Tools

When `create-pull-request` or `push-to-pr-branch` are configured, these Claude tools are automatically added:

- **Edit**: Allows editing existing files
- **MultiEdit**: Allows making multiple edits to files in a single operation
- **Write**: Allows creating new files or overwriting existing files
- **NotebookEdit**: Allows editing Jupyter notebook files

Along with the file editing tools, these Git commands are also automatically whitelisted:

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

By default, safe output jobs use the standard `GITHUB_TOKEN` provided by GitHub Actions. You can specify a custom GitHub token for all safe output jobs:

```yaml
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # Use custom PAT instead of GITHUB_TOKEN
  create-issue:
  add-comment:
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

When using `create-pull-request` or `push-to-pr-branch`, you can configure the maximum allowed size for git patches to prevent workflow failures from excessively large patches:

```yaml
safe-outputs:
  max-patch-size: 512                   # Optional: maximum patch size in KB (default: 1024)
  create-pull-request:
  push-to-pr-branch:
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
  push-to-pr-branch:
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

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
