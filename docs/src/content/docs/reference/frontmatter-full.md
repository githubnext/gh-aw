---
title: Frontmatter Full Reference
description: Complete JSON Schema-based reference for all GitHub Agentic Workflows frontmatter configuration options with YAML examples.
sidebar:
  order: 201
---

This document provides a comprehensive reference for all available frontmatter configuration options in GitHub Agentic Workflows. The examples below are generated from the JSON Schema and include inline comments describing each field.

:::note
This documentation is automatically generated from the JSON Schema. For a more user-friendly guide, see [Frontmatter](/gh-aw/reference/frontmatter/).
:::

## Complete Frontmatter Reference

```yaml
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

# Optional array of workflow specifications to import (similar to @include
# directives but defined in frontmatter). Format: owner/repo/path@ref (e.g.,
# githubnext/agentics/workflows/shared/common.md@v1.0.0).
# (optional)
imports: []
  # Array of Workflow specification in format owner/repo/path@ref

# Workflow triggers that define when the agentic workflow should run. Supports
# standard GitHub Actions trigger events plus special command triggers for
# /commands (required)
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple trigger event name (e.g., 'push', 'issues', 'pull_request',
# 'discussion', 'schedule', 'fork', 'create', 'delete', 'public', 'watch',
# 'workflow_call')
on: "example-value"

# Option 2: Complex trigger configuration with event-specific filters and options
on:
  # Special command trigger for /command workflows (e.g., '/my-bot' in issue
  # comments). Creates conditions to match slash commands automatically.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null command configuration - defaults to using the workflow filename
  # (without .md extension) as the command name
  command: null

  # Option 2: Command name as a string (shorthand format, e.g., 'customname' for
  # '/customname' triggers)
  command: "example-value"

  # Option 3: Command configuration object with custom command name
  command:
    # Custom command name for slash commands (e.g., 'helper-bot' for '/helper-bot'
    # triggers). Defaults to workflow filename without .md extension if not specified.
    # (optional)
    name: "My Workflow"

    # Events where the command should be active. Default is all comment-related events
    # ('*'). Use GitHub Actions event names.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single event name or '*' for all events. Use GitHub Actions event
    # names: 'issues', 'issue_comment', 'pull_request_comment', 'pull_request',
    # 'pull_request_review_comment', 'discussion', 'discussion_comment'.
    events: "*"

    # Option 2: Array of event names where the command should be active. Use GitHub
    # Actions event names.
    events: []
      # Array items: GitHub Actions event name.

  # Push event trigger that runs the workflow when code is pushed to the repository
  # (optional)
  push:
    # Branches to filter on
    # (optional)
    branches: []
      # Array of strings

    # Branches to ignore
    # (optional)
    branches-ignore: []
      # Array of strings

    # Paths to filter on
    # (optional)
    paths: []
      # Array of strings

    # Paths to ignore
    # (optional)
    paths-ignore: []
      # Array of strings

    # List of git tag names or patterns to include for push events (supports
    # wildcards)
    # (optional)
    tags: []
      # Array of strings

    # List of git tag names or patterns to exclude from push events (supports
    # wildcards)
    # (optional)
    tags-ignore: []
      # Array of strings

  # Pull request event trigger that runs the workflow when pull requests are
  # created, updated, or closed
  # (optional)
  pull_request:
    # List of pull request event types to trigger on
    # (optional)
    types: []
      # Array of strings

    # Branches to filter on
    # (optional)
    branches: []
      # Array of strings

    # Branches to ignore
    # (optional)
    branches-ignore: []
      # Array of strings

    # Paths to filter on
    # (optional)
    paths: []
      # Array of strings

    # Paths to ignore
    # (optional)
    paths-ignore: []
      # Array of strings

    # Filter by draft pull request state. Set to false to exclude draft PRs, true to
    # include only drafts, or omit to include both
    # (optional)
    draft: true

    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single fork pattern (e.g., '*' for all forks, 'org/*' for org glob,
    # 'org/repo' for exact match)
    forks: "example-value"

    # Option 2: List of allowed fork repositories with glob support (e.g., 'org/repo',
    # 'org/*', '*' for all forks)
    forks: []
      # Array items: Repository pattern with optional glob support

    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single label name to filter labeled/unlabeled events (e.g., 'bug')
    names: "example-value"

    # Option 2: List of label names to filter labeled/unlabeled events. Only applies
    # when 'labeled' or 'unlabeled' is in the types array
    names: []
      # Array items: Label name

  # Issues event trigger that runs the workflow when repository issues are created,
  # updated, or managed
  # (optional)
  issues:
    # Types of issue events
    # (optional)
    types: []
      # Array of strings

    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single label name to filter labeled/unlabeled events (e.g., 'bug')
    names: "example-value"

    # Option 2: List of label names to filter labeled/unlabeled events. Only applies
    # when 'labeled' or 'unlabeled' is in the types array
    names: []
      # Array items: Label name

  # Issue comment event trigger
  # (optional)
  issue_comment:
    # Types of issue comment events
    # (optional)
    types: []
      # Array of strings

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

  # Scheduled trigger events
  # (optional)
  schedule: []
    # Array items:
      # Cron expression for schedule
      cron: "example-value"

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
  workflow_run:
    # List of workflows to trigger on
    # (optional)
    workflows: []
      # Array of strings

    # Types of workflow run events
    # (optional)
    types: []
      # Array of strings

    # Branches to filter on
    # (optional)
    branches: []
      # Array of strings

    # Branches to ignore
    # (optional)
    branches-ignore: []
      # Array of strings

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
    {}

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
  pull_request_target:
    # List of pull request target event types to trigger on
    # (optional)
    types: []
      # Array of strings

    # Branches to filter on
    # (optional)
    branches: []
      # Array of strings

    # Branches to ignore
    # (optional)
    branches-ignore: []
      # Array of strings

    # Paths to filter on
    # (optional)
    paths: []
      # Array of strings

    # Paths to ignore
    # (optional)
    paths-ignore: []
      # Array of strings

    # Filter by draft pull request state
    # (optional)
    draft: true

    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Single fork pattern
    forks: "example-value"

    # Option 2: List of allowed fork repositories with glob support
    forks: []
      # Array items: string

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
    {}

  # Time when workflow should stop running. Supports multiple formats: absolute
  # dates (YYYY-MM-DD HH:MM:SS, June 1 2025, 1st June 2025, 06/01/2025, etc.) or
  # relative time deltas (+25h, +3d, +1d12h30m)
  # (optional)
  stop-after: "example-value"

  # AI reaction to add/remove on triggering item (one of: +1, -1, laugh, confused,
  # heart, hooray, rocket, eyes). Defaults to 'eyes' if not specified.
  # (optional)
  reaction: "+1"

# GitHub token permissions for the workflow. Controls what the GITHUB_TOKEN can
# access during execution. Use the principle of least privilege - only grant the
# minimum permissions needed.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple permissions string: 'read-all' (all read permissions),
# 'write-all' (all write permissions), 'read' or 'write' (basic level)
permissions: "read-all"

# Option 2: Detailed permissions object with granular control over specific GitHub
# API scopes
permissions:
  # Permission for GitHub Actions workflows and runs (read: view workflows, write:
  # manage workflows, none: no access)
  # (optional)
  actions: "read"

  # Permission for artifact attestations (read: view attestations, write: create
  # attestations, none: no access)
  # (optional)
  attestations: "read"

  # Permission for repository checks and status checks (read: view checks, write:
  # create/update checks, none: no access)
  # (optional)
  checks: "read"

  # Permission for repository contents (read: view files, write: modify
  # files/branches, none: no access)
  # (optional)
  contents: "read"

  # Permission for repository deployments (read: view deployments, write:
  # create/update deployments, none: no access)
  # (optional)
  deployments: "read"

  # Permission for repository discussions (read: view discussions, write:
  # create/update discussions, none: no access)
  # (optional)
  discussions: "read"

  # (optional)
  id-token: "read"

  # Permission for repository issues (read: view issues, write: create/update/close
  # issues, none: no access)
  # (optional)
  issues: "read"

  # Permission for GitHub Copilot models (read: access AI models for agentic
  # workflows, none: no access)
  # (optional)
  models: "read"

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

  # Permission shorthand that applies read access to all permission scopes. Can be
  # combined with specific write permissions to override individual scopes. 'write'
  # is not allowed for all.
  # (optional)
  all: "read"

# Custom name for workflow runs that appears in the GitHub Actions interface
# (supports GitHub expressions like ${{ github.event.issue.title }})
# (optional)
run-name: "example-value"

# Groups together all the jobs that run in the workflow
# (optional)
jobs:
  {}

# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Runner type as string
runs-on: "example-value"

# Option 2: Runner type as array
runs-on: []
  # Array items: string

# Option 3: Runner type as object
runs-on:
  # Runner group name for self-hosted runners
  # (optional)
  group: "example-value"

  # List of runner labels for self-hosted runners
  # (optional)
  labels: []
    # Array of strings

# Workflow timeout in minutes. Defaults to 15 minutes for agentic workflows. Has
# sensible defaults and can typically be omitted.
# (optional)
timeout_minutes: 10

# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple concurrency group name to prevent multiple runs. Agentic
# workflows automatically generate enhanced concurrency policies.
concurrency: "example-value"

# Option 2: Concurrency configuration object with group isolation and cancellation
# control
concurrency:
  # Concurrency group name. Workflows in the same group cannot run simultaneously.
  group: "example-value"

  # Whether to cancel in-progress workflows in the same concurrency group when a new
  # one starts
  # (optional)
  cancel-in-progress: true

# Environment variables for the workflow
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: object
env:
  {}

# Option 2: string
env: "example-value"

# Feature flags to enable experimental or optional features in the workflow. Each
# feature is specified as a key with a boolean value.
# (optional)
features:
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
    # (optional)
    username: "example-value"

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
# allowlists. Controls web fetch and search capabilities.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Use default network permissions (basic infrastructure: certificates,
# JSON schema, Ubuntu, etc.)
network: "defaults"

# Option 2: Custom network access configuration with ecosystem identifiers and
# specific domains
network:
  # List of allowed domains or ecosystem identifiers (e.g., 'defaults', 'python',
  # 'node', '*.example.com')
  # (optional)
  allowed: []
    # Array of Domain name or ecosystem identifier (supports wildcards like
    # '*.example.com' and ecosystem names like 'python', 'node')

  # AWF (Agent Workflow Firewall) configuration for network egress control. Only
  # supported for Copilot engine.
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable AWF with default settings (equivalent to empty object)
  firewall: null

  # Option 2: Enable (true) or explicitly disable (false) AWF firewall
  firewall: true

  # Option 3: Disable AWF firewall (triggers warning if allowed != *, error in
  # strict mode if allowed is not * or engine does not support firewall)
  firewall: "disable"

  # Option 4: Custom AWF configuration with version and arguments
  firewall:
    # Optional additional arguments to pass to AWF wrapper
    # (optional)
    args: []
      # Array of strings

    # AWF version to use (empty = latest release)
    # (optional)
    version: "example-value"

    # AWF log level (default: info). Valid values: debug, info, warn, error
    # (optional)
    log-level: "debug"

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
# executes the markdown content of the workflow. Defaults to 'claude'.
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Simple engine name: 'claude' (default, Claude Code), 'copilot' (GitHub
# Copilot CLI), 'codex' (OpenAI Codex CLI), or 'custom' (user-defined steps)
engine: "claude"

# Option 2: Extended engine configuration object with advanced options for model
# selection, turn limiting, environment variables, and custom steps
engine:
  # AI engine identifier: 'claude' (Claude Code), 'codex' (OpenAI Codex CLI),
  # 'copilot' (GitHub Copilot CLI), or 'custom' (user-defined GitHub Actions steps)
  id: "claude"

  # Optional version of the AI engine action (e.g., 'beta', 'stable'). Has sensible
  # defaults and can typically be omitted.
  # (optional)
  version: "example-value"

  # Optional specific LLM model to use (e.g., 'claude-3-5-sonnet-20241022',
  # 'gpt-4'). Has sensible defaults and can typically be omitted.
  # (optional)
  model: "example-value"

  # Maximum number of chat iterations per run. Helps prevent runaway loops and
  # control costs. Has sensible defaults and can typically be omitted.
  # (optional)
  max-turns: 1

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

  # Custom environment variables to pass to the AI engine, including secret
  # overrides (e.g., OPENAI_API_KEY: ${{ secrets.CUSTOM_KEY }})
  # (optional)
  env:
    {}

  # Custom GitHub Actions steps for 'custom' engine. Define your own deterministic
  # workflow steps instead of using AI processing.
  # (optional)
  steps: []
    # Array items:

  # Custom error patterns for validating agent logs
  # (optional)
  error_patterns: []
    # Array items:
      # Unique identifier for this error pattern
      # (optional)
      id: "example-value"

      # Ecma script regular expression pattern to match log lines
      pattern: "example-value"

      # Capture group index (1-based) that contains the error level. Use 0 to infer from
      # pattern content.
      # (optional)
      level_group: 1

      # Capture group index (1-based) that contains the error message. Use 0 to use the
      # entire match.
      # (optional)
      message_group: 1

      # Human-readable description of what this pattern matches
      # (optional)
      description: "Description of the workflow"

  # Additional TOML configuration text that will be appended to the generated
  # config.toml in the action (codex engine only)
  # (optional)
  config: "example-value"

  # Optional array of command-line arguments to pass to the AI engine CLI. These
  # arguments are injected after all other args but before the prompt.
  # (optional)
  args: []
    # Array of strings

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

  # Option 2: Simple GitHub tool configuration (enables all GitHub API functions)
  github: "example-value"

  # Option 3: GitHub tools object configuration with restricted function access
  github:
    # List of allowed GitHub API functions (e.g., 'create_issue', 'update_issue',
    # 'add_comment')
    # (optional)
    allowed: []
      # Array of strings

    # MCP server mode: 'local' (Docker-based, default) or 'remote' (hosted at
    # api.githubcopilot.com)
    # (optional)
    mode: "local"

    # Optional version specification for the GitHub MCP server (used with 'local'
    # type)
    # (optional)
    version: "example-value"

    # Optional additional arguments to append to the generated MCP server command
    # (used with 'local' type)
    # (optional)
    args: []
      # Array of strings

    # Enable read-only mode to restrict GitHub MCP server to read-only operations only
    # (optional)
    read-only: true

    # Optional custom GitHub token (e.g., '${{ secrets.CUSTOM_PAT }}'). For 'remote'
    # type, defaults to GH_AW_GITHUB_TOKEN if not specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Array of GitHub MCP server toolset names to enable specific groups of GitHub API
    # functionalities
    # (optional)
    toolset: []
      # Array of Toolset name

  # Bash shell command execution tool for running command-line programs and scripts
  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Enable bash tool with all shell commands allowed (security
  # consideration: use restricted list in production)
  bash: null

  # Option 2: Enable bash tool - true allows all commands (equivalent to ['*']),
  # false disables the tool
  bash: true

  # Option 3: List of allowed bash commands and patterns (e.g., ['echo', 'ls', 'git
  # status', 'npm install'])
  bash: []
    # Array items: string

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

  # Option 1: Enable Playwright tool with default settings (localhost access only
  # for security)
  playwright: null

  # Option 2: Playwright tool configuration with custom version and domain
  # restrictions
  playwright:
    # Optional Playwright container version (e.g., 'v1.41.0')
    # (optional)
    version: "example-value"

    # Domains allowed for Playwright browser network access. Defaults to localhost
    # only for security.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: List of allowed domains or patterns (e.g., ['github.com',
    # '*.example.com'])
    allowed_domains: []
      # Array items: string

    # Option 2: Single allowed domain (e.g., 'github.com')
    allowed_domains: "example-value"

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

    # Docker image to use for the memory MCP server (default: mcp/memory)
    # (optional)
    docker-image: "example-value"

    # Number of days to retain uploaded artifacts (1-90 days, default: repository
    # setting)
    # (optional)
    retention-days: 1

  # Option 4: Array of cache-memory configurations for multiple caches
  cache-memory: []
    # Array items: object

  # Enable or disable XPIA (Cross-Prompt Injection Attack) security warnings in the
  # prompt. Defaults to true (enabled). Set to false to disable security warnings.
  # (optional)
  safety-prompt: true

  # Timeout in seconds for tool/MCP server operations. Applies to all tools and MCP
  # servers if supported by the engine. Default varies by engine (Claude: 60s,
  # Codex: 120s).
  # (optional)
  timeout: 1

  # Timeout in seconds for MCP server startup. Applies to MCP server initialization
  # if supported by the engine. Default: 120 seconds.
  # (optional)
  startup-timeout: 1

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

  # This field supports multiple formats (oneOf):

  # Option 1: A single path to cache
  path: "example-value"

  # Option 2: Multiple paths to cache
  path: []
    # Array items: string

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

# Option 2: Multiple cache configurations
cache: []
  # Array items: object

# Safe output processing configuration that automatically creates GitHub issues,
# comments, and pull requests from AI workflow output without requiring write
# permissions in the main job
# (optional)
safe-outputs:
  # List of allowed domains for URI filtering in AI workflow output. URLs from other
  # domains will be replaced with '(redacted)' for security.
  # (optional)
  allowed-domains: []
    # Array of strings

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

    # Maximum number of issues to create (default: 1)
    # (optional)
    max: 1

    # Minimum number of issues to create (default: 0 - no requirement)
    # (optional)
    min: 1

    # Target repository in format 'owner/repo' for cross-repository issue creation.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable issue creation with default configuration
  create-issue: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub Copilot agent tasks from agentic
  # workflow output using gh agent-task CLI. The main job does not need write
  # permissions.
  create-agent-task:
    # Base branch for the agent task pull request. Defaults to the current branch or
    # repository default branch.
    # (optional)
    base: "example-value"

    # Maximum number of agent tasks to create (default: 1)
    # (optional)
    max: 1

    # Minimum number of agent tasks to create (default: 0 - no requirement)
    # (optional)
    min: 1

    # Target repository in format 'owner/repo' for cross-repository agent task
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable agent task creation with default configuration
  create-agent-task: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub discussions from agentic workflow
  # output
  create-discussion:
    # Optional prefix for the discussion title
    # (optional)
    title-prefix: "example-value"

    # Optional discussion category. Can be a category ID (string or number), category
    # name, or category slug/route. If not specified, uses the first available
    # category. Matched first against category IDs, then against category names, then
    # against category slugs.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Discussion category name or ID
    category: "example-value"

    # Option 2: Discussion category ID as a number
    category: 1

    # Maximum number of discussions to create (default: 1)
    # (optional)
    max: 1

    # Minimum number of discussions to create (default: 0 - no requirement)
    # (optional)
    min: 1

    # Target repository in format 'owner/repo' for cross-repository discussion
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable discussion creation with default configuration
  create-discussion: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for automatically creating GitHub issue or pull request
  # comments from AI workflow output. The main job does not need write permissions.
  add-comment:
    # Maximum number of comments to create (default: 1)
    # (optional)
    max: 1

    # Minimum number of comments to create (default: 0 - no requirement)
    # (optional)
    min: 1

    # Target for comments: 'triggering' (default), '*' (any issue), or explicit issue
    # number
    # (optional)
    target: "example-value"

    # Target repository in format 'owner/repo' for cross-repository comments. Takes
    # precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

    # Target discussion comments instead of issue/PR comments. Must be true if
    # present.
    # (optional)
    discussion: true

  # Option 2: Enable issue comment creation with default configuration
  add-comment: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub pull requests from agentic workflow
  # output
  create-pull-request:
    # Optional prefix for the pull request title
    # (optional)
    title-prefix: "example-value"

    # Optional list of labels to attach to the pull request
    # (optional)
    labels: []
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

    # Whether to create pull request as draft (defaults to true)
    # (optional)
    draft: true

    # Behavior when no changes to push: 'warn' (default - log warning but succeed),
    # 'error' (fail the action), or 'ignore' (silent success)
    # (optional)
    if-no-changes: "warn"

    # Target repository in format 'owner/repo' for cross-repository pull request
    # creation. Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable pull request creation with default configuration
  create-pull-request: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating GitHub pull request review comments from
  # agentic workflow output
  create-pull-request-review-comment:
    # Maximum number of review comments to create (default: 1)
    # (optional)
    max: 1

    # Minimum number of review comments to create (default: 0 - no requirement)
    # (optional)
    min: 1

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

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable PR review comment creation with default configuration
  create-pull-request-review-comment: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for creating repository security advisories (SARIF
  # format) from agentic workflow output
  create-code-scanning-alert:
    # Maximum number of security findings to include (default: unlimited)
    # (optional)
    max: 1

    # Minimum number of security findings to include (default: 0 - no requirement)
    # (optional)
    min: 1

    # Driver name for SARIF tool.driver.name field (default: 'GitHub Agentic Workflows
    # Security Scanner')
    # (optional)
    driver: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable code scanning alert creation with default configuration
  # (unlimited findings)
  create-code-scanning-alert: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Null configuration allows any labels
  add-labels: null

  # Option 2: Configuration for adding labels to issues/PRs from agentic workflow
  # output
  add-labels:
    # Optional list of allowed labels that can be added. If omitted, any labels are
    # allowed (including creating new ones).
    # (optional)
    allowed: []
      # Array of strings

    # Optional maximum number of labels to add (default: 3)
    # (optional)
    max: 1

    # Minimum number of labels to add (default: 0 - no requirement)
    # (optional)
    min: 1

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

    # Allow updating issue body - presence of key indicates field can be updated
    # (optional)
    body: null

    # Maximum number of issues to update (default: 1)
    # (optional)
    max: 1

    # Minimum number of issues to update (default: 0 - no requirement)
    # (optional)
    min: 1

    # Target repository in format 'owner/repo' for cross-repository issue updates.
    # Takes precedence over trial target repo settings.
    # (optional)
    target-repo: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable issue updating with default configuration
  update-issue: null

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Use default configuration (branch: 'triggering', if-no-changes:
  # 'warn')
  push-to-pull-request-branch: null

  # Option 2: Configuration for pushing changes to a specific branch from agentic
  # workflow output
  push-to-pull-request-branch:
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

    # Optional suffix to append to generated commit titles (e.g., ' [skip ci]' to
    # prevent triggering CI on the commit)
    # (optional)
    commit-title-suffix: "example-value"

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for reporting missing tools from agentic workflow output
  missing-tool:
    # Maximum number of missing tool reports (default: unlimited)
    # (optional)
    max: 1

    # Minimum number of missing tool reports (default: 0 - no requirement)
    # (optional)
    min: 1

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable missing tool reporting with default configuration
  missing-tool: null

  # Option 3: Explicitly disable missing tool reporting (false). Missing tool
  # reporting is enabled by default when safe-outputs is configured.
  missing-tool: true

  # (optional)
  # This field supports multiple formats (oneOf):

  # Option 1: Configuration for publishing assets to an orphaned git branch
  upload-assets:
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

    # Maximum number of assets to upload (default: 10)
    # (optional)
    max: 1

    # Minimum number of assets to upload (default: 0 - no requirement)
    # (optional)
    min: 1

    # GitHub token to use for this specific output type. Overrides global github-token
    # if specified.
    # (optional)
    github-token: "${{ secrets.GITHUB_TOKEN }}"

  # Option 2: Enable asset publishing with default configuration
  upload-assets: null

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

  # Maximum allowed size for git patches in kilobytes (KB). Defaults to 1024 KB (1
  # MB). If patch exceeds this size, the job will fail.
  # (optional)
  max-patch-size: 1

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
    # workflow engine). Supports same format as main engine field.
    # (optional)
    # This field supports multiple formats (oneOf):

    # Option 1: Simple engine name: 'claude' (default, Claude Code), 'copilot' (GitHub
    # Copilot CLI), 'codex' (OpenAI Codex CLI), or 'custom' (user-defined steps)
    engine: "claude"

    # Option 2: Extended engine configuration object with advanced options for model
    # selection, turn limiting, environment variables, and custom steps
    engine:
      # AI engine identifier: 'claude' (Claude Code), 'codex' (OpenAI Codex CLI),
      # 'copilot' (GitHub Copilot CLI), or 'custom' (user-defined GitHub Actions steps)
      id: "claude"

      # Optional version of the AI engine action (e.g., 'beta', 'stable'). Has sensible
      # defaults and can typically be omitted.
      # (optional)
      version: "example-value"

      # Optional specific LLM model to use (e.g., 'claude-3-5-sonnet-20241022',
      # 'gpt-4'). Has sensible defaults and can typically be omitted.
      # (optional)
      model: "example-value"

      # Maximum number of chat iterations per run. Helps prevent runaway loops and
      # control costs. Has sensible defaults and can typically be omitted.
      # (optional)
      max-turns: 1

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

      # Custom environment variables to pass to the AI engine, including secret
      # overrides (e.g., OPENAI_API_KEY: ${{ secrets.CUSTOM_KEY }})
      # (optional)
      env:
        {}

      # Custom GitHub Actions steps for 'custom' engine. Define your own deterministic
      # workflow steps instead of using AI processing.
      # (optional)
      steps: []
        # Array items:

      # Custom error patterns for validating agent logs
      # (optional)
      error_patterns: []
        # Array items:
          # Unique identifier for this error pattern
          # (optional)
          id: "example-value"

          # Ecma script regular expression pattern to match log lines
          pattern: "example-value"

          # Capture group index (1-based) that contains the error level. Use 0 to infer from
          # pattern content.
          # (optional)
          level_group: 1

          # Capture group index (1-based) that contains the error message. Use 0 to use the
          # entire match.
          # (optional)
          message_group: 1

          # Human-readable description of what this pattern matches
          # (optional)
          description: "Description of the workflow"

      # Additional TOML configuration text that will be appended to the generated
      # config.toml in the action (codex engine only)
      # (optional)
      config: "example-value"

      # Optional array of command-line arguments to pass to the AI engine CLI. These
      # arguments are injected after all other args but before the prompt.
      # (optional)
      args: []
        # Array of strings

    # Array of extra job steps to run after detection
    # (optional)
    steps: []

  # Custom safe-output jobs that can be executed based on agentic workflow output.
  # Job names containing dashes will be automatically normalized to underscores
  # (e.g., 'send-notification' becomes 'send_notification').
  # (optional)
  jobs:
    {}

  # Runner specification for all safe-outputs jobs (activation, create-issue,
  # add-comment, etc.). Single runner label (e.g., 'ubuntu-latest',
  # 'windows-latest', 'self-hosted')
  # (optional)
  runs-on: "example-value"

# Repository access roles required to trigger agentic workflows. Defaults to
# ['admin', 'maintainer', 'write'] for security. Use 'all' to allow any
# authenticated user (⚠️ security consideration).
# (optional)
# This field supports multiple formats (oneOf):

# Option 1: Allow any authenticated user to trigger the workflow (⚠️ disables
# permission checking entirely - use with caution)
roles: "all"

# Option 2: List of repository permission levels that can trigger the workflow.
# Permission checks are automatically applied to potentially unsafe triggers.
roles: []
  # Array items: Repository permission level: 'admin' (full access),
  # 'maintainer'/'maintain' (repository management), 'write' (push access), 'triage'
  # (issue management)

# GitHub Actions workflow step
# (optional)
# This field supports multiple formats (anyOf):

# Option 1: undefined

# Option 2: undefined

# Enable strict mode validation: require timeout, refuse write permissions,
# require network configuration. Defaults to false.
# (optional)
strict: true

# Runtime environment version overrides. Allows customizing runtime versions
# (e.g., Node.js, Python) or defining new runtimes. Runtimes from imported shared
# workflows are also merged.
# (optional)
runtimes:
  {}

# GitHub token expression to use for all steps that require GitHub authentication.
# Typically a secret reference like ${{ secrets.GITHUB_TOKEN }} or ${{
# secrets.CUSTOM_PAT }}. If not specified, defaults to ${{
# secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}. This value can be
# overridden by safe-outputs github-token or individual safe-output github-token
# fields.
# (optional)
github-token: "${{ secrets.GITHUB_TOKEN }}"
---
```

## Additional Information

- Fields marked with `(optional)` are not required
- Fields with multiple options show all possible formats
- See the [Frontmatter guide](/gh-aw/reference/frontmatter/) for detailed explanations and examples
- See individual reference pages for specific topics like [Triggers](/gh-aw/reference/triggers/), [Tools](/gh-aw/reference/tools/), and [Safe Outputs](/gh-aw/reference/safe-outputs/)
