# ADR-70429: OIDC Activation Admission and Runtime Credit Allowances

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Unknown (PR #70429, implementing the admission contract introduced in #70286)

---

### Context

The admission service specification (`docs/src/content/docs/specs/admission-service-specification.md`, introduced in #70286) defines a centralized authority consulted before agent execution. The existing daily AI-credit (AIC) guardrail requires preparatory token minting, observation restoration and ledger access. Organizations need centralized admission and runtime credit allowances without per-workflow source changes or forwarding repository API credentials. The runtime must tolerate authority outages, reuse existing agent/detection/evals jobs and proxy enforcement, and fit into compiled workflows and statically computed reusable caller permission envelopes.

### Decision

We will add a compiler-owned `activation_admission` job with a `Check workflow admission` step (`pkg/workflow/compiler_admission.go`). Managed Actions variables (`GH_AW_DEFAULT_ADMISSION_URL`, audience, and outage policy, settable through `gh aw env` defaults) enable the job at runtime. Its job-level `if:` skips both OIDC acquisition and admission when the endpoint is unset. Only this job receives admission's job-scoped `id-token: write`; activation retains its existing required permissions, and reusable caller permission envelopes include the permission needed by the callee. The client (`actions/setup/js/check_admission.cjs` and `admission_protocol.cjs`) authenticates **only** with a GitHub Actions OIDC token, rejects redirects, strictly validates identity and JSON, makes at most three attempts within one 60-second deadline (implemented by extending the shared `withRetry` in `error_recovery.cjs` with per-attempt timeouts, an absolute deadline, and abortable sleeps), and sends a caller-owned W3C `traceparent` plus a `workflow_hash` built from the compiled frontmatter/body hashes as an untrusted cache hint. The admission job's validated output (`needs.activation_admission.outputs.result == 'admitted'`) **replaces** the daily-AIC check and its preparatory steps in activation; denial or malformed approval blocks activation; disabled admission or an outage under the "ignore" policy falls back to the existing daily-AIC validation.

Validated admission results and allowances propagate from the admission job through activation outputs to the existing AWF budget path. A supplied allowance replaces that component's local limit, including an unlimited setting; missing optional detection/evals allowances preserve independent component policy. Skipped or ignored admission retains existing budgets. Missing required or invalid runtime allowances fail closed. Credentials and request identity stay in the isolated admission job; budget handoff variables are unset before the agent container starts. Lifecycle logs and collapsed step summaries report outcomes, allowance presence and sources, not credentials, identities, amounts or service diagnostics.

### Alternatives Considered

#### Alternative 1: Run admission in the agent job (or in every downstream job)

Admission could be re-checked immediately before each component runs. This was rejected because it would distribute OIDC credentials and change downstream permissions and eligibility. A single admission-only job performs the request once; activation carries only its validated outputs to existing runtime budget inputs.

#### Alternative 2: Authenticate with a GitHub App token or forwarded repository credentials

The client could reuse the existing App-token mint pattern (as the daily-AIC guardrail does) or forward `GITHUB_TOKEN` so the authority can query GitHub itself. This was rejected because it hands a reusable credential to an external, separately deployed service. An audience-bound OIDC token proves workflow identity without granting API access and needs no secrets in the repository.

#### Alternative 3: Grant `id-token: write` only when admission is configured

The compiler could emit `id-token: write` conditionally on a runtime variable. GitHub Actions permissions are static per job and do not accept runtime expressions. Instead, the compiler creates an admission-only job with `id-token: write` and a runtime endpoint condition; when admission is disabled, that job is skipped and the regular activation job has no admission OIDC permission.

#### Alternative 4: Run admission alongside the daily-AIC guardrail instead of replacing it

Both checks could always run, with either able to block. This was rejected because it duplicates accounting that the central authority is meant to own, and it keeps the cost of the App-token mint, cache restore, and ledger clone on every admitted run. Keeping daily-AIC as the fallback preserves protection when admission is disabled or ignored during an outage.

### Consequences

#### Positive
- Organizations can enforce centralized, cross-repository admission policy by setting Actions variables, with no per-workflow source changes and no recompilation to opt in.
- No reusable credential leaves the runner: the authority receives only an audience-scoped OIDC token, and redirects are refused.
- Admitted runs skip the daily-AIC guardrail and its preparatory steps (App token mint, observation restore and publish, ledger clone), which shortens activation.
- Existing agent, detection and evals jobs are reused; one conditional admission job gates activation, while only budget inputs and selection change downstream.
- The bounded retry/deadline support added to the shared `withRetry` helper can be reused by other callers.

#### Negative
- The compiler emits one additional job per workflow. It is conditional on the managed endpoint and contains only compiler-owned admission setup; when disabled, it does not run.
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
