---
name: dictation
description: Dictation instructions for fixing speech-to-text errors and improving text quality in gh-aw workflows
---

# Dictation Instructions

## Technical Context

gh-aw (GitHub Agentic Workflows) is a CLI extension for GitHub that compiles markdown workflow files into GitHub Actions YAML. It enables AI-powered workflows using natural language instructions with support for multiple engines (Copilot, Claude, Codex, Gemini), tools (GitHub API, bash, web-fetch, playwright), and security features (safe-outputs, network permissions, integrity levels).

## Project Glossary

AGENTS.md
GITHUB-TOKEN
README.md
access
accessed
action
actions
activation
actor
add
add-comment
add-wizard
agentic
agentic-workflows
agent
agent-job
agent-output.json
allowed
allowed-domains
allowed-extensions
allowed-files
allowed-github-references
allowed-repos
api-target
append
approval-labels
approved
approval-labels
architecture
around
array
artifact
artifacts
authorization
auto
avoidHourBoundary
avoidPeakMinutes
aw-info.json
aw.json
aw.yml
bare
base
bash
bazel
between
blocked
blocked-users
body
boolean
branch
breadth-first
bun
cache
cache-memory
checkout
choice
chrome
chrome
client-id
cli-proxy
codex
comment-id
comment-memory
commits
compilation
component
concat
concurrency
conclusion
containers
contents
copilot
copilot-github-token
copilot-model
copilot-provider-api-key
copilot-provider-base-url
create-discussion
create-issue
create-pull-request
cron
custom
custom-jobs
dart
default
defaults
deduplicate
deno
dependencies
dependency
deployments
description
detection
dev-tools
discussion
discussion-number
discussions
docker-host
domain
dotnet
download-artifact
ecosystem
edit
effective-tokens
elixir
end-date
endpoint
engine
engineenv
env
error
experiment
experiments
expires
failed
failure
fallback-to-issue
false
feature-flags
field
firewall
firewall-audit-logs
fonts
forecasting
format
frontmatter
gemini
gemini-flash
gemini-pro
gh-aw
gh-host
gh-proxy
github
github-app
github-token
go
go-mod
gpt-5
haiku
haskell
head
headers
hourly
http
https
identifiers
id-token
ignore
ignore-if-missing
import
import-schema
imports
integer
integrity
integrity-reactions
issue
issue-comment
issue-number
issues
java
job
jobs
julia
keepaliveInterval
key
kotlin
label-command
labels
large
latex
lean
linux-distros
local
lock.yml
lockfile
lua
manifest
markdown
max-continuations
max-effective-tokens
max-patch-size
max-runs
max-turns
mcp
mcp-gateway
mcp-gateway-session-timeout
mcp-servers
member
membership
merge
merged
midnight
min-integrity
min-samples
mini
missing-data
missing-tool
model
model-alias
name
nano
needs
network
node
none
noon
noop
notify
null
number
object
observability
ocaml
optional
organization
otel
otlp
override
owner
package
package-json
packages
pages
pattern
payloadDir
pending
perl
permissions
php
playwright
post-steps
pre-activation
pre-steps
prepend
private-key
projected-effective-tokens
prompt.txt
protected-files
protocol
pull-request
pull-request-number
pull-request-target
pull-requests
push
push-to-pull-request-branch
python
qmd
rate-limiting
read
README.md
redirect
refusal-labels
registry
remote
repo
repo-memory
report-incomplete
repository
repository-dispatch
required
required-labels
required-title-prefix
restore
retention-days
ruby
runs
runs-on
runtime
runtimes
rust
safe-output-jobs
safe-outputs
sandbox
save
scala
schedule
security
service-name
session
sessionTimeout
setup
skip-if-match
slash-command
sonnet
source
span
spanId
ssl-bump
start-date
state
state.json
state.runs
statuses
step
steps
stop-after
string
swift
target
target-repo
team
terraform
threat-detection
timeout-minutes
timezone
title
toolsets
tools
topological
total-effective-tokens
trace
traceId
traceparent
true
trusted-users
trustedBots
type
unapproved
update
update-issue
update-project
upload-artifact
upload-asset
url
user
user-rate-limit
uv
value
variants
version
warn
web-fetch
web-search
weekly
weight
wildcard
workflow
workflow-call
workflow-dispatch
workflows
write
YAML
zig

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
- "Gemini" → "gemini" (lowercase in YAML context)
- "engine dot model" → "engine.model"
- "engine dot env" → "engine.env"
- "tools dot GitHub" → "tools.github"
- "MCP gateway" → "mcp-gateway" (hyphenated)
- "cache memory" → "cache-memory" (hyphenated)
- "repo memory" → "repo-memory" (hyphenated)
- "allowed repos" → "allowed-repos" (hyphenated in YAML context)
- "min integrity" → "min-integrity" (hyphenated in YAML context)
- "pre activation" → "pre-activation" (hyphenated)
- "A W dot Y M L" → "aw.yml"
- "A W dot JSON" → "aw.json"
- "GitHub token" → "github-token" (hyphenated in YAML context)
- "GitHub app" → "github-app" (hyphenated in YAML context)

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
