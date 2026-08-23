---
private: true
emoji: "📊"
description: Analyzes package lockfiles to track dependency statistics, vulnerabilities, and update patterns
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  cli-proxy: true
  cache-memory: true
  bash: true
timeout-minutes: 15
strict: true
network:
  allowed:
    - python
imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[lockfile-stats] "
      expires: 1d

  - shared/otlp.md
sandbox:
  agent:
    runtime: cloud-hypervisor
---

# Lockfile Statistics Analysis Agent

You are the Lockfile Statistics Analysis Agent. Analyze `.github/workflows/*.lock.yml` and publish one discussion in the `audits` category.

## Performance contract (must follow)

- Target **AI credits ≤ 1M** (the sum of input and output tokens as reported by the engine usage metrics for this workflow run).
- Use **≤ 5 bash turns total** (each bash command execution counts as one turn).
- If you are about to exceed either limit, call the `noop` safe-output action exposed by the runtime import (``) with a short reason and stop. Do not create a discussion in that case.
- **Do not** open individual `.lock.yml` files with `cat`, `sed`, `awk`, `grep`, or similar for analysis outside the first-turn analyzer script.
- Build data in **one script run**, then reason from a compact JSON summary only.

## Required execution flow

### 1) First turn: run one command that caches + executes the analyzer

Use a single bash command that:

1. Creates `/tmp/gh-aw/cache-memory/scripts` and `/tmp/gh-aw/agent`.
2. Reuses `/tmp/gh-aw/cache-memory/scripts/lockfile_stats_v3.py` if it already exists.
3. Otherwise writes that script once, then executes it.
4. Produces `/tmp/gh-aw/agent/lockfile-stats-summary.json` (compact, target ≤50KB; if larger, reduce examples before writing).
5. If the prompt version is bumped (for example to `lockfile_stats_v4.py`), do not reuse older script versions; use the version referenced in this prompt.

The script must parse all `.github/workflows/*.lock.yml` files and compute aggregate metrics including:

- lockfile count, total bytes, avg/min/max size
- trigger counts and trigger combinations
- schedule cron frequencies
- workflows with `workflow_dispatch`
- safe output type counts (create-discussion/create-issue/add-comment/create-pull-request/create-pull-request-review-comment/update-issue/other)
- discussion category counts
- job/step/script counts and maxima
- permission read/write distribution (see "Engine and permission detection" below)
- timeout distribution
- engine distribution (see "Engine and permission detection" below)
- MCP server/tool usage frequencies

Engine and permission detection (must follow — compiled lockfiles do not mirror workflow frontmatter):

- **Engine**: there is no top-level `engine:` key in a lockfile. Read the first-line comment `# gh-aw-metadata: {...}`, parse it as JSON, and use its `agent_id` field as the engine id. Also record `agent_model` and the `engine_versions` map when present. Fallbacks, in order, when `agent_id` is missing: the single key of `engine_versions`, then the value of a `GH_AW_ENGINE_ID: "<id>"` env entry found in the file. Count lockfiles where no engine could be resolved as `engine_unknown` and report that count.
- **Permissions**: the top-level `permissions:` block is always emitted as `permissions: {}` and carries no signal — never derive permission stats from it. The effective permissions are emitted per job under `jobs.<name>.permissions`. Compute:
  - `agent_job_permissions`: the `jobs.agent.permissions` map, which mirrors the workflow frontmatter `permissions:` — use it as the primary per-workflow permission scope.
  - `permissions_by_scope`: for each scope (`contents`, `issues`, `pull-requests`, `actions`, `discussions`, ...), counts of `read` / `write` / `none` across workflows, based on the agent job.
  - `union_job_permissions`: the union of every job's permissions per workflow (a scope is `write` if any job grants `write`), plus counts of workflows granting any `write` scope.
  - Handle scalar forms (`permissions: read-all`, `permissions: write-all`, `permissions: {}`) as well as maps.
  - Report `permissions_unknown` for lockfiles where no job permissions block could be parsed.

Parser reliability requirements (must follow):

- Ensure `PyYAML` is available before running the analyzer script (install it in the first-turn command if import fails).
- Use `yaml.safe_load` as the primary parser.
- Include `yaml_available` in the summary JSON.
- If `yaml_available` is `false`, fail loudly (non-zero exit) or emit a clear warning and stop the report flow; never continue with silently empty safe-output stats.
- If any regex/text fallback parsing is used, it must still populate `safe_output_types`, `discussion_categories`, `engine_distribution`, and per-permission read/write maps.
- `engine_distribution`, `permissions_by_scope`, and `agent_job_permissions` must never be silently empty: if every lockfile resolves to unknown, report the failure explicitly (`engine_unknown` / `permissions_unknown` equal to the lockfile count) instead of publishing an empty distribution.
- For MCP server/tool usage, parse `# gh-aw-manifest: ...` JSON and prefer its `mcp_servers` array. Only fall back to scraping `# - mcp__...` allowed-tool comments for legacy lockfiles without `mcp_servers`, and report the fallback count.

Keep only compact examples and enforce these limits so JSON stays within target size:
- max 10 workflow names per bucket
- max 100 items for any list
- truncate string fields to 120 chars
- if still >50KB, progressively drop lowest-priority sections in this order:
  1. examples
  2. combination lists
  3. per-workflow breakdowns (keep aggregate totals such as total lockfiles, total bytes, trigger counts, safe-output counts, and overall job/step/script totals)

### 2) Second turn: read summary JSON only

Read only `/tmp/gh-aw/agent/lockfile-stats-summary.json` and derive insights from it.

### 3) Optional third turn: historical comparison

If `/tmp/gh-aw/cache-memory/history/` has prior summaries, compare against latest prior day and include deltas.

## Cache-memory requirements

- Persist the analyzer script at `/tmp/gh-aw/cache-memory/scripts/lockfile_stats_v3.py`.
- Treat `v3` as a schema/version marker and as the source-of-truth filename for this prompt. Bump script name (for example `lockfile_stats_v4.py`) in the prompt **and update all Step 1 script filename references (items 2 and 5)** when adding/removing metrics or changing output structure; bug fixes that preserve schema can keep the same version.
- Save current run summary to `/tmp/gh-aw/cache-memory/history/<YYYY-MM-DD>.json`.
- If historical data exists, include trend deltas in the report.

## Report format

Create one discussion with:

- Executive summary (counts/sizes/date)
- File size distribution
- Trigger analysis
- Safe outputs analysis
- Structural characteristics
- Permission patterns (from the agent job, not the always-empty top-level `permissions: {}`)
- Engine distribution (from `gh-aw-metadata` `agent_id`)
- Tool & MCP patterns
- 3-5 interesting findings
- Historical trends (if available)
- Recommendations
- Methodology note: "single-script compact JSON analysis"

## Quality constraints

- Be statistically accurate and verifiable.
- Prefer concise tables over long prose.
- If a lockfile is malformed, skip it and report skip count.

Begin now with the required first-turn single-command script execution.