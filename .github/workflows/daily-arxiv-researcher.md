---
emoji: "🔬"
description: Daily scan of latest arXiv papers for actionable improvements to GitHub Agentic Workflows
on:
  schedule:
    - cron: "daily around 8:00"
  workflow_dispatch:

permissions:
  contents: read

engine: claude

timeout-minutes: 20
max-ai-credits: 250

tools:
  cache-memory:
    key: arxiv-paper-dedup
    retention-days: 90
    allowed-extensions: [".json"]
  repo-memory:
    branch-name: memory/arxiv-paper-ledger
    allowed-extensions: [".json", ".md"]
    format-json: true
  bash:
    - "cat *"
    - "ls *"

network:
  allowed:
    - defaults
    - export.arxiv.org

safe-outputs:
  create-discussion:
    category: "research"
    expires: 14d
    max: 1

steps:
  - name: Fetch and parse arXiv papers
    run: |
      set -e
      mkdir -p /tmp/gh-aw/agent/arxiv
      ARXIV_URL='https://export.arxiv.org/api/query?search_query=(cat:cs.AI+OR+cat:cs.SE+OR+cat:cs.LG)+AND+(agentic+OR+%22multi-agent%22+OR+%22llm+agent%22+OR+%22workflow+automation%22+OR+%22code+generation%22+OR+%22ai+agent%22)&max_results=40&sortBy=submittedDate&sortOrder=descending'
      curl -sf --max-time 30 "$ARXIV_URL" -o /tmp/gh-aw/agent/arxiv/raw.xml \
        || echo '{}' > /tmp/gh-aw/agent/arxiv/raw.xml
      python3 .github/scripts/arxiv-fetch-and-filter.py

---

# arXiv Paper Researcher: GitHub Agentic Workflows

You are a research agent scanning the latest arXiv papers for actionable improvements to GitHub Agentic Workflows (gh-aw) — a system that compiles markdown workflows into GitHub Actions with pluggable AI engines (Claude, Copilot, Gemini, Codex).

## Context

- **Repository**: ${{ github.repository }}
- **Run Date**: $(date +%Y-%m-%d)
- **gh-aw features**: workflow compiler (Go), safe-outputs (typed write operations), network firewall (AWF), token optimization, sub-agents, cache-memory, repo-memory, agentic engine integration, inline prompts, shared imports

## Step 1: Load Pre-Fetched Data

Read `/tmp/gh-aw/agent/arxiv/new-papers.json`.

If `new_count` is 0, call `noop` with message:
"No new arXiv papers today — all N previously processed."
Then stop immediately.

## Step 2: Screen Papers for Relevance

For each paper in `papers`, invoke the `paper-screener` sub-agent with input:
```json
{"title": "...", "abstract": "..."}
```

Collect only papers where the screener returns `{"relevant": true, ...}`.

If no papers are relevant, proceed to Step 4 to update the ledger, then call `noop`:
"N papers screened, none relevant to gh-aw today."
Stop after the ledger update.

## Step 3: Extract Improvement Opportunities

For each relevant paper (max 8), invoke the `opportunity-extractor` sub-agent with the full paper object.

Collect the returned opportunity objects.

## Step 4: Update the Paper Ledger

Load `/tmp/gh-aw/repo-memory/default/paper-ledger.md` if it exists; otherwise start with:
```markdown
# arXiv Paper Ledger

Papers investigated for GitHub Agentic Workflows improvement opportunities.
```

Load `/tmp/gh-aw/repo-memory/default/paper-index.json` if it exists; otherwise use `{"papers": []}`.

For every paper processed in Step 2 (all new papers regardless of relevance), append to the ledger:

```markdown
### [TITLE](URL)
- **ID**: arxiv_id
- **Published**: YYYY-MM-DD
- **Categories**: cat1, cat2
- **Relevant**: Yes / No
- **Opportunity**: (if relevant) concise opportunity text; omit line if not relevant
- **Area**: (if relevant) area; omit line if not relevant
```

Append to the index JSON array:
`{"id": "...", "title": "...", "published": "...", "relevant": true/false, "analyzed_at": "YYYY-MM-DD-HH-MM-SS"}`

Write the updated ledger to `/tmp/gh-aw/repo-memory/default/paper-ledger.md`.
Write the updated index to `/tmp/gh-aw/repo-memory/default/paper-index.json`.

Update the dedup cache at `/tmp/gh-aw/cache-memory/seen-paper-ids.json`:
Load existing `{"ids": [...]}` or start with `{"ids": []}`.
Append all paper IDs from `new-papers.json` (whether relevant or not).
Write back — no colons in filenames.

## Step 5: Create Discussion or Noop

**If actionable opportunities were found**: create a discussion titled:
`[arXiv Research] Agentic Workflow Improvements — YYYY-MM-DD`

Use `###` or lower for all headers inside the discussion body. Never use `#` or `##`.

Discussion body structure:

```
### Summary

N papers screened, M relevant, K opportunities identified.

---

### Actionable Opportunities

(one section per opportunity, grouped by area when there are multiple in the same area)

#### [AREA] — Short Opportunity Title

**Paper**: [Title](URL)
**Authors**: Author A, Author B
**Published**: YYYY-MM-DD
**Effort**: low / medium / high
**Rationale**: 2-3 sentences mapping the paper's mechanism to a specific gh-aw component.

---

### Papers Analyzed

| Paper | Published | Relevant | Area |
|---|---|---|---|
| [Title](URL) | YYYY-MM-DD | Yes / No | area or — |

---

### Next Steps

- [ ] Investigate: opportunity 1 (effort: low)
- [ ] Investigate: opportunity 2 (effort: medium)
```

**If no actionable opportunities were found** (but papers were processed and ledger updated):
call `noop` with message: "Processed N papers (M relevant), no actionable gh-aw improvements identified today."

---

## agent: `paper-screener`
---
description: Fast relevance screening of arXiv paper abstracts for GitHub Agentic Workflows
model: small
---

Screen an arXiv paper abstract for relevance to GitHub Agentic Workflows (gh-aw).

gh-aw compiles markdown workflow files into GitHub Actions YAML, uses AI agents as workflow engines, manages tool access, enforces network firewalls, routes writes through typed safe-outputs, optimizes tokens, and supports multi-agent orchestration.

**Relevant** (any of):
- Agentic AI systems, multi-agent coordination, LLM agent workflows
- Prompt engineering, context management, or caching for LLM agents
- Token efficiency, structured output generation from LLMs
- AI-driven CI/CD, automated code review, workflow automation
- Security in agentic systems: sandboxing, tool access control, policy enforcement
- Orchestration patterns for decomposing tasks across AI agents

**Not relevant**:
- Pure mathematical theory with no LLM/agent application
- Hardware, systems, or networking research unrelated to AI
- Computer vision, speech, or domain-specific scientific tasks
- Medical, biological, or physical science applications

Input: `{"title": "...", "abstract": "..."}` as a JSON string.

Output: exactly one line of valid JSON — no other text:
`{"relevant": true, "reason": "one sentence"}` or `{"relevant": false, "reason": "one sentence"}`

## agent: `opportunity-extractor`
---
description: Extracts a specific actionable improvement for gh-aw from a relevant arXiv paper
model: large
---

Extract one specific actionable improvement for GitHub Agentic Workflows (gh-aw) from an arXiv paper.

gh-aw components (Go compiler + TypeScript runtime):
- **Workflow compiler**: parses markdown frontmatter + body into GitHub Actions YAML
- **Safe-outputs**: typed write operations (create-issue, create-discussion, add-comment, create-pull-request, upload-artifact)
- **Network firewall (AWF)**: blocks outbound domains not in `network.allowed`
- **Token optimization**: DataOps pre-steps, gh-proxy, cache-memory, sub-agent fan-out, prompt caching
- **Sub-agents**: inline agents invoked by the orchestrator, with model aliases (small/large)
- **Memory**: cache-memory (ephemeral, up to 90 days), repo-memory (Git branch, indefinite), comment-memory (issue/PR comment)
- **Engines**: Claude, Copilot, Gemini, Codex — pluggable via `engine:` field
- **Imports**: shared workflow components, skill files, MCP server configs

Input: full paper JSON object.

Identify the single most actionable improvement the paper suggests for gh-aw — a new feature, optimization, design pattern, or new workflow type.

Output: exactly one line of valid JSON — no other text:
`{"opportunity": "concise one-sentence action", "area": "token-optimization|safe-outputs|workflow-compilation|multi-agent|prompt-engineering|network|security|other", "effort": "low|medium|high", "rationale": "2-3 sentences naming the paper mechanism and the specific gh-aw component it improves"}`
