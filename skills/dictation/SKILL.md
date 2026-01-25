---
name: Dictation Instructions
description: Fix speech-to-text errors and improve text clarity in dictated content related to GitHub Agentic Workflows
applyTo: "**/*"
---

# Dictation Instructions

## Technical Context

GitHub Agentic Workflows (gh-aw) is a CLI tool for writing agentic workflows in natural language using markdown files and running them as GitHub Actions. When fixing dictated text, use these project-specific terms and conventions, and improve text clarity by removing filler words and making it more professional.

## Project Glossary

@copilot
@mention
@ref
@sha
action-mode
actions
activation
actor
admin
agent-finish
agentic
allow-write
allowed-domains
alpine
analyze
api.github.com
approved
args
argument
artifact
assigned
assignee
audit
authentication
authorization
automate
backoff
bash
bashguard
branch
build
bun
cache
cache-memory
campaign
changes-requested
chatops
checkout
checks
cjs
claude
claude-sonnet-4
closed
cmd
codex
collaborator
comment
commented
commit
compile
compliance
concurrency
container
container-image
contents
continue-on-error
contributor
copilot
create
create_code_scanning_alert
create_discussion
create_issue
create_pull_request
create_pull_request_review_comment
created
credentials
cron
cross-repository
custom
debian
deleted
deno
deny-read
deny-write
dependency
deployments
deps
deps-dev
discussion
discussion_comment
discussions
dispatch
docker
draft
edit
edited
engine
entrypoint
env
environment
fail-fast
features
fetch_copilot_cli_documentation
filesystem
firewall
flag
fmt
frontmatter
get_commit
get_file_contents
get_me
gh-aw
github
github.com
glob
go
gpt-4
grep
id-token
if
imports
input
issue
issue_comment
issue_read
issueops
issues
jobs
json
label
labeled
labelops
lifecycle
lint
list_bash
list_issues
list_pull_requests
local
localhost
lock.yml
locked
lockfile
logs
macos-latest
maintain
maintainer
manual
markdown
matrix
max-tokens
mcp
mcp-gateway
mcp-server
md
member
merge
merged
metadata
milestone
missing_data
missing_tool
model
multi-repo
needs
network
node
noop
on
open
options
organization
output
owner
packages
pages
parallel
parameter
permission
permissions
playwright
port
process
project
projectops
pull-request
pull-requests
pull_request
pull_request_read
pull_request_review
pull_request_review_comment
push
pwsh
python
read
read_bash
recompile
ref
registry
remote
reopened
report_intent
repository
retry
review-requested
reviewer
run
runner
runs-on
runtimes
safe-inputs
safe-outputs
sandbox
sanitization
schedule
scheduled
search_issues
search_pull_requests
secret
security
security-events
self-hosted
sequential
service
sh
sha
shell
skill
spec
sse
statuses
stdio
stop_bash
store_memory
strategy
summary
synchronize
task
team
temperature
test
test-unit
timeout
timeout-minutes
token
tools
toolset
toolsets
triage
trigger
ubuntu
ubuntu-latest
unassigned
unlocked
update_todo
uses
uv
validation
variable
version
view
windows-latest
workflow
workflow_dispatch
workflow_run
write
yaml
yml

## Fix Speech-to-Text Errors

Common speech-to-text misrecognitions and their corrections:

### Safe Outputs/Inputs
- "safe output" → safe-output
- "safe outputs" → safe-outputs
- "safe input" → safe-input
- "safe inputs" → safe-inputs
- "save outputs" → safe-outputs
- "save output" → safe-output

### Workflow Terms
- "agent ic workflows" → agentic workflows
- "agent tick workflows" → agentic workflows
- "work flow" → workflow
- "work flows" → workflows
- "G H A W" → gh-aw
- "G age A W" → gh-aw

### Configuration
- "front matter" → frontmatter
- "tool set" → toolset
- "tool sets" → toolsets
- "M C P servers" → MCP servers
- "M C P server" → MCP server
- "lock file" → lockfile

### Commands & Operations
- "re compile" → recompile
- "runs on" → runs-on
- "time out minutes" → timeout-minutes
- "work flow dispatch" → workflow-dispatch
- "pull request" → pull-request (in YAML contexts)

### GitHub Actions
- "add comment" → add-comment
- "add labels" → add-labels
- "close issue" → close-issue
- "create issue" → create-issue
- "pull request review" → pull-request-review

### AI Engines & Bots
- "co-pilot" → copilot (when referring to the engine)
- "Co-Pilot" → Copilot
- "at copilot" → @copilot (when assigning/mentioning the bot)
- "@ copilot" → @copilot
- "copilot" → @copilot (when context indicates assignment or mention)
- "code X" → codex
- "Code X" → Codex

### Spacing/Hyphenation Ambiguity
When context suggests a GitHub Actions key or CLI flag:
- Use hyphens: `timeout-minutes`, `runs-on`, `cache-memory`
- In YAML: prefer hyphenated form
- In prose: either form acceptable, prefer hyphenated for consistency

## Clean Up and Improve Text

Make dictated text clearer and more professional by:

### Remove Filler Words
Common filler words and verbal tics to remove:
- "humm", "hmm", "hm"
- "um", "uh", "uhh", "er", "err"
- "you know"
- "like" (when used as filler, not for comparisons)
- "basically", "actually", "essentially" (when redundant)
- "sort of", "kind of" (when used to hedge unnecessarily)
- "I mean", "I think", "I guess"
- "right?", "yeah", "okay" (at start/end of sentences)
- Repeated words: "the the", "and and", etc.

### Improve Clarity
- Make sentences more direct and concise
- Use active voice instead of passive voice where appropriate
- Remove redundant phrases
- Fix run-on sentences by splitting them appropriately
- Ensure proper sentence structure and punctuation
- Replace vague terms with specific technical terms from the glossary

### Maintain Professional Tone
- Keep technical accuracy
- Preserve the user's intended meaning
- Use neutral, technical language
- Avoid overly casual or conversational tone in technical contexts
- Maintain appropriate formality for documentation and technical discussions

### Examples
- "Um, so like, you need to basically compile the workflow, you know?" → "Compile the workflow."
- "I think we should, hmm, use safe-outputs for this" → "Use safe-outputs for this."
- "The workflow is kind of slow, actually" → "The workflow is slow."
- "You know, the MCP server needs to be configured" → "The MCP server needs to be configured."

## Guidelines

You do not have enough background information to plan or provide code examples.
- Do NOT generate code examples
- Do NOT plan steps or provide implementation guidance
- Focus on fixing speech-to-text errors (misrecognized words, spacing, hyphenation)
- Remove filler words and verbal tics (humm, you know, um, uh, like, etc.)
- Improve clarity and professionalism of the text
- Make text more direct and concise
- When unsure, prefer the hyphenated form for technical terms
- Preserve the user's intended meaning while correcting transcription errors and improving clarity
