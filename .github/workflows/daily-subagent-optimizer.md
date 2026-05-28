---
emoji: "⚡"
name: Daily Sub-Agent Optimizer
description: Identifies high-token workflows lacking inline sub-agents, applies LLM-expert heuristics to locate decomposable tasks, and creates a concrete inline-agent refactoring proposal
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

tracker-id: daily-subagent-optimizer
engine: claude
strict: true

network:
  allowed:
    - defaults
    - github

tools:
  cache-memory: true
  bash:
    - "*"

safe-outputs:
  create-issue:
    title-prefix: "[subagent-optimizer] "
    labels: [automation, optimization, prompt-quality]
    close-older-issues: true
    expires: 7d
    max: 1
  noop:

experiments:
  timeout_setting:
    variants: [tight, default, relaxed]
    description: "Tests whether 20 / 30 / 45 minute timeout affects run success rate and issue-filing rate for this dense multi-phase analysis workflow"
    hypothesis: "H0: no change in run success rate. H1: tight variant increases failure rate ≥15pp; relaxed variant improves issue-filing rate ≥15pp vs default"
    metric: run_success_rate
    secondary_metrics: [issue_filed_rate, run_duration_p95]
    guardrail_metrics:
      - name: empty_output_rate
        threshold: "<0.10"
    min_samples: 30
    weight: [33, 34, 33]
    start_date: "2026-05-28"
    tags: [cost-efficiency, reliability, timeout]
    analysis_type: proportion_test

timeout-minutes: ${{ (needs.activation.outputs.timeout_setting == 'tight' && 20) || (needs.activation.outputs.timeout_setting == 'relaxed' && 45) || 30 }}

imports:
  - uses: shared/meta-analysis-base.md
    with:
      toolsets: [default, repos, actions]
  - shared/reporting.md

  - shared/otlp.md
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Sub-Agent Optimizer

You are an LLM efficiency expert specializing in agentic workflow optimization. Your mission today: identify one workflow that would benefit from inline sub-agent refactoring, reason carefully about which tasks a smaller model (small alias) can handle, and produce a copy-paste-ready proposal issue.

## Context

- Repository: `${{ github.repository }}`
- Run: `${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}`

## Phase 1 — Gather Recent Workflow Run Data

Use the `agentic-workflows` MCP server `logs` tool to fetch recent workflow runs:
- Count: 100
- Start date: `-7d`

From the results, build a ranked list of the top 15 workflows by **total token usage** over the 7-day window. Include: workflow name, run count, total tokens, avg tokens/run, avg turns/run.

If the MCP tool is unavailable or returns no data, fall back to listing candidate workflows from `.github/workflows/*.md` sorted by file size (larger files tend to have more complex prompts and higher token usage).

## Phase 2 — Screen Candidates

Load the cached optimization log from `/tmp/gh-aw/cache-memory/subagent-optimizer/optimization-log.json` if it exists.

From the ranked list, filter out:
- Workflows optimized in the last 14 days (per the log above)
- Smoke-test workflows (filename starts with `smoke-`)
- This optimizer itself (`daily-subagent-optimizer`)
- Workflows already using inline sub-agents

Use the `workflow-screener` sub-agent to check the top 6 remaining candidates. For each, pass the file path and ask the agent to read the file and report all five fields: `has_agent_blocks`, `engine`, `is_smoke_test`, `prompt_phases`, and `notes`.

Select the **highest-token workflow that passes all filters and has at least 3 distinct phases/sections** in its prompt. This is your optimization target.

If no suitable candidate is found, call `noop` with a brief explanation.

## Phase 3 — Read the Target Workflow

Read the full target workflow source:

```bash
cat .github/workflows/<target>.md
```

Extract and note:
- **Frontmatter fields**: engine, tools, features, timeout, imports, safe-outputs
- **Prompt body**: all natural-language content after the closing `---`
- **Token stats**: total tokens last 7 days, avg tokens/run, avg turns/run (from Phase 1 data)

## Phase 4 — Common Tool Prefix Analysis

This is the **highest-value optimization**. Before considering sub-agents, check whether multiple phases in the workflow repeat the same tool invocations at their start. Extracting these into a single shared setup step eliminates redundant context-gathering and reduces per-run token cost more reliably than sub-agent extraction.

Use the `prefix-analyzer` sub-agent to perform this analysis. Pass it the full prompt body text and ask it to:
- Split the body into its major sections (headings starting with `##` or `###`)
- For each section, extract every explicit tool invocation in the first 10 lines: bash commands (code blocks or inline `backtick` calls), MCP tool calls (`Use the ... tool`), file reads (`cat`, `gh api`, `gh repo view`), and any repeated setup phrases
- Find tool calls that appear verbatim (or near-verbatim) as opening instructions in **two or more** sections
- Report the common prefix set and how many sections share it

### Scoring Common Prefixes

| Finding | Score |
|---|---|
| ≥ 3 sections share ≥ 2 common opening tool calls | High (best optimization) |
| 2 sections share ≥ 3 common opening tool calls | High |
| 2 sections share 2 common opening tool calls | Moderate |
| ≤ 1 common call, or only 1 section affected | Low — skip |

**If a High or Moderate prefix is found**, record:
- The shared tool calls (exact text)
- Which sections share them
- The proposed "Setup" step text that would run them once
- Estimated token savings (be conservative: 5–15% per duplicated call removed)

**If no qualifying prefix is found**, note this and proceed to Phase 5.

## Phase 5 — LLM Expert Analysis

As an LLM efficiency expert, identify where the workflow's prompt does work that a smaller model can handle independently.

### Sub-Agent Candidate Scoring

For each major section or phase in the prompt body, use the `opportunity-classifier` sub-agent to score it. Pass the section text and ask for a score on:

| Dimension | Meaning | Max |
|---|---|---|
| **Independence** | Can this run without outputs from other sections? | 3 |
| **Haiku-adequacy** | Simple enough for a smaller model (extractive/classificatory)? | 3 |
| **Parallelism** | Could this run concurrently with other sections? | 2 |
| **Size** | Substantial enough to warrant a separate agent call? | 2 |

Threshold: **≥ 6 → strong candidate, 4–5 → moderate, < 4 → keep in main agent.**

### Heuristics Cheatsheet

Tasks a **`small` model handles well**:
- Summarizing a single file or code section
- Extracting specific fields from structured/semi-structured text
- Classifying items into a predefined set of categories
- Checking whether something meets a stated criterion (yes/no + reason)
- Converting data from one format to another (JSON → markdown table, etc.)
- Listing occurrences of a pattern in text
- Validating that a config block follows expected syntax
- Writing a first-draft fragment from a template

Tasks that **must stay with the main model**:
- Cross-referencing multiple heterogeneous sources to form a conclusion
- Synthesizing findings into a holistic recommendation
- Nuanced judgment requiring the full workflow context
- Writing the final issue/report body (authoritative output)
- Strategic decisions that affect the rest of the workflow

### Selection

Collect all sections scoring ≥ 4. Pick the **top 2–4** by score to propose as sub-agents. Discard candidates whose combined scope covers less than 10% of the prompt body — the savings would be negligible.

## Phase 6 — Design the Refactoring

For each selected sub-agent candidate, design a concrete inline sub-agent:

1. **Name**: lowercase, hyphenated, descriptive (e.g., `file-summarizer`, `category-detector`)
2. **Model**: `small`
3. **Description**: one sentence (≤ 15 words)
4. **Agent prompt**: focused, ≤ 15 lines, imperative mood
5. **Invocation change**: the 1–3 line replacement in the main prompt that calls the sub-agent by name

Also determine:
- Estimated token reduction per run (be conservative: 10–25% per sub-agent extracted)

## Phase 7 — Create the Proposal Issue

Create one GitHub issue with this structure:

Title: `[subagent-optimizer] Optimize <workflow-name> — YYYY-MM-DD`

Body:

```markdown
### Target Workflow

**File**: `.github/workflows/<name>.md`
**Engine**: <engine>
**7-day token usage**: ~N tokens across M runs (~N avg/run)

### Why This Workflow

[2–3 sentences: what makes it a good candidate — high token usage, number of distinct phases,
specific tasks identified as small-appropriate or having repeated tool prefixes]

---

## Optimization 1 — Common Tool Prefix (Highest Priority)

[Include this section only if Phase 4 found a High or Moderate prefix]

### Repeated Tool Calls Found

The following tool invocations appear at the start of **N** sections:

```
[exact text of each shared tool call, one per line]
```

**Sections affected**: [list section names/headings]

### Proposed Setup Step

Add a `## Setup` section at the top of the prompt body that runs these calls once:

```markdown
## Setup

[proposed setup step text]
```

Then remove the duplicate calls from each of the affected sections.

**Estimated savings**: ~N tokens/run (~X% reduction)

---

## Optimization 2 — Inline Sub-Agents

[Include this section only if Phase 5 found sub-agent candidates scoring ≥ 4]

### LLM Expert Reasoning

[3–5 bullet points — which heuristics fired, which scoring dimensions were highest, why smaller models suffice for the proposed sections]

### Proposed Sub-Agents

#### 1. `<agent-name>` (`small`)

**Extracted task**: [1 sentence]
**Why small**: [1 sentence — which heuristic applies]
**Score**: <X>/10 (independence: N, model-adequacy: N, parallelism: N, size: N)
**Estimated savings**: ~N tokens/run

<details>
<summary>Agent definition (copy-paste ready)</summary>

```markdown
## agent: `<agent-name>`
---
description: <description>
model: small
---
<agent prompt>
```

</details>

**Invocation change in main prompt:**

Before:
```
[Original verbose section — first 3–5 lines]
```

After:
```
Use the `<agent-name>` agent to [task].
```

[Repeat for each proposed sub-agent]

### Estimated Impact

| Metric | Before | After (estimated) |
|---|---|---|
| Avg tokens/run | ~N | ~M (~X% reduction) |
| Main-model context saved | — | ~Y tokens/run |
| Parallelism opportunity | None | [N] sections in parallel |

### Implementation Steps

1. **Common prefix** (if applicable): Add `## Setup` section at the top of the prompt body and remove duplicated tool calls from affected sections
2. **Sub-agents** (if applicable): Add each sub-agent block at the bottom of `.github/workflows/<name>.md`, after all workflow content
3. Update the prompt sections listed above to invoke sub-agents by name
4. Compile: `gh aw compile <name>`
5. Test: `gh workflow run <name>.yml`

### References

- Optimizer run: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
```

## Phase 8 — Update Optimization Log

Create the directory if needed and append one entry to `/tmp/gh-aw/cache-memory/subagent-optimizer/optimization-log.json`:

```bash
mkdir -p /tmp/gh-aw/cache-memory/subagent-optimizer
```

```json
{"date":"YYYY-MM-DD","workflow_name":"...","total_tokens_7d":N,"avg_tokens_per_run":N,"prefix_optimization":true,"sub_agents_proposed":N,"estimated_savings_pct":N}
```

Load the existing array from that path if the file is present, append the new entry, keep only the last 30 entries, and save.

## Guardrails

- If no suitable target exists, call `noop` explaining what was checked and why nothing qualified
- Never propose sub-agents for a workflow that already has existing `## agent:` blocks — it already uses inline sub-agents
- Always check for common tool prefixes (Phase 4) before scoring sub-agent candidates — the prefix optimization is cheaper to implement and yields faster returns
- Keep every proposed agent prompt ≤ 15 lines — if it needs more, it belongs in the main model
- Base savings estimates on the Phase 1 token data; if unavailable, omit numerical estimates
- Maximum 4 sub-agents per proposal — larger diffs are harder to review
- Omit the Sub-Agents section from the issue entirely if Phase 5 finds no candidates scoring ≥ 4 — do not pad the issue with weak candidates

{{#runtime-import shared/noop-reminder.md}}

## agent: `prefix-analyzer`
---
description: Detects repeated tool-call prefixes across workflow prompt sections and scores extraction value
model: small
---
You are a prompt-structure analyst. Given the full body text of an agentic workflow prompt (everything after the closing frontmatter `---`), identify repeated tool invocations that appear as opening instructions in multiple sections.

**Steps:**

1. Split the prompt into major sections using `##` and `###` headings. List each section name.
2. For each section, extract the first 10 lines and identify every explicit tool invocation:
   - Bash code blocks or inline `` `backtick` `` shell commands
   - Phrases like "Use the `<name>` tool", "Run `<command>`", "Call `<tool>`"
   - File-read operations: `cat`, `gh api`, `gh repo view`, `head`, `tail`
   - MCP tool calls starting with "Use the ... MCP server"
3. Find tool calls that appear (verbatim or near-verbatim) as opening instructions in **two or more** sections.
4. For each group of matching calls, report:
   - The exact shared text
   - Which sections contain it (by heading name)
   - Whether it appears at the very start of the section (first 5 lines) or later

Score the finding:
- `high`: ≥ 3 sections share ≥ 2 common opening tool calls, OR 2 sections share ≥ 3 common calls
- `moderate`: exactly 2 sections share exactly 2 common opening calls
- `low`: fewer than 2 sections share any common calls — no actionable prefix found

Return in this exact format:
```
score: high/moderate/low
shared_calls:
  - "<exact tool call text>"
  - "<exact tool call text>"
affected_sections:
  - "<section heading 1>"
  - "<section heading 2>"
setup_step_proposal: |
  [proposed ## Setup section text that would replace the repeated calls, or "none" if score is low]
reasoning: <1–2 sentences explaining the finding>
```

## agent: `workflow-screener`
---
description: Reads a workflow .md file and reports sub-agent blocks, engine, and prompt complexity
model: small
---
You are a workflow file scanner. When given a file path, read the file using bash and report the following facts:

1. **has_agent_blocks**: Does the file body contain any `## agent:` section? (yes/no)
2. **engine**: The value of the `engine:` field (e.g., `claude`, `copilot`, `codex`). If `engine:` is an object, report `id:` value.
3. **is_smoke_test**: Is this a smoke-test workflow? (yes if filename starts with `smoke-` or file body is fewer than 40 lines)
4. **prompt_phases**: Count the number of major sections (lines starting with `## ` or `### `) in the prompt body (everything after the closing `---`). Report as a number.
5. **notes**: One sentence about anything notable (e.g., "already has inline sub-agent blocks", "very short prompt", "no distinct phases").

Return your findings in this exact format:
```
has_agent_blocks: yes/no
engine: <value>
is_smoke_test: yes/no
prompt_phases: <number>
notes: <one sentence>
```

## agent: `opportunity-classifier`
---
description: Scores a workflow prompt section on its suitability for extraction into a small sub-agent
model: small
---
You are an LLM task-decomposition expert. Given a section of an agentic workflow prompt, score it on its suitability to be extracted into a sub-agent using a smaller model.

Score each dimension:

- **independence** (0–3): Can this section run without the outputs of other sections? 3 = fully independent, 0 = deeply coupled to earlier results
- **model_adequacy** (0–3): Is the reasoning simple enough for a smaller model? 3 = pure extraction/classification/formatting, 0 = requires deep synthesis or cross-referencing many sources
- **parallelism** (0–2): Could this run concurrently with other sections? 2 = yes, 0 = must be sequential
- **size** (0–2): Is the task substantial enough to warrant a separate agent call? 2 = many tool calls or long output, 0 = trivial (< 2 tool calls)

Compute: `total = independence + model_adequacy + parallelism + size` (max 10)

Verdict: `strong` (≥ 6), `moderate` (4–5), `weak` (< 4)

Return in this exact format:
```
total: <score>/10
independence: <0-3>
model_adequacy: <0-3>
parallelism: <0-2>
size: <0-2>
verdict: strong/moderate/weak
task_type: summarizer/classifier/extractor/validator/formatter/other
reasoning: <1–2 sentences explaining the verdict and why a smaller model suffices or not>
```
