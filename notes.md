# Copilot Session Insights — repo memory

## 2026-07-03 snapshot
- 50 sessions; **4% completion** (2 success, 46 action_required, 2 in_progress) — floor regime continues (20%→8%→4% over 07-01..07-03); below 30d-mean ~13% & 15d-mean ~10%.
- provenance_inversion holds (5th+ obs): 2 successes = agentic (PR Sous Chef 8.62m + Skillet 0.48m, both on lint-monster-targeted-cleanup, 0/4 core CI gates); 46 action_required = CI gate sweeps (median 0m, avg 0.19m).
- Concentration: 8 `copilot/*` branches; top-2 duplicate-code-fix(16)+runtime-cloning(13)=58%. 22.5-min window (07:18–07:40Z).
- Orphans 0 (active runs all on `main`; max gates/copilot-branch=0); 12 open PRs (11 Copilot-assigned, 1 unassigned fresh codeql PR #43148 0-gate) → 0% NORMAL, ~39th healthy day. 0 escalations.
- Conv logs empty (36th day). **EXPERIMENTAL run (roll=8): Gate-Bundle Composition Divergence (GBCD)** — only 3/8 branches fire full 4/4 core CI gate set (Smoke CI+CGO+CWI+Doc-Deploy); lint/doc branches fire 0/4 (lightweight agentic wf); update-checkout fires 2/4 + moderation. Refines per_branch_gate_fanout: bundle is change-TYPE-adaptive, not uniform per PR-open. Effectiveness Medium; recommend Refine.

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

_(Prior peak 06-27: 40% (20 succ). Per-day detail in session-trends.jsonl / session-insights-history.jsonl.)_

## Active patterns
- provenance_inversion (06-07): successes come from agentic runs (PR-comment / cloud-agent), never gate sweeps. Holds 06-28 (all 5).
- inverse_gate_count_to_conclusiveness: Copilot-assigned ⇒ never orphaned (~38th healthy day).
- gate_sweep_zero_duration: snapshots routinely catch 45-50 action_required 0-duration runs.
- recovery_regression_oscillation: saw-tooth persists; spikes (38-40%) between troughs (0-10%).
- conversation_log_fetch_failure: 35th+ day (longest unresolved risk; behavioral/loop/context analysis unavailable — metrics are CI/infra metadata only).
- gate_footprint_refire_signature (06-20): refire ratio = runs/distinct-workflows distinguishes broad CI from narrow re-fire.
- per_branch_gate_fanout (06-26): each PR-open fires a gate bundle; gate_count = f(PR-open), not branch health. **Refined 07-03 (GBCD):** bundle composition is change-TYPE-adaptive — code-change branches fire full 4/4 core CI gates, lint/doc branches fire 0/4 (lightweight agentic wf only), spec branches fire 2/4 + moderation. "~8-workflow uniform bundle" was an over-generalization from a code-heavy snapshot.
- gate_bundle_composition_divergence (07-03, experimental): fraction of open branches deviating from the full core CI gate set; 62.5% (5/8) diverged today. Distinguishes deterministic CI overhead from change-type-specific triggers.
