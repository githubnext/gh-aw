# ADR-70429: OIDC Activation Admission and Runtime Credit Allowances

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Unknown (PR #70429, implementing the admission contract introduced in #70286)

---

### Context

The admission service specification (`docs/src/content/docs/specs/admission-service-specification.md`, introduced in #70286) defines a centralized authority consulted before agent execution. The existing daily AI-credit (AIC) guardrail requires preparatory token minting, observation restoration and ledger access. Organizations need centralized admission and runtime credit allowances without per-workflow source changes or forwarding repository API credentials. The runtime must tolerate authority outages, reuse existing agent/detection/evals jobs and proxy enforcement, and fit into compiled workflows and statically computed reusable caller permission envelopes.

### Decision

Generate a separate compiler-owned `admission` job (`pkg/workflow/compiler_admission.go`), gated at job level by the managed `GH_AW_DEFAULT_ADMISSION_URL` Actions variable. This job gains `id-token: write` for the start-time admission check; activation retains permissions required by its independent features. The admission job runs trusted setup and the OIDC client, without author hooks or workflow-specific authentication. Activation depends on admission and explicitly accepts success or skip, but not failure or cancellation. Existing prechecks remain prerequisites.

The completion counterpart also grants `id-token: write` to the compiler-owned `conclusion` job. Its trusted, endpoint-gated step posts the complete generated unified session to the same authority's `/api/conclusion` route, with a fresh token for the same audience and independently reconstructed invocation identity. No token or request identity crosses jobs through outputs. Artifact upload and any upload retry finish before callback execution, so callback failures cannot delay publication. Failed artifact publication does not suppress direct session delivery. Delivery remains best-effort under either admission outage policy; failures warn without changing completed execution results. Activation and agent do not gain OIDC permission for this callback. The companion `AdmissionConclusion.tla` model checks ordering, fresh credentials, stable retries and durable deduplicated acceptance.

The client (`actions/setup/js/check_admission.cjs` and `admission_protocol.cjs`) authenticates only with GitHub Actions OIDC, rejects redirects, strictly validates identity and JSON, and makes at most three attempts within one 60-second deadline using shared `withRetry` cancellation and timeout support. It sends caller-owned W3C `traceparent` and compiler-generated `workflow_hash` as an untrusted cache hint. Valid approval replaces daily-AIC validation and its preparatory steps; denial or malformed approval prevents activation. Unconfigured admission and ignored outages retain daily validation. Reusable callers must grant the callee's admission OIDC permission.

Validated admission results and allowances propagate through activation outputs to the existing AWF budget path. A supplied allowance replaces that component's local limit, including an unlimited setting; missing optional detection/evals allowances preserve independent component policy. Skipped or ignored admission retains existing budgets. Missing required or invalid runtime allowances fail closed. Credentials and request identity stay in admission; budget handoff variables are unset before the agent container starts. Lifecycle logs and collapsed step summaries report outcomes, allowance presence and sources, not credentials, identities, amounts or service diagnostics. The structured unified-session artifact records the validated numeric allowances with admission provenance.

### Alternatives Considered

#### Alternative 1: Run admission in the agent job (or in every downstream job)

Admission could be re-checked immediately before each component runs. This was rejected because it would distribute OIDC credentials and change downstream permissions and eligibility. Keeping authentication and the decision in one isolated admission job allows existing runtime budget inputs to consume validated component allowances without further service calls.

#### Alternative 2: Authenticate with a GitHub App token or forwarded repository credentials

The client could reuse the existing App-token mint pattern (as the daily-AIC guardrail does) or forward `GITHUB_TOKEN` so the authority can query GitHub itself. This was rejected because it hands a reusable credential to an external, separately deployed service. An audience-bound OIDC token proves workflow identity without granting API access and needs no secrets in the repository.

#### Alternative 3: Grant `id-token: write` only when admission is configured

The compiler cannot conditionally generate permissions from runtime Actions variables. Instead, it always generates the isolated job and gates the whole job on the endpoint variable. A skipped job starts no runner and gives activation no additional OIDC capability.

#### Alternative 4: Run admission alongside the daily-AIC guardrail instead of replacing it

Both checks could always run, with either able to block. This was rejected because it duplicates accounting that the central authority is meant to own, and it keeps the cost of the App-token mint, cache restore, and ledger clone on every admitted run. Keeping daily-AIC as the fallback preserves protection when admission is disabled or ignored during an outage.

### Consequences

#### Positive
- Organizations can enforce centralized, cross-repository admission policy by setting Actions variables, with no per-workflow source changes and no recompilation to opt in.
- No reusable credential leaves the runner: the authority receives only an audience-scoped OIDC token, and redirects are refused.
- Admitted runs skip the daily-AIC guardrail and its preparatory steps (App token mint, observation restore and publish, ledger clone), which shortens activation.
- Existing agent, detection and evals jobs and dependencies are reused; only budget inputs and selection change.
- Admission OIDC is no longer shared with activation hooks; compiler-owned admission rejects author customization.
- The bounded retry/deadline support added to the shared `withRetry` helper can be reused by other callers.

#### Negative
- Configured admission adds a runner/job boundary before activation. Its explicit success-or-skip dependency condition must preserve other prerequisite checks.
- Reusable `workflow_call` callers must grant `id-token: write`. Existing caller envelopes (and the hand-authored Windows probe caller) had to change, and third-party callers that do not grant it will fail.
- Admission remains cooperative, not an unbypassable organization spending cap. Supplied allowances configure existing runtime proxy limits, but deployments still need a separately deployed authority and protected backend policy/accounting.
- When the authority is unavailable, activation can wait up to the 60-second deadline before the outage policy applies.
- The shared `withRetry` helper is more complex (timeouts, absolute deadlines, abortable sleeps, injectable clocks), which affects every caller.

#### Neutral
- `workflow_hash` (frontmatter hash + body hash) is sent as an untrusted cache hint only. The authority must not treat it as authorization.
- The response contract has no trace or correlation ID. Tracing relies on the caller-provided W3C `traceparent`.
- Most of the original diff is regenerated lock files, and the TLA+ model (`specs/workflow-security/ActivationAdmission.tla`) and its checker were revised to match the admission contract.
- `gh aw env` defaults gain three new keys (`default_admission_url`, `default_admission_audience`, `default_admission_on_outage`), with validation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
