# Copilot Session Insights — repo memory

## 2026-06-27 snapshot
- 50 sessions; **40% completion** (20 success, 27 action_required, 2 skipped, 1 cancelled) — **sharp +38pt upturn** from 06-26 (2%); highest since 05-26 (46%). Saw-tooth recovery spike, not yet sustained.
- **Bimodal**: 27/50 zero-dur gate sweeps; median 0 vs mean 3.83m; real-work cluster = 11 sessions ≥5min, 2 ≥15min, max 45.7m.
- **High concentration**: only 6 unique branches; top-2 `fix-nolint-suppression-gap` (13) + `fix-fmterrorfnoverbs-false-negative` (13) = **52%**. Concentration coincided with the completion upturn (matches concentrated_branch_activity).
- Orphans: **0**; 10 open PRs, only 4 in-progress runs (max 2 gates/branch, on `main`) → 0% (vs 40% baseline, NORMAL, ~37th healthy day). 0 escalation candidates.
- Conversation logs still empty (0 files) **30+ consecutive days**; standard run (roll=88).

_(Per-day detail before 2026-06-27 lives in session-trends.jsonl / session-analysis-history.json.)_

## Active patterns
- provenance_inversion (06-07): successes come from agentic runs (PR-comment / cloud-agent), never from gate sweeps. Holds 06-25 (all 4 = PR-comment).
- inverse_gate_count_to_conclusiveness: Copilot-assigned ⇒ never orphaned (0 orphans again 06-25, ~35th healthy day).
- gate_sweep_zero_duration: recurring — snapshots routinely catch 46-50 action_required 0-duration runs.
- recovery_regression_oscillation: saw-tooth persists; spikes (38-40%) between troughs (0-8%); recent week low single-digits.
- conversation_log_fetch_failure: 32nd+ day (longest unresolved risk).
- gate_footprint_refire_signature (06-20): high daily gate counts can come from one branch re-firing a few workflows; refire ratio = runs/distinct-workflows distinguishes broad CI from narrow re-fire.
