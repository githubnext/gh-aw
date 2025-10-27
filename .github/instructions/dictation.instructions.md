# Dictation Instructions

Fix text-to-speech errors in dictated text for creating agentic workflow prompts in the gh-aw repository.

## Project Glossary

The following terms are specific to this project and should be recognized and used correctly:

activation
add-comment
add-labels
agentic
agentic-workflows
allowed
allowed-domains
args
assignees
audit
bash
cache
cache-memory
category
chatops
claude
codex
command
compile
concurrency
contents
copilot
create-agent-task
create-code-scanning-alert
create-discussion
create-issue
create-pull-request
create-pull-request-review-comment
custom
defaults
discussion
discussion-comment
draft
edit
engine
fmt
frontmatter
gh-aw
github
github-token
if-no-changes
imports
issue-comment
issue_comment
issueops
issues
jobs
labelops
labels
lint
lock-yml
lockfile
logs
max-concurrency
max-patch-size
max-turns
mcp
mcp-config
mcp-servers
memory
missing-tool
mode
model
network
npmjs
on
permissions
playwright
post-steps
pull-request
pull-request-comment
pull-request-review-comment
pull_request
purge
push-to-pull-request-branch
pypi
reaction
read-only
recompile
remote
retention-days
reviewers
run-name
runs-on
safe-outputs
sanitized
sarif
schedule
secrets
session
staged
steps
stop-after
disable-workflow-after
strict
target
target-repo
test-unit
timeout
timeout-minutes
title-prefix
toolset
tools
triggers
update-issue
verbose
version
web-fetch
web-search
workflow
workflow-dispatch
workflows

## Technical Context

GitHub Agentic Workflows (gh-aw) - a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files with YAML frontmatter, which compile to GitHub Actions workflows.

## Fix Speech-to-Text Errors

Replace speech-to-text ambiguities with correct technical terms from the glossary:

- "ghaw" → "gh-aw"
- "G H A W" → "gh-aw"
- "work flow" → "workflow"
- "work flows" → "workflows"
- "front matter" → "frontmatter"
- "lock file" → "lockfile" or "lock-yml"
- "co-pilot" → "copilot"
- "safe outputs" → "safe-outputs"
- "cache memory" → "cache-memory"
- "max turns" → "max-turns"
- "max concurrency" → "max-concurrency"
- "max patch size" → "max-patch-size"
- "issue comment" → "issue-comment"
- "pull request" → "pull-request"
- "pull request comment" → "pull-request-comment"
- "pull request review comment" → "pull-request-review-comment"
- "workflow dispatch" → "workflow-dispatch"
- "runs on" → "runs-on"
- "run name" → "run-name"
- "timeout minutes" → "timeout-minutes"
- "allowed domains" → "allowed-domains"
- "title prefix" → "title-prefix"
- "target repo" → "target-repo"
- "read only" → "read-only"
- "if no changes" → "if-no-changes"
- "retention days" → "retention-days"
- "stop after" → "stop-after" or "disable-workflow-after"
- "disable workflow after" → "disable-workflow-after"
- "web fetch" → "web-fetch"
- "web search" → "web-search"
- "test unit" → "test-unit"
- "post steps" → "post-steps"
- "M C P" → "mcp"
- "MCP" → "mcp"
- "MCP servers" → "mcp-servers"
- "MCP config" → "mcp-config"
- "git hub" → "github"
- "git hub token" → "github-token"
- "agent ick" → "agentic"
- "agent tick" → "agentic"
- "label ops" → "labelops"
- "issue ops" → "issueops"
- "chat ops" → "chatops"
- "NPM JS" → "npmjs"
- "pi pi" → "pypi"
- "SAR IF" → "sarif"
- "SARIF" → "sarif"
- "create agent task" → "create-agent-task"
- "create code scanning alert" → "create-code-scanning-alert"
- "create discussion" → "create-discussion"
- "create issue" → "create-issue"
- "create pull request" → "create-pull-request"
- "create pull request review comment" → "create-pull-request-review-comment"
- "push to pull request branch" → "push-to-pull-request-branch"
- "update issue" → "update-issue"
- "missing tool" → "missing-tool"
- "add comment" → "add-comment"
- "add labels" → "add-labels"
- "discussion comment" → "discussion-comment"
- "issue comment" → "issue-comment" or "issue_comment"
- "pull request" → "pull-request" or "pull_request"
