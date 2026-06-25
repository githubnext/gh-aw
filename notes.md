# Copilot Session Insights — repo memory

## 2026-06-25 snapshot
- 50 sessions; **8% completion** (4 success, 46 action_required, 0 failure) — uptick **+6pts** from 06-24 (2%); recent-7d (06-19..25) avg **9.1%** (saw-tooth: 6,0,24,4,20,2,8)
- **GATE-SWEEP**: 46/50 zero-duration; 36-min window 06:47–07:22Z; 4 nonzero = the 4 successes (14.6/14.6/16.2/29.8m)
- All 4 successes = `Addressing comment on PR #41385/41387/41388/41401` (Copilot cloud-agent PR-comment runs), one per distinct copilot/* branch → **provenance_inversion holds**
- 6 branches, **all copilot/***; top-2 retry-loop-drained-tokens-2 (21, 42%) + remove-strict-false-and-fix-env-support (15, 30%) = **72%**
- Orphans: **0**; 6 open PRs all Copilot-assigned, only `main` has active runs → 0% (vs 40% baseline, NORMAL, ~35th healthy day)
- Conversation logs empty/OAuth **32nd+ day**; standard run (roll=68)

## Active patterns
- provenance_inversion (06-07): successes come from agentic runs (PR-comment / cloud-agent), never from gate sweeps. Holds 06-25 (all 4 = PR-comment).
- inverse_gate_count_to_conclusiveness: Copilot-assigned ⇒ never orphaned (0 orphans again 06-25, ~35th healthy day).
- gate_sweep_zero_duration: recurring — snapshots routinely catch 46-50 action_required 0-duration runs.
- recovery_regression_oscillation: saw-tooth persists; spikes (38-40%) between troughs (0-8%); recent week low single-digits.
- conversation_log_fetch_failure: 32nd+ day (longest unresolved risk).
- gate_footprint_refire_signature (06-20): high daily gate counts can come from one branch re-firing a few workflows; refire ratio = runs/distinct-workflows distinguishes broad CI from narrow re-fire.
