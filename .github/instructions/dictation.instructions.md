---
description: Dictation Instructions
---

# Dictation Instructions

Fix text-to-speech errors in dictated text for creating agentic workflow prompts in the gh-aw repository.

## Project Glossary

The following terms are specific to this project and should be recognized and used correctly:

actionlint
activation
add-comment
add-labels
agentic
agentic-workflows
agent
agent-task
allowed
allowed-domains
anthropic
args
assign-milestone
assignees
audit
bash
cache
cache-memory
chatops
checkout
claude
cli
close-discussion
codex
command
compile
concurrency
container
contents
copilot
create-agent-task
create-code-scanning-alert
create-discussion
create-issue
create-pull-request
create-pull-request-review-comment
custom
dailyops
defaults
description
disable
discussion
discussion-comment
docker
draft
edit
enable
engine
env
environment
firewall
fmt
fork
forks
frontmatter
gh-aw
github
github-token
gpt-5
if-no-changes
imports
init
issue-comment
issue_comment
issueops
issues
job
jobs
labelops
labels
lint
lock-yml
lockfile
log-level
logs
manual-approval
markdown
max
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
new
noop
npm
npmjs
on
output
permission
permissions
playwright
post-steps
poutine
projectops
pull-request
pull-request-comment
pull-request-review-comment
pull_request
purge
push
push-to-pull-request-branch
pypi
reaction
read-only
recompile
remote
remove
repo
retention-days
reviewers
run
run-name
runs-on
safe-outputs
sanitized
sarif
schedule
schema
secrets
server
session
skip-if-match
staged
status
steps
stop-after
strict
target
target-repo
test-unit
timeout
timeout-minutes
title-prefix
toolset
toolsets
tools
trigger
triggers
trial
update
update-issue
update-project
update-release
verbose
version
web-fetch
web-search
workflow
workflow-dispatch
workflow_run
workflows
yaml
zizmor

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
- "stop after" → "stop-after"
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
- "project ops" → "projectops"
- "daily ops" → "dailyops"
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
- "update project" → "update-project"
- "update release" → "update-release"
- "close discussion" → "close-discussion"
- "missing tool" → "missing-tool"
- "add comment" → "add-comment"
- "add labels" → "add-labels"
- "assign milestone" → "assign-milestone"
- "discussion comment" → "discussion-comment"
- "issue comment" → "issue-comment" or "issue_comment"
- "pull request" → "pull-request" or "pull_request"
- "skip if match" → "skip-if-match"
- "manual approval" → "manual-approval"
- "log level" → "log-level"
- "workflow run" → "workflow_run"

## Guidelines

You do not have enough background information to plan or provide code examples.
- do NOT generate code examples
- do NOT plan steps
- focus on fixing speech-to-text errors only
