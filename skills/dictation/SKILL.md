---
name: Dictation Instructions
description: Fix speech-to-text errors and improve text clarity in dictated content related to GitHub Agentic Workflows
applyTo: "**/*"
---

# Dictation Instructions

## Technical Context

GitHub Agentic Workflows (gh-aw) is a CLI tool for writing agentic workflows in natural language using markdown files and running them as GitHub Actions. When fixing dictated text, use these project-specific terms and conventions, and improve text clarity by removing filler words and making it more professional.

## Project Glossary

.github
.lock.yml
.md
@copilot
acme-org
activation
add-comment
add-labels
add-reviewer
add_comment
add_labels
agent-performance-analyzer
agent-session
agent_output
agentic
agentic-chat
agentic-workflow
agentic-workflows
agentics-maintenance
allowed-domains
allowed-github-references
allowed-labels
allowed-reasons
allowed_domains
anti-patterns
api_key
app-id
app_id
app_private_key
append-only
append-only-comments
assign-milestone
assign-to-agent
assign-to-user
assigning-an-issue-to-copilot
ast-grep
audit
audit-workflows
authoring-workflows
auto-assign
auto-close
auto-commit
auto-expiration
auto-generated
auto-label
auto-merge
auto-triage
aw_audit
aw_logs
bash
border-radius
branch-name
branch-prefix
breadth-first
breaking-change-checker
cache-memory
campaign_id
campaigns
cancel-in-progress
central-tracker
check_run
check_suite
claude
claude-sonnet-4
cli-consistency-checker
clone-repo
close-discussion
close-issue
close-older-discussions
close-pull-request
close_older
code-review
code_security
codex
coding-agent
command-line
command-triggers
comment-related
comment-triggered
common-issues
common-tools
compilation-time
compile
container
content-type
context-aware
continuous-ai
copilot
copilot-agent-analysis
copilot-pr-nlp-analysis
copilot-session-insights
copilot_github_token
copy-project
create
create-a-pr
create-agent-session
create-code-scanning-alert
create-discussion
create-issue
create-project
create-project-status-update
create-pull-request
create-pull-request-review-comment
create_discussion
create_issue
create_pull_request
cross-component
cross-references
cross-repo
cross-repository
cross_repo_pat
custom
custom-agent
custom-safe-outputs
custom_pat
daily-accessibility-review
daily-doc-updater
daily-fact
daily-file-diet
daily-firewall-report
daily-multi-device-docs-tester
daily-news
daily-perf-improver
daily-qa
daily-repo-chronicle
daily-secrets-analysis
daily-team-status
daily-test-improver
daily-workflow-updater
database_url
defense-in-depth
delete-host-repo-after
design-patterns
deterministic-agentic-patterns
dev-hawk
discussion_comment
discussions
dispatch-only
docker
domain-based
downstream-service
dry-run
duplicate-code-detector
edit
engine-id
engine-specific
event-driven
event-triggered
events-that-trigger-workflows
example-value
file-glob
fine-grained
fmt
framework-upgrade
frontmatter
get_page
getting-started
gh-aw
gh_aw_agent_token
gh_aw_github_mcp_server_token
gh_aw_github_token
gh_aw_project_github_token
gh_token
github
github-agentic-workflows
github-hosted
github-projects-v2
github-token
github-tools-github
github_personal_access_token
github_token
glob
glossary-maintainer
grep
grumpy-reviewer
hide-comment
hide-older-comments
high-priority
high-severity
host-repo
hourly-ci-cleaner
how-tos
how-workflows-work
html_url
http-based
http-only
https-only
hub-and-spoke
human-friendly
human-in-the-loop
human-readable
implementation-dependent
import-b
imports-and-sharing
input_repo
install-gh-aw
issue-arborist
issue-monster
issue-pr-events
issue-triage
issue-triage-agent
issue-triggered
issue_comment
issue_number
issues
job-level
job-specific
json
json-rpc
label-based
link-sub-issue
lint
list-tools
local
lock-for-agent
lock.yml
lockfile
log-level
logical-repo
logs
long-term
main-repo
main_repo_pat
manual
manual-approval
markdown
max-comments-per-run
max-file-count
max-file-size
max-patch-size
max-project-updates-per-run
max_items
mcp
mcp-gateway
mcp-inspector
mcp-model-context-protocol
mcp-server
mcp-servers
meet-the-workflows
meet-the-workflows-advanced-analytics
meet-the-workflows-continuous-simplicity
meet-the-workflows-creative-culture
meet-the-workflows-documentation
meet-the-workflows-interactive-chatops
meet-the-workflows-issue-management
meet-the-workflows-metrics-analytics
meet-the-workflows-multi-phase
meet-the-workflows-operations-release
meet-the-workflows-organization
meet-the-workflows-quality-hygiene

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
