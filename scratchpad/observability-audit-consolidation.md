# Pre-Consolidation Analysis: Observability & Audit Workflows

## Scope

Five workflows all overlap in the "monitor and report on agentic workflow runs" space:

| Short name | File | Lines |
|------------|------|------:|
| **Kit** | `agentic-observability-kit.md` | 569 |
| **Audit** | `audit-workflows.md` | 99 |
| **DOR** | `daily-observability-report.md` | 393 |
| **WHM** | `workflow-health-manager.md` | 456 |
| **SOH** | `safe-output-health.md` | 361 |
| **Total** | | **1,878** |

---

## Table 1 — What Each Workflow Currently Analyzes

| Dimension | Kit | Audit | DOR | WHM | SOH |
|-----------|-----|-------|-----|-----|-----|
| **Primary question** | Are runs operationally healthy & portfolio optimal? | Were there issues, missing tools, or errors in the last 24 h? | Do firewall & MCP gateway logs provide adequate telemetry? | Are all 120+ workflows themselves structurally healthy? | Did safe-output jobs succeed or fail? |
| **Analysis subjects** | Workflow _runs_ (episodes) | Workflow _runs_ (raw logs) | Workflow _runs_ (log artifacts: access.log, gateway.jsonl) | Workflow _files_ + lock files + CI | Safe-output _jobs_ within runs |
| **Scope** | Full repository, all workflows | Full repository, all workflows | Full repository, firewall- and MCP-enabled runs only | Full repository, all `.md` + `.lock.yml` files | Full repository, safe-output job logs only |
| **Data model** | Episode graph (`episodes[]`, `edges[]`) + behavior fingerprint | Raw per-run logs from `logs` tool | File presence checks on log artifact directories | `git ls-files` + Actions API + repo-memory | Per-safe-output-job log lines |
| **What it looks for** | Risk regressions, cost outliers, portfolio overlap, escalation triggers | Missing tools, MCP failures, auth/timeout errors, performance | Missing `access.log`, missing `gateway.jsonl`/`rpc-messages.jsonl` | Missing lock files, failed runs, stale configs, dependency issues | Safe-output errors by type and cluster |
| **Time window (primary)** | 30 days (`count: 300–500`), emphasize last 14 | 24 hours (`start_date: "-1d"`) | 7 days (`start_date: "-7d"`) | 7 days per workflow (last 10 runs) | 24 hours (`start_date: "-1d"`) |
| **Time window (charts)** | 30 days (4 analytical charts) | 30 days (2 trend charts) | N/A (no charts) | 30 days (metrics from repo-memory) | N/A (no charts) |
| **Episode / lineage model** | ✅ Primary (`episodes[]`, `edges[]`) | ❌ Not used | ❌ Not used | ❌ Not used | ❌ Not used |
| **Firewall analysis** | ⚠️ Incidental (blocked requests as risk signal) | ⚠️ Incidental (blocked requests as error signal) | ✅ Primary (`access.log` coverage) | ❌ Not analyzed | ❌ Not analyzed |
| **MCP gateway analysis** | ⚠️ Incidental (MCP failure as risk signal) | ⚠️ Incidental (MCP failures as errors) | ✅ Primary (`gateway.jsonl`/`rpc-messages.jsonl`) | ❌ Not analyzed | ❌ Not analyzed |
| **Safe-output analysis** | ❌ Not analyzed | ❌ Not analyzed | ❌ Not analyzed | ⚠️ Incidental (safe-output job health) | ✅ Primary |
| **Portfolio / overlap** | ✅ Secondary appendix (workflow overlap matrix chart) | ❌ Not analyzed | ❌ Not analyzed | ⚠️ Incidental (stale/redundant detection) | ❌ Not analyzed |
| **Trend charts generated** | 4 (risk frontier, stability matrix, portfolio map, overlap matrix) | 2 (health trend, token/cost trend) | 0 | 0 | 0 |

---

## Table 2 — Workflow Configuration Snapshot

| Property | Kit | Audit | DOR | WHM | SOH |
|----------|-----|-------|-----|-----|-----|
| **Schedule** | Weekly (Mon ~08:00) | Daily | Daily | Daily | Daily |
| **Trigger** | schedule + `workflow_dispatch` | schedule + `workflow_dispatch` | `on: daily` | `on: daily` | schedule + `workflow_dispatch` |
| **Engine** | copilot | claude | codex | copilot | claude |
| **`strict: true`** | ✅ | ❌ | ✅ | ❌ | ✅ |
| **`timeout-minutes`** | 30 | 30 | 45 | 30 | 30 |
| **Tools: `agentic-workflows`** | ✅ | ✅ | ✅ | ❌ | ❌ (via import) |
| **Tools: `github`** | ✅ `[default, discussions]` | ❌ | ✅ `[default, discussions, actions]` | ✅ `[default, actions]` | ❌ |
| **Tools: `bash`** | ❌ | ❌ | ❌ | ✅ `[":*"]` | ❌ |
| **Tools: `edit`** | ❌ | ❌ | ❌ | ✅ | ❌ |
| **Tools: `repo-memory`** | ❌ | Via `repo-memory-standard` import | ❌ | ✅ (branch: `memory/meta-orchestrators`) | ❌ |
| **Tools: `cache-memory`** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **`mount-as-clis: true`** | ✅ | ✅ | ✅ | ❌ | ❌ |
| **`features.mcp-cli`** | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Permissions: `discussions`** | ✅ read | ❌ | ✅ read | ❌ | ❌ |
| **tracker-id** | `agentic-observability-kit` | `audit-workflows-daily` | `daily-observability-report` | _(none)_ | _(none)_ |

---

## Table 3 — Output Artifacts

| Output | Kit | Audit | DOR | WHM | SOH |
|--------|-----|-------|-----|-----|-----|
| **Discussion** | ✅ `[observability] …` expires 7d | ✅ `[audit-workflows] …` expires 1d | ✅ `[observability] …` expires 1d | ❌ | ✅ `[safe-output-health] …` expires 1d |
| **Escalation issue** | ✅ 1 max, `[observability escalation]`, labels: agentics/warning/observability | ❌ | ❌ | ✅ up to 10, `cookie` label, grouped | ❌ |
| **Comment on issue** | ❌ | ❌ | ❌ | ✅ up to 15 | ❌ |
| **Update issue** | ❌ | ❌ | ❌ | ✅ up to 5 | ❌ |
| **Charts (max)** | 4 PNG/SVG | 3 PNG/JPG/SVG | 0 | 0 | 0 |
| **Repo memory write** | ❌ | ✅ `memory/audit-workflows` branch | ❌ | ✅ `memory/meta-orchestrators` branch | ❌ |
| **Cache memory write** | ❌ | ❌ | ❌ | ❌ | ✅ `/tmp/gh-aw/cache-memory/safe-output-health/` |
| **noop** | ✅ (not reported as issue) | ❌ | ❌ | ❌ | ❌ |

---

## Table 4 — Import Dependencies

| Import | Kit | Audit | DOR | WHM | SOH |
|--------|-----|-------|-----|-----|-----|
| `shared/daily-audit-discussion.md` | ✅ (expires 7d) | ✅ (expires 1d) | ❌ (uses `daily-audit-base.md` instead) | ❌ | ✅ (expires 1d) |
| `shared/daily-audit-base.md` | ❌ | ❌ | ✅ | ❌ | ❌ |
| `shared/trending-charts-simple.md` | ✅ | ✅ | ❌ | ❌ | ❌ |
| `shared/reporting.md` | ✅ | ✅ | ❌ | ✅ | ✅ |
| `shared/repo-memory-standard.md` | ❌ | ✅ | ❌ | ❌ | ❌ |
| `shared/jqschema.md` | ❌ | ✅ | ❌ | ❌ | ✅ |
| `shared/aw-logs-24h-fetch.md` | ❌ | ❌ | ❌ | ❌ | ✅ |
| `.github/shared-instructions.md` (runtime import) | ❌ | ❌ | ✅ | ✅ | ❌ |

---

## Table 5 — Overlap Map: KEEP vs. CONSOLIDATE

Key: **✅ KEEP** = unique capability, no other workflow covers it. **⚠️ CONSOLIDATE** = overlaps with another; merge into the preference winner.

| Capability / Signal | Kit | Audit | DOR | WHM | SOH | Preference Winner |
|--------------------|-----|-------|-----|-----|-----|-------------------|
| Daily "what broke in 24 h" run digest | ❌ | ✅ KEEP | ❌ | ⚠️ partial | ❌ | **Audit** |
| Weekly executive health narrative with escalation | ✅ KEEP | ❌ | ❌ | ❌ | ❌ | **Kit** |
| Episode/lineage DAG analysis | ✅ KEEP | ❌ | ❌ | ❌ | ❌ | **Kit** (only user) |
| Portfolio overlap + workflow value map | ✅ KEEP | ❌ | ❌ | ⚠️ stale detection | ❌ | **Kit** |
| 4-chart visual analytical suite | ✅ KEEP | ❌ | ❌ | ❌ | ❌ | **Kit** |
| 2-chart 30-day trend (health + cost) | ⚠️ redundant | ✅ KEEP | ❌ | ❌ | ❌ | **Audit** (charts match its daily cadence) |
| Firewall `access.log` coverage check | ❌ | ❌ | ✅ KEEP | ❌ | ❌ | **DOR** |
| MCP gateway `gateway.jsonl`/`rpc-messages.jsonl` check | ❌ | ❌ | ✅ KEEP | ❌ | ❌ | **DOR** |
| Infrastructure-layer telemetry (Squid, JSONL parsing) | ❌ | ❌ | ✅ KEEP | ❌ | ❌ | **DOR** |
| Workflow file + lock file structural health | ❌ | ❌ | ❌ | ✅ KEEP | ❌ | **WHM** |
| CI / Actions run health per workflow | ⚠️ partial | ⚠️ partial | ❌ | ✅ KEEP | ❌ | **WHM** |
| Repo-memory trend accumulation (30+ days) | ❌ | ✅ KEEP | ❌ | ✅ KEEP | ❌ | Both (different branches) |
| Safe-output job error clustering | ❌ | ❌ | ❌ | ⚠️ incidental | ✅ KEEP | **SOH** |
| Cache-memory persistent error patterns | ❌ | ❌ | ❌ | ❌ | ✅ KEEP | **SOH** |
| `[observability]` discussion prefix | ✅ | ❌ | ✅ **CONFLICT** | ❌ | ❌ | **Kit** (weekly); **DOR** should rename |
| Escalation issue creation | ✅ KEEP | ❌ | ❌ | ⚠️ overlap | ❌ | **Kit** for agentic escalation; **WHM** for structural |
| Missing-tool pattern detection | ❌ | ✅ KEEP | ❌ | ❌ | ❌ | **Audit** |
| MCP failure error analysis | ⚠️ risk signal | ✅ KEEP | ❌ | ❌ | ❌ | **Audit** (daily granularity is more actionable) |
| Token / cost reporting | ✅ (episode-level) | ✅ (daily trend) | ❌ | ❌ | ❌ | Both serve different cadences |
| Blocked-request reporting | ✅ (risk signal) | ✅ (error signal) | ✅ (coverage signal) | ❌ | ❌ | Distinct angles — all KEEP |

---

## Table 6 — Key Conflicts Requiring Resolution

| Conflict | Workflows | Impact | Recommended Resolution |
|----------|-----------|--------|------------------------|
| **Shared `[observability]` discussion title prefix** | Kit (expires 7d) + DOR (expires 1d, same prefix) | DOR's daily close-older-discussions call closes Kit's weekly report prematurely | Rename DOR prefix to `[firewall-mcp-coverage]` or `[telemetry-coverage]` |
| **Both analyze MCP failures** | Kit (risk score) + Audit (error log) | Dual-counting; readers get two different numbers for same failures | Separate concerns: Kit = trend/episode, Audit = raw daily errors |
| **Both escalate operational issues** | Kit (observability escalation issue) + WHM (cookie issues) | Two escalation channels, different labels, risk of duplication | Clarify: Kit escalates agentic behavior; WHM escalates structural/lock-file issues |
| **Both produce 30-day trend charts** | Kit (4 charts) + Audit (2 charts, same chart type: health + cost) | Workflow health and token/cost charts duplicated weekly vs. daily | Remove `trending-charts-simple` from Kit; Kit charts are already superior analytical replacements |
| **Both use `shared/trending-charts-simple.md`** | Kit + Audit | Shared import but Kit's charts supersede what `trending-charts-simple` adds | Drop `trending-charts-simple` import from Kit (use its own 4 custom charts) |
| **Audit has no `strict: true`** | Audit only | Audit can output to stdout without safe-output guard | Add `strict: true` to Audit if it should match Kit/SOH hygiene |

---

## Table 7 — Engine Selection Rationale (Current State)

| Workflow | Engine | Why (inferred) | Concerns |
|----------|--------|----------------|----------|
| Kit | copilot | Broad analytical reasoning over episode DAGs; portfolio judgments | Weekly cadence means higher per-run cost is acceptable |
| Audit | claude | Strong at structured log reading, pattern matching, natural language error summaries | Daily run; cost per run matters more than Kit |
| DOR | codex | Systematic file-existence checks; coverage math; tabular output | Least LLM-intensive task in this group; codex overhead may be overkill |
| WHM | copilot | Broad multi-step orchestration; file inspection + issue triage | `bash` and `edit` tools indicate it needs an agent that can act, not just report |
| SOH | claude | Error clustering and root-cause classification from log text | Daily run; structured output oriented |

**Consolidation implication**: DOR is the strongest candidate for a model downgrade (codex → smaller/cheaper) since its task is primarily mechanical file-presence checking, not open-ended reasoning.

---

## Table 8 — Summary: Five Workflows, Four Distinct Concerns

| Concern | Primary Workflow | Secondary / Redundant | Action |
|---------|-----------------|----------------------|--------|
| **Agentic health narrative + portfolio** | Kit (weekly) | Audit (daily) partially overlaps | KEEP both; Kit is strategic, Audit is tactical |
| **Log artifact / telemetry coverage** | DOR | — | KEEP; rename discussion prefix |
| **Structural workflow file health** | WHM | — | KEEP; no overlapping workflow |
| **Safe-output job failures** | SOH | — | KEEP; no overlapping workflow |

The pair with the most consolidation opportunity is **Kit ↔ Audit**. The four remaining workflows each own a distinct concern and have minimal overlap with each other.

---

## Appendix: Raw Signal Inventory by Workflow

### Kit — Unique Signals (not in Audit)
- `episodes[].episode_id`, `.kind`, `.confidence`, `.reasons[]`
- `episodes[].escalation_eligible`, `.escalation_reason`, `.suggested_route`
- `edges[].edge_type`, `.confidence`
- `behavior_fingerprint.*` (execution_style, tool_breadth, actuation_style, …)
- `comparison.baseline.selection`, `.matched_on[]`
- `comparison.classification.label`, `.reason_codes[]`
- `agentic_assessments[].kind`, `.severity`
- `workflow_instability_score`, `workflow_value_proxy`, `episode_risk_score` (derived)
- `workflow_overlap_score(a, b)` (pairwise, derived)
- Portfolio map chart (quadrant: keep/optimize/simplify/review)
- Overlap matrix chart (pairwise similarity heatmap)

### Audit — Unique Signals (not in Kit)
- Per-run missing-tool pattern frequency
- Auth failure categorization
- `summary.engine_counts` from `logs` tool (engine classification)
- `repo-memory` pattern storage (`patterns/errors.json`, `patterns/missing-tools.json`)
- 24-hour granularity (Kit's minimum is ~7 days of meaningful signal)

### DOR — Unique Signals (not in Kit or Audit)
- `access.log` file presence + Squid log entry count
- `gateway.jsonl` file presence + JSONL field validation
- `rpc-messages.jsonl` as canonical fallback telemetry source
- Firewall coverage percentage (`firewall_logs_present / firewall_enabled_workflows`)
- Gateway coverage percentage (`gateway_logs_present / mcp_enabled_workflows`)
- Per-run severity: CRITICAL / WARNING / HEALTHY / N/A classification

### WHM — Unique Signals (not in any other workflow)
- Lock file existence for every `.md` file (structural completeness)
- Dependency graph between workflows (who imports whom)
- Health score per workflow (0–100, based on recent run success rate + error patterns)
- `memory/meta-orchestrators` shared repo memory
- Direct `edit` capability (can write fixes to workflow files)

### SOH — Unique Signals (not in any other workflow)
- Safe-output job type breakdown (create_discussion, create_issue, add_comment, …)
- Error cluster analysis across safe-output job types
- Recurring failure tracking in cache-memory
- `error-patterns.json` + `recurring-failures.json` persistent state
