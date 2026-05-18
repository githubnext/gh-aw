---
name: Dictation Instructions
description: Instructions for fixing speech-to-text errors and improving text quality in gh-aw documentation and workflows
---

# Dictation Instructions

## Technical Context

GitHub Agentic Workflows (gh-aw) is a Go-based GitHub CLI extension for writing AI-powered workflows in natural language using markdown files that compile to GitHub Actions YAML.

## Project Glossary

The following project-specific technical terms should be corrected when encountered in speech-to-text input:
.github
.github/agents/
.github/aw/imports/
.github/workflows/
.lock.yml
.md
@copilot
ACTIONS_STEP_DEBUG
ANTHROPIC_API_KEY
CLAUDE_CODE_OAUTH_TOKEN
CODEX_API_KEY
COPILOT_GITHUB_TOKEN
COPILOT_MODEL
COPILOT_PROVIDER_API_KEY
COPILOT_PROVIDER_BASE_URL
DEBUG
FUZZY:BI-WEEKLY
FUZZY:DAILY
FUZZY:WEEKLY
GEMINI_API_KEY
GH_AW_AGENT_TOKEN
GH_AW_ALLOWED_DOMAINS
GH_AW_GITHUB_TOKEN
GH_AW_PROMPT
GH_AW_SAFE_OUTPUTS
GH_AW_VERSION
GH_AW_WORKFLOW_ID
GH_HOST
GH_TOKEN
GITHUB_TOKEN
MCP_GATEWAY_SESSION_TIMEOUT
OPENAI_API_KEY
OTEL_EXPORTER_OTLP_ENDPOINT
RUNNER_TEMP
SARIF
actionlint
activation
activation-job
add-comment
add-wizard
agent-job
agentic
agentic-workflows
allowed
allowed-domains
allowed-files
allowed-labels
allowed-repos
allowlist
api-target
api.github.com
api.githubcopilot.com
approval-labels
artifact
assign-to-agent
assign-to-copilot
audit
auth
auto-merge
auto-triage-issues
automation
aw.yml
bash
blocked
blocked-users
branch
bun
bypassPermissions
cache
cache-key
cache-memory
checkout
checks
claude
code-review
code-scanning
codex
coding-agent
comment
compile
compile-workflow
compiler
concurrency
concurrency-group
contents
copilot
create-agent-session
create-discussion
create-issue
create-pull-request
create-pull-request-review-comment
cron
cross-repository
custom
custom-agent
daily
debug
default-branch
defaults
deno
dependabot
detection
discussions
dispatch-workflow
docker
dotnet
draft
engine
engine-config
env
events
experiment
experiments
expires
fallback-as-issue
fallback-to-issue
features
firewall
firewall-audit-logs
footer
frontmatter
fuzzy
fuzzy-schedule
gateway
gemini
gh aw
gh aw audit
gh aw compile
gh aw logs
gh aw update
gh aw upgrade
gh-aw
github
github-actions
github-app
github-token
github/gh-aw
headers
hourly
id-token
imports
inlined-imports
inputs
integrity-reactions
issue
issue-ops
issue_comment
issueops
issues
javascript
job-discriminator
jobs
json
json-schema
label
label-ops
labelops
labels
lockfile
logs
markdown
max-continuations
max-patch-size
max-turns
mcp
mcp-gateway
mcp-inspect
mcp-list
mcp-registry
mcp-scripts
mcp-server
mcp-servers
metadata
milestone
min-integrity
model
needs.activation
network
network.allowed
network.firewall
node
noop
on-demand
opentelemetry
organization
outputs
permissions
pip
playwright
poutine
prompt
prompt-injection
protected-files
pull-requests
pull_request
pull_request_target
python
qmd
recompile
refusal-labels
repo
repo-memory
repository
repository_dispatch
runs-on
runtime
runtimes
safe-inputs
safe-outputs
sandbox
schedule
secrets
security
setup
shared-workflow
skip-if-match
slash_command
staged
staged-mode
stale
steps.sanitized.outputs.body
steps.sanitized.outputs.text
steps.sanitized.outputs.title
target-repo
timeout
timeout-minutes
title-prefix
token-weights
toolsets
traceId
trigger
triggers
trusted
trusted-users
ubuntu-latest
unapproved
update-issue
update-pull-request
validation
version
wasm
web-fetch
web-search
webhook
weekly
workflow
workflow-dispatch
workflow-run
workflow_call
workflow_dispatch
workflow_run
workflows
workspace
write-all
yaml
zizmor

## Fix Speech-to-Text Errors

When fixing dictated text, correct these common misrecognitions:

### GitHub and Git Terms
- "get hub" → github
- "git lab" → gitlab
- "get actions" → github-actions
- "pull request" → pull-request (when used as compound modifier)
- "issue ops" → issueops
- "label ops" → labelops
- "chat ops" → chatops
- "multi repo ops" → multirepoops
- "project ops" → projectops
- "data ops" → data-ops
- "dispatch ops" → dispatch-ops
- "daily ops" → daily-ops

### Workflow Configuration
- "front matter" → frontmatter
- "safe outputs" → safe-outputs (in configuration context)
- "safe inputs" → safe-inputs (in configuration context)
- "lock file" → .lock.yml or lockfile (depending on context)
- "tool sets" → toolsets
- "M.C.P." or "M C P" → MCP
- "repo memory" → repo-memory (in configuration context)
- "cache memory" → cache-memory (in configuration context)
- "work flow" → workflow
- "timeout minutes" → timeout-minutes
- "runs on" → runs-on
- "min integrity" → min-integrity (in configuration context)
- "mcp gateway" → mcp-gateway
- "mcp scripts" → mcp-scripts
- "staged mode" → staged-mode
- "token weights" → token-weights
- "effective tokens" → effective-tokens

### AI Engines
- "co-pilot" → @copilot
- "code x" → codex
- "cloud" → claude (when referring to the AI engine)
- "gem ini" → gemini (when referring to the AI engine)
- "serena" → serena (code intelligence MCP server)

### Commands and Operations
- "G.H. A.W." → gh-aw or `gh aw` (depending on context)
- "re-compile" → recompile
- "work flow dispatch" → workflow_dispatch
- "action lint" → actionlint
- "ziz more" → zizmor
- "poo teen" → poutine
- "queue M.D." → qmd

### File Formats and Extensions
- "dot M.D." → .md
- "dot Y.A.M.L." or "dot Y M L" → .yaml or .yml
- "dot lock dot Y M L" → .lock.yml
- "jason" → JSON (when referring to format)
- "wasm" → WebAssembly or wasm (depending on context)

### Technical Patterns
- "A.P.I." → API
- "U.R.L." → URL
- "H.T.T.P." → HTTP
- "H.T.T.P.S." → HTTPS
- "S.H.A." → SHA
- "C.I." → CI
- "G.H." → GH (when referring to GitHub CLI)
- "Y.A.M.L." → YAML
- "O.I.D.C." → OIDC
- "S.A.R.I.F." → SARIF

### Hyphenation Rules
Use hyphens for compound modifiers:
- "safe outputs" → safe-outputs
- "safe inputs" → safe-inputs
- "cache memory" → cache-memory
- "timeout minutes" → timeout-minutes
- "cross repository" → cross-repository
- "pull request" → pull-request (when used as adjective)
- "mcp gateway" → mcp-gateway
- "mcp scripts" → mcp-scripts
- "token weights" → token-weights

### Environment Variables
Capitalize fully: GITHUB_TOKEN, GH_TOKEN, COPILOT_GITHUB_TOKEN, GH_AW_GITHUB_TOKEN, ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY, CLAUDE_CODE_OAUTH_TOKEN, CODEX_API_KEY

### Common Ambiguities
- "their/there/they're" → use context to determine correct spelling
- "its/it's" → its (possessive), it's (it is)
- "your/you're" → your (possessive), you're (you are)

## Clean Up and Improve Text

Remove filler words and improve clarity:

### Remove These Filler Words
- humm, um, uh, uhh, umm
- you know, like, basically, actually, literally
- kind of, sort of, I mean, I think
- right?, okay?, so yeah, well

### Improve Clarity
1. Remove redundant phrases:
   - "in order to" → "to"
   - "at this point in time" → "now"
   - "due to the fact that" → "because"
   - "in the event that" → "if"

2. Make text more concise:
   - Remove unnecessary qualifiers (very, really, quite)
   - Use active voice instead of passive voice
   - Replace wordy phrases with simpler alternatives

3. Maintain technical accuracy:
   - Keep all technical terms from the glossary
   - Preserve code examples and commands exactly
   - Don't simplify technical concepts

## Guidelines

You do not have enough background information to plan or provide code examples.
- do NOT generate code examples
- do NOT plan steps
- focus on fixing speech-to-text errors and improving text quality
- remove filler words (humm, you know, um, uh, like, basically, actually, etc.)
- improve clarity and make text more professional
- maintain the user's intended meaning
