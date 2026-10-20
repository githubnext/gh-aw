# ADR-70429: Activation-Only OIDC Admission Service Runtime

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Unknown (PR #70429, implementing the admission contract introduced in #70286)

---

### Context

The admission service specification (`docs/src/content/docs/specs/admission-service-specification.md`, introduced in #70286) defines a contract for a centralized authority that participating workflows can consult before agent execution. Until now, the only pre-execution spending control in the activation job was the per-workflow daily AI-credit (AIC) guardrail, which runs locally and requires its own preparatory steps (App token mint, scan-observation restore, repo-memory ledger clone). Organizations want one authority to approve or deny runs across many repositories without changing each workflow's source, without forwarding repository credentials to a third party, and without altering downstream agent, detection, or eval jobs. The runtime must also tolerate an unreachable authority without hanging activation, and it has to fit into 344 compiled lock files and reusable `workflow_call` callers whose permission envelopes are computed statically by the compiler.

### Decision

We will add a single compiler-emitted `Check workflow admission` step (`pkg/workflow/compiler_admission.go`) to the **activation job only**. It is enabled by managed Actions variables (`GH_AW_DEFAULT_ADMISSION_URL`, audience, and outage policy, settable through `gh aw env` defaults) and gated with an `if:` on the endpoint variable, so no OIDC token is requested when admission is unconfigured. The client (`actions/setup/js/check_admission.cjs` and `admission_protocol.cjs`) authenticates **only** with a GitHub Actions OIDC token, rejects redirects, strictly validates identity and JSON, makes at most three attempts within one 60-second deadline (implemented by extending the shared `withRetry` in `error_recovery.cjs` with per-attempt timeouts, an absolute deadline, and abortable sleeps), and sends a caller-owned W3C `traceparent` plus a `workflow_hash` built from the compiled frontmatter/body hashes as an untrusted cache hint. A validated approval (`steps.admission.outputs.result == 'admitted'`) **replaces** the daily-AIC check and its preparatory steps; denial or malformed approval blocks activation; disabled admission or an outage under the "ignore" policy falls back to the existing daily-AIC validation. To support this, the activation job always receives job-scoped `id-token: write`, and reusable caller permission envelopes are widened accordingly.

### Alternatives Considered

#### Alternative 1: Run admission in the agent job (or in every downstream job)

Admission could be checked immediately before the agent runs, or re-checked in detection/eval jobs, giving the authority a later and more precise view. It was rejected because it would change downstream job structure, permissions, and dependencies across all 344 lock files, and because returned threat-detection/eval allowances are intended as activation-only metadata rather than as runtime reconfiguration of inference limits. Keeping the decision in activation leaves all generated agent jobs unchanged.

#### Alternative 2: Authenticate with a GitHub App token or forwarded repository credentials

The client could reuse the existing App-token mint pattern (as the daily-AIC guardrail does) or forward `GITHUB_TOKEN` so the authority can query GitHub itself. This was rejected because it hands a reusable credential to an external, separately deployed service. An audience-bound OIDC token proves workflow identity without granting API access and needs no secrets in the repository.

#### Alternative 3: Grant `id-token: write` only when admission is configured

The compiler could emit `id-token: write` conditionally (as it previously did only for OTLP GitHub OIDC auth). This was rejected because admission is enabled at run time through Actions variables that the compiler cannot see, so a compile-time condition would either miss enabled repositories or require recompilation whenever an organization opts in. The trade-off is that the permission is present even when the step is skipped.

#### Alternative 4: Run admission alongside the daily-AIC guardrail instead of replacing it

Both checks could always run, with either able to block. This was rejected because it duplicates accounting that the central authority is meant to own, and it keeps the cost of the App-token mint, cache restore, and ledger clone on every admitted run. Keeping daily-AIC as the fallback preserves protection when admission is disabled or ignored during an outage.

### Consequences

#### Positive
- Organizations can enforce centralized, cross-repository admission policy by setting Actions variables, with no per-workflow source changes and no recompilation to opt in.
- No reusable credential leaves the runner: the authority receives only an audience-scoped OIDC token, and redirects are refused.
- Admitted runs skip the daily-AIC guardrail and its preparatory steps (App token mint, observation restore and publish, ledger clone), which shortens activation.
- Downstream agent, detection, and eval jobs and their dependencies are unchanged, which limits the blast radius of the change.
- The bounded retry/deadline support added to the shared `withRetry` helper can be reused by other callers.

#### Negative
- Every activation job now carries `id-token: write`, even when admission is disabled. The step `if:` is not a permission sandbox, so this widens the activation job's capability surface.
- Reusable `workflow_call` callers must grant `id-token: write`. Existing caller envelopes (and the hand-authored Windows probe caller) had to change, and third-party callers that do not grant it will fail.
- Admission is cooperative, not a hard spending cap. Its effectiveness depends on a separately deployed authority and policy/accounting backend that this repository does not provide, and returned allowances do not reconfigure inference limits.
- When the authority is unavailable, activation can wait up to the 60-second deadline before the outage policy applies.
- The shared `withRetry` helper is more complex (timeouts, absolute deadlines, abortable sleeps, injectable clocks), which affects every caller.

#### Neutral
- `workflow_hash` (frontmatter hash + body hash) is sent as an untrusted cache hint only. The authority must not treat it as authorization.
- The response contract has no trace or correlation ID. Tracing relies on the caller-provided W3C `traceparent`.
- Most of the diff is regenerated lock files, and the TLA+ model (`specs/workflow-security/ActivationAdmission.tla`) and its checker were revised to match the new contract.
- `gh aw env` defaults gain three new keys (`default_admission_url`, `default_admission_audience`, `default_admission_on_outage`), with validation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
