# GitHub MCP Server Instructions

**Source**: [github/github-mcp-server](https://github.com/github/github-mcp-server/tree/main/pkg/github)
**Mapping File**: [pkg/workflow/data/github_toolsets_permissions.json](https://github.com/github/gh-aw/blob/main/pkg/workflow/data/github_toolsets_permissions.json)
**Last Updated**: 2026-03-01

## Overview

The GitHub MCP server provides tools to interact with GitHub APIs through the Model Context Protocol (MCP). It operates in two modes:

- **Remote mode**: Connects to GitHub's hosted MCP endpoint (`https://api.githubcopilot.com/mcp/`)
- **Local mode**: Runs `gh mcp` (GitHub CLI) as a local subprocess

### Authentication

**Remote mode**: Uses a Bearer token in the Authorization header:
```
Authorization: Bearer <github-token>
```

**Read-only mode**: Add the `X-MCP-Readonly: true` header to restrict to read operations only:
```
X-MCP-Readonly: true
```

**Local mode**: Uses the GitHub CLI's existing authentication (`gh auth login`).

## Configuration

### In Agentic Workflows

```yaml
tools:
  github:
    mode: "remote"          # or "local"
    toolsets: [default]     # or specific toolsets
    # Optional: GitHub App authentication
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

### Toolset Options

- `[default]` — Recommended defaults: `context`, `repos`, `issues`, `pull_requests`
- `[all]` — Enable all toolsets
- Specific toolsets: `[repos, issues, pull_requests, discussions]`
- Extend defaults: `[default, discussions, actions]`

## Recommended Default Toolsets

The following toolsets are recommended as defaults for typical agentic workflows:

| Toolset | Rationale |
|---------|-----------|
| `context` | Identity and team awareness (`get_me`, `get_teams`) — essential for any GitHub-aware agent |
| `repos` | Core repository operations (read files, list commits/branches) — most workflows need file access |
| `issues` | Issue management (read, comment, create) — common in CI/CD and automation workflows |
| `pull_requests` | PR operations (read, create, review) — critical for code review and merge automation |

**Enable explicitly when needed** (not in defaults):

| Toolset | When to Enable |
|---------|---------------|
| `actions` | Workflow introspection, triggering runs |
| `code_security` | Code scanning alert management |
| `dependabot` | Dependency vulnerability management |
| `discussions` | Community discussion workflows |
| `experiments` | Dynamic toolset management |
| `gists` | Gist creation and management |
| `labels` | Label management automation |
| `notifications` | Notification processing agents |
| `orgs` | Organization-level security advisories |
| `projects` | GitHub Projects automation (requires PAT) |
| `search` | Cross-repository search operations |
| `secret_protection` | Secret scanning alert management |
| `security_advisories` | Advisory database queries |
| `stargazers` | Star/unstar repository operations |
| `users` | (currently empty — no tools registered) |

## Tools by Toolset

### context
**Description**: GitHub context and environment (current user, teams)
**Source**: [`pkg/github/context_tools.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/context_tools.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_me` | Get details of the authenticated user | — |
| `get_team_members` | List members of a GitHub team | `org`, `team_slug` |
| `get_teams` | List teams the authenticated user belongs to | `org` |

---

### repos
**Description**: Repository operations
**Source**: [`pkg/github/repositories.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/repositories.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `create_branch` | Create a new branch | `owner`, `repo`, `branch`, `from_branch` |
| `create_or_update_file` | Create or update a file in a repository | `owner`, `repo`, `path`, `content`, `message`, `branch` |
| `create_repository` | Create a new GitHub repository | `name`, `description`, `private`, `auto_init` |
| `delete_file` | Delete a file from a repository | `owner`, `repo`, `path`, `message`, `sha`, `branch` |
| `fork_repository` | Fork a repository | `owner`, `repo`, `organization` |
| `get_commit` | Get details of a specific commit | `owner`, `repo`, `sha` |
| `get_file_contents` | Read file or directory contents | `owner`, `repo`, `path`, `ref` |
| `get_latest_release` | Get the latest release for a repository | `owner`, `repo` |
| `get_release_by_tag` | Get a release by its tag name | `owner`, `repo`, `tag` |
| `get_tag` | Get details of a specific tag | `owner`, `repo`, `tag` |
| `list_branches` | List branches in a repository | `owner`, `repo`, `page`, `per_page` |
| `list_commits` | List commits in a repository | `owner`, `repo`, `sha`, `path`, `page` |
| `list_releases` | List all releases for a repository | `owner`, `repo`, `page`, `per_page` |
| `list_tags` | List tags in a repository | `owner`, `repo`, `page`, `per_page` |
| `push_files` | Push multiple files in a single commit | `owner`, `repo`, `branch`, `files`, `message` |

---

### issues
**Description**: Issue management
**Source**: [`pkg/github/issues.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/issues.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `add_issue_comment` | Add a comment to an issue | `owner`, `repo`, `issue_number`, `body` |
| `issue_read` | Read issue details and comments | `owner`, `repo`, `issue_number` |
| `issue_write` | Create or update an issue | `owner`, `repo`, `title`, `body`, `labels`, `assignees` |
| `list_issue_types` | List available issue types for a repository | `owner`, `repo` |
| `list_issues` | List issues in a repository | `owner`, `repo`, `state`, `labels`, `page` |
| `search_issues` | Search issues across GitHub | `query`, `page`, `per_page` |
| `sub_issue_write` | Create or manage sub-issues | `owner`, `repo`, `issue_number` |

---

### pull_requests
**Description**: Pull request operations
**Source**: [`pkg/github/pullrequests.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/pullrequests.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `add_comment_to_pending_review` | Add a comment to a pending PR review | `owner`, `repo`, `pull_number`, `review_id` |
| `add_reply_to_pull_request_comment` | Reply to a PR review comment | `owner`, `repo`, `pull_number`, `comment_id`, `body` |
| `create_pull_request` | Create a new pull request | `owner`, `repo`, `title`, `body`, `head`, `base` |
| `list_pull_requests` | List pull requests in a repository | `owner`, `repo`, `state`, `head`, `base` |
| `merge_pull_request` | Merge a pull request | `owner`, `repo`, `pull_number`, `merge_method` |
| `pull_request_read` | Read PR details, reviews, and comments | `owner`, `repo`, `pull_number` |
| `pull_request_review_write` | Create or submit a PR review | `owner`, `repo`, `pull_number`, `event`, `body` |
| `search_pull_requests` | Search pull requests across GitHub | `query`, `page`, `per_page` |
| `update_pull_request` | Update PR title, body, or state | `owner`, `repo`, `pull_number`, `title`, `body` |
| `update_pull_request_branch` | Update PR branch with latest base | `owner`, `repo`, `pull_number` |

---

### actions
**Description**: GitHub Actions workflows
**Source**: [`pkg/github/actions.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/actions.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `actions_get` | Get details of a specific workflow run | `owner`, `repo`, `run_id` |
| `actions_list` | List GitHub Actions workflows and runs | `owner`, `repo`, `workflow_id` |
| `actions_run_trigger` | Trigger a workflow run | `owner`, `repo`, `workflow_id`, `ref`, `inputs` |
| `get_job_logs` | Download logs for a specific workflow job | `owner`, `repo`, `job_id` |

---

### code_security
**Description**: Code scanning alerts
**Source**: [`pkg/github/code_scanning.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/code_scanning.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_code_scanning_alert` | Get details of a specific code scanning alert | `owner`, `repo`, `alert_number` |
| `list_code_scanning_alerts` | List code scanning alerts for a repository | `owner`, `repo`, `state`, `severity` |

---

### dependabot
**Description**: Dependabot alerts
**Source**: [`pkg/github/dependabot.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/dependabot.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_dependabot_alert` | Get details of a specific Dependabot alert | `owner`, `repo`, `alert_number` |
| `list_dependabot_alerts` | List Dependabot alerts for a repository | `owner`, `repo`, `state`, `severity` |

---

### discussions
**Description**: GitHub Discussions
**Source**: [`pkg/github/discussions.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/discussions.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_discussion` | Get details of a specific discussion | `owner`, `repo`, `discussion_number` |
| `get_discussion_comments` | Get comments for a specific discussion | `owner`, `repo`, `discussion_number` |
| `list_discussion_categories` | List discussion categories for a repository | `owner`, `repo` |
| `list_discussions` | List discussions in a repository | `owner`, `repo`, `category_id` |

---

### experiments
**Description**: Experimental features — dynamic toolset management
**Source**: [`pkg/github/dynamic_tools.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/dynamic_tools.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `enable_toolset` | Dynamically enable a toolset | `toolset` |
| `get_toolset_tools` | Get tools available in a specific toolset | `toolset` |
| `list_available_toolsets` | List all available toolsets | — |

---

### gists
**Description**: Gist operations
**Source**: [`pkg/github/gists.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/gists.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `create_gist` | Create a new gist | `description`, `files`, `public` |
| `get_gist` | Get a specific gist by ID | `gist_id` |
| `list_gists` | List gists for a user | `username`, `page`, `per_page` |
| `update_gist` | Update an existing gist | `gist_id`, `description`, `files` |

---

### labels
**Description**: Label management
**Source**: [`pkg/github/labels.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/labels.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_label` | Get details of a specific label | `owner`, `repo`, `name` |
| `label_write` | Create or update a label | `owner`, `repo`, `name`, `color`, `description` |
| `list_label` | List labels in a repository | `owner`, `repo`, `page`, `per_page` |

---

### notifications
**Description**: Notification management
**Source**: [`pkg/github/notifications.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/notifications.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `dismiss_notification` | Dismiss a specific notification | `notification_id` |
| `get_notification_details` | Get details of a specific notification | `notification_id` |
| `list_notifications` | List user notifications | `all`, `participating`, `page` |
| `manage_notification_subscription` | Manage notification subscription for a thread | `thread_id`, `subscribed` |
| `manage_repository_notification_subscription` | Manage notifications for a repository | `owner`, `repo`, `subscribed` |
| `mark_all_notifications_read` | Mark all notifications as read | `last_read_at` |

---

### orgs
**Description**: Organization operations
**Source**: [`pkg/github/security_advisories.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/security_advisories.go) (for `list_org_repository_security_advisories`)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `list_org_repository_security_advisories` | List security advisories for all repos in an org | `org`, `state` |

---

### projects
**Description**: GitHub Projects (requires PAT — not supported by GITHUB_TOKEN)
**Source**: [`pkg/github/projects.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/projects.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `projects_get` | Get details of a specific project | `owner`, `project_number` |
| `projects_list` | List GitHub Projects for a user or organization | `owner`, `per_page` |
| `projects_write` | Create or update project items/fields | `owner`, `project_number` |

---

### search
**Description**: Advanced search across GitHub
**Source**: [`pkg/github/search.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/search.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `search_code` | Search code across repositories | `query`, `page`, `per_page` |
| `search_orgs` | Search GitHub organizations | `query`, `page`, `per_page` |
| `search_repositories` | Search for repositories | `query`, `page`, `per_page` |
| `search_users` | Search GitHub users | `query`, `page`, `per_page` |

---

### secret_protection
**Description**: Secret scanning
**Source**: [`pkg/github/secret_scanning.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/secret_scanning.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_secret_scanning_alert` | Get details of a specific secret scanning alert | `owner`, `repo`, `alert_number` |
| `list_secret_scanning_alerts` | List secret scanning alerts for a repository | `owner`, `repo`, `state` |

---

### security_advisories
**Description**: Security advisories
**Source**: [`pkg/github/security_advisories.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/security_advisories.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `get_global_security_advisory` | Get a specific global security advisory | `ghsa_id` |
| `list_global_security_advisories` | List advisories from the GitHub Advisory Database | `type`, `severity`, `ecosystem` |
| `list_repository_security_advisories` | List security advisories for a specific repository | `owner`, `repo`, `state` |

---

### stargazers
**Description**: Repository stars
**Source**: [`pkg/github/repositories.go`](https://github.com/github/github-mcp-server/blob/main/pkg/github/repositories.go)

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `list_starred_repositories` | List repositories starred by a user | `username`, `page`, `per_page` |
| `star_repository` | Star a repository | `owner`, `repo` |
| `unstar_repository` | Unstar a repository | `owner`, `repo` |

---

### users
**Description**: User information
**Source**: N/A (currently no tools registered)

> **Note**: No tools are currently registered in the `users` toolset. User search is available via the `search` toolset (`search_users`).

---

## Best Practices

### Toolset Selection

1. **Start with defaults** (`context`, `repos`, `issues`, `pull_requests`) for most workflows
2. **Add toolsets incrementally** based on actual needs rather than enabling `all`
3. **Security toolsets** (`code_security`, `dependabot`, `secret_protection`, `security_advisories`) require `security-events` permission
4. **Write operations** require appropriate GitHub token permissions (see `write_permissions` in the JSON mapping)
5. **Projects toolset** requires a PAT (Personal Access Token) — `GITHUB_TOKEN` lacks the required `project` scope

### Permission Requirements

Most toolsets work with the default `GITHUB_TOKEN` in GitHub Actions. Exceptions:

- `projects` — Requires a PAT with `project` scope
- `security_advisories` (write) — Requires `security-events: write` permission
- `actions` (write for `actions_run_trigger`) — Requires `actions: write` permission

### Token Scopes for Remote Mode

When using remote mode with a PAT:
- Basic read: `repo` scope
- Issues/PRs write: `repo` scope (covers everything)
- Projects: `project` scope
- Gists: `gist` scope
- Notifications: `notifications` scope

## Tool Count Summary

| Toolset | Tool Count |
|---------|-----------|
| actions | 4 |
| code_security | 2 |
| context | 3 |
| dependabot | 2 |
| discussions | 4 |
| experiments | 3 |
| gists | 4 |
| issues | 7 |
| labels | 3 |
| notifications | 6 |
| orgs | 1 |
| projects | 3 |
| pull_requests | 10 |
| repos | 15 |
| search | 4 |
| secret_protection | 2 |
| security_advisories | 3 |
| stargazers | 3 |
| users | 0 |
| **Total** | **79** |
