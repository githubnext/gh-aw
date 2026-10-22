# ADR-71043: Attribute Agent Proxy Requests via x-awf-caller Header Only at Per-Process Boundaries

**Date**: 2026-10-22
**Status**: Proposed
**Deciders**: gh-aw maintainers

---

### Context

The AWF API proxy records one accounting entry per LLM request (`token-usage.jsonl`). AWF (github/gh-aw-firewall#10222) can attribute each record to a caller when the client sends an `x-awf-caller` header (`main` or `subagent/<name>`), but gh-aw never sent it, and the unified-session collector dropped AWF's `agent_name` / `caller_source` fields. As a result, `gh aw audit` / `logs` had to attribute sub-agent usage by heuristically correlating tokens, models and timestamps. That works poorly when the main agent and sub-agents use the same model or overlap in time. The engines differ in where headers can be set. Pi runs each sub-agent as a separate process. Copilot SDK, Copilot CLI, Claude Code and Codex set headers per provider or per process, and that scope covers sub-agent requests too.

### Decision

We will send `x-awf-caller` only from harnesses that have a real per-caller process boundary, which today means Pi only. Pi's `before_provider_headers` hook sends `main` from the parent process and `subagent/<name>` from each sub-agent process (the name is passed through `GH_AW_PI_SUBAGENT_NAME`). The header is registered only when the AWF api-proxy is present (`AWF_REFLECT_ENABLED=1`), so caller names never reach a provider directly (for example with `sandbox.agent: false`). Invalid values are omitted, never truncated. The collector will keep AWF's caller fields (`agentName`, `callerSource`) in the unified session, bumped to spec 2.4.0 with requirement T-UAS-083. Audit and logs will treat explicit caller records as authoritative: they are credited first and reserved. Token/model/time correlation then runs only over the unlabelled records and only for sub-agents that have no explicit records. AWF-owned records (`routing_classification` or `callerSource: awf`) are never counted as agent requests. The main driver is correctness: an attribution label must never be wrong, even if that means some requests stay unattributed.

### Alternatives Considered

#### Alternative 1: Send a provider/process-wide `main` header for all engines

Every engine can set a static header on its provider or process, so this would label traffic for all engines immediately. It was rejected because sub-agents in Copilot SDK/CLI, Claude Code and Codex share that provider or process. Their requests would be mislabelled as `main`, which is worse than sending no attribution. `TestUnsupportedEngineDoesNotMislabelSubagents` guards against this.

#### Alternative 2: Give each declared Copilot SDK agent its own provider entry

A separate provider per agent would give the SDK a per-caller header boundary. It was rejected because it changes the qualified model IDs that model resolution and audit depend on, and those effects reach beyond attribution.

#### Alternative 3: Keep correlation-only attribution (status quo)

This needs no harness or schema changes. It was rejected as the primary mechanism because AWF can now provide explicit caller identity and correlation is ambiguous. Correlation is kept as a fallback for older records and headerless engines, so existing behaviour stays the same.

### Consequences

#### Positive
- Pi main and sub-agent usage is attributed exactly from AWF records instead of by heuristic matching.
- The scheme is backward compatible. Records without caller fields (older AWF versions or unsupported engines) still use the existing correlation path, and absent fields are never synthesized.
- AWF-owned classifier/router traffic is clearly separated from agent requests, and totals are unchanged.
- The unified session contract now carries caller identity, so other consumers can use it.

#### Negative
- Attribution is uneven across engines. Copilot SDK/CLI, Claude Code and Codex requests stay unattributed until those harnesses get a per-caller header mechanism.
- Audit has two attribution paths (explicit first, correlation as fallback), which makes the precedence logic in `pkg/cli/token_usage_subagent.go` more complex.
- Records naming an unknown sub-agent only produce a warning and are not credited, so some usage can be left out of per-agent rows.
- The gh-aw validation regex and 128-character limit duplicate AWF's own rules and must be kept in sync with AWF.

#### Neutral
- The unified session specification is bumped to 2.4.0 with a per-engine support table and a compliance-matrix row.
- Pi runs without the AWF api-proxy send no `x-awf-caller` header.
- Pi sub-agent names that do not match `^(?:main|subagent/[A-Za-z0-9_.:/-]+)$` lead to the header being omitted rather than to an error.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Proposed to Accepted.*
