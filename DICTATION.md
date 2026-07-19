---
name: dictation
description: Dictation instructions for fixing speech-to-text errors and improving text quality in gh-aw workflows
---

# Dictation Instructions

## Technical Context

gh-aw (GitHub Agentic Workflows) is a CLI extension for GitHub that compiles markdown workflow files into GitHub Actions YAML. It enables AI-powered workflows using natural language instructions with support for multiple engines (Copilot, Claude, Codex, Gemini), tools (GitHub API, bash, web-fetch, playwright), and security features (safe-outputs, network permissions, integrity levels).

## Project Glossary

acceptEdits
action_required
actionlint
activation
actor
add-comment
add-labels
add-reviewer
add_comment
agentic
agentic-ops
agentic-token-audit
agentic-token-optimizer
agentic-workflows
AGENTS.md
ai-credits
aic
allow-action-refs
allow-tool
allowed-branches
allowed-domains
allowed-repos
AllowedDomains
anthropic
ANTHROPIC_API_KEY
api-proxy
api.github.com
api.githubcopilot.com
approval-labels
artifact
artifacts
audit
aw.json
aw.yml
bash
batch-ops
blocked-users
branch
bypassPermissions
cache
cache-memory
chat-ops
checkout
claude
close-issue
close-pull-request
codex
compilation
compile
compiler
concurrency
@copilot
copilot
COPILOT_GITHUB_TOKEN
create-issue
create-pull-request
cross-repository
dependabot
discussion
discussion-number
discussions
dispatch-repository
dispatch-workflow
docker-host
domains
dry-run
edit
effective-tokens
engine
engine.api-target
engine.auth
engine.driver
engine.env
engine.harness
engine.mcp
engine.model
engine.permission-mode
EngineConfig
env
experiment
experiments
fail-fast
fallback-to-issue
feature-flags
fetch-depth
file-glob
firewall
firewall-audit-logs
FirewallConfig
footer
frontmatter
fuzzy-schedule
gateway-api-key
gateway-port
gemini
gemini-flash
gemini-flash-lite
GEMINI_API_KEY
gemma
gh-aw
gh-aw-audit
gh-aw-compile
gh-aw-logs
gh-proxy
github-app
github-token
github.com
GITHUB_TOKEN
head-repo
hide-older-comments
inline-sub-agents
integrity
issue
issue-number
issue-ops
issues
json
label
label-ops
labels
lock.yml
lockdown-mode
logs
mai-code
markdown
max-daily-ai-credits
mcp
mcp-gateway
mcp-scripts
memory-ops
merge
min-integrity
missing-data
missing-tool
model
monitor-ops
multi-repo-ops
network
network.allowed
NetworkPermissions
no-op
noop
on.github-app
on.needs
on.skip-author-associations
openai
OPENAI_API_KEY
opencode
opus
orchestrator-ops
organization
otlp
outcomes
outputs.jsonl
PAT
paths-ignore
permissions
PermissionScope
persist-credentials
playwright
pre-activation
pre-agent
pre-steps
pull-number
pull-request
pull-request-number
pull-requests
pull_request
pull_request_number
pull_request_target
push-to-pull-request-branch
rate-limit
reasoning
refusal-labels
remove-labels
replay
report-as-issue
report-incomplete
repository_dispatch
required-labels
resolve-pull-request-review-thread
run-failure
run-name
run-success
run_id
runner-guard
runs-on
runtime
safe-output
safe-outputs
safe-rollout
safeoutputs.jsonl
sandbox
SandboxConfig
schedule
secret-masking
secrets
self-hosted
set-issue-type
settings.json
sink-visibility
slash-command
sonnet
spec-ops
staged
staged-output
start-date
state.json
stop-after
sub-agent
sub-agents
sub-issues
submit-pull-request-review
target-repo
threat-detection
timeout-minutes
timezone
title-prefix
token-usage
tool-timeout
tools.bash
tools.github
tools.mcp-servers
tools.playwright
tools.timeout
toolsets
trial-ops
trigger
trusted-users
ubuntu-latest
update-discussion
update-issue
update-project
update-pull-request
upload-artifact
usage.jsonl
variants
vulnerability-alerts
web-fetch
web-search
weekly
workflow
workflow-call
workflow-dispatch
workflow-logs
workflow-name
workflow_call
workflow_dispatch
workflow_id
workflow_run
workflows
workflows-dir
working-directory
workqueue-ops
YAML
zizmor

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
- "engine dot model" → "engine.model"
- "engine dot env" → "engine.env"
- "engine dot MCP" → "engine.mcp"
- "tools dot GitHub" → "tools.github"
- "MCP gateway" → "mcp-gateway" (hyphenated)
- "MCP scripts" → "mcp-scripts" (hyphenated)
- "cache memory" → "cache-memory" (hyphenated)
- "allowed repos" → "allowed-repos" (hyphenated in YAML context)
- "min integrity" → "min-integrity" (hyphenated in YAML context)
- "pre activation" → "pre-activation" (hyphenated)
- "A W dot Y M L" → "aw.yml"
- "A W dot JSON" → "aw.json"
- "GitHub token" → "github-token" (hyphenated in YAML context)
- "GitHub app" → "github-app" (hyphenated in YAML context)
- "fuzzy schedule" → "fuzzy-schedule" (hyphenated)
- "cross repository" → "cross-repository" (hyphenated)
- "sub agent" → "sub-agent" (hyphenated)
- "sub issues" → "sub-issues" (hyphenated)
- "max daily AI credits" → "max-daily-ai-credits" (hyphenated)
- "AI credits" → "ai-credits" (hyphenated)
- "open code" → "opencode" (one word when used as engine name)
- "open AI API key" → "OPENAI_API_KEY" (uppercase with underscores)
- "anthropic API key" → "ANTHROPIC_API_KEY" (uppercase with underscores)
- "gemini API key" → "GEMINI_API_KEY" (uppercase with underscores)
- "gemini flash lite" → "gemini-flash-lite" (hyphenated)
- "gemini flash" → "gemini-flash" (hyphenated)
- "agentic ops" → "agentic-ops" (hyphenated)
- "at copilot" → "@copilot" (GitHub bot mention)
- "depend a bot" → "Dependabot" (one word, capital D)
- "run time" → "runtime" (one word)
- "self hosted" → "self-hosted" (hyphenated)
- "fire wall" → "firewall" (one word)
- "action lint" → "actionlint" (one word, linter name)
- "inline sub agents" → "inline-sub-agents" (hyphenated)
- "lockdown mode" → "lockdown-mode" (hyphenated)
- "push to pull request branch" → "push-to-pull-request-branch" (hyphenated)
- "resolve pull request review thread" → "resolve-pull-request-review-thread" (hyphenated)
- "mai code" → "mai-code" (hyphenated)
- "safe rollout" → "safe-rollout" (hyphenated)
- "target repo" → "target-repo" (hyphenated)

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
