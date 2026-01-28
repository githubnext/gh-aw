package workflow

// Safe Outputs Module Organization
//
// This file serves as documentation for the safe_outputs_* module organization.
// Safe outputs provide secure handling of workflow outputs that perform write operations
// (create issue, add comment, update PR, etc.) with sanitization and permission boundaries.
//
// # Architecture
//
// The safe_outputs functionality has been split into multiple focused files:
//
//   - safe_outputs_config.go: Configuration parsing and validation for safe output definitions
//   - safe_outputs_steps.go: Step builders for GitHub Script and custom actions
//   - safe_outputs_env.go: Environment variable helpers for output data passing
//   - safe_outputs_jobs.go: Job assembly and orchestration for safe output execution
//
// # Key Concepts
//
// Safe Outputs are write operations that run in separate GitHub Actions jobs with
// minimal permissions (e.g., issues:write, pull-requests:write) to prevent prompt
// injection attacks. The main agent job runs read-only, and writes are gated through
// explicit safe output jobs that sanitize inputs.
//
// Example safe output types:
//   - create_issue: Creates GitHub issues with sanitized title/body
//   - add_comment: Adds comments to issues/PRs
//   - update_pull_request: Updates PR title, body, or labels
//   - create_pull_request: Creates new pull requests
//
// # Related Documentation
//
// See compiler_safe_outputs.go for the main safe outputs compilation logic that
// orchestrates these modules during workflow generation.
