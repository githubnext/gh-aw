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
.github/aw
.github/workflows
.lock.yml
.md
@copilot
ACTIONS_STEP_DEBUG
ANTHROPIC_API_KEY
CLAUDE_CODE_OAUTH_TOKEN
CODEX_API_KEY
COPILOT_GITHUB_TOKEN
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
GH_TOKEN
GITHUB_TOKEN
OPENAI_API_KEY
RUNNER_TEMP
SARIF
activation
activation-job
agent-job
agentic
agentic-workflows
allow-workflows
allowed
allowed-domains
allowed-files
allowed-labels
allowed-repos
allowlist
api.github.com
artifact
assign-to-agent
assign-to-copilot
audit
audit-workflows
auto-merge
automation
bash
branch
branch-name
build
bun
cache
cache-key
cache-memory
checkout
checks
claude
code-scanning
codex
coding-agent
comment
compile
compile-workflow
compiled
compiler
concurrency
concurrency-group
config
contents
create-discussion
create-issue
create-pull-request
cron
custom
custom-agent
daily
debug
default-branch
defaults
deno
dependabot
description
dispatch-workflow
discussions
docker
documentation
dotnet
draft
edit
engine
engine-config
engines
environment
environment-variables
events
expires
fail-fast
fallback-as-issue
feature-flag
features
frontmatter
fuzzy
fuzzy-schedule
gateway
gemini
gh-aw
git-branch
git-commit
github
github-actions
github-app
github-token
github.repository
github.run_id
github/gh-aw
hourly
id-token
imports
input
inputs
inspect
installation
integration
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
latest
lockfile
logs
main
markdown
max-continuations
max-turns
mcp
mcp-gateway
mcp-inspect
mcp-list
mcp-registry
mcp-scripts
mcp-server
mcp-servers
merge
metadata
milestone
min-integrity
mode
model
needs.activation
network
network.allowed
network.firewall
node
noop
on-demand
org
organization
outputs
override
owner
package
parallel
parsing
permissions
phase
pip
pipeline
playwright
prompt
prompt-injection
protected-files
pull-requests
pull_request
pull_request_target
python
python3
recompile
registry
release
repo
repo-memory
report-summary
repository
repository_dispatch
review
runs-on
runtime
runtimes
safe-inputs
safe-outputs
sandbox
sanitized
schedule
scheduled
schema
scripts
search
secrets
security
session
setup
shared
shared-workflow
shell
slack
staged
staged-mode
stale
steps.sanitized.outputs.body
steps.sanitized.outputs.text
steps.sanitized.outputs.title
timeout
timeout-minutes
token-weights
toolsets
trigger
triggers
trusted
trusted-users
ubuntu
ubuntu-latest
update-issue
update-pull-request
validate
validation
version
version-bump
vulnerability-scan
web-fetch
web-search
webhook
weekly
workflow
workflow-dispatch
workflow-run
workflow-status
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
