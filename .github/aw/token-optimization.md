---
description: Guide for reducing token consumption in agentic workflows — DataOps, gh-proxy, inline sub-agents, caveman experiments, and audit-based measurement.
---

# Token Consumption Optimization

Tokens are the primary cost driver for agentic workflows. Apply the techniques below to reduce effective token consumption while preserving output quality.

## Quick-Reference Checklist

Apply these in order — each check can halve costs:

- [ ] **DataOps**: Move data fetching into `steps:` — agent reads compact JSON, not raw API responses
- [ ] **gh-proxy**: Set `tools.github.mode: gh-proxy` — skips Docker MCP server startup and extra tool definitions
- [ ] **cli-proxy**: Mount additional MCP servers as CLIs via `cli-proxy: true` — agent pipes output through `jq` before it enters context
- [ ] **Sub-agents**: Delegate repetitive per-item tasks to `model: small` sub-agents (~10–20× cheaper)
- [ ] **Prompt size**: Strip redundant instructions, examples, and pleasantries from the prompt body
- [ ] **Dynamic context**: Inject only required fields — `${{ github.event.issue.number }}` not the full event payload
- [ ] **Prompt caching**: Put stable instructions before dynamic content to maximize cache hits
- [ ] **Cadence**: If the result is not time-sensitive, schedule less often (`hourly` → `daily`, `daily` → `weekly`)
- [ ] **Batching**: Prefer scheduled batch processing over reactive events when delayed processing is acceptable
- [ ] **Telemetry**: Configure `observability.otlp` so token usage and run phases are measurable outside individual run logs
- [ ] **AgenticOps**: Add `copilot-token-audit` / `copilot-token-optimizer` workflows so the repository keeps finding waste automatically
- [ ] **Measure first**: Back every change with an `experiments:` field and `metric: "effective_tokens"` before promoting

---

## How to Measure Token Usage

Establish a baseline before optimizing. The audit command is your main instrument.

### Single-run audit

```bash
gh aw audit <run-id> --json
```

Key fields in the output:

- `agent_usage.effective_tokens` — the normalized cost metric (accounts for model price differences and cache discounts)
- `agent_usage.input_tokens` / `agent_usage.output_tokens` — raw token counts
- `agent_usage.cache_read_tokens` / `agent_usage.cache_write_tokens` — tokens served from the prompt cache

Or via MCP tool:

```
Use the audit tool with run_id: <run-id>
```

### Comparing two runs (regression detection)

Run the baseline and the optimized variant, then diff them:

```bash
gh aw audit <base-run-id> <optimized-run-id> --json
# Or compare multiple variants at once:
gh aw audit <base-run-id> <variant-a-run-id> <variant-b-run-id> --json
```

Or via MCP tool:

```
Use the audit tool with run_ids_or_urls: ["<base-run-id>", "<optimized-run-id>"]
```

The diff output highlights changes in effective tokens, tool calls, and safe outputs between runs — making it easy to confirm that an optimization actually reduced cost without degrading behavior.

### Per-request token detail

`gh aw audit <run-id>` downloads all artifacts into `logs/run-<run-id>/`. Read the per-call breakdown from there:

```bash
gh aw audit <run-id>
cat logs/run-<run-id>/firewall-audit-logs/api-proxy-logs/token-usage.jsonl
```

Each line is one API call with `model`, `input_tokens`, `output_tokens`, `cache_read_tokens`, and `cache_write_tokens`. Use this to find which API calls are the most expensive.

---

## Technique 1 — DataOps: Move Compute to Steps

**The single biggest optimization.** Replace agentic data fetching with deterministic shell commands in `steps:`. Shell steps run outside the AI sandbox (no tokens), produce structured output the agent reads directly, and are fast and reproducible.

### Before (agent does all the work)

```markdown
---
engine: copilot
tools:
  github:
    mode: gh-proxy
    toolsets: [default, pull_requests]
---

Fetch all open PRs in ${{ github.repository }}, compute the merge rate, identify authors with the most contributions, and create a weekly summary discussion.
```

The agent calls GitHub APIs iteratively, consuming tokens for each call and for processing raw API responses.

### After (DataOps pattern)

```markdown
---
engine: copilot
tools:
  github:
    mode: gh-proxy
  bash: ["*"]

steps:
  - name: Fetch and aggregate PR data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/gh-aw/data
      gh pr list --repo "${{ github.repository }}" \
        --state all --limit 100 \
        --json number,title,state,author,createdAt,mergedAt,additions,deletions \
        > /tmp/gh-aw/data/prs.json

      jq '{
        total: length,
        merged: [.[] | select(.state=="MERGED")] | length,
        open: [.[] | select(.state=="OPEN")] | length,
        top_authors: ([.[].author.login] | group_by(.) | map({author:.[0], count:length}) | sort_by(-.count) | .[0:5])
      }' /tmp/gh-aw/data/prs.json > /tmp/gh-aw/data/stats.json

safe-outputs:
  create-discussion:
    title-prefix: "[weekly-pr] "
    category: "General"
    close-older-discussions: true
---

Read the pre-computed stats at `/tmp/gh-aw/data/stats.json` and `/tmp/gh-aw/data/prs.json`.
Create a concise weekly PR summary discussion.
```

**Why this saves tokens:**

- API calls happen in shell steps — zero AI tokens spent on fetching
- The agent receives compact, aggregated JSON instead of raw API responses
- The agent's context window stays small, reducing per-turn input token counts

**Best practices:**

- Write one JSON file per data source; use `jq` to pre-aggregate
- Store files under `/tmp/gh-aw/` — this directory is available to the agent
- Document file locations and schema in the prompt body so the agent doesn't need to explore

See also: [DataOps pattern docs](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/patterns/data-ops.md)

---

## Technique 2 — Use `gh-proxy` and `cli-proxy` Instead of the MCP Server

**Eliminates Docker startup overhead and reduces per-call context overhead.**

### `mode: gh-proxy` (GitHub reads)

```yaml
tools:
  github:
    mode: gh-proxy      # ✅ preferred — pre-authenticated gh CLI, no MCP server startup
    toolsets: [default]
```

`gh-proxy` makes a pre-authenticated `gh` CLI available in bash. The agent reads GitHub data with `gh issue list`, `gh pr view`, etc. — no Docker container, no MCP server initialization, and tighter bash output that the agent can pipe through `jq` before reading.

The alternative (`mode: local`) starts a Docker-based GitHub MCP Server, which:
- Adds startup latency
- Registers extra tool descriptions that expand the system prompt
- Returns verbose JSON that the agent must process in full

### `cli-proxy: true` (other MCP servers as CLIs)

When a workflow uses additional MCP servers (e.g., a custom Notion or Slack MCP), `cli-proxy: true` mounts each server as a standalone CLI tool on `PATH`:

```yaml
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
  my-custom-mcp:
    ...
```

With `cli-proxy`, the agent can call `my-custom-mcp <tool> <args>` from bash, capturing output as text and processing it before passing results back into the conversation. This lets the agent use `jq` or `grep` to extract only the fields it needs — rather than receiving the full MCP tool response as a structured tool result in the conversation context.

**Summary:**

| Mode | Docker startup | Extra tool definitions | Agent output processing |
|---|---|---|---|
| `mode: local` + MCP tools | Yes | Yes | Tool result (full JSON) |
| `mode: gh-proxy` + bash | No | No | Agent pipes through jq |
| `cli-proxy: true` + bash | Yes (once) | Reduced | Agent pipes through jq |

---

## Technique 3 — Inline Sub-Agents with Smaller Models

**Delegate narrow, repetitive tasks to cheap models; reserve the large model for synthesis.**

Sub-agents defined inside the workflow file with `model: small` cost 10–20× less than the parent model and are well-suited for classification, one-sentence summarization, structured extraction, and scoring.

### Pattern

```
steps:        → deterministic shell (zero AI tokens)
sub-agents:   → small model per item (cheap, parallelizable)
main agent:   → synthesizes compact sub-agent results (one high-quality pass)
```

### Example

```markdown
---
engine: copilot
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
  bash: ["*"]

steps:
  - name: Split issues into per-item files
    run: |
      mkdir -p /tmp/gh-aw/issues
      gh issue list --repo "${{ github.repository }}" \
        --state open --limit 50 \
        --json number,title,body,labels \
        | jq -c '.[]' \
        | while IFS= read -r issue; do
            num=$(echo "$issue" | jq -r '.number')
            echo "$issue" > /tmp/gh-aw/issues/issue-${num}.json
          done
---

## Step 1 — classify each issue

For every `/tmp/gh-aw/issues/issue-*.json`, use the `classifier` agent.
Write output to `/tmp/gh-aw/issues/cat-<number>.json`.

## Step 2 — synthesize

Read all `cat-*.json` files and create a triage report grouped by category.

## agent: `classifier`
---
description: Classifies a GitHub issue into a single category
model: small
---
Read the JSON file provided. Return only:
`{"number": <n>, "category": "bug|feature|question|docs|security|other"}`
Nothing else.
```

**Why this saves tokens:**

- 50 issues × ~200 tokens at `small` pricing = ~10,000 tokens (cheap model)
- Main agent only reads compact `{"number":…, "category":…}` JSON — far fewer input tokens than reading 50 raw issue bodies
- Parallelism: the main agent can dispatch multiple sub-agents concurrently

**Sub-agent model aliases:**

| Alias | Use when |
|---|---|
| `small` | Classification, extraction, one-sentence summaries, scoring |
| `large` | Complex reasoning, multi-step synthesis, code generation |
| `inherited` | Sub-agent needs same capability as the parent (default) |

Always use aliases, not model IDs — aliases resolve to the best available model per provider.

See also: [Inline Sub-Agents](subagents.md)

---

## Technique 4 — Apply the Caveman Technique

**Measure the cost of verbosity with a prompt-style experiment.**

The "caveman" technique uses an A/B experiment to compare a verbose prompt against a stripped-down minimal prompt. If the minimal variant produces equally useful output, adopt it permanently.

```yaml
experiments:
  prompt_style: [verbose, minimal]
```

```markdown
{{#if experiments.prompt_style == "verbose" }}
Please analyze all of the open issues in this repository and provide a comprehensive, detailed report covering: the number of open issues, any significant trends or patterns you observe, the most frequently occurring labels, the oldest unresolved issues, a prioritized list of the most critical items, and any recommendations for the team.
{{#else}}
List open issues by priority. Top 5 critical items. Be brief.
{{/if}}
```

Measure `effective_tokens` in each variant's run summary or via `gh aw audit`. If the `minimal` variant uses fewer tokens at acceptable quality, promote it as the baseline.

**What to minimize:**

- Remove redundant instructions (the model already knows common conventions)
- Replace prose explanations with bullet constraints
- Cut examples that don't constrain behavior
- Remove hedging language and pleasantries

---

## Technique 5 — Use Experiments to Measure Impact

**Never guess — measure every optimization with A/B experiments.**

Declare an experiment before making any prompt or configuration change. Let both variants run for enough cycles to be statistically meaningful (≥ 20 runs per variant for high-frequency workflows).

```yaml
experiments:
  optimization_v1:
    variants: [control, optimized]
    description: "DataOps refactor — move issue fetching to steps:"
    metric: "effective_tokens"
    issue: "123"
```

Reference the active variant in the prompt:

```markdown
{{#if experiments.optimization_v1 == "optimized" }}
Read the pre-fetched data from `/tmp/gh-aw/data/`.
{{#else}}
Fetch open issues from ${{ github.repository }} using the GitHub tools.
{{/if}}
```

**After enough runs:**

1. Compare variants using `gh aw audit <control-run-id> <optimized-run-id>`
2. Check effective token deltas in the diff output
3. If the optimized variant wins, rewrite the baseline prompt and remove the `experiments:` field

**Key experiment dimensions for token optimization:**

| Dimension | Example variants |
|---|---|
| Prompt verbosity | `verbose` / `concise` / `minimal` |
| Data source | `agentic-fetch` / `dataops-steps` |
| Model tier | Run separate workflows for each engine |
| Sub-agent usage | `single-agent` / `with-subagents` |
| Tool mode | `mcp-local` / `gh-proxy` |

See also: [A/B Testing Experiments](experiments.md)

---

## Technique 6 — Reduce Trigger Frequency and Batch Work

**The cheapest run is the one you do not execute.** If a workflow does not need near-real-time feedback, run it less often and process multiple items in one pass.

### Prefer slower schedules when latency is acceptable

Move high-frequency schedules down to the slowest cadence that still meets the operational need:

- `hourly` → `daily on weekdays` for team-facing summaries or audits
- `daily` → `weekly` for trend reports, optimization reviews, and backlog hygiene
- `every N hours` → a daily or weekly batch when the workflow only produces guidance or reports

This reduces total workflow runs, token usage, GitHub Actions minutes, and notification noise all at once.

### Prefer scheduled batches over reactive triggers

Reactive triggers (`issues:`, `pull_request:`, comment commands) are appropriate when maintainers need immediate feedback. Otherwise, prefer `schedule:` and batch work:

```yaml
on:
  schedule: daily on weekdays
```

Typical batch-friendly tasks:

- daily or weekly triage summaries
- stale backlog review
- token usage audits
- repository-wide quality or security digests

Combine batching with `cache-memory` or `repo-memory` to track what was already processed so each scheduled run only handles new items.

---

## Technique 7 — Measure Continuously with OpenTelemetry and AgenticOps

**Don't rely only on ad hoc audits.** Export telemetry automatically, then add workflows that keep looking for token waste over time.

### Enable OTLP export

Add workflow-level OpenTelemetry export so each run emits token and phase data to your observability backend:

```yaml
observability:
  otlp:
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
```

`gh-aw` emits setup, agent, and conclusion spans with token usage attributes, which makes it easier to:

- compare workflows over time
- identify expensive phases before opening logs
- validate that an optimization reduced cost after rollout

See also: [Frontmatter syntax](syntax.md#observability)

### Add AgenticOps token workflows

Use the token-focused workflows from the AgenticOps pattern to optimize continuously at the repository level:

- `copilot-token-audit` — scheduled audit of token usage across workflows
- `copilot-token-optimizer` — scheduled follow-up that identifies one expensive workflow and proposes concrete savings

This turns token optimization into an ongoing loop:

1. export OTEL data
2. collect and summarize repository-wide token usage
3. open optimization issues for the highest-value fixes
4. re-measure after changes land

See the `gh-aw` repository for derived `copilot-token-audit` and `copilot-token-optimizer` examples under `.github/workflows/`.

---

## Technique 8 — Enable Prompt Caching

**Repeated context (system prompt, shared preamble) is charged at ~10× less when cached.**

Prompt caching is automatically enabled by the AWF gateway. Effective cached input tokens are weighted at `0.1` in the effective token formula (versus `1.0` for uncached input).

To maximize cache hits:

- **Keep stable content at the top of the prompt.** Instructions that don't change between runs (role description, output format rules, JSON schema) should appear before dynamic content (issue body, event context).
- **Use `cache-memory`** for workflows that re-read the same large knowledge base across runs. The memory server avoids injecting duplicate context into every turn.
- **Minimize dynamic context.** Inject only the fields the agent actually needs: use `${{ github.event.issue.number }}` instead of dumping the full event payload.

---

## Additional Resources

| Topic | File |
|---|---|
| Inline sub-agents syntax | [subagents.md](subagents.md) |
| A/B experiments | [experiments.md](experiments.md) |
| Persistent memory | [memory.md](memory.md) |
| DataOps pattern | [DataOps guide](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/patterns/data-ops.md) |
| Audit command reference | [cli-commands.md](cli-commands.md) |
| Frontmatter syntax | [syntax.md](syntax.md) |
