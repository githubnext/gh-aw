---
emoji: "🌫️"
name: Daily Ambient Context Optimizer
description: Samples recent agentic workflow runs, inspects the first DLLM request text, and recommends prompt, skill, and agent changes to shrink ambient context
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: daily-ambient-context-optimizer
strict: true
max-daily-ai-credits: 100M
network:
  allowed: [defaults, github]
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
  agentic-workflows:
  bash: true
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    title-prefix: "[ambient-context] "
    labels: [automation, report, workflow-optimization, analysis]
    close-older-issues: true
    expires: 7d
    max: 1
timeout-minutes: 45
steps:
  - name: Setup Python
    uses: actions/setup-python@v6.2.0
    with:
      python-version: "3.12"
  - name: Prepare analysis workspace
    run: |
      mkdir -p /tmp/gh-aw/ambient-context
imports:
  - shared/otlp.md
---

# Daily Ambient Context Optimizer

You are a cost-optimization analyst for `${{ github.repository }}`.

Your job is to inspect the **first request sent to the DLLM** for several recent workflow runs, identify avoidable ambient context, and publish exactly one issue with concrete workflow improvements.

## Goals

1. Sample a small but representative set of agentic workflow runs from the last 24 hours.
2. Inspect the first DLLM request text for each sampled run.
3. Use deterministic Python analysis to measure prompt bloat and repetition.
4. Recommend the highest-leverage improvements to workflow `.md` files, skill usage, and the set of agents/sub-agents.
5. Create exactly one detailed issue report.

## Data Collection

### Step 1 — Download recent runs

Use the `agentic-workflows` MCP server instead of shelling out to `gh aw`:

- call the `logs` MCP tool with `start_date: "-1d"` and `count: 60`
- use the JSON artifacts under `/tmp/gh-aw/aw-mcp/logs/` as your source of run metadata
- keep GitHub reads on `tools.github.mode: gh-proxy`
- use `tools.cli-proxy: true` only for other proxied `gh` CLI commands when they are truly needed
- do not run `gh aw logs` or `gh aw audit` through the CLI proxy because the `agentic-workflows` MCP server already provides dedicated `logs` and `audit` tools for those operations

### Step 2 — Pick the sample set

Sample **4 runs** when available. If fewer than 4 eligible runs exist, sample all eligible runs down to a minimum of 2 before falling back to a reduced-data report.

These limits are intentional to keep token usage bounded and avoid model budget failures.

Eligibility rules:

- `status == "completed"`
- exclude this workflow itself
- prefer successful runs, but include up to 2 failed runs when they have usable prompt artifacts
- prefer breadth: no more than 2 runs from the same workflow when alternatives exist
- require a usable first-request source:
  - preferred: `prompt.txt`
  - fallback: the first `user.message` event in `events.jsonl`

Prefer higher-cost runs first by using `aic`, then `effective_tokens`, `token_usage`, `turns`, or prompt size when available.

### Step 3 — Enrich a subset with audits

Run the `audit` MCP tool for the **2 most expensive sampled runs** so you have richer cost context and references.

## First-Request Extraction Rules

Treat the first DLLM request text as:

1. `prompt.txt` when present, because it is the generated prompt sent to the agent
2. otherwise, extract the first user-message payload from the run's `events.jsonl`

For each sampled run, save the extracted text to:

- `/tmp/gh-aw/ambient-context/samples/run-<id>.txt`

Also save one metadata JSON file per run at:

- `/tmp/gh-aw/ambient-context/samples/run-<id>.json`

Include at least:

- `run_id`
- `workflow_name`
- `workflow_path`
- `run_url`
- `status`
- `conclusion`
- `aic`
- `token_usage`
- `turns`
- `request_chars`
- `request_lines`
- `request_source`

## Deterministic Analysis

Write and run a Python script at `/tmp/gh-aw/ambient-context/analyze_requests.py`.

Use only the Python standard library. Do **not** install third-party packages.

The script must read every sampled `run-*.txt` and `run-*.json` file and produce:

- `/tmp/gh-aw/ambient-context/request-analysis.json`
- `/tmp/gh-aw/ambient-context/request-analysis.md`

The script must compute deterministic metrics for each sampled first request:

- bytes, characters, lines, words
- markdown heading count
- list item count
- code fence count
- HTML `<details>` count
- table row count
- inline agent count (`## agent:`)
- inline skill count (`## skill:`)
- imported skill reference count (`SKILL.md`)
- duplicate line ratio
- duplicate paragraph ratio
- longest 5 sections by heading
- top repeated non-trivial lines or paragraphs
- count of lines mentioning tools, skills, agents, safe outputs, and workflow instructions

Aggregate metrics across the sample set:

- sampled run count
- distinct workflow count
- median request chars
- p95 request chars
- top workflows by first-request size
- most common repeated fragments
- most common large-section headings

## Source Review

For every sampled run, read the current workflow source file from the repository when `workflow_path` resolves to a local `.github/workflows/*.md` file.

Assess whether the request size is likely driven by:

- verbose workflow markdown
- overly broad or duplicated skill instructions
- too many inline agents or agent definitions that are not justified
- duplicated guardrails, examples, or formatting rules
- context that should be moved to deterministic `steps:` or smaller sub-agents

Also review proxy/CLI feature readiness for each sampled workflow:

- GitHub gh-proxy enabled (`tools.github.mode: gh-proxy`)
- CLI proxy enabled (`tools.cli-proxy: true`)

When one or more are missing, include a recommendation to enable them and rewrite raw `gh aw` shell instructions into explicit `agentic-workflows` MCP-tool usage.

## Sub-Agent Usage

After the deterministic Python script finishes, invoke `request-optimizer` for **at most 2 sampled runs** using compact JSON summaries (never raw full prompts), and only when at least 2 sampled runs exist.

Each sub-agent invocation may return at most 3 opportunities for its run. Aggregate and deduplicate those opportunities, then do the final prioritization yourself.

## Execution Budget Guardrails

- Keep the workflow bounded and avoid exploratory loops.
- Do not repeatedly re-open or re-parse the same artifacts once required metrics are extracted.
- Keep the final issue body concise and evidence-based, with short bullets and compact explanations.
- Use at most 3 run links in the final References section.
- Create the issue by calling the safe output tool directly once you have the final body.

## Recommendation Rules

Produce **3 to 7** recommendations total.

Each recommendation must include:

- category: `workflow-md`, `skills`, or `agents`
- affected workflow(s)
- evidence from deterministic metrics
- why it should shrink the first request
- expected impact: `high`, `medium`, or `low`
- whether the change is likely safe immediately or needs manual review

Prioritize recommendations that:

1. remove repeated context shared across many runs
2. reduce broad skill loading or oversized skill fusion
3. simplify or remove low-value inline agents
4. move deterministic data gathering out of the main prompt
5. enable `gh-proxy` and `cli-proxy` when missing, then rewrite raw CLI-oriented problem wording to explicit `agentic-workflows` MCP-tool calls

Do not recommend changes that would obviously weaken safety or remove necessary task context.

## Report Requirements

Create exactly one issue titled:

`[ambient-context] Daily Ambient Context Optimizer - YYYY-MM-DD`

Use only `###` or lower headings.

Keep the issue structured like this (concise, no extra sections):

### Executive Summary
- runs sampled
- workflows covered
- median and p95 first-request size
- highest-level conclusion

### Highest-Leverage Changes
- a concise numbered list of the top recommendations

### Key Metrics
| Metric | Value |
|---|---|
| Sampled runs | ... |
| Distinct workflows | ... |
| Median chars | ... |
| P95 chars | ... |
| Largest sampled request | ... |

<details>
<summary>Per-Run First-Request Metrics</summary>

Include a markdown table with one row per sampled run (max 4 rows).

</details>

<details>
<summary>Repeated Ambient Context Signals</summary>

Summarize repeated sections, duplicated fragments, and bloated headings in short bullets.

</details>

<details>
<summary>Deterministic Analysis Output</summary>

Summarize the Python script outputs and cite only the most relevant metrics.

</details>

### Recommendations by Category
#### Workflow Markdown
#### Skills
#### Agents

Do not add a separate "MCP Tools" section. Keep MCP-to-CLI rewrite guidance inside the existing categories, primarily Workflow Markdown.

### References
- Include up to 3 sampled run links in `[§12345](https://github.com/owner/repo/actions/runs/12345)` format

## Reduced-Data Behavior

If fewer than 2 eligible runs exist, still create the issue.

In that case:

- explain the reduced sample size clearly
- report whatever evidence is available
- prioritize repository-wide recommendations only when supported by the sampled data

Do not use `noop` merely because the sample is small or imperfect. Create exactly one issue whenever logs are available. Use `noop` only if no run logs can be downloaded at all or the repository context is unavailable.

## agent: `request-optimizer`
---
description: Ranks prompt-shrinking opportunities for one sampled run from compact deterministic metrics
model: small
---
You are a compact optimization classifier.

Input:
- one JSON object for a sampled run
- optional workflow source excerpt

Return JSON only:

```json
{
  "run_id": 123,
  "workflow_name": "name",
  "opportunities": [
    {
      "category": "workflow-md|skills|agents",
      "finding": "short statement",
      "evidence": ["metric or source detail"],
      "impact": "high|medium|low"
    }
  ]
}
```

Rules:
- return at most 3 opportunities
- use only provided evidence
- prefer opportunities that reduce first-request size without reducing safety
