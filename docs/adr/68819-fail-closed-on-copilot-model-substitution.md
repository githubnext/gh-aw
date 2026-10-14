# ADR-68819: Fail closed when the Copilot CLI substitutes the configured model

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (PR #68819, authored by @pelikhan)

---

### Context

gh-aw lets a workflow pin an exact model (for example `copilot/gemini-...`) for the Copilot engine. Two paths could silently diverge from that configuration. First, AWF alias resolution (`actions/setup/js/resolve_model_alias.cjs`) conflated two very different failures: a successfully fetched catalog that contains no matching model, and a catalog fetch that failed outright — both surfaced as a generic engine outage. Second, on the native direct-CLI path the Copilot CLI can start with a different model than the one requested (Gemini silently downgraded to Claude) and the run would still succeed, producing results attributed to a model the author never selected. The related upstream limitation (#66894) is that Copilot/CAPI may not offer a policy-enabled model to the `agentic-workflows` integration at all, even when the standalone CLI and the bundled static catalog list it. Log and audit output (`pkg/cli/logs_models.go`) also had only one notion of "the model", so substitution was invisible after the fact.

### Decision

We will fail closed on model substitution rather than continue on a substituted model. Concretely: a new `actions/setup/js/copilot_model_selection.cjs` module tails only session events appended by the current native main-agent attempt, compares the observed model identifier against the resolved one, and terminates the CLI without retrying when they differ (including the quick-exit case); alias resolution now distinguishes "configured model unavailable" from "catalog retrieval failed" and reports an unavailable-model *configuration* error; `aw_info.json` records configured, resolved, and observed effective models separately so logs and audit attribution stay accurate; and the compiler (`pkg/workflow/configured_model_validation.go`) emits a compile-time warning — not an error — when Gemini is configured against Copilot's GitHub integration. The primary driver is correctness of attribution: a run that did not use the configured model must not be reported as a successful run of that model.

### Alternatives Considered

#### Alternative 1: Warn on substitution and let the run continue

The observer could log a warning and allow the CLI to proceed on the fallback model, keeping runs green while the upstream integration gap is resolved. It was rejected because the failure mode this PR addresses is precisely a *silent* success: a tolerated substitution leaves cost, capability, and output attributed to a model the workflow never selected, and gives the author no signal that their pin is ineffective. Failing closed was chosen even though it converts previously "passing" runs into explicit failures.

#### Alternative 2: Reject unavailable Gemini/Copilot model configurations at compile time

Since `warnCopilotIntegrationModelAvailability` already detects a Gemini model configured for Copilot's GitHub integration, the compiler could hard-error and never generate the lock file. It was rejected because the bundled static pricing/model catalog does not describe runtime integration access — it can be stale in both directions — and a hard error would break legitimate BYOK configurations (explicit `CopilotProviderBaseURL`) and any model that Copilot later enables. The compile-time check therefore stays a warning and the authoritative decision is deferred to the runtime observer. A third option, changing the integration ID to one that does offer Gemini, was rejected outright as a policy-bypass workaround.

### Consequences

#### Positive
- A configured model is now either honoured or the run fails loudly; no result is ever attributed to a silently substituted model.
- Operators get a distinct, actionable "configured model unavailable" configuration error instead of a generic engine-outage message, while the bounded catalog refresh is retained.
- `aw_info.json` carries configured, resolved, and observed effective model metadata, so `gh aw logs` and audit output remain accurate after the fact.
- A compile-time warning surfaces the likely Gemini/`agentic-workflows` integration gap before a run is ever dispatched.

#### Negative
- Workflows that previously completed on a substituted model will now fail, and because substitution is terminal and not retried the failure is immediate and unrecoverable within the run.
- Detection depends on parsing newly appended Copilot CLI session events from a set of candidate `session-state` directories; an upstream change to that event format or layout silently degrades the guard to a no-op (false negatives).
- Adds a second notion of "model" (resolved vs observed) that every consumer of `aw_info.json`, logs, and audit output must now understand.

#### Neutral
- The observer deliberately ignores cached sessions, subagents, SDK mode, and the threat-detection phase, so the guard applies only to the native main-agent attempt.
- Lock-file model metadata remains explicitly compile-time configuration and is not rewritten with the observed model.
- The upstream Copilot/CAPI limitation (no Gemini for the `agentic-workflows` integration) and the separately reported Gemini BYOK `refusal: null` replay failure are unchanged by this decision; authorized AWF fallback precedence is preserved.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
