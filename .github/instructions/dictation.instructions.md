Fix text-to-speech errors in dictated text for creating agentic workflow prompts in the gh-aw repository.

## Project Glossary

The following terms are specific to this project and should be recognized and used correctly:

activation
add-comment
add-labels
agent-finish
agentic
agentic-workflows
allowed
allowed-domains
args
assignees
audit
bash
cache-memory
changeset
checkout
claude
codex
compile
compiler
concurrency
copilot
create-discussion
create-issue
create-pull-request
custom
defaults
deps
deps-dev
discussion
downstream
draft
edit
engine
fmt
frontmatter
gh-aw
github
github-token
issue_comment
issues
jobs
labels
lint
lock.yml
lockfile
logger
logs
main
max-concurrency
max-turns
mcp
mcp-config
mcp-servers
metadata
missing-tool
model
network
on
permissions
playwright
post-steps
pull_request
pull_request_review_comment
recompile
refs
remote
rendering
repository
reviewers
roles
run-name
runner
runs-on
safe-outputs
sanitized
schema
secrets
session
staged
step
steps
stop-after
strict
struct
tag
target
target-repo
tavily
test-unit
timeout
timeout_minutes
title-prefix
toolset
tools
trial
trigger
update-issue
upload
validation
version
web-fetch
web-search
workflow
workflow_dispatch
workflows

## Technical Context

GitHub Agentic Workflows (gh-aw) - a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files with YAML frontmatter, which compile to GitHub Actions workflows.

## Fix Speech-to-Text Errors

Replace speech-to-text ambiguities with correct technical terms from the glossary:

- "ghaw" → "gh-aw"
- "work flow" → "workflow"
- "front matter" → "frontmatter"
- "lock file" → "lockfile" or ".lock.yml"
- "co-pilot" → "copilot"
- "safe outputs" → "safe-outputs"
- "cache memory" → "cache-memory"
- "max turns" → "max-turns"
- "issue comment" → "issue_comment"
- "pull request" → "pull_request"
- "workflow dispatch" → "workflow_dispatch"
- "runs on" → "runs-on"
- "timeout minutes" → "timeout_minutes"
