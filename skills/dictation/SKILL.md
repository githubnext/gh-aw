---
name: dictation
description: Fix speech-to-text errors in gh-aw documentation and code
---


# Dictation Instructions

Fix text-to-speech errors in dictated text for creating agentic workflow prompts in the gh-aw repository.

## Project Glossary

The following terms are specific to this project and should be recognized and used correctly:

activation
add-comment
add-labels
add-reviewer
agentic
agentic-workflows
agent
agent-files
assign-milestone
assign-to-agent
assign-to-user
audit
auto-expiration
bash
cache-memory
campaign
chatops
close-discussion
close-issue
close-pull-request
codex
compile
concurrency
copilot
create-agent-task
create-code-scanning-alert
create-discussion
create-issue
create-pull-request
cron
cross-repository
custom-agents
dailyops
domains
draft
ecosystem
edit
engine
environment-variables
expires
features
firewall
fork
forks
frontmatter
fuzzy
gh-aw
github
github-token
glossary
hide-comment
if-no-changes
imports
issue
issue-comment
issue_comment
issueops
labels
link-sub-issue
lockdown
lockfile
log-level
logs
markdown
max-patch-size
mcp
mcp-server
mcp-servers
metadata
minimize-comment
missing-tool
monthly
network
noop
npm
npx
permissions
pip
playwright
pull-request
pull_request
push-to-pull-request-branch
pypi
reaction
read-only
recompile
repo-memory
roles
runs-on
safe-inputs
safe-outputs
sanitize
sarif
sbom
scheduled
secrets
slash-command
staged
status
strict
sub-issue
timeout-minutes
toolset
toolsets
tracker-id
trial
triggers
update-issue
update-project
update-pull-request
update-release
upload-asset
web-fetch
web-search
weekly
workflow-dispatch
workflow-run
workflow_run
workflows

## Technical Context

GitHub Agentic Workflows (gh-aw) - a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files with YAML frontmatter, which compile to GitHub Actions workflows.

## Fix Speech-to-Text Errors

Replace speech-to-text ambiguities with correct technical terms from the glossary:

- "ghaw" → "gh-aw"
- "G H A W" → "gh-aw"
- "G H dash A W" → "gh-aw"
- "gee awe" → "gh-aw"
- "work flow" → "workflow"
- "work flows" → "workflows"
- "front matter" → "frontmatter"
- "lock file" → "lockfile" or "lock-yml"
- "co-pilot" → "copilot"
- "safe inputs" → "safe-inputs"
- "safe outputs" → "safe-outputs"
- "cache memory" → "cache-memory"
- "repo memory" → "repo-memory"
- "max turns" → "max-turns"
- "max concurrency" → "max-concurrency"
- "max patch size" → "max-patch-size"
- "issue comment" → "issue-comment"
- "pull request" → "pull-request"
- "pull request comment" → "pull-request-comment"
- "pull request review comment" → "pull-request-review-comment"
- "workflow dispatch" → "workflow-dispatch"
- "work flow dispatch" → "workflow_dispatch"
- "runs on" → "runs-on"
- "run name" → "run-name"
- "timeout minutes" → "timeout-minutes"
- "allowed domains" → "allowed-domains"
- "title prefix" → "title-prefix"
- "target repo" → "target-repo"
- "read only" → "read-only"
- "if no changes" → "if-no-changes"
- "retention days" → "retention-days"
- "stop after" → "stop-after"
- "web fetch" → "web-fetch"
- "web search" → "web-search"
- "test unit" → "test-unit"
- "post steps" → "post-steps"
- "M C P" → "MCP"
- "em see pee" → "MCP"
- "MCP servers" → "mcp-servers"
- "MCP server" → "mcp-server"
- "MCP config" → "mcp-config"
- "git hub" → "github"
- "git hub token" → "github-token"
- "agentive" → "agentic"
- "agent ick" → "agentic"
- "agent tick" → "agentic"
- "label ops" → "labelops"
- "issue ops" → "issueops"
- "chat ops" → "chatops"
- "project ops" → "projectops"
- "daily ops" → "dailyops"
- "NPM JS" → "npmjs"
- "NPX" → "npx"
- "pi pi" → "pypi"
- "pip" → "pip"
- "SAR IF" → "SARIF"
- "SARIF" → "SARIF"
- "SBOM" → "SBOM"
- "es bom" → "SBOM"
- "create agent task" → "create-agent-task"
- "create code scanning alert" → "create-code-scanning-alert"
- "create discussion" → "create-discussion"
- "create issue" → "create-issue"
- "create pull request" → "create-pull-request"
- "create pull request review comment" → "create-pull-request-review-comment"
- "push to pull request branch" → "push-to-pull-request-branch"
- "update issue" → "update-issue"
- "update project" → "update-project"
- "update pull request" → "update-pull-request"
- "update release" → "update-release"
- "upload asset" → "upload-asset"
- "close discussion" → "close-discussion"
- "close issue" → "close-issue"
- "close pull request" → "close-pull-request"
- "hide comment" → "hide-comment"
- "minimize comment" → "minimize-comment"
- "missing tool" → "missing-tool"
- "add comment" → "add-comment"
- "add labels" → "add-labels"
- "add reviewer" → "add-reviewer"
- "assign milestone" → "assign-milestone"
- "assign to agent" → "assign-to-agent"
- "assign to user" → "assign-to-user"
- "link sub issue" → "link-sub-issue"
- "sub issue" → "sub-issue"
- "discussion comment" → "discussion-comment"
- "issue comment" → "issue-comment" or "issue_comment"
- "pull request" → "pull-request" or "pull_request"
- "skip if match" → "skip-if-match"
- "manual approval" → "manual-approval"
- "log level" → "log-level"
- "workflow run" → "workflow_run"
- "tracker ID" → "tracker-id"
- "custom agents" → "custom-agents"
- "agent files" → "agent-files"
- "slash command" → "slash-command" (in frontmatter)
- "environment variables" → "environment-variables"
- "auto expiration" → "auto-expiration"
- "cross repository" → "cross-repository"
- "tool set" → "toolset"
- "tool sets" → "toolsets"

## Guidelines

You do not have enough background information to plan or provide code examples.
- do NOT generate code examples
- do NOT plan steps
- focus on fixing speech-to-text errors only
