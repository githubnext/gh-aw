---
title: Frontmatter Reference
description: Complete JSON Schema-based reference for all GitHub Agentic Workflows frontmatter configuration options with YAML examples.
sidebar:
  order: 201
---

This document provides a comprehensive reference for all available frontmatter configuration options in GitHub Agentic Workflows. The examples below are generated from the JSON Schema and include inline comments describing each field.

> [!NOTE]
> This documentation is automatically generated from the JSON Schema. For a more user-friendly guide, see [Frontmatter](/gh-aw/reference/frontmatter/).

## Schema Description

JSON Schema for validating agentic workflow frontmatter configuration

## Complete Frontmatter Reference

```yaml wrap
---
# Workflow name that appears in the GitHub Actions interface. If not specified,
# defaults to the filename without extension.
# (optional)
name: "My Workflow"

# Optional workflow description that is rendered as a comment in the generated
# GitHub Actions YAML file (.lock.yml)
# (optional)
description: "Description of the workflow"

# Optional source reference indicating where this workflow was added from. Format:
# owner/repo/path@ref (e.g., githubnext/agentics/workflows/ci-doctor.md@v1.0.0).
# Rendered as a comment in the generated lock file.
# (optional)
source: "example-value"

# Optional workflow location redirect for updates. Format: workflow spec or GitHub
# URL (e.g., owner/repo/path@ref or
# https://github.com/owner/repo/blob/main/path.md). When present, update follows
# this location and rewrites source.
# (optional)
redirect: "example-value"

# Optional tracker identifier to tag all created assets (issues, discussions,
# comments, pull requests). Must be at least 8 characters and contain only
# alphanumeric characters, hyphens, and underscores. This identifier will be
# inserted in the body/description of all created assets to enable searching and
# retrieving assets associated with this workflow.
# (optional)
tracker-id: "example-value"

# Optional array of labels to categorize and organize workflows. Labels can be
# used to filter workflows in status/list commands.
# (optional)
labels: []
  # Array of strings

# Optional metadata field for storing custom key-value pairs compatible with the
# custom agent spec. Key names are limited to 64 characters, and values are
# limited to 1024 characters.
# (optional)
metadata:
  {}

# Workflow specifications to import. Supports array form (list of paths) or object
# form with 'aw' (agentic workflow paths) subfield. Path resolution: (1) relative
# paths (e.g., 'shared/file.md') are resolved relative to the workflow's
# directory; (2) paths starting with '.github/' or '/' are resolved from the
# repository root (repo-root-relative); (3) paths matching 'owner/repo/path@ref'
# are fetched from GitHub at compile time (cross-repo).
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Array of workflow specifications to import. Three path formats are
# supported: relative paths ('shared/file.md'), repo-root-relative paths
# ('.github/agents/my-agent.md'), and cross-repo paths ('owner/repo/path@ref').
# Any markdown files under .github/agents directory are treated as custom agent
# files and only one agent file is allowed per workflow.
imports: []
  # Array items: undefined

# Option 2: Object form of imports with 'aw' subfield for shared agentic workflow
# paths.
imports:
  # Array of shared agentic workflow specifications to import. Format:
  # owner/repo/path@ref or relative paths.
  # (optional)
  aw: []

# Optional list of additional workflow or action files that should be fetched
# alongside this workflow when running 'gh aw add'. Entries are relative paths
# (from the same directory as this workflow in the source repository) to agentic
# workflow .md files or GitHub Actions .yml/.yaml files. GitHub Actions expression
# syntax (${{) is not allowed in resource paths.
# (optional)
resources: []
  # Array of Relative path to a workflow .md file or action .yml/.yaml file. Must be
  # a static path; GitHub Actions expression syntax (${{) is not allowed.

# If true, inline all imports (including those without inputs) at compilation time
# in the generated lock.yml instead of using runtime-import macros. When enabled,
# the frontmatter hash covers the entire markdown body so any change to the
# content will invalidate the hash.
# (optional)
inlined-imports: true

# Workflow triggers that define when the agentic workflow should run. Supports
# standard GitHub Actions trigger events plus special command triggers for
# /commands (required)
# This field supports multiple formats (oneOf):

# Option 1: Simple trigger event name (e.g., 'push', 'issues', 'pull_request',
# 'discussion', 'schedule', 'fork', 'create', 'delete', 'public', 'watch',
# 'workflow_call'), schedule shorthand (e.g., 'daily', 'weekly'), or slash command
# shorthand (e.g., '/my-bot' expands to slash_command + workflow_dispatch)
on: "example-value"

# Option 2: Complex trigger configuration with event-specific filters and options
on:
  # Special slash command trigger for /command workflows (e.g., '/my-bot' in issue
  # comments). Creates conditions to match slash commands automatically. Note: Can
  # be combined with issues/pull_request events if those events only use 'labeled'
  # or 'unlabeled' types.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null command configuration - defaults to using the workflow filename
  # (without .md extension) as the command name
  slash_command: null

  # Option 2: Command name as a string (shorthand format, e.g., 'customname' for
  # '/customname' triggers). Command names must not start with '/' as the slash is
  # automatically added when matching commands.
  slash_command: "example-value"

  # Option 3: Command configuration object with custom command name
  slash_command:
    # Name of the slash command that triggers the workflow (e.g., '/help',
    # '/analyze'). Used for comment-based workflow activation.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single command name for slash commands (e.g., 'helper-bot' for
    # '/helper-bot' triggers). Command names must not start with '/' as the slash is
    # automatically added when matching commands. Defaults to workflow filename
    # without .md extension if not specified.
    name: "My Workflow"

    # Option 2: Array of command names that trigger this workflow (e.g., ['cmd.add',
    # 'cmd.remove'] for '/cmd.add' and '/cmd.remove' triggers). Each command name must
    # not start with '/'.
    name: []
      # Array items: Command name without leading slash

    # Events where the command should be active. Default is all comment-related events
    # ('*'). Use GitHub Actions event names.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single event name or '*' for all events. Use GitHub Actions event
    # names: 'issues', 'issue_comment', 'pull_request_comment', 'pull_request',
    # 'pull_request_review_comment', 'discussion', 'discussion_comment'.
    events: "*"

    # Option 2: Array of event names where the command should be active (requires at
    # least one). Use GitHub Actions event names.
    events: []
      # Array items: GitHub Actions event name.

  # DEPRECATED: Use 'slash_command' instead. Special command trigger for /command
  # workflows (e.g., '/my-bot' in issue comments). Creates conditions to match slash
  # commands automatically.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null command configuration - defaults to using the workflow filename
  # (without .md extension) as the command name
  command: null

  # Option 2: Command name as a string (shorthand format, e.g., 'customname' for
  # '/customname' triggers). Command names must not start with '/' as the slash is
  # automatically added when matching commands.
  command: "example-value"

  # Option 3: Command configuration object with custom command name
  command:
    # Name of the slash command that triggers the workflow (e.g., '/deploy', '/test').
    # Used for command-based workflow activation.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Custom command name for slash commands (e.g., 'helper-bot' for
    # '/helper-bot' triggers). Command names must not start with '/' as the slash is
    # automatically added when matching commands. Defaults to workflow filename
    # without .md extension if not specified.
    name: "My Workflow"

    # Option 2: Array of command names that trigger this workflow (e.g., ['cmd.add',
    # 'cmd.remove'] for '/cmd.add' and '/cmd.remove' triggers). Each command name must
    # not start with '/'.
    name: []
      # Array items: Command name without leading slash

    # Events where the command should be active. Default is all comment-related events
    # ('*'). Use GitHub Actions event names.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single event name or '*' for all events. Use GitHub Actions event
    # names: 'issues', 'issue_comment', 'pull_request_comment', 'pull_request',
    # 'pull_request_review_comment', 'discussion', 'discussion_comment'.
    events: "*"

    # Option 2: Array of event names where the command should be active (requires at
    # least one). Use GitHub Actions event names.
    events: []
      # Array items: GitHub Actions event name.

  # On Label Command trigger: fires when a specific label is added to an issue, pull
  # request, or discussion. The triggering label is automatically removed at
  # workflow start so it can be applied again to re-trigger. Use the 'events' field
  # to restrict which item types (issues, pull_request, discussion) activate the
  # trigger.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Label name as a string (shorthand format). The workflow fires when
  # this label is added to any supported item type (issue, pull request, or
  # discussion).
  label_command: "example-value"

  # Option 2: Label command configuration object with label name(s) and optional
  # event filtering.
  label_command:
    # Label name(s) that trigger the workflow when added to an issue, pull request, or
    # discussion.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single label name that acts as a command (e.g., 'deploy' triggers the
    # workflow when the 'deploy' label is added).
    name: "My Workflow"

    # Option 2: Array of label names — any of these labels will trigger the workflow.
    name: []
      # Array items: A label name

    # Alternative to 'name': label name(s) that trigger the workflow.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single label name.
    names: "example-value"

    # Option 2: Array of label names — any of these labels will trigger the workflow.
    names: []
      # Array items: A label name

    # Item types where the label-command trigger should be active. Default is all
    # supported types: issues, pull_request, discussion.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single item type or '*' for all types.
    events: "*"

    # Option 2: Array of item types where the trigger is active.
    events: []
      # Array items: Item type.

    # Whether to automatically remove the triggering label after the workflow starts.
    # Defaults to true. Set to false to keep the label on the item and skip the
    # label-removal step. When false, the issues:write and discussions:write
    # permissions required for label removal are also omitted.
    # (optional)
    remove_label: true

  # Push event trigger that runs the workflow when code is pushed to the repository
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: undefined

  # Option 2: undefined

  # Option 3: undefined

  # Pull request event trigger that runs the workflow when pull requests are
  # created, updated, or closed
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: undefined

  # Option 2: undefined

  # Option 3: undefined

  # Issues event trigger that runs when repository issues are created, updated, or
  # managed
  # (optional)
  issues:
    # Types of issue events
    # (optional)
    types: []
      # Array of strings

    # Array of issue type names that trigger the workflow. Filters workflow execution
    # to specific issue categories.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single label name to filter labeled/unlabeled events (e.g., 'bug')
    names: "example-value"

    # Option 2: List of label names to filter labeled/unlabeled events. Only applies
    # when 'labeled' or 'unlabeled' is in the types array
    names: []
      # Array items: Label name

    # Whether to lock the issue for the agent when the workflow runs (prevents
    # concurrent modifications)
    # (optional)
    lock-for-agent: true

  # Issue comment event trigger
  # (optional)
  issue_comment:
    # Types of issue comment events
    # (optional)
    types: []
      # Array of strings

    # Whether to lock the parent issue for the agent when the workflow runs (prevents
    # concurrent modifications)
    # (optional)
    lock-for-agent: true

  # Discussion event trigger that runs the workflow when repository discussions are
  # created, updated, or managed
  # (optional)
  discussion:
    # Types of discussion events
    # (optional)
    types: []
      # Array of strings

  # Discussion comment event trigger that runs the workflow when comments on
  # discussions are created, updated, or deleted
  # (optional)
  discussion_comment:
    # Types of discussion comment events
    # (optional)
    types: []
      # Array of strings

  # Scheduled trigger events using fuzzy schedules or standard cron expressions.
  # Supports shorthand string notation (e.g., 'daily', 'daily around 2pm') or array
  # of schedule objects. Fuzzy schedules automatically distribute execution times to
  # prevent load spikes.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Shorthand schedule string using fuzzy or cron format. Examples:
  # 'daily', 'daily around 14:00', 'daily between 9:00 and 17:00', 'weekly', 'weekly
  # on monday', 'weekly on friday around 5pm', 'hourly', 'every 2h', 'every 10
  # minutes', '0 9 * * 1'. Fuzzy schedules distribute execution times to prevent
  # load spikes. For fixed times, use standard cron syntax. Minimum interval is 5
  # minutes.
  schedule: "example-value"

  # Option 2: Array of schedule objects with cron expressions (standard cron or
  # fuzzy format)
  schedule: []
    # Array items: object

  # Manual workflow dispatch trigger
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple workflow dispatch trigger
  workflow_dispatch: null

  # Option 2: object
  workflow_dispatch:
    # Input parameters for manual dispatch
    # (optional)
    inputs:
      {}

  # Workflow run trigger
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: undefined

  # Option 2: undefined

  # Option 3: undefined

  # Release event trigger
  # (optional)
  release:
    # Types of release events
    # (optional)
    types: []
      # Array of strings

  # Pull request review comment event trigger
  # (optional)
  pull_request_review_comment:
    # Types of pull request review comment events
    # (optional)
    types: []
      # Array of strings

  # Branch protection rule event trigger that runs when branch protection rules are
  # changed
  # (optional)
  branch_protection_rule:
    # Types of branch protection rule events
    # (optional)
    types: []
      # Array of strings

  # Check run event trigger that runs when a check run is created, rerequested,
  # completed, or has a requested action
  # (optional)
  check_run:
    # Types of check run events
    # (optional)
    types: []
      # Array of strings

  # Check suite event trigger that runs when check suite activity occurs
  # (optional)
  check_suite:
    # Types of check suite events
    # (optional)
    types: []
      # Array of strings

  # Create event trigger that runs when a Git reference (branch or tag) is created
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple create event trigger
  create: null

  # Option 2: object
  create:
    {}

  # Delete event trigger that runs when a Git reference (branch or tag) is deleted
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple delete event trigger
  delete: null

  # Option 2: object
  delete:
    {}

  # Deployment event trigger that runs when a deployment is created
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple deployment event trigger
  deployment: null

  # Option 2: object
  deployment:
    {}

  # Deployment status event trigger that runs when a deployment status is updated
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple deployment status event trigger
  deployment_status: null

  # Option 2: object
  deployment_status:
    # Filter to specific deployment states (compiled into if condition). Use a string
    # for one state or an array for multiple states.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: string
    state: "error"

    # Option 2: array
    state: []
      # Array items: string

  # Fork event trigger that runs when someone forks the repository
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple fork event trigger
  fork: null

  # Option 2: object
  fork:
    {}

  # Gollum event trigger that runs when someone creates or updates a Wiki page
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple gollum event trigger
  gollum: null

  # Option 2: object
  gollum:
    {}

  # Label event trigger that runs when a label is created, edited, or deleted
  # (optional)
  label:
    # Types of label events
    # (optional)
    types: []
      # Array of strings

  # Merge group event trigger that runs when a pull request is added to a merge
  # queue
  # (optional)
  merge_group:
    # Types of merge group events
    # (optional)
    types: []
      # Array of strings

  # Milestone event trigger that runs when a milestone is created, closed, opened,
  # edited, or deleted
  # (optional)
  milestone:
    # Types of milestone events
    # (optional)
    types: []
      # Array of strings

  # Page build event trigger that runs when someone pushes to a GitHub Pages
  # publishing source branch
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple page build event trigger
  page_build: null

  # Option 2: object
  page_build:
    {}

  # Public event trigger that runs when a repository changes from private to public
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple public event trigger
  public: null

  # Option 2: object
  public:
    {}

  # Pull request target event trigger that runs in the context of the base
  # repository (secure for fork PRs)
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: undefined

  # Option 2: undefined

  # Option 3: undefined

  # Pull request review event trigger that runs when a pull request review is
  # submitted, edited, or dismissed
  # (optional)
  pull_request_review:
    # Types of pull request review events
    # (optional)
    types: []
      # Array of strings

  # Registry package event trigger that runs when a package is published or updated
  # (optional)
  registry_package:
    # Types of registry package events
    # (optional)
    types: []
      # Array of strings

  # Repository dispatch event trigger for custom webhook events
  # (optional)
  repository_dispatch:
    # Custom event types to trigger on
    # (optional)
    types: []
      # Array of strings

  # Status event trigger that runs when the status of a Git commit changes
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple status event trigger
  status: null

  # Option 2: object
  status:
    {}

  # Watch event trigger that runs when someone stars the repository
  # (optional)
  watch:
    # Types of watch events
    # (optional)
    types: []
      # Array of strings

  # Workflow call event trigger that allows this workflow to be called by another
  # workflow
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple workflow call event trigger
  workflow_call: null

  # Option 2: object
  workflow_call:
    # Input parameters that can be passed to the workflow when it is called
    # (optional)
    inputs:
      {}

    # Secrets that can be passed to the workflow when it is called
    # (optional)
    secrets:
      {}

  # Time when workflow should stop running. Supports multiple formats: absolute
  # dates (YYYY-MM-DD HH:MM:SS, June 1 2025, 1st June 2025, 06/01/2025, etc.) or
  # relative time deltas (+25h, +3d, +1d12h30m). Maximum values for time deltas:
  # 12mo, 52w, 365d, 8760h (365 days). Note: Minute unit 'm' is not allowed for
  # stop-after; minimum unit is hours 'h'.
  # (optional)
  stop-after: "example-value"

  # Conditionally skip workflow execution when a GitHub search query has matches.
  # Can be a string (query only, implies max=1) or an object with 'query', optional
  # 'max', and 'scope' fields. Use top-level on.github-token or on.github-app for
  # custom authentication.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: GitHub search query string to check before running workflow (implies
  # max=1). If the search returns any results, the workflow will be skipped. Query
  # is automatically scoped to the current repository. Example: 'is:issue is:open
  # label:bug'
  skip-if-match: "example-value"

  # Option 2: Skip-if-match configuration object with query, maximum match count,
  # and optional scope. For custom authentication use the top-level on.github-token
  # or on.github-app fields.
  skip-if-match:
    # GitHub search query string to check before running workflow. Query is
    # automatically scoped to the current repository.
    query: "example-value"

    # Maximum number of items that must be matched for the workflow to be skipped.
    # Defaults to 1 if not specified. Supports integer or GitHub Actions expression
    # (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Scope for the search query. Set to 'none' to disable the automatic
    # 'repo:owner/repo' scoping, enabling org-wide or cross-repo queries.
    # (optional)
    scope: "none"

  # Conditionally skip workflow execution when a GitHub search query has no matches
  # (or fewer than minimum). Can be a string (query only, implies min=1) or an
  # object with 'query', optional 'min', and 'scope' fields. Use top-level
  # on.github-token or on.github-app for custom authentication.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: GitHub search query string to check before running workflow (implies
  # min=1). If the search returns no results, the workflow will be skipped. Query is
  # automatically scoped to the current repository. Example: 'is:pr is:open
  # label:ready-to-deploy'
  skip-if-no-match: "example-value"

  # Option 2: Skip-if-no-match configuration object with query, minimum match count,
  # and optional scope. For custom authentication use the top-level on.github-token
  # or on.github-app fields.
  skip-if-no-match:
    # GitHub search query string to check before running workflow. Query is
    # automatically scoped to the current repository.
    query: "example-value"

    # Minimum number of items that must be matched for the workflow to proceed.
    # Defaults to 1 if not specified.
    # (optional)
    min: 1

    # Scope for the search query. Set to 'none' to disable the automatic
    # 'repo:owner/repo' scoping, enabling org-wide or cross-repo queries.
    # (optional)
    scope: "none"

  # Skip workflow execution if any CI checks on the target branch are failing or
  # pending. Accepts true (check all) or an object to filter specific checks by name
  # and optionally specify a branch or allow pending checks.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Bare key with no value — equivalent to true. Skips workflow execution
  # if any CI checks on the target branch are currently failing.
  skip-if-check-failing: null

  # Option 2: Skip workflow execution if any CI checks on the target branch are
  # currently failing. For pull_request events, checks the base branch. For other
  # events, checks the current ref.
  skip-if-check-failing: true

  # Option 3: Skip-if-check-failing configuration object with optional
  # include/exclude filter lists, an optional branch name, and an allow-pending
  # flag.
  skip-if-check-failing:
    # List of check names to evaluate. When specified, only these named checks are
    # considered. If omitted, all checks are evaluated.
    # (optional)
    include: []
      # Array of strings

    # List of check names to ignore. Checks in this list are not considered when
    # determining whether to skip the workflow.
    # (optional)
    exclude: []
      # Array of strings

    # Branch name to check for failing CI checks. When omitted, defaults to the base
    # branch of a pull_request event or the current ref for other events.
    # (optional)
    branch: "example-value"

    # When true, pending or in-progress checks are not treated as failing. By default
    # (false), any check that has not yet completed is treated as failing and will
    # block the workflow.
    # (optional)
    allow-pending: true

  # Skip workflow execution for users with specific repository roles. Useful for
  # workflows that should only run for external contributors or specific permission
  # levels.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Single role to skip workflow for (e.g., 'admin'). If the triggering
  # user has this role, the workflow will be skipped.
  skip-roles: "example-value"

  # Option 2: List of roles to skip workflow for (e.g., ['admin', 'maintainer',
  # 'write']). If the triggering user has any of these roles, the workflow will be
  # skipped.
  skip-roles: []
    # Array items: string

  # Skip workflow execution for specific GitHub users. Useful for preventing
  # workflows from running for specific accounts (e.g., bots, specific team
  # members).
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Single GitHub username to skip workflow for (e.g., 'user1'). If the
  # triggering user matches, the workflow will be skipped.
  skip-bots: "example-value"

  # Option 2: List of GitHub usernames to skip workflow for (e.g., ['user1',
  # 'user2']). If the triggering user is in this list, the workflow will be skipped.
  skip-bots: []
    # Array items: string

  # Repository access roles required to trigger agentic workflows. Defaults to
  # ['admin', 'maintainer', 'write'] for security. Use 'all' to allow any
  # authenticated user (⚠️ security consideration).
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Single repository permission level that can trigger the workflow. Use
  # 'all' to allow any authenticated user (⚠️ disables permission checking entirely
  # - use with caution)
  roles: "admin"

  # Option 2: List of repository permission levels that can trigger the workflow.
  # Permission checks are automatically applied to potentially unsafe triggers.
  roles: []
    # Array items: Repository permission level: 'admin' (full access),
    # 'maintainer'/'maintain' (repository management), 'write' (push access), 'triage'
    # (issue management), 'read' (read-only access)

  # Allow list of bot identifiers that can trigger the workflow even if they don't
  # meet the required role permissions. When the actor is in this list, the bot must
  # be active (installed) on the repository to trigger the workflow.
  # (optional)
  bots: []
    # Array of Bot identifier/name (e.g., 'dependabot[bot]', 'renovate[bot]',
    # 'github-actions[bot]')

  # Filter workflows triggered by pull_request_target (or other labeled events) to
  # only fire when the triggering label matches one of these names. Generates a
  # job-level if: condition on the pre-activation job so unmatched label events show
  # as Skipped (⊘) rather than Failed (❌).
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Single label name that must match the triggering label (e.g.,
  # 'panel-review')

  # Option 2: List of label names; the workflow fires when the triggering label
  # matches any entry.
  labels: []
    # Array items: undefined

  # Environment name that requires manual approval before the workflow can run. Must
  # match a valid environment configured in the repository settings.
  # (optional)
  manual-approval: "example-value"

  # AI reaction to add/remove on triggering item. Scalar form accepts one of: +1,
  # -1, laugh, confused, heart, hooray, rocket, eyes, none. Object form implies
  # enabled reactions and supports optional `issues`, `pull-requests`, and
  # `discussions` fields to control trigger groups independently; use `type` to
  # choose the reaction emoji (defaults to `eyes` when omitted). Use 'none' to
  # disable reactions.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: string
  reaction: "+1"

  # Option 2: YAML parses +1 and -1 without quotes as integers. These are converted
  # to +1 and -1 strings respectively.
  reaction: 1

  # Option 3: object
  reaction:
    # Reaction type. Defaults to 'eyes' when omitted.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: string
    type: "+1"

    # Option 2: YAML parses +1 and -1 without quotes as integers. These are converted
    # to +1 and -1 strings respectively.
    type: 1

    # Whether reactions are allowed for issue triggers (issues, issue_comment).
    # (optional)
    issues: true

    # Whether reactions are allowed for pull request triggers (pull_request,
    # pull_request_review_comment).
    # (optional)
    pull-requests: true

    # Whether reactions are allowed for discussion and discussion_comment triggers.
    # (optional)
    discussions: true

  # Whether to post status comments (started/completed) on the triggering item.
  # Boolean form enables/disables status comments globally. Object form implies
  # enabled status comments and supports optional `issues`, `pull-requests`, and
  # `discussions` fields to control trigger groups independently. Automatically
  # enabled for slash_command and label_command triggers when not explicitly
  # configured.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: boolean
  status-comment: true

  # Option 2: object
  status-comment:
    # Whether status comments are allowed for issue triggers (issues, issue_comment).
    # (optional)
    issues: true

    # Whether status comments are allowed for pull request triggers (pull_request,
    # pull_request_review_comment).
    # (optional)
    pull-requests: true

    # Whether status comments are allowed for discussion and discussion_comment
    # triggers.
    # (optional)
    discussions: true

  # Custom GitHub token for pre-activation reactions, activation status comments,
  # and skip-if search queries. When specified, overrides the default GITHUB_TOKEN
  # for these operations.
  # (optional)
  github-token: "${{ secrets.GITHUB_TOKEN }}"

  # GitHub App configuration for minting a token used in pre-activation reactions,
  # activation status comments, and skip-if search queries. When configured, a
  # single GitHub App installation access token is minted and shared across all
  # these operations instead of using the default GITHUB_TOKEN. Can be defined in a
  # shared agentic workflow and inherited by importing workflows.
  # (optional)
  # This field supports multiple formats (anyOf):

  # Option 1: undefined

  # Option 2: undefined

  # Explicit additional custom workflow jobs that pre_activation and activation
  # should depend on.
  # (optional)
  needs: []
    # Array of strings

  # Steps to inject into the pre-activation job. These steps run after all built-in
  # checks (membership, stop-time, skip-if, etc.) and their results are exposed as
  # pre-activation outputs. Use 'id' on steps to reference their results via
  # needs.pre_activation.outputs.<id>_result.
  # (optional)
  steps: []
    # Array items:
      # Optional name for the step
      # (optional)
      name: "My Workflow"

      # Optional step ID. When set, the step result is exposed as
      # needs.pre_activation.outputs.<id>_result
      # (optional)
      id: "example-value"

      # Shell command to run
      # (optional)
      run: "example-value"

      # Action to use (e.g., 'actions/checkout@v4')
      # (optional)
      uses: "example-value"

      # Input parameters for the action
      # (optional)
      with:
        {}

      # Environment variables for the step
      # (optional)
      env:
        {}

      # Conditional expression for the step
      # (optional)
      if: "example-value"

      # Whether to continue if the step fails
      # (optional)
      continue-on-error: true

  # Additional permissions for the pre-activation job. Use to declare extra scopes
  # required by on.steps (e.g., issues: read for GitHub API calls in steps).
  # (optional)
  # Map of permission scope to level
  # (optional)
  permissions:
    # (optional)
    actions: "read"

    # (optional)
    checks: "read"

    # (optional)
    contents: "read"

    # (optional)
    deployments: "read"

    # (optional)
    discussions: "read"

    # (optional)
    issues: "read"

    # (optional)
    packages: "read"

    # (optional)
    pages: "read"

    # (optional)
    pull-requests: "read"

    # (optional)
    repository-projects: "read"

    # (optional)
    security-events: "read"

    # (optional)
    statuses: "read"

  # When set to false, disables the frontmatter hash check step in the activation
  # job. Default is true (check is enabled). Useful when the workflow source files
  # are managed outside the default GitHub repo context (e.g. cross-repo org
  # rulesets) and the stale check is not needed.
  # (optional)
  stale-check: true

# GitHub token permissions for the workflow. Controls what the GITHUB_TOKEN can
# access during execution. Use the principle of least privilege - only grant the
# minimum permissions needed.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple permissions string: 'read-all' (all read permissions) or
# 'write-all' (all write permissions)
permissions: "read-all"

# Option 2: undefined

# Custom name for workflow runs that appears in the GitHub Actions interface
# (supports GitHub expressions like ${{ github.event.issue.title }})
# (optional)
run-name: "example-value"

# Groups together all the jobs that run in the workflow
# (optional)
jobs:
  {}

# Runner type for workflow execution (GitHub Actions standard field). Supports
# multiple forms: simple string for single runner label (e.g., 'ubuntu-latest'),
# array for runner selection with fallbacks, or object for GitHub-hosted runner
# groups with specific labels. For agentic workflows, runner selection matters
# when AI workloads require specific compute resources or when using self-hosted
# runners with specialized capabilities. Typically configured at the job level
# instead. See
# https://docs.github.com/en/actions/using-jobs/choosing-the-runner-for-a-job
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple runner label string. Use for standard GitHub-hosted runners
# (e.g., 'ubuntu-latest', 'windows-latest', 'macos-latest') or self-hosted runner
# labels. Most common form for agentic workflows.
runs-on: "example-value"

# Option 2: Array of runner labels for selection with fallbacks. GitHub Actions
# will use the first available runner that matches any label in the array. Useful
# for high-availability setups or when multiple runner types are acceptable.
runs-on: []
  # Array items: string

# Option 3: Runner group configuration for GitHub-hosted runners. Use this form to
# target specific runner groups (e.g., larger runners with more CPU/memory) or
# self-hosted runner pools with specific label requirements. Agentic workflows may
# benefit from larger runners for complex AI processing tasks.
runs-on:
  # Runner group name for self-hosted runners or GitHub-hosted runner groups
  # (optional)
  group: "example-value"

  # List of runner labels for self-hosted runners or GitHub-hosted runner selection
  # (optional)
  labels: []
    # Array of strings

# Runner for all framework/generated jobs (activation, pre-activation,
# safe-outputs, unlock, APM, etc.). Provides a compile-stable override for
# generated job runners without requiring a safe-outputs section. Overridden by
# safe-outputs.runs-on when both are set. Defaults to 'ubuntu-slim'. Use this when
# your infrastructure does not provide the default runner or when you need
# consistent runner selection across all jobs.
# (optional)
runs-on-slim: "example-value"

# Workflow timeout in minutes (GitHub Actions standard field). Defaults to 20
# minutes for agentic workflows. Has sensible defaults and can typically be
# omitted. Custom runners support longer timeouts beyond the GitHub-hosted runner
# limit. Supports GitHub Actions expressions (e.g. '${{ inputs.timeout }}') for
# reusable workflow_call workflows.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: integer
timeout-minutes: 1

# Option 2: GitHub Actions expression that resolves to an integer (e.g. '${{
# inputs.timeout }}')
timeout-minutes: "example-value"

# Concurrency control to limit concurrent workflow runs (GitHub Actions standard
# field). Supports two forms: simple string for basic group isolation, or object
# with cancel-in-progress option for advanced control. Agentic workflows enhance
# this with automatic per-engine concurrency policies (defaults to single job per
# engine across all workflows) and token-based rate limiting. Default behavior:
# workflows in the same group queue sequentially unless cancel-in-progress is
# true. See https://docs.github.com/en/actions/using-jobs/using-concurrency
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple concurrency group name to prevent multiple runs in the same
# group. Use expressions like '${{ github.workflow }}' for per-workflow isolation
# or '${{ github.ref }}' for per-branch isolation. Agentic workflows automatically
# generate enhanced concurrency policies using 'gh-aw-{engine-id}' as the default
# group to limit concurrent AI workloads across all workflows using the same
# engine.
concurrency: "example-value"

# Option 2: Concurrency configuration object with group isolation and cancellation
# control. Use object form when you need fine-grained control over whether to
# cancel in-progress runs. For agentic workflows, this is useful to prevent
# multiple AI agents from running simultaneously and consuming excessive resources
# or API quotas.
concurrency:
  # Concurrency group name. Workflows in the same group cannot run simultaneously.
  # Supports GitHub Actions expressions for dynamic group names based on branch,
  # workflow, or other context.
  # (optional)
  group: "example-value"

  # Whether to cancel in-progress workflows in the same concurrency group when a new
  # one starts. Default: false (queue new runs). Set to true for agentic workflows
  # where only the latest run matters (e.g., PR analysis that becomes stale when new
  # commits are pushed).
  # (optional)
  cancel-in-progress: true

  # Additional discriminator expression appended to compiler-generated job-level
  # concurrency groups (agent, output jobs). Use this when multiple workflow
  # instances are dispatched concurrently with different inputs (fan-out pattern) to
  # prevent job-level concurrency groups from colliding. For example, '${{
  # inputs.finding_id }}' ensures each dispatched run gets a unique job-level group.
  # Supports GitHub Actions expressions. This field is stripped from the compiled
  # lock file (it is a gh-aw extension, not a GitHub Actions field).
  # (optional)
  job-discriminator: "example-value"

# Environment variables for the workflow
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
env:
  {}

# Option 2: string
env: "example-value"

# Feature flags and configuration options for experimental or optional features in
# the workflow. Each feature can be a boolean flag or a string value. The
# 'action-tag' feature (string) specifies the tag or SHA to use when referencing
# actions/setup in compiled workflows (for testing purposes only).
# (optional)
features:
  {}

# DEPRECATED: Use 'disable-model-invocation' instead. Controls whether the custom
# agent should infer additional context from the conversation. This field is
# maintained for backward compatibility with existing custom agent files.
# (optional)
infer: true

# Controls whether the custom agent should disable model invocation. When set to
# true, the agent will not make additional model calls. This is the preferred
# field name for custom agent files (replaces the deprecated 'infer' field).
# (optional)
disable-model-invocation: true

# Secret values passed to workflow execution. Secrets can be defined as simple
# strings (GitHub Actions expressions) or objects with 'value' and 'description'
# properties. Typically used to provide secrets to MCP servers or custom engines.
# Note: For passing secrets to reusable workflows, use the jobs.<job_id>.secrets
# field instead.
# (optional)
secrets:
  {}

# Environment that the job references (for protected environments and deployments)
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Environment name as a string
environment: "example-value"

# Option 2: Environment object with name and optional URL
environment:
  # The name of the environment configured in the repo
  name: "My Workflow"

  # A deployment URL
  # (optional)
  url: "example-value"

# Container to run the job steps in
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Docker image name (e.g., 'node:18', 'ubuntu:latest')
container: "example-value"

# Option 2: Container configuration object
container:
  # The Docker image to use as the container
  image: "example-value"

  # Credentials for private registries
  # (optional)
  credentials:
    # Username for Docker registry authentication when pulling private container
    # images.
    # (optional)
    username: "example-value"

    # Password or access token for Docker registry authentication. Should use secrets
    # syntax: ${{ secrets.DOCKER_PASSWORD }}
    # (optional)
    password: "example-value"

  # Environment variables for the container
  # (optional)
  env:
    {}

  # Ports to expose on the container
  # (optional)
  ports: []

  # Volumes for the container
  # (optional)
  volumes: []
    # Array of strings

  # Additional Docker container options
  # (optional)
  options: "example-value"

# Service containers for the job
# (optional)
services:
  {}

# Network access control for AI engines using ecosystem identifiers and domain
# allowlists. Supports wildcard patterns like '*.example.com' to match any
# subdomain. Controls web fetch and search capabilities. IMPORTANT: For workflows
# that build/install/test code, always include the language ecosystem identifier
# alongside 'defaults' — 'defaults' alone only covers basic infrastructure, not
# package registries. Key ecosystem identifiers by runtime: 'dotnet' (.NET/NuGet),
# 'python' (pip/PyPI), 'node' (npm/yarn), 'go' (go modules), 'java'
# (Maven/Gradle), 'ruby' (Bundler), 'rust' (Cargo), 'swift' (Swift PM). Example: a
# .NET project needs network: { allowed: [defaults, dotnet] }.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Use default network permissions (basic infrastructure: certificates,
# JSON schema, Ubuntu, etc.)
network: "defaults"

# Option 2: Custom network access configuration with ecosystem identifiers and
# specific domains
network:
  # List of allowed domains or ecosystem identifiers (e.g., 'defaults', 'python',
  # 'node', '*.example.com'). Wildcard patterns match any subdomain AND the base
  # domain.
  # (optional)
  allowed: []
    # Array of Domain name or ecosystem identifier. Supports wildcards like
    # '*.example.com' (matches sub.example.com, deep.nested.example.com, and
    # example.com itself). Ecosystem identifiers by runtime: 'dotnet' (.NET/NuGet),
    # 'python' (pip/PyPI), 'node' (npm/yarn), 'go' (go modules), 'java'
    # (Maven/Gradle), 'ruby' (RubyGems), 'rust' (Cargo), 'swift' (Swift PM), 'php'
    # (Composer), 'dart' (pub.dev), 'haskell' (Hackage), 'perl' (CPAN), 'containers'
    # (Docker/GHCR), 'github' (GitHub domains), 'terraform' (HashiCorp),
    # 'linux-distros' (apt/yum), 'playwright' (browser testing), 'defaults' (basic
    # infrastructure).

  # List of blocked domains or ecosystem identifiers (e.g., 'python', 'node',
  # 'tracker.example.com'). Blocked domains take precedence over allowed domains.
  # (optional)
  blocked: []
    # Array of Domain name or ecosystem identifier to block. Supports wildcards like
    # '*.example.com' (matches sub.example.com, deep.nested.example.com, and
    # example.com itself) and ecosystem names like 'python', 'node'.

# Sandbox configuration for AI engines. Controls agent sandbox (AWF) and MCP
# gateway. The MCP gateway is always enabled and cannot be disabled.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: String format for sandbox type: 'default' for no sandbox, 'awf' for
# Agent Workflow Firewall. Note: Legacy 'srt' and 'sandbox-runtime' values are
# automatically migrated to 'awf'
sandbox: "default"

# Option 2: Object format for full sandbox configuration with agent and mcp
# options
sandbox:
  # Legacy sandbox type field (use agent instead). Note: Legacy 'srt' and
  # 'sandbox-runtime' values are automatically migrated to 'awf'
  # (optional)
  type: "default"

  # Agent sandbox type: 'awf' uses AWF (Agent Workflow Firewall), or false to
  # disable agent sandbox. Defaults to 'awf' if not specified. Note: Disabling the
  # agent sandbox (false) removes firewall protection but keeps the MCP gateway
  # enabled.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Set to false to disable the agent sandbox (firewall). Warning: This
  # removes firewall protection but keeps the MCP gateway enabled. Not allowed in
  # strict mode.
  agent: true

  # Option 2: Sandbox type: 'awf' for Agent Workflow Firewall
  agent: "awf"

  # Option 3: Custom sandbox runtime configuration
  agent:
    # Agent identifier (replaces 'type' field in new format): 'awf' for Agent Workflow
    # Firewall
    # (optional)
    id: "awf"

    # Legacy: Sandbox type to use (use 'id' instead)
    # (optional)
    type: "awf"

    # AWF version override used to install and run the matching firewall version.
    # (optional)
    version: "example-value"

    # Container mounts to add when using AWF. Each mount is specified using Docker
    # mount syntax: 'source:destination:mode' where mode can be 'ro' (read-only) or
    # 'rw' (read-write). Example: '/host/path:/container/path:ro'
    # (optional)
    mounts: []
      # Array of Mount specification in format 'source:destination:mode'

    # Memory limit for the AWF container (e.g., '4g', '8g'). Passed as --memory-limit
    # to AWF. If not specified, AWF's default memory limit is used.
    # (optional)
    memory: "example-value"

    # Custom sandbox runtime configuration. Note: Network configuration is controlled
    # by the top-level 'network' field, not here.
    # (optional)
    config:
      # Filesystem access control configuration for the agent within the sandbox.
      # Controls read/write permissions and path restrictions.
      # (optional)
      filesystem:
        # List of paths to deny read access
        # (optional)
        denyRead: []
          # Array of strings

        # List of paths to allow write access
        # (optional)
        allowWrite: []
          # Array of strings

        # List of paths to deny write access
        # (optional)
        denyWrite: []
          # Array of strings

      # Map of command patterns to paths that should ignore violations
      # (optional)
      ignoreViolations:
        {}

      # Enable weaker nested sandbox mode (recommended: true for Docker access)
      # (optional)
      enableWeakerNestedSandbox: true

  # Legacy custom Sandbox Runtime configuration (use agent.config instead). Note:
  # Network configuration is controlled by the top-level 'network' field, not here.
  # (optional)
  config:
    # Filesystem access control configuration for sandboxed workflows. Controls
    # read/write permissions and path restrictions for file operations.
    # (optional)
    filesystem:
      # Array of path patterns that deny read access in the sandboxed environment. Takes
      # precedence over other read permissions.
      # (optional)
      denyRead: []
        # Array of strings

      # Array of path patterns that allow write access in the sandboxed environment.
      # Paths outside these patterns are read-only.
      # (optional)
      allowWrite: []
        # Array of strings

      # Array of path patterns that deny write access in the sandboxed environment.
      # Takes precedence over other write permissions.
      # (optional)
      denyWrite: []
        # Array of strings

    # When true, log sandbox violations without blocking execution. Useful for
    # debugging and gradual enforcement of sandbox policies.
    # (optional)
    ignoreViolations:
      {}

    # When true, allows nested sandbox processes to run with relaxed restrictions.
    # Required for certain containerized tools that spawn subprocesses.
    # (optional)
    enableWeakerNestedSandbox: true

  # MCP Gateway configuration for routing MCP server calls through a unified HTTP
  # gateway. Requires the 'mcp-gateway' feature flag to be enabled. Per MCP Gateway
  # Specification v1.0.0: Only container-based execution is supported.
  # (optional)
  mcp:
    # Volume mounts for the MCP gateway container. Each mount is specified using
    # Docker mount syntax: 'source:destination:mode' where mode can be 'ro'
    # (read-only) or 'rw' (read-write). Example: '/host/data:/container/data:ro'
    # (optional)
    mounts: []
      # Array of Mount specification in format 'source:destination:mode'

    # Environment variables for MCP gateway
    # (optional)
    env:
      {}

    # Port number for the MCP gateway HTTP server (default: 8080)
    # (optional)
    port: 1

    # API key for authenticating with the MCP gateway (supports ${{ secrets.* }}
    # syntax)
    # (optional)
    api-key: "example-value"

    # Gateway domain for URL generation (default: 'host.docker.internal' when agent is
    # enabled, 'localhost' when disabled)
    # (optional)
    domain: "localhost"

    # Keepalive ping interval in seconds for HTTP MCP backends. Sends periodic pings
    # to prevent session expiry during long-running agent tasks. Set to -1 to disable
    # keepalive pings. Unset or 0 uses the gateway default (1500 seconds = 25
    # minutes).
    # (optional)
    keepalive-interval: 1

# Conditional execution expression
# (optional)
if: "example-value"

# Custom workflow steps
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
steps:
  {}

# Option 2: array
steps: []
  # Array items: undefined

# Custom workflow steps to run at the very beginning of the agent job, before
# checkout and any other built-in steps. Use pre-steps to mint short-lived tokens
# or perform any setup that must happen before the repository is checked out. Step
# outputs are available via ${{ steps.<id>.outputs.<name> }} and can be referenced
# in checkout.token to avoid masked-value cross-job-boundary issues.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
pre-steps:
  {}

# Option 2: array
pre-steps: []
  # Array items: undefined

# Custom workflow steps to run immediately before AI execution, after all
# initialization and setup steps in the agent job.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
pre-agent-steps:
  {}

# Option 2: array
pre-agent-steps: []
  # Array items: undefined

# Custom workflow steps to run after AI execution
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
post-steps:
  {}

# Option 2: array
post-steps: []
  # Array items: undefined

# AI engine configuration that specifies which AI processor interprets and
# executes the markdown content of the workflow. Defaults to 'copilot'.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Engine name: built-in ('claude', 'codex', 'copilot', 'gemini',
# 'crush') or a named catalog entry
engine: "example-value"

# Option 2: Extended engine configuration object with advanced options for model
# selection, turn limiting, environment variables, and custom steps
engine:
  # AI engine identifier: built-in ('claude', 'codex', 'copilot', 'gemini', 'crush')
  # or a named catalog entry
  id: "example-value"

  # Optional version of the AI engine action (e.g., 'beta', 'stable', 20). Has
  # sensible defaults and can typically be omitted. Numeric values are automatically
  # converted to strings at runtime. GitHub Actions expressions (e.g., '${{
  # inputs.engine-version }}') are accepted and compiled with injection-safe env var
  # handling.
  # (optional)
  version: null

  # Optional specific LLM model to use (e.g., 'claude-3-5-sonnet-20241022',
  # 'gpt-4'). Has sensible defaults and can typically be omitted.
  # (optional)
  model: "example-value"

  # Maximum number of chat iterations per run. Helps prevent runaway loops and
  # control costs. Has sensible defaults and can typically be omitted. Note: Only
  # supported by the claude engine.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Maximum number of chat iterations per run as an integer value
  max-turns: 1

  # Option 2: Maximum number of chat iterations per run as a string value
  max-turns: "example-value"

  # Maximum number of continuations for multi-run autopilot mode. Default is 1
  # (single run, no autopilot). Values greater than 1 enable --autopilot mode for
  # the copilot engine with --max-autopilot-continues set to this value. Note: Only
  # supported by the copilot engine.
  # (optional)
  max-continuations: 1

  # Agent job concurrency configuration. Defaults to single job per engine across
  # all workflows (group: 'gh-aw-{engine-id}'). Supports full GitHub Actions
  # concurrency syntax.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple concurrency group name. Gets converted to GitHub Actions
  # concurrency format with the specified group.
  concurrency: "example-value"

  # Option 2: GitHub Actions concurrency configuration for the agent job. Controls
  # how many agentic workflow runs can run concurrently.
  concurrency:
    # Concurrency group identifier. Use GitHub Actions expressions like ${{
    # github.workflow }} or ${{ github.ref }}. Defaults to 'gh-aw-{engine-id}' if not
    # specified.
    group: "example-value"

    # Whether to cancel in-progress runs of the same concurrency group. Defaults to
    # false for agentic workflow runs.
    # (optional)
    cancel-in-progress: true

  # Custom user agent string for GitHub MCP server configuration (codex engine only)
  # (optional)
  user-agent: "example-value"

  # Custom executable path for the AI engine CLI. When specified, the workflow will
  # skip the standard installation steps and use this command instead. The command
  # should be the full path to the executable or a command available in PATH.
  # (optional)
  command: "example-value"

  # Custom Node.js harness script filename for an agentic engine. This replaces the
  # engine's built-in harness wrapper (when the engine supports one) and must end
  # with .js, .cjs, or .mjs.
  # (optional)
  harness: "example-value"

  # Custom environment variables to pass to the AI engine, including secret
  # overrides (e.g., OPENAI_API_KEY: ${{ secrets.CUSTOM_KEY }})
  # (optional)
  env:
    {}

  # Additional TOML configuration text that will be appended to the generated
  # config.toml in the action (codex engine only)
  # (optional)
  config: "example-value"

  # Agent identifier to pass to copilot --agent flag (copilot engine only).
  # Specifies which custom agent to use for the workflow.
  # (optional)
  agent: "example-value"

  # Custom API endpoint hostname for the agentic engine. Used for GitHub Enterprise
  # Cloud (GHEC), GitHub Enterprise Server (GHES), or custom AI endpoints. Example:
  # 'api.acme.ghe.com' for GHEC, 'api.enterprise.githubcopilot.com' for GHES, or
  # custom endpoint hostnames.
  # (optional)
  api-target: "example-value"

  # Custom model token weights for effective token computation. Overrides or extends
  # the built-in model multipliers from model_multipliers.json. Useful for custom
  # models or adjusted cost ratios.
  # (optional)
  token-weights:
    # Per-model cost multipliers relative to the reference model (claude-sonnet-4.5 =
    # 1.0). Keys are model names (case-insensitive, prefix matching supported). Values
    # are numeric multipliers.
    # (optional)
    multipliers:
      {}

    # Per-token-class weights applied before the model multiplier. Any specified
    # weight overrides the corresponding default.
    # (optional)
    token-class-weights:
      # Weight for input tokens (default: 1.0)
      # (optional)
      input: 1

      # Weight for cached input tokens (default: 0.1)
      # (optional)
      cached-input: 1

      # Weight for output tokens (default: 4.0)
      # (optional)
      output: 1

      # Weight for reasoning tokens (default: 4.0)
      # (optional)
      reasoning: 1

      # Weight for cache write tokens (default: 1.0)
      # (optional)
      cache-write: 1

  # Optional array of command-line arguments to pass to the AI engine CLI. These
  # arguments are injected after all other args but before the prompt.
  # (optional)
  args: []
    # Array of strings

  # When true, disables automatic loading of context and custom instructions by the
  # AI engine. The engine-specific flag depends on the engine: copilot uses
  # --no-custom-instructions (suppresses .github/AGENTS.md and user-level custom
  # instructions), claude uses --bare (suppresses CLAUDE.md memory files), codex
  # uses --no-system-prompt (suppresses the default system prompt), gemini sets
  # GEMINI_SYSTEM_MD=/dev/null (overrides the built-in system prompt with an empty
  # one). Defaults to false.
  # (optional)
  bare: true

# Option 3: Inline engine definition: specifies a runtime adapter and optional
# provider settings directly in the workflow frontmatter, without requiring a
# named catalog entry
engine:
  # Runtime adapter reference for the inline engine definition
  runtime:
    # Runtime adapter identifier (e.g. 'codex', 'claude', 'copilot', 'gemini',
    # 'crush')
    id: "example-value"

    # Optional version of the runtime adapter (e.g. '0.105.0', 'beta')
    # (optional)
    version: null

  # Optional provider configuration for the inline engine definition
  # (optional)
  provider:
    # Provider identifier (e.g. 'openai', 'anthropic', 'github', 'google')
    # (optional)
    id: "example-value"

    # Optional specific LLM model to use (e.g. 'gpt-5', 'claude-3-5-sonnet-20241022')
    # (optional)
    model: "example-value"

    # Authentication configuration for the provider
    # (optional)
    auth:
      # Name of the GitHub Actions secret that contains the API key for this provider
      # (optional)
      secret: "example-value"

      # Authentication strategy for the provider (default: api-key when secret is set)
      # (optional)
      strategy: "api-key"

      # OAuth 2.0 token endpoint URL. Required when strategy is
      # 'oauth-client-credentials'.
      # (optional)
      token-url: "example-value"

      # GitHub Actions secret name that holds the OAuth client ID. Required when
      # strategy is 'oauth-client-credentials'.
      # (optional)
      client-id: "example-value"

      # GitHub Actions secret name that holds the OAuth client secret. Required when
      # strategy is 'oauth-client-credentials'.
      # (optional)
      client-secret: "example-value"

      # JSON field name in the token response that contains the access token. Defaults
      # to 'access_token'.
      # (optional)
      token-field: "example-value"

      # HTTP header name to inject the API key or token into (e.g. 'api-key',
      # 'x-api-key'). Required when strategy is not 'bearer'.
      # (optional)
      header-name: "example-value"

    # Request shaping configuration for non-standard provider URL and body
    # transformations
    # (optional)
    request:
      # URL path template with {model} and other variable placeholders (e.g.
      # '/openai/deployments/{model}/chat/completions')
      # (optional)
      path-template: "example-value"

      # Static or template query-parameter values appended to every request
      # (optional)
      query:
        {}

      # Key/value pairs injected into the JSON request body before sending
      # (optional)
      body-inject:
        {}

  # When true, disables automatic loading of context and custom instructions by the
  # AI engine. The engine-specific flag depends on the engine: copilot uses
  # --no-custom-instructions, claude uses --bare, codex uses --no-system-prompt,
  # gemini sets GEMINI_SYSTEM_MD=/dev/null. Defaults to false.
  # (optional)
  bare: true

# Option 4: Engine definition: full declarative metadata for a named engine entry
# (used in builtin engine shared workflow files such as @builtin:engines/*.md)
engine:
  # Unique engine identifier (e.g. 'copilot', 'claude', 'codex', 'gemini', 'crush')
  id: "example-value"

  # Human-readable display name for the engine
  display-name: "example-value"

  # Human-readable description of the engine
  # (optional)
  description: "Description of the workflow"

  # Runtime adapter identifier. Maps to the CodingAgentEngine registered in the
  # engine registry. Defaults to id when omitted.
  # (optional)
  runtime-id: "example-value"

  # Provider metadata for the engine
  # (optional)
  provider:
    # Provider name (e.g. 'anthropic', 'github', 'google', 'openai')
    # (optional)
    name: "My Workflow"

    # Default authentication configuration for the provider
    # (optional)
    auth:
      # Name of the GitHub Actions secret that contains the API key
      # (optional)
      secret: "example-value"

      # Authentication strategy
      # (optional)
      strategy: "api-key"

      # OAuth 2.0 token endpoint URL
      # (optional)
      token-url: "example-value"

      # GitHub Actions secret name for the OAuth client ID
      # (optional)
      client-id: "example-value"

      # GitHub Actions secret name for the OAuth client secret
      # (optional)
      client-secret: "example-value"

      # JSON field name in the token response containing the access token
      # (optional)
      token-field: "example-value"

      # HTTP header name to inject the API key or token into
      # (optional)
      header-name: "example-value"

    # Request shaping configuration
    # (optional)
    request:
      # URL path template with variable placeholders
      # (optional)
      path-template: "example-value"

      # Static query parameters
      # (optional)
      query:
        {}

      # Key/value pairs injected into the JSON request body
      # (optional)
      body-inject:
        {}

  # Model selection configuration for the engine
  # (optional)
  models:
    # Default model identifier
    # (optional)
    default: "example-value"

    # List of supported model identifiers
    # (optional)
    supported: []
      # Array of strings

  # Authentication bindings — maps logical roles (e.g. 'api-key') to GitHub Actions
  # secret names
  # (optional)
  auth: []
    # Array items:
      # Logical authentication role (e.g. 'api-key', 'token')
      role: "example-value"

      # Name of the GitHub Actions secret that provides credentials for this role
      secret: "example-value"

  # Additional engine-specific options
  # (optional)
  options:
    {}

# MCP server definitions
# (optional)
mcp-servers:
  {}

# Tools and MCP (Model Context Protocol) servers available to the AI engine for
# GitHub API access, browser automation, file editing, and more
# (optional)
tools:
  # GitHub API tools for repository operations (issues, pull requests, content
  # management)
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Empty GitHub tool configuration (enables all read-only GitHub API
  # functions)
  github: null

  # Option 2: Boolean to explicitly enable (true) or disable (false) the GitHub MCP
  # server. When set to false, the GitHub MCP server is not mounted.
  github: true

  # Option 3: Simple GitHub tool configuration (enables all GitHub API functions)
  github: "example-value"

  # Option 4: GitHub tools object configuration with restricted function access
  github:
    # List of allowed GitHub API functions (e.g., 'create_issue', 'update_issue',
    # 'add_comment')
    # (optional)
    allowed: []
      # Array of strings

    # GitHub access mode. Prefer 'gh-proxy' for better performance (uses
    # pre-authenticated gh CLI prompt guidance). Legacy MCP transport values 'local'
    # and 'remote' are accepted for backward compatibility and use GitHub MCP server
    # prompt guidance.
    # (optional)
    mode: "gh-proxy"

    # GitHub MCP transport type: 'local' (Docker-based, default) or 'remote' (hosted
    # at api.githubcopilot.com)
    # (optional)
    type: "local"

    # Optional version specification for the GitHub MCP server (used with 'local'
    # type). Can be a string (e.g., 'v1.0.0', 'latest') or number (e.g., 20, 3.11).
    # Numeric values are automatically converted to strings at runtime.
    # (optional)
    version: null

    # Optional additional arguments to append to the generated MCP server command
    # (used with 'local' type)
    # (optional)
    args: []
      # Array of strings

    # Enable read-only mode to restrict GitHub MCP server to read-only operations only
    # (optional)
    read-only: true

    # Enable lockdown mode to limit content surfaced from public repositories (only
    # items authored by users with push access). Default: false
    # (optional)
    lockdown: true

    # Controls DIFC proxy injection for pre-agent gh CLI steps when guard policies
    # (min-integrity) are configured. Default: true (enabled). Set to false to disable
    # proxy injection and rely solely on MCP gateway-level filtering.
    # (optional)
    integrity-proxy: true

    # Optional custom GitHub token (e.g., '${{ secrets.CUSTOM_PAT }}'). For 'remote'
    # type, defaults to GH_AW_GITHUB_TOKEN if not specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # GitHub MCP server toolset name(s) to enable. Accepts a single toolset name
    # (string) or an array of toolset names.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: A single GitHub MCP server toolset name (shorthand for a one-element
    # array)

    # Option 2: Array of GitHub MCP server toolset names to enable specific groups of
    # GitHub API functionalities
    toolsets: []
      # Array items: undefined

    # Volume mounts for the containerized GitHub MCP server (format:
    # 'host:container:mode' where mode is 'ro' for read-only or 'rw' for read-write).
    # Applies to local mode only. Example: '/data:/data:ro'
    # (optional)
    mounts: []
      # Array of Mount specification in format 'host:container:mode'

    # Guard policy: repository access configuration. Restricts which repositories the
    # agent can access. Use 'all' to allow all repos, 'public' for public repositories
    # only, or an array of repository patterns (e.g., 'owner/repo', 'owner/*',
    # 'owner/prefix*').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Allow access to all repositories ('all') or only public repositories
    # ('public')
    allowed-repos: "all"

    # Option 2: Allow access to specific repositories using patterns (e.g.,
    # 'owner/repo', 'owner/*', 'owner/prefix*')
    allowed-repos: []
      # Array items: Repository pattern in the format 'owner/repo', 'owner/*' (all repos
      # under owner), or 'owner/prefix*' (repos with name prefix)

    # Guard policy: minimum required integrity level for repository access. Restricts
    # the agent to users with at least the specified permission level.
    # (optional)
    min-integrity: "none"

    # Guard policy: GitHub usernames whose content is unconditionally blocked. Items
    # from these users receive 'blocked' integrity (below 'none') and are always
    # denied, even when 'min-integrity' is 'none'. Cannot be overridden by
    # 'approval-labels'. Requires 'min-integrity' to be set. Accepts an array of
    # usernames, a comma-separated string, a newline-separated string, or a GitHub
    # Actions expression (e.g. '${{ vars.BLOCKED_USERS }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Array of GitHub usernames to block
    blocked-users: []
      # Array items: GitHub username to block

    # Option 2: Comma- or newline-separated list of usernames, or a GitHub Actions
    # expression resolving to such a list (e.g. '${{ vars.BLOCKED_USERS }}')
    blocked-users: "example-value"

    # Guard policy: GitHub usernames whose content is elevated to 'approved' integrity
    # regardless of author_association. Allows specific external contributors to
    # bypass 'min-integrity' checks without lowering the global policy. Precedence:
    # blocked-users > trusted-users > approval-labels > author_association. Requires
    # 'min-integrity' to be set. Accepts an array of usernames, a comma-separated
    # string, a newline-separated string, or a GitHub Actions expression (e.g. '${{
    # vars.TRUSTED_USERS }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Array of GitHub usernames to trust
    trusted-users: []
      # Array items: GitHub username to elevate to approved integrity

    # Option 2: Comma- or newline-separated list of usernames, or a GitHub Actions
    # expression resolving to such a list (e.g. '${{ vars.TRUSTED_USERS }}')
    trusted-users: "example-value"

    # Guard policy: GitHub label names that promote a content item's effective
    # integrity to 'approved' when present. Enables human-review gates where a
    # maintainer labels an item to allow it through. Uses max(base, approved) so it
    # never lowers integrity. Does not override 'blocked-users'. Requires
    # 'min-integrity' to be set. Accepts an array of label names, a comma-separated
    # string, a newline-separated string, or a GitHub Actions expression (e.g. '${{
    # vars.APPROVAL_LABELS }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Array of GitHub label names
    approval-labels: []
      # Array items: GitHub label name

    # Option 2: Comma- or newline-separated list of label names, or a GitHub Actions
    # expression resolving to such a list (e.g. '${{ vars.APPROVAL_LABELS }}')
    approval-labels: "example-value"

    # Guard policy: GitHub reaction types that promote a content item's integrity to
    # 'approved' when added by maintainers. Only enforced in proxy mode (DIFC/CLI
    # proxy); ignored in MCP gateway mode because reaction authors cannot be
    # identified. Optional; defaults to ["THUMBS_UP", "HEART"] when the
    # integrity-reactions feature flag is enabled. Requires 'min-integrity' to be set
    # and MCPG >= v0.2.18.
    # (optional)
    endorsement-reactions: []
      # Array of GitHub ReactionContent enum value

    # Guard policy: GitHub reaction types that demote content integrity when added by
    # maintainers. Only enforced in proxy mode (DIFC/CLI proxy); ignored in MCP
    # gateway mode because reaction authors cannot be identified. Optional; defaults
    # to ["THUMBS_DOWN", "CONFUSED"] when the integrity-reactions feature flag is
    # enabled. Disapproval overrides endorsement (safe default). Requires
    # 'min-integrity' to be set and MCPG >= v0.2.18.
    # (optional)
    disapproval-reactions: []
      # Array of GitHub ReactionContent enum value

    # Guard policy: integrity level assigned when a disapproval reaction is present.
    # Optional, defaults to 'none'. Requires the 'integrity-reactions' feature flag
    # and MCPG >= v0.2.18.
    # (optional)
    disapproval-integrity: "none"

    # Guard policy: minimum integrity level required for an endorser (reactor) to
    # promote content. Optional, defaults to 'approved'. Requires the
    # 'integrity-reactions' feature flag and MCPG >= v0.2.18.
    # (optional)
    endorser-min-integrity: "unapproved"

    # GitHub App configuration for token minting. When configured, a GitHub App
    # installation access token is minted at workflow start and used instead of the
    # default token. This token overrides any custom github-token setting and provides
    # fine-grained permissions matching the agent job requirements.
    # (optional)
    # This field supports multiple formats (anyOf):

    # Option 1: undefined

    # Option 2: undefined

  # Bash shell command execution tool. Supports wildcards: '*' (all commands),
  # 'command *' (command with any args, e.g., 'date *', 'echo *'). Default safe
  # commands: echo, ls, pwd, cat, head, tail, grep, wc, sort, uniq, date.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable bash tool with all shell commands allowed (security
  # consideration: use restricted list in production)
  bash: null

  # Option 2: Enable bash tool - true allows all commands (equivalent to ['*']),
  # false disables the tool
  bash: true

  # Option 3: List of allowed commands and patterns. Wildcards: '*' allows all
  # commands, 'command *' allows command with any args (e.g., 'date *', 'echo *').
  bash: []
    # Array items: Command or pattern: 'echo' (exact match), 'echo *' (command with
    # any args)

  # Web content fetching tool for downloading web pages and API responses (subject
  # to network permissions)
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable web fetch tool with default configuration
  web-fetch: null

  # Option 2: Web fetch tool configuration object
  web-fetch:
    {}

  # Web search tool for performing internet searches and retrieving search results
  # (subject to network permissions)
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable web search tool with default configuration
  web-search: null

  # Option 2: Web search tool configuration object
  web-search:
    {}

  # File editing tool for reading, creating, and modifying files in the repository
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable edit tool
  edit: null

  # Option 2: Edit tool configuration object
  edit:
    {}

  # Playwright browser automation tool for web scraping, testing, and UI
  # interactions in containerized browsers
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable Playwright tool with default settings
  playwright: null

  # Option 2: Playwright tool configuration with custom version and arguments
  playwright:
    # Optional Playwright container version (e.g., 'v1.41.0', 1.41, 20). Numeric
    # values are automatically converted to strings at runtime.
    # (optional)
    version: null

    # Optional additional arguments to append to the generated MCP server command
    # (optional)
    args: []
      # Array of strings

  # GitHub Agentic Workflows MCP server for workflow introspection and analysis.
  # Provides tools for checking status, compiling workflows, downloading logs, and
  # auditing runs.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable agentic-workflows tool with default settings
  agentic-workflows: true

  # Option 2: Enable agentic-workflows tool with default settings (same as true)
  agentic-workflows: null

  # Cache memory MCP configuration for persistent memory storage
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable cache-memory with default settings
  cache-memory: true

  # Option 2: Enable cache-memory with default settings (same as true)
  cache-memory: null

  # Option 3: Cache-memory configuration object
  cache-memory:
    # Custom cache key for memory MCP data (restore keys are auto-generated by
    # splitting on '-')
    # (optional)
    key: "example-value"

    # Optional description for the cache that will be shown in the agent prompt
    # (optional)
    description: "Description of the workflow"

    # Number of days to retain uploaded artifacts (1-90 days, default: repository
    # setting)
    # (optional)
    retention-days: 1

    # If true, only restore the cache without saving it back. Uses
    # actions/cache/restore instead of actions/cache. No artifact upload step will be
    # generated.
    # (optional)
    restore-only: true

    # Cache restore key scope: 'workflow' (default, only restores from same workflow)
    # or 'repo' (restores from any workflow in the repository). Use 'repo' with
    # caution as it allows cross-workflow cache sharing.
    # (optional)
    scope: "workflow"

    # List of allowed file extensions (e.g., [".json", ".txt"]). Default: [".json",
    # ".jsonl", ".txt", ".md", ".csv"]
    # (optional)
    allowed-extensions: []
      # Array of strings

  # Option 4: Array of cache-memory configurations for multiple caches
  cache-memory: []
    # Array items: object

  # Comment memory configuration for managed comment persistence
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for persisting memory in a managed issue/PR comment.
  # Memory is materialized to files for agent editing and synchronized back after
  # execution.
  comment-memory:
    # Maximum number of comment_memory updates to process (default: 1). Supports
    # integer or GitHub Actions expression.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for comment memory: 'triggering' (default), '*' (current issue/PR), or
    # explicit issue/PR number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository memory storage.
    # (optional)
    target-repo: "example-value"

    # Additional repositories in format 'owner/repo' allowed for comment memory
    # operations.
    # (optional)
    allowed-repos: []
      # Array of strings

    # Default memory identifier when output items omit memory_id.
    # (optional)
    memory-id: "example-value"

    # Controls whether AI-generated footer is added to the managed comment. Defaults
    # to true.
    # (optional)
    footer: true

    # GitHub token to use for comment-memory operations. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable (true) or disable (false) comment-memory.
  comment-memory: true

  # Option 3: Explicitly disable comment-memory
  comment-memory: null

  # Timeout in seconds for tool/MCP server operations. Applies to all tools and MCP
  # servers if supported by the engine. Default: 60 seconds (for both Claude and
  # Codex). Supports GitHub Actions expressions for reusable workflow_call
  # workflows.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: integer
  timeout: 1

  # Option 2: GitHub Actions expression (e.g. '${{ inputs.tool-timeout }}')
  timeout: "example-value"

  # Timeout in seconds for MCP server startup. Applies to MCP server initialization
  # if supported by the engine. Default: 120 seconds. Supports GitHub Actions
  # expressions for reusable workflow_call workflows.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: integer
  startup-timeout: 1

  # Option 2: GitHub Actions expression (e.g. '${{ inputs.startup-timeout }}')
  startup-timeout: "example-value"

  # When true, each user-facing MCP server is mounted as a standalone CLI tool on
  # PATH. The agent can then call MCP servers via shell commands (e.g. 'github
  # issue_read --method get ...'). CLI-mounted servers remain in the MCP gateway
  # config so their containers can start, and are removed only from the agent's
  # final config during convert_gateway_config_*.sh processing. Default: false.
  # (optional)
  cli-proxy: true

  # Repo memory configuration for git-based persistent storage
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable repo-memory with default settings
  repo-memory: true

  # Option 2: Enable repo-memory with default settings (same as true)
  repo-memory: null

  # Option 3: Repo-memory configuration object
  repo-memory:
    # Branch prefix for memory storage (default: 'memory'). Must be 4-32 characters,
    # alphanumeric with hyphens/underscores, and cannot be 'copilot'. Branch will be
    # named {branch-prefix}/{id}
    # (optional)
    branch-prefix: "example-value"

    # Target repository for memory storage (default: current repository). Format:
    # owner/repo
    # (optional)
    target-repo: "example-value"

    # Git branch name for memory storage (default: {branch-prefix}/default or
    # memory/default if branch-prefix not set)
    # (optional)
    branch-name: "example-value"

    # Glob patterns for files to include in repository memory. Supports wildcards
    # (e.g., '**/*.md', 'docs/**/*.json') to filter cached files.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single file glob pattern for allowed files
    file-glob: "example-value"

    # Option 2: Array of file glob patterns for allowed files
    file-glob: []
      # Array items: string

    # Maximum size per file in bytes (default: 10240 = 10KB)
    # (optional)
    max-file-size: 1

    # Maximum file count per commit (default: 100)
    # (optional)
    max-file-count: 1

    # Maximum total patch size in bytes (default: 10240 = 10KB, max: 102400 = 100KB).
    # The total size of the git diff must not exceed this value.
    # (optional)
    max-patch-size: 1

    # Optional description for the memory that will be shown in the agent prompt
    # (optional)
    description: "Description of the workflow"

    # Create orphaned branch if it doesn't exist (default: true)
    # (optional)
    create-orphan: true

    # Use the GitHub Wiki git repository instead of the regular repository. When
    # enabled, files are stored in and read from the wiki, and the agent will be
    # instructed to follow GitHub Wiki markdown syntax (default: false)
    # (optional)
    wiki: true

    # List of allowed file extensions (e.g., [".json", ".txt"]). Default: [".json",
    # ".jsonl", ".txt", ".md", ".csv"]
    # (optional)
    allowed-extensions: []
      # Array of strings

  # Option 4: Array of repo-memory configurations for multiple memory locations
  repo-memory: []
    # Array items: object

# Command name for the workflow
# (optional)
command: "example-value"

# Cache configuration for workflow (uses actions/cache syntax)
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Single cache configuration
cache:
  # An explicit key for restoring and saving the cache
  key: "example-value"

  # File path or directory to cache for faster workflow execution. Can be a single
  # path or an array of paths to cache multiple locations.
  # This field supports multiple formats (oneOf):

  # Option 1: A single path to cache
  path: "example-value"

  # Option 2: Multiple paths to cache
  path: []
    # Array items: string

  # Optional list of fallback cache key patterns to use if exact cache key is not
  # found. Enables partial cache restoration for better performance.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: A single restore key
  restore-keys: "example-value"

  # Option 2: Multiple restore keys
  restore-keys: []
    # Array items: string

  # The chunk size used to split up large files during upload, in bytes
  # (optional)
  upload-chunk-size: 1

  # Fail the workflow if cache entry is not found
  # (optional)
  fail-on-cache-miss: true

  # If true, only checks if cache entry exists and skips download
  # (optional)
  lookup-only: true

  # Optional custom name for the cache step (overrides auto-generated name)
  # (optional)
  name: "My Workflow"

# Option 2: Multiple cache configurations
cache: []
  # Array items: object

# Safe output processing configuration that automatically creates GitHub issues,
# comments, and pull requests from AI workflow output without requiring write
# permissions in the main job
# (optional)
safe-outputs:
  # List of allowed domains for URL redaction in safe output handlers. Supports
  # ecosystem identifiers (e.g., "python", "node", "default-safe-outputs") like
  # network.allowed. These domains are unioned with the engine defaults and
  # network.allowed when computing the final allowed domain set. localhost and
  # github.com are always included.
  # (optional)
  allowed-domains: []
    # Array of strings

  # List of allowed repositories for GitHub references (e.g., #123 or
  # owner/repo#456). Use 'repo' to allow current repository. References to other
  # repositories will be escaped with backticks. If not specified, all references
  # are allowed.
  # (optional)
  allowed-github-references: []
    # Array of strings

  # Enable AI agents to create GitHub issues from workflow output. Supports title
  # prefixes, automatic labeling, assignees, and cross-repository creation. Does not
  # require 'issues: write' permission.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for automatically creating GitHub issues from AI
  # workflow output. The main job does not need 'issues: write' permission.
  create-issue:
    # Optional prefix to add to the beginning of the issue title (e.g., '[ai] ' or
    # '[analysis] ')
    # (optional)
    title-prefix: "example-value"

    # Optional list of labels to automatically attach to created issues (e.g.,
    # ['automation', 'ai-generated'])
    # (optional)
    labels: []
      # Array of strings

    # Optional list of allowed labels that can be used when creating issues. If
    # omitted, any labels are allowed (including creating new ones). When specified,
    # the agent can only use labels from this list.
    # (optional)
    allowed-labels: []
      # Array of strings

    # GitHub usernames to assign the created issue to. Can be a single username string
    # or array of usernames. Use 'copilot' to assign to GitHub Copilot.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single GitHub username to assign the created issue to (e.g., 'user1'
    # or 'copilot'). Use 'copilot' to assign to GitHub Copilot using the @copilot
    # special value.
    assignees: "example-value"

    # Option 2: List of GitHub usernames to assign the created issue to (e.g.,
    # ['user1', 'user2', 'copilot']). Use 'copilot' to assign to GitHub Copilot using
    # the @copilot special value.
    assignees: []
      # Array items: string

    # Maximum number of issues to create (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository issue creation.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that issues can be
    # created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the issue in. The target repository (current
    # or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # Time until the issue expires and should be automatically closed. Supports
    # integer (days), relative time format, or false to disable expiration. Minimum
    # duration: 2 hours. When set, a maintenance workflow will be generated.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Number of days until expires
    expires: 1

    # Option 2: Relative time (e.g., '2h', '7d', '2w', '1m', '1y'); minimum 2h for
    # hour values
    expires: "example-value"

    # Option 3: Set to false to explicitly disable expiration
    expires: false

    # If true, group issues as sub-issues under a parent issue. The workflow ID is
    # used as the group identifier. Parent issues are automatically created and
    # managed, with a maximum of 64 sub-issues per parent.
    # (optional)
    group: true

    # When true, automatically close older issues with the same workflow-id marker as
    # 'not planned' with a comment linking to the new issue. Searches for issues
    # containing the workflow-id marker in their body. Maximum 10 issues will be
    # closed. Only runs if issue creation succeeds.
    # (optional)
    close-older-issues: true

    # Optional explicit deduplication key for close-older matching. When set, a `<!--
    # gh-aw-close-key: <value> -->` marker is embedded in the issue body and used as
    # the primary key for searching and filtering older issues instead of the
    # workflow-id markers. This gives deterministic isolation across caller workflows
    # and is stable across workflow renames. The value is normalized to identifier
    # style (lowercase alphanumeric, dashes, underscores).
    # (optional)
    close-older-key: "example-value"

    # When true, if an open issue with the same close-older-key (or workflow-id marker
    # when no key is set) was already created today (UTC), post the new content as a
    # comment on that existing issue instead of creating a new one. Groups multiple
    # same-day runs into a single issue. Works best when combined with
    # close-older-issues: true.
    # (optional)
    group-by-day: true

    # Controls whether AI-generated footer is added to the issue. When false, the
    # visible footer content is omitted but XML markers (workflow-id, tracker-id,
    # metadata) are still included for searchability. Defaults to true.
    # (optional)
    footer: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable issue creation with default configuration
  create-issue: null

  # Enable creation of GitHub Copilot coding agent tasks from workflow output.
  # Allows workflows to spawn new agent sessions for follow-up work.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: DEPRECATED: Use 'create-agent-session' instead. Configuration for
  # creating GitHub Copilot coding agent sessions from agentic workflow output using
  # gh agent-task CLI. The main job does not need write permissions.
  create-agent-task:
    # Base branch for the agent session pull request. Defaults to the current branch
    # or repository default branch.
    # (optional)
    base: "example-value"

    # Maximum number of agent sessions to create (default: 1) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository agent session
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that agent sessions can
    # be created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the agent session in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable agent session creation with default configuration
  create-agent-task: null

  # Enable creation of GitHub Copilot coding agent sessions from workflow output.
  # Allows workflows to start interactive agent conversations.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub Copilot coding agent sessions from
  # agentic workflow output using gh agent-task CLI. The main job does not need
  # write permissions.
  create-agent-session:
    # Base branch for the agent session pull request. Defaults to the current branch
    # or repository default branch.
    # (optional)
    base: "example-value"

    # Maximum number of agent sessions to create (default: 1) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository agent session
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that agent sessions can
    # be created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the agent session in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable agent session creation with default configuration
  create-agent-session: null

  # Enable AI agents to add items to GitHub Projects, update custom fields, and
  # manage project structure. Use this for organizing work into projects with status
  # tracking, priority management, and custom metadata.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for managing GitHub Projects boards. Enable agents to
  # add issues and pull requests to projects, update custom field values (status,
  # priority, effort, dates), create project fields and views. By default it is
  # update-only: if the project does not exist, the job fails with instructions to
  # create it. To allow workflows to create missing projects, explicitly opt in via
  # agent output field create_if_missing=true. Requires a Personal Access Token
  # (PAT) or GitHub App token with Projects permissions (default GITHUB_TOKEN cannot
  # be used). Agent output includes: project (full URL or temporary project ID like
  # aw_XXXXXXXXXXXX or #aw_XXXXXXXXXXXX from create_project), content_type
  # (issue|pull_request|draft_issue), content_number, fields, create_if_missing. For
  # specialized operations, agent can also provide: operation
  # (create_fields|create_view), field_definitions (array of field configs when
  # operation=create_fields), view (view config object when operation=create_view).
  update-project:
    # Maximum number of project operations to perform (default: 10). Each operation
    # may add a project item, or update its fields. Supports integer or GitHub Actions
    # expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Target project URL for update-project operations. This is required in the
    # configuration for documentation purposes. Agent messages MUST explicitly include
    # the project field in their output - the configured value is not used as a
    # fallback. Must be a valid GitHub Projects v2 URL.
    project: "example-value"

    # Default repository in format 'owner/repo' for cross-repository content
    # resolution. When specified, the agent can use 'target_repo' in agent output to
    # resolve issues or PRs from this repository. Wildcards ('*') are not allowed.
    # Supports GitHub Actions expression syntax (e.g., '${{ vars.TARGET_REPO }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: string
    target-repo: "example-value"

    # Option 2: GitHub Actions expression that resolves to owner/repo at runtime
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' allowed for
    # cross-repository content resolution via 'target_repo'. The target-repo (or
    # current repo) is always implicitly allowed. Supports wildcard patterns (e.g.,
    # 'org/*', '*/repo', '*') and GitHub Actions expression syntax for individual
    # entries.
    # (optional)
    allowed-repos: []

    # Optional array of project views to create. Each view must have a name and
    # layout. Views are created during project setup.
    # (optional)
    views: []
      # Array items:
        # The name of the view (e.g., 'Sprint Board', 'Roadmap')
        name: "My Workflow"

        # The layout type of the view
        layout: "table"

        # Optional filter query for the view (e.g., 'is:issue is:open', 'label:bug')
        # (optional)
        filter: "example-value"

        # Optional array of field IDs that should be visible in the view (table/board
        # only, not applicable to roadmap)
        # (optional)
        visible-fields: []

        # Optional human description for the view. Not supported by the GitHub Views API
        # and may be ignored.
        # (optional)
        description: "Description of the workflow"

    # Optional array of project custom fields to create up-front.
    # (optional)
    field-definitions: []
      # Array items:
        # The field name to create (e.g., 'status', 'priority')
        name: "My Workflow"

        # The GitHub Projects v2 custom field type
        data-type: "DATE"

        # Options for SINGLE_SELECT fields. GitHub does not support adding options later.
        # (optional)
        options: []
          # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable project management with default configuration (max=10)
  update-project: null

  # Enable AI agents to create new GitHub Projects for organizing and tracking work
  # across issues and pull requests.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating new GitHub Projects boards. Enables agents
  # to create new project boards with optional custom fields, views, and an initial
  # item. Requires a Personal Access Token (PAT) or GitHub App token with Projects
  # write permission (default GITHUB_TOKEN cannot be used). Agent output includes:
  # title (project name), owner (org/user login, uses default if omitted),
  # owner_type ('org' or 'user'), optional item_url (issue to add as first item),
  # and optional field_definitions. Returns a temporary project ID for use in
  # subsequent update_project operations.
  create-project:
    # Maximum number of create operations to perform (default: 1). Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Must have Projects write
    # permission. Overrides global github-token if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Optional default target owner (organization or user login, e.g., 'myorg' or
    # 'username') for the new project. If specified, the agent can omit the owner
    # field in the tool call and this default will be used. The agent can still
    # override by providing an owner in the tool call.
    # (optional)
    target-owner: "example-value"

    # Optional prefix for auto-generated project titles (default: 'Project'). When the
    # agent doesn't provide a title, the project title is auto-generated as
    # '<title-prefix>: <issue-title>' or '<title-prefix> #<issue-number>' based on the
    # issue context.
    # (optional)
    title-prefix: "example-value"

    # Optional array of project views to create automatically after project creation.
    # Each view must have a name and layout. Views are created immediately after the
    # project is created.
    # (optional)
    views: []
      # Array items:
        # The name of the view (e.g., 'Sprint Board', 'Roadmap')
        name: "My Workflow"

        # The layout type of the view
        layout: "table"

        # Optional filter query for the view (e.g., 'is:issue is:open', 'label:bug')
        # (optional)
        filter: "example-value"

        # Optional array of field IDs that should be visible in the view (table/board
        # only, not applicable to roadmap)
        # (optional)
        visible-fields: []

        # Optional human description for the view. Not supported by the GitHub Views API
        # and may be ignored.
        # (optional)
        description: "Description of the workflow"

    # Optional array of project custom fields to create automatically after project
    # creation.
    # (optional)
    field-definitions: []
      # Array items:
        # The field name to create (e.g., 'Priority', 'Classification')
        name: "My Workflow"

        # The GitHub Projects v2 custom field type
        data-type: "DATE"

        # Options for SINGLE_SELECT fields. GitHub does not support adding options later.
        # (optional)
        options: []
          # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable project creation with default configuration (max=1)
  create-project: null

  # Enable AI agents to post status updates to GitHub Projects for progress tracking
  # and stakeholder communication.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for posting status updates to GitHub Projects. Status
  # updates provide stakeholder communication about project progress, health, and
  # timeline. Each update appears in the project's Updates tab and creates a
  # historical record. Requires a Personal Access Token (PAT) or GitHub App token
  # with Projects read & write permission (default GITHUB_TOKEN cannot be used).
  # Typically used by scheduled workflows or orchestrators to post regular progress
  # summaries with status indicators (on-track, at-risk, off-track, complete,
  # inactive), dates, and progress details.
  create-project-status-update:
    # Maximum number of status updates to create (default: 1). Typically 1 per
    # orchestrator run. Supports integer or GitHub Actions expression (e.g. '${{
    # inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified. Must have Projects: Read+Write permission.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Target project URL for status update operations. This is required in the
    # configuration for documentation purposes. Agent messages MUST explicitly include
    # the project field in their output - the configured value is not used as a
    # fallback. Must be a valid GitHub Projects v2 URL.
    project: "example-value"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable project status updates with default configuration (max=1)
  create-project-status-update: null

  # Enable AI agents to create GitHub Discussions from workflow output. Supports
  # categorization, labeling, and automatic closure of older discussions. Does not
  # require 'discussions: write' permission.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub discussions from agentic workflow
  # output
  create-discussion:
    # Optional prefix for the discussion title
    # (optional)
    title-prefix: "example-value"

    # Optional discussion category. Can be a category ID (string or numeric value),
    # category name, or category slug/route. If not specified, uses the first
    # available category. Matched first against category IDs, then against category
    # names, then against category slugs. Numeric values are automatically converted
    # to strings at runtime.
    # (optional)
    category: null

    # Optional list of labels to attach to created discussions. Also used for matching
    # when close-older-discussions is enabled - discussions must have ALL specified
    # labels (AND logic).
    # (optional)
    labels: []
      # Array of strings

    # Optional list of allowed labels that can be used when creating discussions. If
    # omitted, any labels are allowed (including creating new ones). When specified,
    # the agent can only use labels from this list.
    # (optional)
    allowed-labels: []
      # Array of strings

    # Maximum number of discussions to create (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository discussion
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that discussions can be
    # created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the discussion in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # When true, automatically close older discussions matching the same title prefix
    # or labels as 'outdated' with a comment linking to the new discussion. Requires
    # title-prefix or labels to be set. Maximum 10 discussions will be closed. Only
    # runs if discussion creation succeeds. When fallback-to-issue is enabled and
    # discussion creation fails, older issues will be closed instead.
    # (optional)
    close-older-discussions: true

    # Optional explicit deduplication key for close-older matching. When set, a `<!--
    # gh-aw-close-key: <value> -->` marker is embedded in the discussion body and used
    # as the primary key for searching and filtering older discussions instead of the
    # workflow-id markers. This gives deterministic isolation across caller workflows
    # and is stable across workflow renames. The value is normalized to identifier
    # style (lowercase alphanumeric, dashes, underscores).
    # (optional)
    close-older-key: "example-value"

    # When true (default), fallback to creating an issue if discussion creation fails
    # due to permissions. The fallback issue will include a note indicating it was
    # intended to be a discussion. If close-older-discussions is enabled, the
    # close-older-issues logic will be applied to the fallback issue.
    # (optional)
    fallback-to-issue: true

    # Controls whether AI-generated footer is added to the discussion. When false, the
    # visible footer content is omitted but XML markers (workflow-id, tracker-id,
    # metadata) are still included for searchability. Defaults to true.
    # (optional)
    footer: true

    # Time until the discussion expires and should be automatically closed. Supports
    # integer (days), relative time format like '2h' (2 hours), '7d' (7 days), '2w' (2
    # weeks), '1m' (1 month), '1y' (1 year), or false to disable expiration. Minimum
    # duration: 2 hours. When set, a maintenance workflow will be generated. Defaults
    # to 7 days if not specified.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Number of days until expires
    expires: 1

    # Option 2: Relative time (e.g., '2h', '7d', '2w', '1m', '1y'); minimum 2h for
    # hour values
    expires: "example-value"

    # Option 3: Set to false to explicitly disable expiration
    expires: false

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable discussion creation with default configuration
  create-discussion: null

  # Enable AI agents to close GitHub Discussions based on workflow analysis or
  # conditions.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for closing GitHub discussions with comment and
  # resolution from agentic workflow output
  close-discussion:
    # Only close discussions that have all of these labels
    # (optional)
    required-labels: []
      # Array of strings

    # Only close discussions with this title prefix
    # (optional)
    required-title-prefix: "example-value"

    # Only close discussions in this category
    # (optional)
    required-category: "example-value"

    # Target for closing: 'triggering' (default, current discussion), or '*' (any
    # discussion with discussion_number field)
    # (optional)
    target: "example-value"

    # Maximum number of discussions to close (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository operations. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable discussion closing with default configuration
  close-discussion: null

  # Enable AI agents to edit and update existing GitHub Discussion content, titles,
  # and metadata.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for updating GitHub discussions from agentic workflow
  # output
  update-discussion:
    # Target for updates: 'triggering' (default), '*' (any discussion), or explicit
    # discussion number
    # (optional)
    target: "example-value"

    # Allow updating discussion title - presence of key indicates field can be updated
    # (optional)
    title: null

    # Allow updating discussion body - presence of key indicates field can be updated
    # (optional)
    body: null

    # Allow updating discussion labels - presence of key indicates field can be
    # updated
    # (optional)
    labels: null

    # Optional list of allowed labels. If omitted, any labels are allowed (including
    # creating new ones).
    # (optional)
    allowed-labels: []
      # Array of strings

    # Maximum number of discussions to update (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository discussion
    # updates. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # Controls whether AI-generated footer is added when updating the discussion body.
    # When false, the visible footer content is omitted. Defaults to true. Only
    # applies when 'body' is enabled.
    # (optional)
    footer: true

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable discussion updating with default configuration
  update-discussion: null

  # Enable AI agents to close GitHub issues based on workflow analysis, resolution
  # detection, or automated triage.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for closing GitHub issues with comment from agentic
  # workflow output
  close-issue:
    # Only close issues that have all of these labels
    # (optional)
    required-labels: []
      # Array of strings

    # Only close issues with this title prefix
    # (optional)
    required-title-prefix: "example-value"

    # Target for closing: 'triggering' (default, current issue), or '*' (any issue
    # with issue_number field)
    # (optional)
    target: "example-value"

    # Maximum number of issues to close (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository operations. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that issues can be closed
    # in. When specified, the agent can use a 'repo' field in the output to specify
    # which repository to close the issue in. The target repository (current or
    # target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

    # Reason for closing the issue (default: completed)
    # (optional)
    state-reason: "completed"

  # Option 2: Enable issue closing with default configuration
  close-issue: null

  # Enable AI agents to close pull requests based on workflow analysis or automated
  # review decisions.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for closing GitHub pull requests without merging, with
  # comment from agentic workflow output
  close-pull-request:
    # Only close pull requests that have any of these labels
    # (optional)
    required-labels: []
      # Array of strings

    # Only close pull requests with this title prefix
    # (optional)
    required-title-prefix: "example-value"

    # Target for closing: 'triggering' (default, current PR), or '*' (any PR with
    # pull_request_number field)
    # (optional)
    target: "example-value"

    # Maximum number of pull requests to close (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository operations. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable pull request closing with default configuration
  close-pull-request: null

  # Enable AI agents to mark draft pull requests as ready for review when criteria
  # are met.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for marking draft pull requests as ready for review,
  # with comment from agentic workflow output
  mark-pull-request-as-ready-for-review:
    # Only mark pull requests that have any of these labels
    # (optional)
    required-labels: []
      # Array of strings

    # Only mark pull requests with this title prefix
    # (optional)
    required-title-prefix: "example-value"

    # Target for marking: 'triggering' (default, current PR), or '*' (any PR with
    # pull_request_number field)
    # (optional)
    target: "example-value"

    # Maximum number of pull requests to mark as ready (default: 1) Supports integer
    # or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository operations. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable marking pull requests as ready for review with default
  # configuration
  mark-pull-request-as-ready-for-review: null

  # Enable AI agents to add comments to GitHub issues, pull requests, or
  # discussions. Supports templating, cross-repository commenting, and automatic
  # mentions.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for automatically creating GitHub issue or pull request
  # comments from AI workflow output. The main job does not need write permissions.
  add-comment:
    # Maximum number of comments to create (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for comments: 'triggering' (default), '*' (any issue), or explicit issue
    # number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository comments. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that comments can be
    # created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the comment in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # When true, minimizes/hides all previous comments from the same agentic workflow
    # (identified by tracker-id) before creating the new comment. Default: false.
    # (optional)
    hide-older-comments: true

    # List of allowed reasons for hiding older comments when hide-older-comments is
    # enabled. Default: all reasons allowed (spam, abuse, off_topic, outdated,
    # resolved, low_quality).
    # (optional)
    allowed-reasons: []
      # Array of strings

    # Controls whether the workflow requests discussions:write permission for
    # add-comment. Default: true (includes discussions:write). Set to false if your
    # GitHub App lacks Discussions permission to prevent 422 errors during token
    # generation.
    # (optional)
    discussions: true

    # Controls whether the workflow requests issues:write permission for add-comment.
    # Default: true (includes issues:write). Set to false to disable issue commenting
    # permissions.
    # (optional)
    issues: true

    # Controls whether the workflow requests pull-requests:write permission for
    # add-comment. Default: true (includes pull-requests:write). Set to false to
    # disable pull request commenting permissions.
    # (optional)
    pull-requests: true

    # Controls whether AI-generated footer is added to the comment. When false, the
    # visible footer content is omitted but XML markers (workflow-id, metadata) are
    # still included for searchability. Defaults to true.
    # (optional)
    footer: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable issue comment creation with default configuration
  add-comment: null

  # Enable AI agents to create GitHub pull requests from workflow-generated code
  # changes, patches, or analysis results.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub pull requests from agentic workflow
  # output. Supports creating multiple PRs in a single run when max > 1.
  create-pull-request:
    # Maximum number of pull requests to create (default: 1). Each PR requires
    # distinct changes on a separate branch. Supports integer or GitHub Actions
    # expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Optional prefix for the pull request title
    # (optional)
    title-prefix: "example-value"

    # Optional list of labels to attach to the pull request
    # (optional)
    labels: []
      # Array of strings

    # Optional list of allowed labels that can be used when creating pull requests. If
    # omitted, any labels are allowed (including creating new ones). When specified,
    # the agent can only use labels from this list.
    # (optional)
    allowed-labels: []
      # Array of strings

    # Optional reviewer(s) to assign to the pull request. Accepts either a single
    # string or an array of usernames. Use 'copilot' to request a code review from
    # GitHub Copilot.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single reviewer username to assign to the pull request. Use 'copilot'
    # to request a code review from GitHub Copilot using the
    # copilot-pull-request-reviewer[bot].
    reviewers: "example-value"

    # Option 2: List of reviewer usernames to assign to the pull request. Use
    # 'copilot' to request a code review from GitHub Copilot using the
    # copilot-pull-request-reviewer[bot].
    reviewers: []
      # Array items: string

    # Optional team reviewer(s) to assign to the pull request. Accepts either a single
    # string or an array of team slugs.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single team slug to assign as a reviewer to the pull request.
    team-reviewers: "example-value"

    # Option 2: List of team slugs to assign as reviewers to the pull request.
    team-reviewers: []
      # Array items: string

    # Optional assignee(s) for a fallback issue created when pull request creation
    # cannot proceed, including protected-files fallback-to-issue and pull request
    # creation or push failures. Accepts either a single string or an array of
    # usernames.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single username to assign to a fallback issue created when pull
    # request creation cannot proceed, including protected-files fallback-to-issue and
    # pull request creation or push failures.
    assignees: "example-value"

    # Option 2: List of usernames to assign to a fallback issue created when pull
    # request creation cannot proceed, including protected-files fallback-to-issue and
    # pull request creation or push failures.
    assignees: []
      # Array items: string

    # Optional labels to apply to fallback issues created when pull request creation
    # cannot proceed. When omitted, fallback issues reuse pull request labels. A
    # managed label is always added for triage.
    # (optional)
    fallback-labels: []
      # Array of strings

    # Whether to create pull request as draft (defaults to true). Accepts a boolean or
    # a GitHub Actions expression.
    # (optional)
    draft: null

    # Behavior when no changes to push: 'warn' (default - log warning but succeed),
    # 'error' (fail the action), or 'ignore' (silent success)
    # (optional)
    if-no-changes: "warn"

    # When true, allows creating a pull request without any initial changes or git
    # patch. This is useful for preparing a feature branch that an agent can push
    # changes to later. The branch will be created from the base branch without
    # applying any patch. Defaults to false.
    # (optional)
    allow-empty: true

    # Target repository in format 'owner/repo' for cross-repository pull request
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that pull requests can be
    # created in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to create the pull request in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Time until the pull request expires and should be automatically closed (only for
    # same-repo PRs without target-repo). Supports integer (days) or relative time
    # format. Minimum duration: 2 hours.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Number of days until expires
    expires: 1

    # Option 2: Relative time (e.g., '2h', '7d', '2w', '1m', '1y'); minimum 2h for
    # hour values
    expires: "example-value"

    # Enable auto-merge for the pull request. When enabled, the PR will be
    # automatically merged once all required checks pass and required approvals are
    # met. Defaults to false.
    # (optional)
    auto-merge: true

    # Base branch for the pull request. Defaults to the workflow's branch
    # (github.ref_name) if not specified. Useful for cross-repository PRs targeting
    # non-default branches (e.g., 'vnext', 'release/v1.0').
    # (optional)
    base-branch: "example-value"

    # Optional list of allowed base branch patterns (glob syntax, e.g. 'main',
    # 'release/*'). When configured, the agent may provide a `base` field in
    # create_pull_request output to override base-branch for a single run, but only if
    # it matches one of these patterns.
    # (optional)
    allowed-base-branches: []
      # Array of strings

    # Controls whether AI-generated footer is added to the pull request. When false,
    # the visible footer content is omitted but XML markers (workflow-id, tracker-id,
    # metadata) are still included for searchability. Defaults to true.
    # (optional)
    footer: true

    # Controls the fallback behavior when pull request creation fails. When true
    # (default), an issue is created as a fallback with the patch content. When false,
    # no issue is created and the workflow fails with an error. Setting to false also
    # removes the issues:write permission requirement.
    # (optional)
    fallback-as-issue: true

    # When true (default), automatically appends a closing keyword ("Fixes #N") to the
    # PR description when the workflow is triggered from an issue and no closing
    # keyword is already present. This causes GitHub to auto-close the triggering
    # issue when the PR is merged. Set to false to prevent this behavior, e.g., for
    # partial-work PRs or multi-PR workflows. Accepts a boolean or a GitHub Actions
    # expression.
    # (optional)
    auto-close-issue: null

    # Token used to push an empty commit after PR creation to trigger CI events. Works
    # around the GITHUB_TOKEN limitation where pushes don't trigger workflow runs.
    # Defaults to the magic secret GH_AW_CI_TRIGGER_TOKEN if set in the repository.
    # Use a secret expression (e.g. '${{ secrets.CI_TOKEN }}') for a custom token, or
    # 'app' for GitHub App auth.
    # (optional)
    github-token-for-extra-empty-commit: "example-value"

    # Controls protected-file protection. String form: blocked (default), allowed, or
    # fallback-to-issue. Object form: { policy, exclude } to customise the
    # protected-file set.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Controls protected-file protection. blocked (default): hard-block any
    # patch that modifies package manifests (e.g. package.json, go.mod), engine
    # instruction files (e.g. AGENTS.md, CLAUDE.md) or .github/ files. allowed: allow
    # all changes. fallback-to-issue: push the branch but create a review issue
    # instead of a PR, so a human can review the manifest changes before merging.
    protected-files: "blocked"

    # Option 2: Object form for granular control over the protected-file set. Use the
    # exclude list to remove specific files from the default protection while keeping
    # the rest.
    protected-files:
      # Protection policy. blocked (default): hard-block any patch that modifies
      # protected files. allowed: allow all changes. fallback-to-issue: push the branch
      # but create a review issue instead of a PR.
      # (optional)
      policy: "blocked"

      # List of filenames or path prefixes to remove from the default protected-file
      # set. Items are matched by basename (e.g. "AGENTS.md") or path prefix (e.g.
      # ".agents/"). Use this to allow the agent to modify specific files that are
      # otherwise blocked by default.
      # (optional)
      exclude: []
        # Array of strings

    # Exclusive allowlist of glob patterns. When set, every file in the patch must
    # match at least one pattern — files outside the list are always refused,
    # including normal source files. This is a restriction, not an exception: setting
    # allowed-files: [".github/workflows/*"] blocks all other files. To allow multiple
    # sets of files, list all patterns explicitly. Acts independently of the
    # protected-files policy; both checks must pass. To modify a protected file, it
    # must both match allowed-files and be permitted by protected-files (e.g.
    # protected-files: allowed). Supports * (any characters except /) and ** (any
    # characters including /).
    # (optional)
    allowed-files: []
      # Array of strings

    # When true, the random salt suffix is not appended to the agent-specified branch
    # name. Invalid characters are still replaced for security, and casing is always
    # preserved regardless of this setting. Useful when the target repository enforces
    # branch naming conventions (e.g. Jira keys in uppercase such as
    # 'bugfix/BR-329-red'). Defaults to false.
    # (optional)
    preserve-branch-name: true

    # When true (and preserve-branch-name is true), allows the handler to force-delete
    # an existing remote branch ref and recreate it from the agent's local HEAD. When
    # false (default), if the agent-specified branch already exists on the remote with
    # preserve-branch-name enabled, the handler falls back (e.g. opens an issue)
    # rather than overwriting the remote ref. Useful for long-lived reusable branches
    # whose previous PR was merged.
    # (optional)
    recreate-ref: true

    # List of glob patterns for files to exclude from the patch. Each pattern is
    # passed to `git format-patch` as a `:(exclude)<pattern>` magic pathspec, so
    # matching files are stripped by git at generation time and will not appear in the
    # commit. Excluded files are also not subject to `allowed-files` or
    # `protected-files` checks. Supports * (any characters except /) and ** (any
    # characters including /).
    # (optional)
    excluded-files: []
      # Array of strings

    # Transport format for packaging changes. "am" (default) uses git format-patch/git
    # am. "bundle" uses git bundle, which preserves merge commit topology, per-commit
    # authorship, and merge-resolution-only content.
    # (optional)
    patch-format: "am"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

    # When true, adds workflows: write to the GitHub App token permissions. Required
    # when allowed-files targets .github/workflows/ paths. Requires
    # safe-outputs.github-app to be configured because the workflows permission is a
    # GitHub App-only permission and cannot be granted via GITHUB_TOKEN.
    # (optional)
    allow-workflows: true

  # Option 2: Enable pull request creation with default configuration
  create-pull-request: null

  # Enable AI agents to add review comments to specific lines in pull request diffs
  # during code review workflows.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub pull request review comments from
  # agentic workflow output
  create-pull-request-review-comment:
    # Maximum number of review comments to create (default: 10) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Side of the diff for comments: 'LEFT' or 'RIGHT' (default: 'RIGHT')
    # (optional)
    side: "LEFT"

    # Target for review comments: 'triggering' (default, only on triggering PR), '*'
    # (any PR, requires pull_request_number in agent output), or explicit PR number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository PR review
    # comments. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that PR review comments
    # can be created in. When specified, the agent can use a 'repo' field in the
    # output to specify which repository to create the review comment in. The target
    # repository (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable PR review comment creation with default configuration
  create-pull-request-review-comment: null

  # Enable AI agents to submit consolidated pull request reviews with a status
  # decision. Works with create-pull-request-review-comment to batch inline comments
  # into a single review.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for submitting a consolidated PR review with a status
  # decision (APPROVE, REQUEST_CHANGES, COMMENT). All
  # create-pull-request-review-comment outputs are collected and submitted as part
  # of this review.
  submit-pull-request-review:
    # Maximum number of reviews to submit (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Controls when AI-generated footer is added to the review body. Accepts boolean
    # (true/false) or string ('always', 'none', 'if-body'). The 'if-body' mode is
    # useful for clean approval reviews without body text. Defaults to 'always'.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Controls whether AI-generated footer is added to the review body. true
    # maps to 'always', false maps to 'none'.
    footer: true

    # Option 2: Controls when AI-generated footer is added to the review body:
    # 'always' (default), 'none' (never), or 'if-body' (only when review has body
    # text).
    footer: "always"

    # Target PR for the review: 'triggering' (default, current PR), '*' (any PR,
    # requires pull_request_number in agent output), or explicit PR number (e.g. ${{
    # github.event.inputs.pr_number }}). Required when workflow is not triggered by a
    # pull request (e.g. workflow_dispatch).
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository PR review
    # submission. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that PR reviews can be
    # submitted in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to submit the review in. The target repository (current
    # or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # Optional list of allowed review event types. If omitted, all event types
    # (APPROVE, COMMENT, REQUEST_CHANGES) are allowed. Use this to restrict the agent
    # to specific event types, e.g. [COMMENT, REQUEST_CHANGES] to prevent approvals.
    # (optional)
    allowed-events: []
      # Array of strings

    # When true, after posting a replacement review this workflow dismisses older
    # REQUEST_CHANGES reviews previously posted by the same workflow on the same pull
    # request. This is best-effort and requires workflow markers in prior review
    # bodies.
    # (optional)
    supersede-older-reviews: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable PR review submission with default configuration
  submit-pull-request-review: null

  # Enable AI agents to reply to existing review comments on pull requests.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for replying to existing pull request review comments
  reply-to-pull-request-review-comment:
    # Maximum number of replies to create (default: 10) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for replies: 'triggering' (default), '*' (any PR), or explicit PR number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository operations
    # (optional)
    target-repo: "example-value"

    # List of additional repositories that replies can target
    # (optional)
    allowed-repos: []
      # Array of strings

    # Controls whether AI-generated footer is added to the reply body. When false, the
    # footer is omitted. Defaults to true.
    # (optional)
    footer: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable with default configuration
  reply-to-pull-request-review-comment: null

  # Enable AI agents to resolve review threads on the triggering pull request after
  # addressing feedback.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for resolving review threads on pull requests.
  # Resolution is scoped to the triggering PR only — threads on other PRs cannot be
  # resolved.
  resolve-pull-request-review-thread:
    # Maximum number of review threads to resolve (default: 10) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable review thread resolution with default configuration
  resolve-pull-request-review-thread: null

  # Enable AI agents to create GitHub Advanced Security code scanning alerts for
  # detected vulnerabilities or security issues.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating repository security advisories (SARIF
  # format) from agentic workflow output
  create-code-scanning-alert:
    # Maximum number of security findings to include (default: unlimited) Supports
    # integer or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Driver name for SARIF tool.driver.name field (default: 'GitHub Agentic Workflows
    # Security Scanner')
    # (optional)
    driver: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Target repository in format 'owner/repo' for cross-repository code scanning
    # alert creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that code scanning alerts
    # can be created in. When specified, the agent can use a 'repo' field in the
    # output to specify which repository to create the alert in. The target repository
    # (current or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable code scanning alert creation with default configuration
  # (unlimited findings)
  create-code-scanning-alert: null

  # Enable AI agents to create autofixes for code scanning alerts using the GitHub
  # REST API.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating autofixes for code scanning alerts
  autofix-code-scanning-alert:
    # Maximum number of autofixes to create (default: 10) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable code scanning autofix creation with default configuration (max:
  # 10)
  autofix-code-scanning-alert: null

  # Enable AI agents to add labels to GitHub issues or pull requests based on
  # workflow analysis or classification.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows any labels. Labels will be created if they
  # don't already exist in the repository.
  add-labels: null

  # Option 2: Configuration for adding labels to issues/PRs from agentic workflow
  # output. Labels will be created if they don't already exist in the repository.
  add-labels:
    # Optional list of allowed labels that can be added. Labels will be created if
    # they don't already exist in the repository. If omitted, any labels are allowed
    # (including creating new ones).
    # (optional)
    allowed: []
      # Array of strings

    # Optional list of blocked label patterns (supports glob patterns like '~*',
    # '*[bot]'). Labels matching these patterns will be rejected. Applied before
    # allowed list filtering for security.
    # (optional)
    blocked: []
      # Array of strings

    # Optional maximum number of labels to add (default: 3) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for labels: 'triggering' (default), '*' (any issue/PR), or explicit
    # issue/PR number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository label addition.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # List of additional repositories in format 'owner/repo' that labels can be added
    # to. When specified, the agent can use a 'repo' field in the output to specify
    # which repository to add labels to. The target repository (current or
    # target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to remove labels from GitHub issues or pull requests.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows any labels to be removed.
  remove-labels: null

  # Option 2: Configuration for removing labels from issues/PRs from agentic
  # workflow output.
  remove-labels:
    # Optional list of allowed labels that can be removed. If omitted, any labels can
    # be removed.
    # (optional)
    allowed: []
      # Array of strings

    # Optional list of blocked label patterns (supports glob patterns like '~*',
    # '*[bot]'). Labels matching these patterns will be rejected. Applied before
    # allowed list filtering for security.
    # (optional)
    blocked: []
      # Array of strings

    # Optional maximum number of labels to remove (default: 3) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for labels: 'triggering' (default), '*' (any issue/PR), or explicit
    # issue/PR number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository label removal.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # List of additional repositories in format 'owner/repo' that labels can be
    # removed from. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to remove labels from. The target repository (current
    # or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to request reviews from users or teams on pull requests based
  # on code changes or expertise matching.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows any reviewers
  add-reviewer: null

  # Option 2: Configuration for adding reviewers to pull requests from agentic
  # workflow output
  add-reviewer:
    # Optional list of allowed reviewers. If omitted, any reviewers are allowed.
    # (optional)
    reviewers: []
      # Array of strings

    # Optional allowed team reviewer or list of allowed team reviewers. If omitted,
    # any team reviewers are allowed.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: string
    team-reviewers: "example-value"

    # Option 2: array
    team-reviewers: []
      # Array items: string

    # Optional maximum number of reviewers to add (default: 3) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for reviewers: 'triggering' (default), '*' (any PR), or explicit PR
    # number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository reviewer addition.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to assign GitHub milestones to issues or pull requests based on
  # workflow analysis or project planning.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows assigning any milestones
  assign-milestone: null

  # Option 2: Configuration for assigning issues to milestones from agentic workflow
  # output
  assign-milestone:
    # Optional list of allowed milestone titles that can be assigned. If omitted, any
    # milestones are allowed.
    # (optional)
    allowed: []
      # Array of strings

    # Optional maximum number of milestone assignments (default: 1) Supports integer
    # or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository milestone
    # assignment. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to assign issues or pull requests to GitHub Copilot (@copilot)
  # for automated handling.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration uses default agent (copilot)
  assign-to-agent: null

  # Option 2: Configuration for assigning GitHub Copilot coding agent to issues from
  # agentic workflow output
  assign-to-agent:
    # Default agent name to assign (default: 'copilot')
    # (optional)
    name: "My Workflow"

    # Default AI model to use for the agent (e.g., 'auto', 'claude-sonnet-4.5',
    # 'claude-opus-4.5', 'claude-opus-4.6', 'gpt-5.1-codex-max', 'gpt-5.2-codex').
    # Defaults to 'auto' if not specified.
    # (optional)
    model: "example-value"

    # Default custom agent ID to use when assigning custom agents. This is used for
    # specialized agent configurations beyond the standard Copilot agent.
    # (optional)
    custom-agent: "example-value"

    # Default custom instructions to provide to the agent. These instructions will
    # guide the agent's behavior when working on the task.
    # (optional)
    custom-instructions: "example-value"

    # Optional list of allowed agent names. If specified, only these agents can be
    # assigned. When configured, existing agent assignees not in the list are removed
    # while regular user assignees are preserved.
    # (optional)
    allowed: []
      # Array of strings

    # Optional maximum number of agent assignments (default: 1) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target issue/PR to assign agents to. Use 'triggering' (default) for the
    # triggering issue/PR, '*' to require explicit issue_number/pull_number, or a
    # specific issue/PR number. With 'triggering', auto-resolves from
    # github.event.issue.number or github.event.pull_request.number.
    # (optional)
    target: null

    # Target repository in format 'owner/repo' for cross-repository agent assignment.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # Target repository where the pull request should be created, in format
    # 'owner/repo'. If omitted, the PR will be created in the same repository as the
    # issue (specified by target-repo or the workflow's repository). This allows
    # issues and code to live in different repositories.
    # (optional)
    pull-request-repo: "example-value"

    # List of additional repositories that pull requests can be created in beyond
    # pull-request-repo. Each entry should be in 'owner/repo' format. The repository
    # specified by pull-request-repo is automatically allowed without needing to be
    # listed here.
    # (optional)
    allowed-pull-request-repos: []
      # Array of strings

    # If true, the workflow continues gracefully when agent assignment fails (e.g.,
    # due to missing token or insufficient permissions), logging a warning instead of
    # failing. Default is false. Useful for workflows that should not fail when agent
    # assignment is optional.
    # (optional)
    ignore-if-error: true

    # Base branch for pull request creation in the target repository. Defaults to the
    # target repo's default branch. Only relevant when pull-request-repo is
    # configured.
    # (optional)
    base-branch: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to assign issues or pull requests to specific GitHub users
  # based on workflow logic or expertise matching.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable user assignment with default configuration
  assign-to-user: null

  # Option 2: Configuration for assigning users to issues from agentic workflow
  # output
  assign-to-user:
    # Optional list of allowed usernames. If specified, only these users can be
    # assigned.
    # (optional)
    allowed: []
      # Array of strings

    # Optional list of blocked usernames or patterns (e.g., 'copilot', '*[bot]').
    # Users matching these patterns cannot be assigned. Supports exact matches and
    # glob patterns.
    # (optional)
    blocked: []
      # Array of strings

    # Optional maximum number of user assignments (default: 1) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target issue to assign users to. Use 'triggering' (default) for the triggering
    # issue, '*' to allow any issue, or a specific issue number.
    # (optional)
    target: null

    # Target repository in format 'owner/repo' for cross-repository user assignment.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # If true, unassign all current assignees before assigning new ones. Useful for
    # reassigning issues from one user to another (default: false).
    # (optional)
    unassign-first: true

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # List of allowed repositories in format 'owner/repo' for cross-repository user
    # assignment operations. Use with 'repo' field in tool calls.
    # (optional)
    allowed-repos: []
      # Array of strings

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to unassign users from issues or pull requests. Useful for
  # reassigning work or removing users from issues.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable user unassignment with default configuration
  unassign-from-user: null

  # Option 2: Configuration for removing assignees from issues in agentic workflow
  # output
  unassign-from-user:
    # Optional list of allowed usernames. If specified, only these users can be
    # unassigned.
    # (optional)
    allowed: []
      # Array of strings

    # Optional list of blocked usernames or patterns (e.g., 'copilot', '*[bot]').
    # Users matching these patterns cannot be unassigned. Supports exact matches and
    # glob patterns.
    # (optional)
    blocked: []
      # Array of strings

    # Optional maximum number of unassignment operations (default: 1) Supports integer
    # or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target issue to unassign users from. Use 'triggering' (default) for the
    # triggering issue, '*' to allow any issue, or a specific issue number.
    # (optional)
    target: null

    # Target repository in format 'owner/repo' for cross-repository user unassignment.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of allowed repositories in format 'owner/repo' for cross-repository
    # unassignment operations. Use with 'repo' field in tool calls.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to create hierarchical relationships between issues using
  # GitHub's sub-issue (tasklist) feature.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable sub-issue linking with default configuration
  link-sub-issue: null

  # Option 2: Configuration for linking issues as sub-issues from agentic workflow
  # output
  link-sub-issue:
    # Maximum number of sub-issue links to create (default: 5) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Optional list of labels that parent issues must have to be eligible for linking
    # (optional)
    parent-required-labels: []
      # Array of strings

    # Optional title prefix that parent issues must have to be eligible for linking
    # (optional)
    parent-title-prefix: "example-value"

    # Optional list of labels that sub-issues must have to be eligible for linking
    # (optional)
    sub-required-labels: []
      # Array of strings

    # Optional title prefix that sub-issues must have to be eligible for linking
    # (optional)
    sub-title-prefix: "example-value"

    # Target repository in format 'owner/repo' for cross-repository sub-issue linking.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to edit and update existing GitHub issue content, titles,
  # labels, assignees, and metadata.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for updating GitHub issues from agentic workflow output
  update-issue:
    # Allow updating issue status (open/closed) - presence of key indicates field can
    # be updated
    # (optional)
    status: null

    # Target for updates: 'triggering' (default), '*' (any issue), or explicit issue
    # number
    # (optional)
    target: "example-value"

    # Allow updating issue title - presence of key indicates field can be updated
    # (optional)
    title: null

    # Allow updating issue body. Set to true to enable body updates, false to disable.
    # For backward compatibility, null (body:) also enables body updates.
    # (optional)
    body: null

    # Controls whether AI-generated footer is added when updating the issue body. When
    # false, the visible footer content is omitted but XML markers are still included.
    # Defaults to true. Only applies when 'body' is enabled.
    # (optional)
    footer: true

    # Maximum number of issues to update (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository issue updates.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # Required prefix for issue title. Only issues with this title prefix can be
    # updated.
    # (optional)
    title-prefix: "example-value"

    # List of additional repositories in format 'owner/repo' that issues can be
    # updated in. When specified, the agent can use a 'repo' field in the output to
    # specify which repository to update the issue in. The target repository (current
    # or target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable issue updating with default configuration
  update-issue: null

  # Enable AI agents to edit and update existing pull request content, titles,
  # labels, reviewers, and metadata.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for updating GitHub pull requests from agentic workflow
  # output. Both title and body updates are enabled by default.
  update-pull-request:
    # Target for updates: 'triggering' (default), '*' (any PR), or explicit PR number
    # (optional)
    target: "example-value"

    # Allow updating pull request title - defaults to true, set to false to disable
    # (optional)
    title: true

    # Allow updating pull request body - defaults to true, set to false to disable
    # (optional)
    body: true

    # When true, update the pull request branch with the latest base branch changes
    # before applying other updates. Defaults to false.
    # (optional)
    update-branch: true

    # Default operation for body updates: 'append' (add to end), 'prepend' (add to
    # start), or 'replace' (overwrite completely). Defaults to 'replace' if not
    # specified.
    # (optional)
    operation: "append"

    # Controls whether AI-generated footer is added when updating the pull request
    # body. When false, the visible footer content is omitted but XML markers are
    # still included. Defaults to true. Only applies when 'body' is enabled.
    # (optional)
    footer: true

    # Maximum number of pull requests to update (default: 1) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository pull request
    # updates. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable pull request updating with default configuration (title and
  # body updates enabled)
  update-pull-request: null

  # Enable AI agents to merge pull requests under configured policy gates.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable pull request merge with default policy configuration
  merge-pull-request: null

  # Option 2: Configuration for controlled pull request merges. The merge is blocked
  # unless all configured gates pass.
  merge-pull-request:
    # Maximum number of pull request merges to perform per run (default: 1). Supports
    # integer or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # List of labels that must all be present on the pull request before merge is
    # allowed.
    # (optional)
    required-labels: []
      # Array of strings

    # Exact pull request label names. At least one existing PR label must exactly
    # match one of these values when configured.
    # (optional)
    allowed-labels: []
      # Array of strings

    # Glob patterns for allowed source branch names (pull request head ref).
    # (optional)
    allowed-branches: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, evaluate merge gates and emit preview results without executing the
    # merge API call.
    # (optional)
    staged: true

  # Enable AI agents to push commits directly to pull request branches for automated
  # fixes or improvements.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Use default configuration (branch: 'triggering', if-no-changes:
  # 'warn')
  push-to-pull-request-branch: null

  # Option 2: Configuration for pushing changes to a specific branch from agentic
  # workflow output. Supports pushing to multiple PRs in a single run when max > 1.
  push-to-pull-request-branch:
    # Maximum number of push operations to perform (default: 1). Each push targets a
    # different pull request branch. Supports integer or GitHub Actions expression
    # (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # The branch to push changes to (defaults to 'triggering')
    # (optional)
    branch: "example-value"

    # Target for push operations: 'triggering' (default), '*' (any pull request), or
    # explicit pull request number
    # (optional)
    target: "example-value"

    # Required prefix for pull request title. Only pull requests with this prefix will
    # be accepted.
    # (optional)
    title-prefix: "example-value"

    # Required labels for pull request validation. Only pull requests with all these
    # labels will be accepted.
    # (optional)
    labels: []
      # Array of strings

    # Behavior when no changes to push: 'warn' (default - log warning but succeed),
    # 'error' (fail the action), or 'ignore' (silent success)
    # (optional)
    if-no-changes: "warn"

    # When true, treat deleted/missing pull request branch errors as a skipped push
    # instead of a hard failure. Useful when the PR branch may be deleted before safe
    # outputs run.
    # (optional)
    ignore-missing-branch-failure: true

    # Optional suffix to append to generated commit titles (e.g., ' [skip ci]' to
    # prevent triggering CI on the commit)
    # (optional)
    commit-title-suffix: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

    # Token used to push an empty commit after pushing changes to trigger CI events.
    # Works around the GITHUB_TOKEN limitation where pushes don't trigger workflow
    # runs. Defaults to the magic secret GH_AW_CI_TRIGGER_TOKEN if set in the
    # repository. Use a secret expression (e.g. '${{ secrets.CI_TOKEN }}') for a
    # custom token, or 'app' for GitHub App auth.
    # (optional)
    github-token-for-extra-empty-commit: "example-value"

    # When true (default), if pushing to the PR branch fails due to a
    # non-fast-forward/diverged branch, create a fallback pull request that targets
    # the original PR branch. Set to false to disable this behavior and avoid
    # requiring pull-requests: write permission.
    # (optional)
    fallback-as-pull-request: true

    # Target repository in format 'owner/repo' for cross-repository push to pull
    # request branch. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' that push to pull request
    # branch can target. When specified, the agent can use a 'repo' field in the
    # output to specify which repository to push to. The target repository (current or
    # target-repo) is always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # Controls protected-file protection. String form: blocked (default), allowed, or
    # fallback-to-issue. Object form: { policy, exclude } to customise the
    # protected-file set.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Controls protected-file protection. blocked (default): hard-block any
    # patch that modifies package manifests (e.g. package.json, go.mod), engine
    # instruction files (e.g. AGENTS.md, CLAUDE.md) or .github/ files. allowed: allow
    # all changes. fallback-to-issue: create a review issue instead of pushing to the
    # PR branch, so a human can review the changes before applying.
    protected-files: "blocked"

    # Option 2: Object form for granular control over the protected-file set. Use the
    # exclude list to remove specific files from the default protection while keeping
    # the rest.
    protected-files:
      # Protection policy. blocked (default): hard-block any patch that modifies
      # protected files. allowed: allow all changes. fallback-to-issue: create a review
      # issue instead of pushing.
      # (optional)
      policy: "blocked"

      # List of filenames or path prefixes to remove from the default protected-file
      # set. Items are matched by basename (e.g. "AGENTS.md") or path prefix (e.g.
      # ".agents/"). Use this to allow the agent to modify specific files that are
      # otherwise blocked by default.
      # (optional)
      exclude: []
        # Array of strings

    # Exclusive allowlist of glob patterns. When set, every file in the patch must
    # match at least one pattern — files outside the list are always refused,
    # including normal source files. This is a restriction, not an exception: setting
    # allowed-files: [".github/workflows/*"] blocks all other files. To allow multiple
    # sets of files, list all patterns explicitly. Acts independently of the
    # protected-files policy; both checks must pass. To modify a protected file, it
    # must both match allowed-files and be permitted by protected-files (e.g.
    # protected-files: allowed). Supports * (any characters except /) and ** (any
    # characters including /).
    # (optional)
    allowed-files: []
      # Array of strings

    # List of glob patterns for files to exclude from the patch. Each pattern is
    # passed to `git format-patch` as a `:(exclude)<pattern>` magic pathspec, so
    # matching files are stripped by git at generation time and will not appear in the
    # commit. Excluded files are also not subject to `allowed-files` or
    # `protected-files` checks. Supports * (any characters except /) and ** (any
    # characters including /).
    # (optional)
    excluded-files: []
      # Array of strings

    # Transport format for packaging changes. "am" (default) uses git format-patch/git
    # am. "bundle" uses git bundle, which preserves merge commit topology, per-commit
    # authorship, and merge-resolution-only content.
    # (optional)
    patch-format: "am"

    # When true, adds workflows: write to the GitHub App token permissions. Required
    # when allowed-files targets .github/workflows/ paths. Requires
    # safe-outputs.github-app to be configured because the workflows permission is a
    # GitHub App-only permission and cannot be granted via GITHUB_TOKEN.
    # (optional)
    allow-workflows: true

    # When false, skips the branch protection API pre-flight check before pushing. Set
    # to false to avoid requiring administration: read permission. The GitHub platform
    # will still enforce branch protection at push time. Default is true (check
    # enabled).
    # (optional)
    check-branch-protection: true

  # Enable AI agents to minimize (hide) comments on issues or pull requests based on
  # relevance, spam detection, or moderation rules.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable comment hiding with default configuration
  hide-comment: null

  # Option 2: Configuration for hiding comments on GitHub issues, pull requests, or
  # discussions from agentic workflow output
  hide-comment:
    # Maximum number of comments to hide (default: 5) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository in format 'owner/repo' for cross-repository comment hiding.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of allowed reasons for hiding comments. Default: all reasons allowed (spam,
    # abuse, off_topic, outdated, resolved, low_quality).
    # (optional)
    allowed-reasons: []
      # Array of strings

    # Controls whether the workflow requests discussions:write permission for
    # hide-comment. Default: true (includes discussions:write). Set to false if your
    # GitHub App lacks Discussions permission to prevent 422 errors during token
    # generation.
    # (optional)
    discussions: true

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Enable AI agents to set or clear the type of GitHub issues. Use an empty string
  # to clear the current type.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows setting any issue type
  set-issue-type: null

  # Option 2: Configuration for setting the type of GitHub issues from agentic
  # workflow output
  set-issue-type:
    # Optional list of allowed issue type names (e.g. 'Bug', 'Feature'). If omitted,
    # any type is allowed. Empty string is always allowed to clear the type.
    # (optional)
    allowed: []
      # Array of strings

    # Optional maximum number of set-issue-type operations (default: 5). Supports
    # integer or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target for issue type: 'triggering' (default), '*' (any issue), or explicit
    # issue number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository issue type
    # setting. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # List of additional repositories in format 'owner/repo' where issue types can be
    # set. When specified, the agent can use a 'repo' field in the output to specify
    # which repository to target. The target repository (current or target-repo) is
    # always implicitly allowed.
    # (optional)
    allowed-repos: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Dispatch workflow_dispatch events to other workflows. Used by orchestrators to
  # delegate work to worker workflows with controlled maximum dispatch count.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for dispatching workflow_dispatch events to other
  # workflows. Orchestrators use this to delegate work to worker workflows.
  dispatch-workflow:
    # List of workflow names (without .md extension) to allow dispatching. Each
    # workflow must exist in .github/workflows/.
    workflows: []
      # Array of strings

    # Maximum number of workflow dispatch operations per run (default: 1, max: 50)
    # Supports integer or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for dispatching workflows. Overrides global github-token if
    # specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Target repository in format 'owner/repo' for cross-repository workflow dispatch.
    # When specified, the workflow will be dispatched to the target repository instead
    # of the current one.
    # (optional)
    target-repo: "example-value"

    # Git ref (branch, tag, or SHA) to use when dispatching the workflow. For
    # workflow_call relay scenarios this is auto-injected by the compiler from
    # needs.activation.outputs.target_ref. Overrides the caller's GITHUB_REF.
    # (optional)
    target-ref: "example-value"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Shorthand array format: list of workflow names (without .md extension)
  # to allow dispatching
  dispatch-workflow: []
    # Array items: string

  # Dispatch repository_dispatch events to external repositories. Each sub-key
  # defines a named dispatch tool with its own event_type, target repository, input
  # schema, and execution limits.
  # (optional)
  dispatch_repository:
    {}

  # Call reusable workflows via workflow_call fan-out. The compiler generates static
  # conditional jobs; the agent selects which worker to activate. Use this for
  # orchestrator/dispatcher patterns within the same repository.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for calling reusable workflows via workflow_call
  # fan-out. The compiler generates conditional `uses:` jobs at compile time; the
  # agent selects which worker to activate at runtime.
  call-workflow:
    # List of workflow names (without .md extension) to allow calling. Each workflow
    # must exist in .github/workflows/ and declare a workflow_call trigger.
    workflows: []
      # Array of strings

    # Maximum number of workflow_call fan-out operations per run (default: 1, max:
    # 50). Supports integer or GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token passed to called workflows. Overrides global github-token if
    # specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Shorthand array format: list of workflow names (without .md extension)
  # to allow calling
  call-workflow: []
    # Array items: string

  # Enable AI agents to report when required MCP tools are unavailable. Used for
  # workflow diagnostics and tool discovery.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for reporting missing tools from agentic workflow output
  missing-tool:
    # Maximum number of missing tool reports (default: unlimited) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Whether to create or update GitHub issues when tools are missing (default: true)
    # (optional)
    create-issue: true

    # Prefix for issue titles when creating issues for missing tools (default:
    # '[missing tool]')
    # (optional)
    title-prefix: "example-value"

    # Labels to add to created issues for missing tools
    # (optional)
    labels: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable missing tool reporting with default configuration
  missing-tool: null

  # Option 3: Explicitly disable missing tool reporting (false). Missing tool
  # reporting is enabled by default when safe-outputs is configured.
  missing-tool: true

  # Enable AI agents to report when required data or context is missing. Used for
  # workflow troubleshooting and data validation.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for reporting missing data required to achieve workflow
  # goals. Encourages AI agents to be truthful about data gaps instead of
  # hallucinating information.
  missing-data:
    # Maximum number of missing data reports (default: unlimited) Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Whether to create or update GitHub issues when data is missing (default: true)
    # (optional)
    create-issue: true

    # Prefix for issue titles when creating issues for missing data (default:
    # '[missing data]')
    # (optional)
    title-prefix: "example-value"

    # Labels to add to created issues for missing data
    # (optional)
    labels: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable missing data reporting with default configuration
  missing-data: null

  # Option 3: Explicitly disable missing data reporting (false). Missing data
  # reporting is enabled by default when safe-outputs is configured.
  missing-data: true

  # Enable AI agents to explicitly indicate no action is needed. Used for workflow
  # control flow and conditional logic.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for no-op safe output (logging only, no GitHub API
  # calls). Always available as a fallback to ensure human-visible artifacts.
  noop:
    # Maximum number of noop messages (default: 1) Supports integer or GitHub Actions
    # expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Controls whether noop runs are reported as issue comments (default: true). Set
    # to false to disable posting to the no-op runs issue.
    # (optional)
    report-as-issue: true

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable noop output with default configuration (max: 1)
  noop: null

  # Option 3: Explicitly disable noop output (false). Noop is enabled by default
  # when safe-outputs is configured.
  noop: true

  # Enable AI agents to publish files (images, charts, reports) to an orphaned git
  # branch for persistent storage and web access.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for publishing assets to an orphaned git branch
  upload-asset:
    # Branch name (default: 'assets/${{ github.workflow }}')
    # (optional)
    branch: "example-value"

    # Maximum file size in KB (default: 10240 = 10MB)
    # (optional)
    max-size: 1

    # Allowed file extensions (default: common non-executable types)
    # (optional)
    allowed-exts: []
      # Array of strings

    # Maximum number of assets to upload (default: 10) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable asset publishing with default configuration
  upload-asset: null

  # Enable AI agents to upload files as run-scoped GitHub Actions artifacts. Returns
  # a temporary artifact ID rather than a raw download URL, keeping authorization
  # centralized.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for uploading files as run-scoped GitHub Actions
  # artifacts
  upload-artifact:
    # Maximum number of upload_artifact tool calls allowed per run (default: 1)
    # (optional)
    max-uploads: 1

    # Artifact retention period in days (fixed; the agent cannot override this value).
    # Supports integer or GitHub Actions expression (e.g. '${{ inputs.retention-days
    # }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    retention-days: 1

    # Option 2: string
    retention-days: "example-value"

    # Upload files directly without zip archiving (fixed; the agent cannot override
    # this value). Only valid for single-file uploads. Supports boolean or GitHub
    # Actions expression (e.g. '${{ inputs.skip-archive }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: boolean
    skip-archive: true

    # Option 2: string
    skip-archive: "example-value"

    # Maximum total upload size in bytes per slot (default: 104857600 = 100 MB)
    # (optional)
    max-size-bytes: 1

    # Glob patterns restricting which paths relative to the staging directory the
    # model may upload
    # (optional)
    allowed-paths: []
      # Array of strings

    # Default include/exclude glob filters applied on top of allowed-paths
    # (optional)
    filters:
      # Glob patterns for files to include
      # (optional)
      include: []
        # Array of strings

      # Glob patterns for files to exclude
      # (optional)
      exclude: []
        # Array of strings

    # Default values injected when the model omits a field
    # (optional)
    defaults:
      # Behaviour when no files match: 'error' (default) or 'ignore'
      # (optional)
      if-no-files: "error"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub Actions artifact
    # uploads (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable artifact uploads with default configuration
  upload-artifact: null

  # Enable AI agents to edit and update GitHub release content, including release
  # notes, assets, and metadata.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for updating GitHub release descriptions
  update-release:
    # Maximum number of releases to update (default: 1) Supports integer or GitHub
    # Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Target repository for cross-repo release updates (format: owner/repo). If not
    # specified, updates releases in the workflow's repository.
    # (optional)
    target-repo: "example-value"

    # Controls whether AI-generated footer is added when updating the release body.
    # When false, the visible footer content is omitted. Defaults to true.
    # (optional)
    footer: true

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable release updates with default configuration
  update-release: null

  # If true, emit step summary messages instead of making GitHub API calls (preview
  # mode)
  # (optional)
  staged: true

  # Environment variables to pass to safe output jobs
  # (optional)
  env:
    {}

  # GitHub token to use for safe output jobs. Typically a secret reference like ${{
  # secrets.GITHUB_TOKEN }} or ${{ secrets.CUSTOM_PAT }}
  # (optional)
  github-token: "${{ secrets.GITHUB_TOKEN }}"

  # GitHub App credentials for minting installation access tokens. When configured,
  # a token will be generated using the app credentials and used for all safe output
  # operations.
  # (optional)
  # This field supports multiple formats (anyOf):

  # Option 1: undefined

  # Option 2: undefined

  # Maximum allowed size for git patches in kilobytes (KB). Defaults to 1024 KB (1
  # MB). If patch exceeds this size, the job will fail.
  # (optional)
  max-patch-size: 1

  # Maximum allowed number of unique files in a create-pull-request patch. Defaults
  # to 100. The check counts unique file paths (deduplicated across multi-commit
  # patches), so it reflects how many distinct files the agent is pushing in this
  # iteration.
  # (optional)
  max-patch-files: 1

  # Enable AI agents to report detected security threats, policy violations, or
  # suspicious patterns for security review.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable or disable threat detection for safe outputs (defaults to true
  # when safe-outputs are configured)
  threat-detection: true

  # Option 2: Threat detection configuration object
  threat-detection:
    # Whether threat detection is enabled
    # (optional)
    enabled: true

    # Additional custom prompt instructions to append to threat detection analysis
    # (optional)
    prompt: "example-value"

    # AI engine configuration specifically for threat detection (overrides main
    # workflow engine). Set to false to disable AI-based threat detection. Supports
    # same format as main engine field when not false.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Disable AI engine for threat detection (only run custom steps)
    engine: true

    # Option 2: undefined

    # Array of extra job steps to run before engine execution
    # (optional)
    steps: []

    # Array of extra job steps to run after engine execution
    # (optional)
    post-steps: []

    # Runner specification for the detection job. Overrides agent.runs-on for the
    # detection job only. Defaults to agent.runs-on.
    # (optional)
    runs-on: "example-value"

    # When true (default), detection failures produce warnings and allow safe outputs
    # to proceed with a caution notice and 'needs-review' label. When false, detection
    # failures block safe outputs entirely.
    # (optional)
    continue-on-error: true

  # Custom safe-output jobs that can be executed based on agentic workflow output.
  # Job names containing dashes will be automatically normalized to underscores
  # (e.g., 'send-notification' becomes 'send_notification').
  # (optional)
  jobs:
    {}

  # Inline JavaScript script handlers that run inside the consolidated safe-outputs
  # job handler loop. Unlike 'jobs' (which create separate GitHub Actions jobs),
  # scripts execute in-process alongside the built-in handlers. Users write only the
  # body of the main function — the compiler wraps it with 'async function
  # main(config = {}) { ... }' and 'module.exports = { main };' automatically.
  # Script names containing dashes will be automatically normalized to underscores
  # (e.g., 'post-slack-message' becomes 'post_slack_message').
  # (optional)
  scripts:
    {}

  # Custom message templates for safe-output footer and notification messages.
  # Available placeholders: {workflow_name} (workflow name), {run_url} (GitHub
  # Actions run URL), {triggering_number} (issue/PR/discussion number),
  # {workflow_source} (owner/repo/path@ref), {workflow_source_url} (GitHub URL to
  # source), {operation} (safe-output operation name for staged mode).
  # (optional)
  messages:
    # Custom footer message template for AI-generated content. Available placeholders:
    # {workflow_name}, {run_url}, {triggering_number}, {workflow_source},
    # {workflow_source_url}. Example: '> Generated by [{workflow_name}]({run_url})'
    # (optional)
    footer: "example-value"

    # Custom installation instructions template appended to the footer. Available
    # placeholders: {workflow_source}, {workflow_source_url}. Example: '> Install: `gh
    # aw add {workflow_source}`'
    # (optional)
    footer-install: "example-value"

    # Custom footer message template for workflow recompile issues. Available
    # placeholders: {workflow_name}, {run_url}, {repository}. Example: '> Workflow
    # sync report by [{workflow_name}]({run_url}) for {repository}'
    # (optional)
    footer-workflow-recompile: "example-value"

    # Custom footer message template for comments on workflow recompile issues.
    # Available placeholders: {workflow_name}, {run_url}, {repository}. Example: '>
    # Update from [{workflow_name}]({run_url}) for {repository}'
    # (optional)
    footer-workflow-recompile-comment: "example-value"

    # Custom title template for staged mode preview. Available placeholders:
    # {operation}. Example: '🎭 Preview: {operation}'
    # (optional)
    staged-title: "example-value"

    # Custom description template for staged mode preview. Available placeholders:
    # {operation}. Example: 'The following {operation} would occur if staged mode was
    # disabled:'
    # (optional)
    staged-description: "example-value"

    # Custom message template for workflow activation comment. Available placeholders:
    # {workflow_name}, {run_url}, {event_type}. Default: 'Agentic
    # [{workflow_name}]({run_url}) triggered by this {event_type}.'
    # (optional)
    run-started: "example-value"

    # Custom message template for successful workflow completion. Available
    # placeholders: {workflow_name}, {run_url}. Default: '✅ Agentic
    # [{workflow_name}]({run_url}) completed successfully.'
    # (optional)
    run-success: "example-value"

    # Custom message template for failed workflow. Available placeholders:
    # {workflow_name}, {run_url}, {status}. Default: '❌ Agentic
    # [{workflow_name}]({run_url}) {status} and wasn't able to produce a result.'
    # (optional)
    run-failure: "example-value"

    # Custom message template for detection job failure. Available placeholders:
    # {workflow_name}, {run_url}. Default: '⚠️ Security scanning failed for
    # [{workflow_name}]({run_url}). Review the logs for details.'
    # (optional)
    detection-failure: "example-value"

    # Custom footer template for agent failure tracking issues. Available
    # placeholders: {workflow_name}, {run_url}. Default: '> Agent failure tracked by
    # [{workflow_name}]({run_url})'
    # (optional)
    agent-failure-issue: "example-value"

    # Custom footer template for comments on agent failure tracking issues. Available
    # placeholders: {workflow_name}, {run_url}. Default: '> Agent failure update from
    # [{workflow_name}]({run_url})'
    # (optional)
    agent-failure-comment: "example-value"

    # Custom message template for pull request creation link appended to the
    # activation comment. Available placeholders: {item_number}, {item_url}. Default:
    # 'Pull request created: [#{item_number}]({item_url})'
    # (optional)
    pull-request-created: "example-value"

    # Custom message template for issue creation link appended to the activation
    # comment. Available placeholders: {item_number}, {item_url}. Default: 'Issue
    # created: [#{item_number}]({item_url})'
    # (optional)
    issue-created: "example-value"

    # Custom message template for commit push link appended to the activation comment.
    # Available placeholders: {commit_sha}, {short_sha}, {commit_url}. Default:
    # 'Commit pushed: [`{short_sha}`]({commit_url})'
    # (optional)
    commit-pushed: "example-value"

    # When enabled, workflow completion notifier creates a new comment instead of
    # editing the activation comment. Creates an append-only timeline of workflow
    # runs. Default: false
    # (optional)
    append-only-comments: true

  # Configuration for @mention filtering in safe outputs. Controls whether and how
  # @mentions in AI-generated content are allowed or escaped.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Simple boolean mode: false = always escape mentions, true = always
  # allow mentions (error in strict mode)
  mentions: true

  # Option 2: Advanced configuration for @mention filtering with fine-grained
  # control
  mentions:
    # Allow mentions of repository team members (collaborators with any permission
    # level, excluding bots). Default: true
    # (optional)
    allow-team-members: true

    # Allow mentions inferred from event context (issue/PR authors, assignees,
    # commenters). Default: true
    # (optional)
    allow-context: true

    # List of user/bot names always allowed to be mentioned. Bots are not allowed by
    # default unless listed here.
    # (optional)
    allowed: []
      # Array of strings

    # Maximum number of mentions allowed per message. Default: 50 Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

  # Global footer control for all safe outputs. When false, omits visible
  # AI-generated footer content from all created/updated entities (issues, PRs,
  # discussions, releases) while still including XML markers for searchability.
  # Individual safe-output types (create-issue, update-issue, etc.) can override
  # this by specifying their own footer field. Defaults to true.
  # (optional)
  footer: true

  # When set to false or "false", disables all activation and fallback comments
  # entirely (run-started, run-success, run-failure, PR/issue creation links).
  # Supports templatable boolean values including GitHub Actions expressions (e.g.
  # ${{ inputs.activation-comments }}). Default: true
  # (optional)
  activation-comments: null

  # When true, creates a parent '[aw] Failed runs' issue that tracks all workflow
  # failures as sub-issues. Helps organize failure tracking but may be unnecessary
  # in smaller repositories. Defaults to false.
  # (optional)
  group-reports: true

  # When false, disables creating failure tracking issues when workflows fail.
  # Useful for workflows where failures are expected or handled elsewhere. Defaults
  # to true.
  # (optional)
  report-failure-as-issue: true

  # Repository to create failure tracking issues in, in the format 'owner/repo'.
  # Useful when the current repository has issues disabled. Defaults to the current
  # repository.
  # (optional)
  failure-issue-repo: "example-value"

  # Maximum number of bot trigger references (e.g. 'fixes #123', 'closes #456')
  # allowed in output before all of them are neutralized. Default: 10. Supports
  # integer or GitHub Actions expression (e.g. '${{ inputs.max-bot-mentions }}').
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: integer
  max-bot-mentions: 1

  # Option 2: GitHub Actions expression that resolves to an integer at runtime
  max-bot-mentions: "example-value"

  # Override the id-token permission for the safe-outputs job. Use 'write' to
  # force-enable the id-token: write permission (required for OIDC authentication
  # with cloud providers). Use 'none' to suppress automatic detection and prevent
  # adding id-token: write even when vault/OIDC actions are detected in steps. By
  # default, the compiler auto-detects known OIDC/vault actions
  # (aws-actions/configure-aws-credentials, azure/login, google-github-actions/auth,
  # hashicorp/vault-action, cyberark/conjur-action) and adds id-token: write
  # automatically.
  # (optional)
  id-token: "write"

  # Concurrency group for the safe-outputs job. When set, the safe-outputs job will
  # use this concurrency group with cancel-in-progress: false. Supports GitHub
  # Actions expressions.
  # (optional)
  concurrency-group: "example-value"

  # Explicit additional custom workflow jobs that the consolidated safe_outputs job
  # should depend on.
  # (optional)
  needs: []
    # Array of strings

  # Override the GitHub deployment environment for the safe-outputs job. When set,
  # this environment is used instead of the top-level environment: field. When not
  # set, the top-level environment: field is propagated automatically so that
  # environment-scoped secrets are accessible in the safe-outputs job.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Environment name as a string
  environment: "example-value"

  # Option 2: Environment object with name and optional URL
  environment:
    # The name of the environment configured in the repo
    name: "My Workflow"

    # A deployment URL
    # (optional)
    url: "example-value"

  # Runner specification for all safe-outputs jobs (activation, create-issue,
  # add-comment, etc.). Single runner label (e.g., 'ubuntu-slim', 'ubuntu-latest',
  # 'windows-latest', 'self-hosted'). Defaults to 'ubuntu-slim'. See
  # https://github.blog/changelog/2025-10-28-1-vcpu-linux-runner-now-available-in-github-actions-in-public-preview/
  # (optional)
  runs-on: "example-value"

  # Custom steps to inject into all safe-output jobs. These steps run after checking
  # out the repository and setting up the action, and before any safe-output code
  # executes.
  # (optional)
  steps: []

  # Custom GitHub Actions to mount as once-callable MCP tools. Each action is
  # resolved at compile time to derive its input schema from action.yml, and a
  # guarded `uses:` step is injected in the safe_outputs job. Action names
  # containing dashes will be automatically normalized to underscores (e.g.,
  # 'add-smoked-label' becomes 'add_smoked_label').
  # (optional)
  actions:
    {}

  # Enable AI agents to signal that a task could not be completed due to
  # infrastructure or tool failures (e.g., MCP crash, missing auth, inaccessible
  # repository). Activates failure handling even when the agent exits 0.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for report_incomplete safe output
  report-incomplete:
    # Maximum number of report_incomplete signals (default: 5). Supports integer or
    # GitHub Actions expression (e.g. '${{ inputs.max }}').
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: integer
    max: 1

    # Option 2: GitHub Actions expression that resolves to an integer at runtime
    max: "example-value"

    # Whether to create or update GitHub issues when the task was incomplete (default:
    # true)
    # (optional)
    create-issue: true

    # Prefix for issue titles when creating issues for incomplete runs (default:
    # '[incomplete]')
    # (optional)
    title-prefix: "example-value"

    # Labels to add to created issues for incomplete runs
    # (optional)
    labels: []
      # Array of strings

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # If true, emit step summary messages instead of making GitHub API calls for this
    # specific output type (preview mode)
    # (optional)
    staged: true

  # Option 2: Enable report_incomplete with default configuration
  report-incomplete: null

  # Option 3: Explicitly disable report_incomplete (false). report_incomplete is
  # enabled by default when safe-outputs is configured.
  report-incomplete: true

# Configuration for secret redaction behavior in workflow outputs and artifacts
# (optional)
secret-masking:
  # Additional secret redaction steps to inject after the built-in secret redaction.
  # Use this to mask secrets in generated files using custom patterns.
  # (optional)
  steps: []

# Optional observability output settings for workflow runs.
# (optional)
observability:
  # OTLP (OpenTelemetry Protocol) trace export configuration.
  # (optional)
  otlp:
    # OTLP collector endpoint URL (e.g. 'https://traces.example.com:4317'). Supports
    # GitHub Actions expressions such as ${{ secrets.OTLP_ENDPOINT }}. When a static
    # URL is provided, its hostname is automatically added to the network firewall
    # allowlist.
    # (optional)
    endpoint: "example-value"

    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Map of HTTP header names to values to include with every OTLP export
    # request. Values support GitHub Actions expressions such as ${{ secrets.TOKEN }}.
    # Injected as the OTEL_EXPORTER_OTLP_HEADERS environment variable.
    headers:
      {}

    # Option 2: Deprecated: use the map form instead. Comma-separated list of
    # key=value HTTP headers to include with every OTLP export request (e.g.
    # 'Authorization=Bearer <token>'). Supports GitHub Actions expressions such as ${{
    # secrets.OTLP_HEADERS }}. Injected as the OTEL_EXPORTER_OTLP_HEADERS environment
    # variable.
    headers: "example-value"

# Allow list of bot identifiers that can trigger the workflow even if they don't
# meet the required role permissions. When the actor is in this list, the bot must
# be active (installed) on the repository to trigger the workflow.
# (optional)
bots: []
  # Array of Bot identifier/name (e.g., 'dependabot[bot]', 'renovate[bot]',
  # 'github-actions[bot]')

# Rate limiting configuration to restrict how frequently users can trigger the
# workflow. Helps prevent abuse and resource exhaustion from programmatically
# triggered events.
# (optional)
rate-limit:
  # Maximum number of workflow runs allowed per user within the time window.
  # Required field. Supports integer or GitHub Actions expression (e.g. '${{
  # inputs.max }}').
  # This field supports multiple formats (oneOf):

  # Option 1: integer
  max: 1

  # Option 2: GitHub Actions expression that resolves to an integer at runtime
  max: "example-value"

  # Time window in minutes for rate limiting. Defaults to 60 (1 hour). Maximum: 180
  # (3 hours).
  # (optional)
  window: 1

  # Optional list of event types to apply rate limiting to. If not specified, rate
  # limiting applies to all programmatically triggered events (e.g.,
  # workflow_dispatch, issue_comment, pull_request_review).
  # (optional)
  events: []
    # Array of strings

  # Optional list of roles that are exempt from rate limiting. Defaults to ['admin',
  # 'maintain', 'write'] if not specified. Users with any of these roles will not be
  # subject to rate limiting checks. To apply rate limiting to all users, set to an
  # empty array: []
  # (optional)
  ignored-roles: []
    # Array of strings

# Enable strict mode validation for enhanced security and compliance. Strict mode
# enforces: (1) Write Permissions - refuses contents:write, issues:write,
# pull-requests:write; requires safe-outputs instead, (2) Network Configuration -
# requires explicit network configuration with no standalone wildcard '*' in
# allowed domains (patterns like '*.example.com' are allowed), (3) Action Pinning
# - enforces actions pinned to commit SHAs instead of tags/branches, (4) MCP
# Network - requires network configuration for custom MCP servers with containers,
# (5) Deprecated Fields - refuses deprecated frontmatter fields. Can be enabled
# per-workflow via 'strict: true' in frontmatter, or disabled via 'strict: false'.
# CLI flag takes precedence over frontmatter (gh aw compile --strict enforces
# strict mode). Defaults to true. See:
# https://github.github.com/gh-aw/reference/frontmatter/#strict-mode-strict
# (optional)
strict: true

# Mark the workflow as private, preventing it from being added to other
# repositories via 'gh aw add'. A workflow with private: true is not meant to be
# shared outside its repository.
# (optional)
private: true

# Control whether the compile-agentic version update check runs in the activation
# job. When true (default), the activation job downloads config.json from the
# gh-aw repository and verifies the compiled version is not blocked and meets the
# minimum supported version. Set to false to disable the check (not allowed in
# strict mode). See:
# https://github.github.com/gh-aw/reference/frontmatter/#check-for-updates
# (optional)
check-for-updates: true

# Allow npm pre/post install scripts to execute during package installation. By
# default, --ignore-scripts is added to all generated npm install commands to
# prevent supply chain attacks via malicious install hooks. Setting
# run-install-scripts: true disables this protection globally (all runtimes). A
# supply chain security warning is emitted at compile time; in strict mode this is
# an error. Per-runtime control is also available via
# runtimes.<runtime>.run-install-scripts. See:
# https://github.github.com/gh-aw/reference/frontmatter/#run-install-scripts
# (optional)
run-install-scripts: true

# MCP Scripts configuration for defining custom lightweight MCP tools as
# JavaScript, shell scripts, or Python scripts. Tools are mounted in an MCP server
# and have access to secrets specified by the user. Only one of 'script'
# (JavaScript), 'run' (shell), or 'py' (Python) must be specified per tool.
# (optional)
mcp-scripts:
  {}

# Runtime environment version overrides. Allows customizing runtime versions
# (e.g., Node.js, Python) or defining new runtimes. Runtimes from imported shared
# workflows are also merged.
# (optional)
runtimes:
  {}

# Checkout configuration for the agent job. Controls how actions/checkout is
# invoked. Can be a single checkout configuration, an array for multiple
# checkouts, or false to disable the default checkout step entirely (dev-mode
# checkouts are unaffected).
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Single checkout configuration for the default workspace

# Option 2: Multiple checkout configurations
checkout: []
  # Array items: undefined

# Option 3: Set to false to disable the default checkout step. The agent job will
# not check out any repository (dev-mode checkouts are unaffected).
checkout: false

# APM package references to install. Supports array format (list of package slugs)
# or object format with packages and isolated fields.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple array of APM package references.
dependencies: []
  # Array items: APM package reference in the format 'org/repo' or
  # 'org/repo/path/to/skill'

# Option 2: Object format with packages and optional isolated flag.
dependencies:
  # List of APM package references to install.
  packages: []
    # Array of APM package reference in the format 'org/repo' or
    # 'org/repo/path/to/skill'

  # If true, agent restore step clears primitive dirs before unpacking.
  # (optional)
  isolated: true

  # GitHub App credentials for minting installation access tokens used by APM to
  # access cross-org private repositories.
  # (optional)
  # This field supports multiple formats (anyOf):

  # Option 1: undefined

  # Option 2: undefined

  # Environment variables to set on the APM pack step (e.g., tokens or registry
  # URLs).
  # (optional)
  env:
    {}

  # GitHub token expression to authenticate APM with private package repositories.
  # Uses cascading fallback (GH_AW_PLUGINS_TOKEN → GH_AW_GITHUB_TOKEN →
  # GITHUB_TOKEN) when not specified. Takes effect unless github-app is also
  # configured (which takes precedence).
  # (optional)
  github-token: "${{ secrets.GITHUB_TOKEN }}"

# Top-level GitHub App configuration used as a fallback for all nested github-app
# token minting operations (on, safe-outputs, checkout, tools.github,
# dependencies). When a nested section does not define its own github-app, this
# top-level configuration is used automatically.
# (optional)
# This field supports multiple formats (anyOf):

# Option 1: undefined

# Option 2: undefined

# Schema for validating 'with' input values when this workflow is imported by
# another workflow using the 'uses'/'with' syntax. Defines the expected
# parameters, their types, and whether they are required. Scalar inputs are
# accessible via '${{ github.aw.import-inputs.<name> }}' expressions. Object
# inputs (type: object) allow one-level deep sub-fields accessible via '${{
# github.aw.import-inputs.<name>.<subkey> }}' expressions.
# (optional)
import-schema:
  {}
---
```

## Additional Information

- Fields marked with `(optional)` are not required
- Fields with multiple options show all possible formats
- See the [Frontmatter guide](/gh-aw/reference/frontmatter/) for detailed explanations and examples
- See individual reference pages for specific topics like [Triggers](/gh-aw/reference/triggers/), [Tools](/gh-aw/reference/tools/), and [Safe Outputs](/gh-aw/reference/safe-outputs/)
