---
name: dictation
description: Dictation instructions for fixing speech-to-text errors and improving text quality in gh-aw workflows
---

# Dictation Instructions

## Technical Context

gh-aw (GitHub Agentic Workflows) is a CLI extension for GitHub that compiles markdown workflow files into GitHub Actions YAML. It enables AI-powered workflows using natural language instructions with support for multiple engines (Copilot, Claude, Codex, Gemini), tools (GitHub API, bash, web-fetch, playwright), and security features (safe-outputs, network permissions, integrity levels).

## Project Glossary

AGENTS.md
CLAUDE.md
COPILOT_GITHUB_TOKEN
COPILOT_MODEL
DOCKER_HOST
GH_AW_DEFAULT_DETECTION_MODEL
GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS
GH_HOST
GITHUB_TOKEN
MCP_GATEWAY_SESSION_TIMEOUT
README.md
YAML
acceptEdits
actions
activation
actor
add-comment
add-wizard
agentic
agentic-workflows
agent
agent-output.json
allowed
allowed-domains
allowed-files
allowed-github-references
allowed-repos
api-target
append
approval-labels
approved
artifact
artifacts
audit
avoidHourBoundary
avoidPeakMinutes
aw-info.json
aw.json
aw.yml
bash
blocked
blocked-users
body
branch
bun
bypassPermissions
cache
cache-memory
call-workflow
checkout
client-id
codex
comment-memory
compile
compilation
concurrency
conclusion
contents
copilot
create-discussion
create-issue
create-pull-request
create-pull-request-review-comment
cron
custom
custom-safe-outputs
daily
default
defaults
deno
dependencies
detection
dev-tools
discussion
discussion-number
discussions
dispatch-workflow
domain
download-artifact
edit
effective-tokens
end-date
endpoint
engine
engine.bare
engine.env
engine.mcp
engine.model
engine.permission-mode
env
ephemerals
experiment
experiments
expires
fallback-to-issue
feature-flags
field_node_id
firewall
firewall-audit-logs
footer
frontmatter
fuzzy-schedule
gemini
gemini-flash
gemini-pro
gh-aw
github
github-app
github-token
go.mod
gpt-5
gpt-5-mini
haiku
headers
hourly
http
id-token
ignore
ignore-if-missing
import-schema
imports
inlined-imports
integrity
integrity-reactions
issue
issue-comment
issue-number
issues
job
jobs
keepaliveInterval
label-command
labels
large
lock.yml
lockfile
manifest
markdown
max-continuations
max-effective-tokens
max-patch-size
max-runs
max-turns
mcp
mcp-gateway
mcp-scripts
mcp-servers
member
merge
merged
midnight
min-integrity
min-samples
mini
model
model-alias
network
network.allowed
node
noop
noon
notify
observability
opentelemetry
opus
organization
otel
otlp
owner
package.json
packages
payloadDir
pending
permissions
playwright
post-steps
pre-activation
pre-steps
prepend
private-key
projected-effective-tokens
prompt.txt
protected-files
pull-request
pull-request-number
pull-request-target
pull-requests
pull_request
pull_request_target
push
push-to-pull-request-branch
python
rate-limiting
redirect
refusal-labels
registry
replace
repo
repo-memory
report-incomplete
repository
repository-dispatch
required-labels
required-title-prefix
retention-days
runs-on
runs-on-slim
runtime
runtimes
safe-outputs
sandbox
schedule
security
serena
sessionTimeout
skip-if-match
slash-command
sonnet
spanId
staged
start-date
state.json
state.runs
steps.sanitized.outputs.body
steps.sanitized.outputs.text
steps.sanitized.outputs.title
stop-after
target-repo
threat-detection
timeout-minutes
timezone
title-prefix
toolsets
tools
tools.github
tools.timeout
total-effective-tokens
traceId
traceparent
trusted-users
trustedBots
unapproved
update-issue
update-project
upload-artifact
upload-asset
user-rate-limit
uv
variants
version
warn
web-fetch
web-search
weekly
weight
workflow-call
workflow-dispatch
workflow_call
workflow_dispatch
workflows

## Fix Speech-to-Text Errors

Common misrecognitions to correct:

- "GH away" → "gh-aw"
- "G H A W" → "gh-aw"
- "lock Y M L" → "lock.yml"
- "Y A M L" → "YAML"
- "MCP" → "MCP" (not "M C P")
- "front matter" → "frontmatter" (one word)
- "safe outputs" → "safe-outputs" (hyphenated)
- "workflow dispatch" → "workflow-dispatch" (hyphenated in YAML context)
- "pull request" → "pull-request" (hyphenated in YAML context)
- "cop pilot" → "copilot" (one word)
- "code X" → "codex" (one word)
- "bypass permissions" → "bypassPermissions" (camelCase)
- "accept edits" → "acceptEdits" (camelCase)
- "Gemini" → "gemini" (lowercase in YAML context)
- "engine dot model" → "engine.model"
- "engine dot env" → "engine.env"
- "engine dot MCP" → "engine.mcp"
- "tools dot GitHub" → "tools.github"
- "MCP gateway" → "mcp-gateway" (hyphenated)
- "MCP scripts" → "mcp-scripts" (hyphenated)
- "cache memory" → "cache-memory" (hyphenated)
- "repo memory" → "repo-memory" (hyphenated)
- "allowed repos" → "allowed-repos" (hyphenated in YAML context)
- "min integrity" → "min-integrity" (hyphenated in YAML context)
- "pre activation" → "pre-activation" (hyphenated)
- "A W dot Y M L" → "aw.yml"
- "A W dot JSON" → "aw.json"
- "GitHub token" → "github-token" (hyphenated in YAML context)
- "GitHub app" → "github-app" (hyphenated in YAML context)
- "steps dot sanitized dot outputs dot text" → "steps.sanitized.outputs.text"
- "fuzzy schedule" → "fuzzy-schedule" (hyphenated)

## Clean Up and Improve Text

- Remove filler words: humm, you know, um, uh, like, basically, actually, so, well, right, okay
- Remove false starts and repeated words
- Improve clarity and sentence structure
- Make text more professional and concise
- Fix run-on sentences
- Correct grammar and punctuation
- Maintain the user's intended meaning and tone
- Preserve technical terminology exactly as provided

## Guidelines

You do not have enough background information to plan or provide code examples.
- do NOT generate code examples
- do NOT plan steps
- focus on fixing speech-to-text errors and improving text quality
- remove filler words (humm, you know, um, uh, like, basically, actually, etc.)
- improve clarity and make text more professional
- maintain the user's intended meaning
