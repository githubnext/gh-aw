---
description: Guide for reducing token consumption in agentic workflows — DataOps, gh-proxy, inline sub-agents, caveman experiments, and audit-based measurement.
---

# Token Consumption Optimization

If a task can be solved using deterministic tools, use deterministic tools. Only use agents when necessary, as they incur higher cost. Agentic workflows allow you to run deterministic tools first and gate agent execution using conditions, enabling workflows that avoid triggering agents most of the time and only use them when needed.

## Quick-Reference Checklist

Apply these in order — each check can halve costs:

- [ ] **Cheap triage first**: classify duplicates, stale items, low-value events, and known cases before escalating
- [ ] **Frontier model as planner**: use frontier models for planning, synthesis, ambiguous decisions, and final judgment — not bulk extraction
- [ ] **DataOps**: Move data fetching into `steps:` — agent reads compact JSON, not raw API responses
- [ ] **gh-proxy**: Set `tools.github.mode: gh-proxy` — skips Docker MCP server startup and extra tool definitions
- [ ] **cli-proxy**: Mount additional MCP servers as CLIs via `cli-proxy: true` — agent pipes output through `jq` before it enters context
- [ ] **Sub-agents**: Delegate repetitive per-item tasks to `model: small` sub-agents (~10–20× cheaper)
- [ ] **Sub-skills**: Keep the main prompt as a short execution plan; move detailed playbooks/output layouts into `## skill:` blocks the agent invokes only when needed
- [ ] **Prompt size**: Strip redundant instructions, examples, and pleasantries from the prompt body
- [ ] **Dynamic context**: Inject only required fields — `${{ github.event.issue.number }}` not the full event payload
- [ ] **Pull context on demand**: query logs/data only after a hypothesis forms; avoid preloading large raw dumps into the initial prompt
- [ ] **Prompt caching**: Put stable instructions before dynamic content to maximize cache hits
- [ ] **Context hygiene**: keep the orchestrator context compact; prefer short worker summaries over raw output
- [ ] **Cadence**: If the result is not time-sensitive, schedule less often (`hourly` → `daily`, `daily` → `weekly`)
- [ ] **Batching**: Prefer scheduled batch processing over reactive events when delayed processing is acceptable
- [ ] **Telemetry**: Configure `observability.otlp` so token usage and run phases are measurable outside individual run logs
- [ ] **AgenticOps**: Add `copilot-token-audit` / `copilot-token-optimizer` workflows so the repository keeps finding waste automatically
- [ ] **Measure first**: Back every change with an `experiments:` field and `metric: "aic"` before promoting

---

## Frontier-Model Cost Pattern

A frontier model can reduce **total** cost when architecture prevents unnecessary invocations and keeps expensive context narrow.

- use frontier model for planning, hypothesis selection, synthesis, ambiguous decisions, final judgment
- do not spend frontier turns on repetitive extraction, duplicate detection, or broad first-pass scanning
- add a cheap triage stage for known/duplicate/stale/low-value events; stop with `noop` when escalation is unnecessary
- escalate to frontier model only when triage is uncertain or the case is genuinely new/high-value
- cap sub-agent fan-out so escalations cannot recurse without bound

Cost wins come from architecture and selective execution, not model tier alone.

---

## Pull Context, Do Not Push Context

Avoid front-loading large raw context when data can be fetched on demand. Prefer deterministic pre-steps that materialize compact files under `/tmp/gh-aw/`, `gh` + filtering (`jq`, `grep`) before context reaches the model, pre-aggregated summaries over full API payloads, and directed tool calls issued only after the agent forms a hypothesis. Anchoring warning: preselecting raw logs too early can make the model over-focus and miss the actual cause.

---

## How to Measure Token Usage

`gh aw audit` reports per-run cost. See [cli-commands.md](cli-commands.md#gh-aw-audit) for full command syntax (single run, `--json`, multi-run diff) and the MCP `audit` equivalent.

Token-specific fields in `gh aw audit <run-id> --json`:

- `agent_usage.aic` — AI Credits (AIC), the normalized cost metric (1 AIC = $0.01; accounts for model price differences and cache discounts)
- `agent_usage.input_tokens` / `agent_usage.output_tokens` — raw token counts
- `agent_usage.cache_read_tokens` / `agent_usage.cache_write_tokens` — tokens served from the prompt cache

For per-call detail, `gh aw audit <run-id>` downloads artifacts into `logs/run-<run-id>/`; read `firewall-audit-logs/api-proxy-logs/token-usage.jsonl` (one API call per line, with `model` and token counts) to find the most expensive calls. Diff two runs with `gh aw audit <base-id> <optimized-id>` to detect AI-credit regressions.

Treat optimization as successful only when quality remains acceptable. A quality regression is a failure even if AI Credits decrease.

---

## Technique 1 — DataOps: Move Compute to Steps

The single biggest optimization. Replace agentic data fetching with deterministic shell commands in `steps:`. Shell steps run outside the AI sandbox (no tokens) and produce structured output the agent reads directly.

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

**Best practices:**

- One JSON file per data source; `jq` to pre-aggregate
- Store files under `/tmp/gh-aw/`
- Document file locations and schema in the prompt body so the agent doesn't need to explore

See also: [DataOps pattern docs](https://github.com/github/gh-aw/blob/main/docs/src/content/docs/patterns/data-ops.md)

---

## Technique 2 — Use `gh-proxy` and `cli-proxy` Instead of the MCP Server

### `mode: gh-proxy` (GitHub reads)

```yaml
tools:
  github:
    mode: gh-proxy      # ✅ preferred — pre-authenticated gh CLI, no MCP server startup
    toolsets: [default]
```

Agent reads GitHub via `gh issue list`, `gh pr view`, etc. and pipes through `jq` before data enters context. `mode: local` starts a Docker-based MCP server with startup latency and verbose tool results.

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

With `cli-proxy`, the agent calls `my-custom-mcp <tool> <args>` from bash and pipes output through `jq`/`grep` to extract only needed fields — instead of receiving the full MCP tool response in context.

**Summary:**

| Mode | Docker startup | Extra tool definitions | Agent output processing |
|---|---|---|---|
| `mode: local` + MCP tools | Yes | Yes | Tool result (full JSON) |
| `mode: gh-proxy` + bash | No | No | Agent pipes through jq |
| `cli-proxy: true` + bash | Yes (once) | Reduced | Agent pipes through jq |

---

## Technique 3 — Inline Sub-Agents with Smaller Models

Sub-agents with `model: small` cost 10–20× less than the parent model. Use them for classification, one-sentence summarization, structured extraction, and scoring; reserve the large model for synthesis.

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

**Why this saves tokens:** sub-agents run the cheap `small` model; main agent reads only compact `{"number":…, "category":…}` JSON; dispatches run in parallel.

### Pair sub-agents with sub-skills (progressive disclosure)

- Keep the main prompt short and plan-like (what to do, in what order).
- Put verbose instructions (report layout, rubric details, formatting constraints) into `## skill:` blocks.
- Invoke skills only when needed (e.g., producing final output), so early turns stay lean.

This delays expensive instruction payloads until the final phase, lowering ambient context.

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

A/B compare a verbose prompt against a minimal one. Adopt minimal if quality holds.

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

Measure AIC via run summary or `gh aw audit`. If `minimal` wins on cost at acceptable quality, promote as baseline.

---

## Technique 5 — Use Experiments to Measure Impact

Declare an experiment before making any prompt or configuration change, and compare before/after cost and quality. Run ≥ 20 cycles per variant for statistical significance on high-frequency workflows.

```yaml
experiments:
  optimization_v1:
    variants: [control, optimized]
    description: "DataOps refactor — move issue fetching to steps:"
    metric: "aic"
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
2. Inspect `aic`, `input_tokens`, `output_tokens`, `cache_read_tokens`, and `cache_write_tokens`
3. Validate output quality and decision accuracy against the control run
4. If the optimized variant wins on cost **and** quality, rewrite the baseline prompt and remove the `experiments:` field

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

The cheapest run is the one you don't execute. If a workflow doesn't need near-real-time feedback, run it less often and batch.

### Prefer slower schedules when latency is acceptable

- `hourly` → `daily on weekdays` for team-facing summaries or audits
- `daily` → `weekly` for trend reports, optimization reviews, backlog hygiene
- `every N hours` → daily/weekly batch when the workflow only produces guidance

### Prefer scheduled batches over reactive triggers

Reactive triggers (`issues:`, `pull_request:`, comment commands) suit immediate feedback. Otherwise prefer `schedule: daily on weekdays` and batch work. Typical batch-friendly tasks: triage summaries, stale backlog review, token audits, security digests. Combine with `cache-memory` or `repo-memory` to track processed items.

---

## Technique 7 — Measure Continuously with OpenTelemetry and AgenticOps

Export telemetry automatically and add workflows that keep finding token waste over time.

### Enable OTLP export

Add workflow-level OpenTelemetry export so each run emits token and phase data to your observability backend:

```yaml
observability:
  otlp:
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
```

Setup, agent, and conclusion spans carry token usage attributes. See [Frontmatter syntax](syntax-agentic.md#agentic-workflow-specific-fields).

### Add AgenticOps token workflows

- `copilot-token-audit` — scheduled audit of token usage across workflows
- `copilot-token-optimizer` — scheduled follow-up that identifies one expensive workflow and proposes concrete savings

Loop: export OTEL → summarize usage → open optimization issues → re-measure. See `.github/workflows/` for examples.

---

## Technique 8 — Enable Prompt Caching

Prompt caching is automatic via the AWF gateway. Cached input tokens are weighted at `0.1` versus `1.0` for uncached input — repeated context (system prompt, shared preamble) costs ~10× less when cached.

To maximize cache hits:

- **Keep stable content at the top of the prompt** — instructions that don't change between runs (role, output format, schema) before dynamic content (issue body, event context).
- **Use `cache-memory`** for workflows that re-read the same large knowledge base across runs; avoids duplicate context every turn.
- **Minimize dynamic context** — inject only the fields the agent needs: `${{ github.event.issue.number }}` instead of the full event payload.

---

## Technique 9 — Cap Spend with AI-Credit Guardrails

Two top-level frontmatter fields enforce AI Credit budgets directly, independent of the techniques above. Both accept an integer or a `K`/`M` short-form string (e.g. `100M`, `500K`). Typical workflow range: `100` to `2500`.

- **`max-ai-credits:`** — Per-run AI credit budget enforced by the AWF firewall/API proxy (default `1000`). The agent is steered to stay within budget; set a negative value to disable enforcement and steering.
- **`max-daily-ai-credits:`** — Per-user 24-hour guardrail. At activation, gh-aw sums the triggering user's AI credits across their runs of this workflow over the last 24 hours and blocks execution once the total exceeds the threshold. Enabled by default with a system default threshold; set `-1` to disable, or an explicit value to override the default.

```yaml
max-ai-credits: 100M        # per-run cap (short-form string)
max-daily-ai-credits: 500M  # per-user 24h cap; -1 disables
```

For custom or private models, the top-level **`models:`** frontmatter field supplies pricing in the same structure as `models.json` (keyed `providers.<provider>.models.<model>.cost` with `input`/`output`/`cache_read`/`cache_write` per-token costs). Entries are merged with the built-in `models.json` at runtime — they override matching models and fill gaps for unknown ones — so AI Credit accounting stays accurate for models gh-aw does not price by default.

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
