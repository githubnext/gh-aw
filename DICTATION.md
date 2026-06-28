---
name: dictation
description: Dictation instructions for fixing speech-to-text errors and improving text quality in gh-aw workflows
---

# Dictation Instructions

## Technical Context

gh-aw (GitHub Agentic Workflows) is a CLI extension for GitHub that compiles markdown workflow files into GitHub Actions YAML. It enables AI-powered workflows using natural language instructions with support for multiple engines (Copilot, Claude, Codex, Gemini), tools (GitHub API, bash, web-fetch, playwright), and security features (safe-outputs, network permissions, integrity levels).

## Project Glossary

acceptEdits
activation
actor
add-comment
add-labels
add-wizard
agent
agent-output.json
agentic
agentic-ops
agentic-workflows
AGENTS.md
ai-credits
allowed-domains
allowed-files
allowed-github-references
allowed-repos
ANTHROPIC_API_KEY
api-target
append
approval-labels
artifact
artifacts
assign-milestone
assign-to-agent
audit
auto-triage-issues
aw-info.json
aw.json
aw.yml
bash
batch-ops
blocked
blocked-users
bypassPermissions
cache
cache-memory
call-workflow
central-repo-ops
chat-ops
checkout
claude
CLAUDE.md
client-id
close-discussion
close-issue
close-pull-request
codex
comment-memory
compilation
compile
conclusion
concurrency
contents
copilot
COPILOT_GITHUB_TOKEN
COPILOT_MODEL
correction-ops
create-discussion
create-issue
create-pull-request
create-pull-request-review-comment
cron
cross-repository
crush
custom-safe-outputs
daily
dependencies
detection
deterministic-ops
discussion
discussion-number
discussions
dispatch-ops
dispatch-workflow
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
firewall
firewall-audit-logs
footer
frontmatter
fuzzy-schedule
gemini
gemini-flash
gemini-pro
GEMINI_API_KEY
gh-aw
GH_AW_DEFAULT_DETECTION_MODEL
GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS
GH_HOST
github
github-app
github-token
github-tools
GITHUB_TOKEN
gpt-5
gpt-5-mini
haiku
headers
hide-comment
hide-older-comments
id-token
import-schema
imports
inline-sub-agents
inlined-imports
integrity
integrity-reactions
issue
issue-comment
issue-number
issue-ops
issues
label-command
label-ops
labels
link-sub-issue
lock.yml
lockdown-mode
markdown
max-ai-credits
max-continuations
max-daily-ai-credits
max-patch-size
max-runs
max-turns
mcp
mcp-gateway
mcp-scripts
mcp-servers
MCP_GATEWAY_SESSION_TIMEOUT
memory-ops
merge
merge-pull-request
min-integrity
min-samples
min-version
model-alias
monitor-ops
multi-phase
multi-repo-ops
network
network.allowed
noop
opencode
OPENAI_API_KEY
opentelemetry
opus
orchestration
orchestrator-ops
organization
otel
otlp
payloadDir
permissions
pi
playwright
post-steps
pre-activation
pre-steps
prepend
preserve-branch-name
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
push-to-pull-request-branch
rate-limiting
README.md
refusal-labels
registry
remove-labels
replace
repo-memory
reply-to-pull-request-review-comment
report-incomplete
required-labels
required-title-prefix
research-plan-assign-ops
resolve-pull-request-review-thread
runs-on
runs-on-slim
safe-outputs
safe-rollout
sandbox
schedule
serena
sessionTimeout
skip-if-match
slash-command
sonnet
spanId
spec-ops
staged
start-date
state.json
state.runs
steps.sanitized.outputs.body
steps.sanitized.outputs.text
steps.sanitized.outputs.title
stop-after
sub-agent
sub-issues
target-repo
threat-detection
timeout-minutes
timezone
title-prefix
tools.github
tools.timeout
toolsets
total-effective-tokens
traceId
traceparent
trial-ops
trusted-users
trustedBots
unapproved
update-discussion
update-issue
update-project
update-pull-request
user-rate-limit
variants
web-fetch
web-search
weekly
weight
workflow-call
workflow-dispatch
workflow_call
workflow_dispatch
workqueue-ops
YAML

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
- "cross repository" → "cross-repository" (hyphenated)
- "sub agent" → "sub-agent" (hyphenated)
- "sub issues" → "sub-issues" (hyphenated)
- "GitHub tools" → "github-tools" (hyphenated in YAML context)
- "hide older comments" → "hide-older-comments" (hyphenated)
- "preserve branch name" → "preserve-branch-name" (hyphenated)
- "max daily AI credits" → "max-daily-ai-credits" (hyphenated)
- "AI credits" → "ai-credits" (hyphenated)
- "open code" → "opencode" (one word when used as engine name)
- "open AI API key" → "OPENAI_API_KEY" (uppercase with underscores)
- "crush" → "crush" (CLI engine name, lowercase)
- "anthropic API key" → "ANTHROPIC_API_KEY" (uppercase with underscores)
- "gemini API key" → "GEMINI_API_KEY" (uppercase with underscores)
- "close issue" → "close-issue" (hyphenated in YAML context)
- "close pull request" → "close-pull-request" (hyphenated in YAML context)
- "batch ops" → "batch-ops" (hyphenated)
- "chat ops" → "chat-ops" (hyphenated)
- "issue ops" → "issue-ops" (hyphenated)
- "label ops" → "label-ops" (hyphenated)
- "memory ops" → "memory-ops" (hyphenated)
- "monitor ops" → "monitor-ops" (hyphenated)
- "multi repo ops" → "multi-repo-ops" (hyphenated)
- "orchestrator ops" → "orchestrator-ops" (hyphenated)
- "spec ops" → "spec-ops" (hyphenated)
- "trial ops" → "trial-ops" (hyphenated)
- "workqueue ops" → "workqueue-ops" (hyphenated)
- "safe rollout" → "safe-rollout" (hyphenated)
- "inline sub agents" → "inline-sub-agents" (hyphenated)
- "lockdown mode" → "lockdown-mode" (hyphenated)
- "agentic ops" → "agentic-ops" (hyphenated)

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
