# Copilot Session Insights — repo memory

## 2026-06-20 snapshot
- 50 sessions; **0% completion** (0 success, 49 action_required, 1 failure) — NEW zero-day; recent-7d avg **6.0%** vs prior-7d **20.6%** (clear downtrend)
- **PURE GATE-SWEEP**: 50/50 zero-duration; window 05:58–06:22Z (24 min)
- 4 copilot/* branches; dominant **update-custom-css-themes 27/50 (54%)**
- The only failure = `.github/workflows/skillet.lock.yml` on the dominant branch
- Orphans: **0**; 10 open PRs (8 copilot all Copilot-assigned, 2 idle housekeeping w/ 0 gates) → orphan_rate 0% (vs 40% baseline, NORMAL, ~28th healthy day)
- Conversation logs empty **27th+ day**; experimental run (roll=20)

## Experimental finding (2026-06-20): Gate Footprint Signature Classification
- Classify a branch's gate activity by **refire ratio = total runs / distinct workflows**.
- **Narrow high-refire** (update-custom-css-themes: 27 runs / 5 distinct, ratio 5.4 — Q×12 + Agentic Commands×12): one branch re-firing the same 2 workflows; inflates daily gate-sweep count and co-located the only failure.
- Implication: GSII (sessions/branches=12.5) is bimodal by branch; "50 gate sweeps" ≠ 50 distinct gates. Effectiveness Medium-High; recommend Refine (track refire ratio per branch over time, watch if high-refire branches correlate with failures).

## Active patterns
- provenance_inversion (06-07): recovery days (>=40%) gate/moderator workflows CAN produce successes; on pure gate-sweep days they do not. 06-20 is a 0% gate-sweep — no successes from any source.
- copilot_cloud_agent_reliability: holds on productive days; 06-20 had zero substantive sessions so untested
- inverse_gate_count_to_conclusiveness: holds; Copilot-assigned ⇒ never orphaned (0 orphans again 06-20)
- gate_sweep_zero_duration: recurring — snapshots routinely catch 49-50 action_required 0-duration runs
- recovery_regression_oscillation: saw-tooth persists; spikes (06-07/06-10/06-13 @38-40%) between troughs (0-8%), but recent week trending DOWN (20.6%→6.0%)
- conversation_log_fetch_failure: 27th+ day (longest unresolved risk)
- NEW gate_footprint_refire_signature (06-20): high daily gate counts can come from one branch re-firing a few workflows, not broad CI; refire ratio distinguishes the two
