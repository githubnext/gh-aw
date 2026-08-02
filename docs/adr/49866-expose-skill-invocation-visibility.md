# ADR-49866: Expose Skill Invocation Visibility via Dual-Source Extraction

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan (via PR #49866)

---

### Context

APM-restored skills are installed into agent workflow runs, but no first-class signal existed to confirm whether a skill was actually invoked by the agent at runtime — only that it was present. This observability gap prevents operators and experiment pipelines from distinguishing "skill was available" from "skill was exercised," making it impossible to assess skill adoption or correlate skill usage with outcome metrics. The audit pipeline already collects similar runtime signals (MCP failures, missing tools, noops), so skill activation records fit naturally into the same data flow.

### Decision

We will add a `SkillActivation` data type and a two-phase extraction function (`extractSkillActivationsFromRun`) that first reads explicit `skill_invocation` items emitted by agents to `agent_output.json` (Phase 1), then scans raw agent log files for engine-specific invocation patterns (Phase 2). Results from both phases are merged, with `agent_output` entries taking precedence on name collision. `SkillActivation` records are wired into `ProcessedRun`, `RunSummary`, `DownloadResult`, `AuditData`, and `auditAnalysisResults`, and rendered in both `gh aw audit` console output and `--json` mode.

### Alternatives Considered

#### Alternative 1: Explicit `agent_output.json` only (no log-parse fallback)

Agents opt in to declaring skill invocations by emitting `{"type": "skill_invocation", ...}` items to the safe-output mechanism. Only those explicit records are surfaced.

**Why considered**: Explicit declarations are reliable, unambiguous, and already supported by the safe-output contract.

**Why not chosen**: Many agents — especially legacy and third-party engines — do not emit `skill_invocation` items, so coverage would be sparse. The log-parse fallback materially improves recall for the current population of real-world agent runs without requiring agent-side changes.

#### Alternative 2: Log-file pattern scanning only (no `agent_output` path)

Walk agent log files and detect skill invocations via regex patterns; return all matches as `SkillActivation` records.

**Why considered**: Log files are already available in `runOutputDir`; no safe-output contract changes are needed.

**Why not chosen**: Log parsing is inherently heuristic and cannot capture rich per-invocation metadata (timestamp, experiment provenance) that agents can provide explicitly. `agent_output` records are higher fidelity and should be authoritative when available. Using only log parsing sacrifices that fidelity for all agents that already emit explicit signals.

#### Alternative 3: No skill activation tracking

Leave observability at the installation level; accept that "skill was available" is the only signal.

**Why considered**: Minimal implementation cost; no new data types or extraction logic needed.

**Why not chosen**: This directly blocks the observability goal stated in the linked issue (#39369). Without runtime invocation records, the APM pipeline cannot determine whether skills function as intended or whether usage correlates with experiment outcomes.

### Consequences

#### Positive
- Operators can confirm at a per-run level whether a skill was actually invoked, not just installed.
- Dual-source extraction maximises recall: modern agents providing explicit signals get authoritative records; legacy agents relying only on log output are still covered.
- Provenance fields (run ID, workflow name, experiment/variant, timestamp) are carried through to all output formats, enabling downstream metric correlation.
- The new `SkillActivations` field in `run_summary.json` makes historical data queryable without re-processing raw logs.

#### Negative
- Log parsing is regex-based and tied to three specific log patterns; invocations logged in novel formats will be missed until new patterns are added.
- Phase 2 walks the entire run directory on every processing call; runs with many large log files may incur measurable I/O overhead.
- The `SkillActivation.Status` field is currently always `"invoked"`, limiting expressiveness if future needs require distinguishing failure modes.

#### Neutral
- The `SkillActivations` field is added to `RunSummary` with `omitempty`, so existing cached `run_summary.json` files that predate this change remain valid — the field will simply be absent on cache restore.
- Consumers of `gh aw audit --json` will see a new `skill_activations` key; downstream scripts that do strict schema validation may need updating.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
