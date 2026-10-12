# ADR-67837: Assert Threat Detection Stays Unrouted in the Routed Copilot Smoke

**Date**: 2026-10-11
**Status**: Draft
**Proposed deciders**: copilot-swe-agent (PR author), gh-aw maintainers

---

### Context

The routed smoke workflows set `features: gh-aw-detection: false`, so CI never exercised the default (external) threat-detection engine while model routing was active. Nothing checked at runtime that the detection job stays *unrouted*, and that invariant has regressed before (issue #66224, tracked by #67789). Detection is a security gate: if model routing silently applies to it, the classifier can run on an unintended model or endpoint without any signal in CI. The constraint is that such a check must be deterministic and must read runner- and proxy-written evidence rather than the agent's own reply, matching the existing `R*`/`E*` assertion style in `actions/setup/js/smoke_model_routing_assertions.cjs`.

### Decision

We will enable the default threat-detection engine in `smoke-copilot-routed` and add a detection-job post-step that asserts three evidence-based checks (`D1`–`D3`) against artifacts under `/tmp/gh-aw/threat-detection/`. `safe-outputs.threat-detection.model` is pinned to `claude-haiku-5.5`, a model deliberately outside the smoke's `allowed-models` set, so the compiled step uses `COPILOT_MODEL: claude-haiku-5.5` instead of the `detection` alias — the alias can resolve to an allowed model, which would make the `D2` assertion ambiguous. `D1` verifies the detection step succeeded and `detection_result.json` has the expected schema; `D2` verifies every served model in the detection api-proxy `token-usage.jsonl` equals the pinned, non-allowed model; `D3` verifies no `routing_classification` record, no routing `selection` event, and no `modelRouting` block in the resolved AWF config. A failing `D` check turns the detection job red and skips safe outputs. The audit CLI gains a separate `detection_model_usage` field so detection token usage is reported distinctly from `firewall_token_usage` and the routed agent model.

### Alternatives Considered

These alternatives are inferred from the diff and PR description; they do not record maintainer approval or rejection.

#### Alternative 1: Keep `gh-aw-detection: false` and add a compile-time or unit-level assertion

Assert in Go/vitest that the compiler never emits routing configuration into the detection job. This is cheaper and runs on every PR, but it only checks generated YAML, not what the proxy actually served at runtime. The regression class in #66224 is a runtime one (model resolution and routing applied to a live detection run), so a compile-time check would not have caught it.

#### Alternative 2: Leave `COPILOT_MODEL` as the `detection` alias instead of pinning a concrete model

This matches production configuration most closely and requires no frontmatter change. It was rejected because the `detection` alias (`small` → `mini` → `haiku`/`gpt-5-mini`/…) can resolve to a model that is also in the smoke's `allowed-models` list, so `D2` could pass even if routing had leaked into the detection job. Pinning a model outside `allowed-models` makes the assertion unambiguous.

#### Alternative 3: Add the detection assertions to all three routed smokes at once

Covering `smoke-pi-routed` and `smoke-copilot-sdk-routed` too would give broader protection. It was deferred to keep the blast radius of the first runtime detection check to a single workflow, since the detection job wiring (binary install, artifact paths, post-step pinning) is new and unproven in CI.

### Consequences

#### Positive
- Routing leakage into the threat-detection job is now caught at runtime by CI rather than in production, with a failing check printing the observed model/endpoint in the familiar `PASS/FAIL <id>` format.
- The routed smoke now exercises the default external threat-detection engine end to end (binary install, `threat-detect` execution, `detection_result.json` parsing), a path previously never covered by CI.
- `detection_model_usage` in the audit report makes detection-side model spend visible and separable from agent and firewall usage.

#### Negative
- The pinned `claude-haiku-5.5` model is a hardcoded value duplicated in the workflow frontmatter and the post-step arguments; it must be updated in both places when the model is retired, or `D2` will fail for an unrelated reason.
- The `uses:` reference in the threat-detection `post-steps` has to be pinned by commit hash in the workflow source because the compiler does not pin actions in threat-detection steps (it does for top-level `post-steps`). This is a known compiler gap left unaddressed, and the manifest now records an unversioned `actions/github-script` entry.
- Enabling the detection engine makes the smoke longer and adds new failure modes (binary install, artifact availability) that can turn the job red for reasons unrelated to routing.

#### Neutral
- The assertion module `smoke_model_routing_assertions.cjs` grows a second entry point (`detectionMain`) alongside the agent-job entry point, reusing `servedModel`, `normalizeEndpoint`, `CLASSIFIER_PURPOSE`, `TOKEN_USAGE_PATHS` and `readJsonl`; the existing `E1`, `E2` and `R1`–`R4` checks are unchanged.
- Vitest fixtures are copies of real records (runs 38173405739 and 38174054652) with request IDs and timestamps redacted, so fixture refresh is a manual step tied to specific runs.
- Only `smoke-copilot-routed` changes; the SDK and pi routed smokes keep `gh-aw-detection: false`, so coverage is intentionally asymmetric until the pattern is proven.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
