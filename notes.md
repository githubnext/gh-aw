# Copilot Session Insights — repo memory

## 2026-07-02 snapshot
- 50 sessions; **8% completion** (4 success, 46 action_required) — floor regime after 20% on 07-01.
- provenance_inversion holds (4th+ obs): 4 successes = agent-exec (2 cloud-agent + 2 PR-comment), only non-zero-dur runs (max 11.98m, median 0); 46 action_required = CI/infra gates.
- Orphans 0 (max gates/branch=3); 18 open PRs (11 Copilot-assigned, 7 unassigned, none with active gate sweep) → 0% NORMAL. Conv logs OAuth stub. Standard run (roll=85).

## 2026-06-28 snapshot
- 50 sessions; **10% completion** (5 success, 45 action_required) — **saw-tooth pullback −30pt** from 06-27 (40%); 30d-avg 13.2%.
- Bimodal: 45/50 zero-dur gate sweeps; median 0 vs mean 1.42m; nonzero=5 (max 21.65m); 128-min window.
- provenance_inversion holds cleanly: all 5 successes = agentic (4 cloud-agent + 1 PR-comment), **exactly 1 per branch**; every gate firing = action_required.
- Concentration: 5 `copilot/*` branches; top-3 @12 each = 72%. Full gate bundle per PR-open (AI Moderator+Agentic Commands+Q+Smoke CI+CGO+CWI+Content Mod+Doc-Deploy).
- Orphans 0; 14 open PRs (11 Copilot-assigned, 3 idle 0-gate), 1 in-progress run (main) → 0% NORMAL, ~38th healthy day. 0 escalations.
- Conv logs empty 35+ days; standard run (roll=67).

## 2026-06-27 snapshot (prior peak)
- 40% completion (20 succ); +38pt upturn from 06-26 (2%), highest since 05-26 (46%); 6 branches top-2 nolint+fmterror @13=52%; orphan 0/10 ~37th healthy day.

_(Per-day detail lives in session-trends.jsonl / session-insights-history.jsonl.)_

## Active patterns
- provenance_inversion (06-07): successes come from agentic runs (PR-comment / cloud-agent), never gate sweeps. Holds 06-28 (all 5).
- inverse_gate_count_to_conclusiveness: Copilot-assigned ⇒ never orphaned (~38th healthy day).
- gate_sweep_zero_duration: snapshots routinely catch 45-50 action_required 0-duration runs.
- recovery_regression_oscillation: saw-tooth persists; spikes (38-40%) between troughs (0-10%).
- conversation_log_fetch_failure: 35th+ day (longest unresolved risk; behavioral/loop/context analysis unavailable — metrics are CI/infra metadata only).
- gate_footprint_refire_signature (06-20): refire ratio = runs/distinct-workflows distinguishes broad CI from narrow re-fire.
- per_branch_gate_fanout (06-26): each PR-open fires a full ~8-workflow gate bundle; gate_count = f(PR-open), not branch health.
