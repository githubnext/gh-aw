# ADR-70875: Require Successful Redaction for All Agent Artifact Uploads

**Date**: 2026-10-21
**Status**: Proposed (implemented in PR #70875; maintainer acceptance pending)
**Deciders**: pelikhan (PR author); maintainer acceptance pending

---

### Context

The compiled agent job can upload a unified `agent` artifact and, when applicable, a smaller `agent-output-fallback` artifact used by downstream safe-output processing. Built-in secret redaction and configured custom `secret-masking` steps precede these uploads, but their presence alone does not prevent an upload after redaction fails.

Before this PR, the unified upload required successful redaction only for custom safe-job artifact paths, work-queue workflows, or paths containing the AWF resolved config. The fallback upload required it only when the agent firewall (AWF) was enabled. Otherwise, the upload used plain `if: always()`, allowing publication even if redaction failed, was skipped, or was cancelled. Reported heap exhaustion and file-permission failures motivated closing this gap; this decision does not diagnose or repair those runtime failures.

### Decision

Whenever either upload is generated, gate it with `always() && ` followed by `secretRedactionSuccessCondition(data)`. The generated condition requires `steps.redact_secrets.outcome == 'success'` and a `success` outcome for every configured custom secret-masking step, independently of AWF, artifact paths, or work-queue configuration.

Use `outcome`, not `conclusion`, so a failed custom hook still blocks upload when it has `continue-on-error: true`. Preserve the existing explicit custom-hook IDs and stable, non-conflicting IDs assigned to unnamed hooks. Retain the existing single built-in step with ID `redact_secrets` before the uploads so their conditions reference the actual redaction step; the historical missing-ID issue was already corrected before this PR.

Remove the unified upload's `requireSuccessfulRedaction` parameter and the path- and firewall-dependent choice of upload condition. AWF-specific artifact path collection remains unchanged. This makes successful redaction a prerequisite for these two uploads rather than a feature-dependent exception. An engine failure alone still permits diagnostic uploads if redaction succeeds.

### Alternatives Considered

#### Alternative 1: Keep conditional gating and add more trigger cases

Retaining feature- or path-specific gates would preserve more diagnostic uploads when redaction fails in configurations treated as lower risk. It would also require each new artifact path or feature to opt into the prerequisite. The unconditional rule avoids repeating the no-AWF gap and does not assume that other agent output is safe without redaction.

#### Alternative 2: Require only built-in redaction success

Checking only the built-in step would simplify the condition and allow uploads when an optional custom hook is skipped or fails. However, custom hooks may provide masking beyond the built-in redactor. Requiring every configured hook's actual successful outcome preserves that contract rather than treating `continue-on-error` as permission to publish.

### Consequences

#### Positive
- Both uploads fail closed across AWF and no-AWF configurations unless built-in redaction and every configured custom hook succeed.
- The upload generators share one condition without path- or firewall-specific gating branches.
- Custom masking hooks with `continue-on-error: true` can no longer hide a failure from the upload gate.

#### Negative
- When redaction fails (for example, from heap exhaustion or permission errors), no-AWF workflows lose the diagnostic artifacts they used to get, which makes those failures harder to debug.
- A deliberately skipped custom hook also blocks these uploads. Redaction heap exhaustion and EACCES failures remain unresolved by this change.
- Losing both artifacts can also leave downstream safe-output processing without its normal artifact inputs.

#### Neutral
- An engine failure alone still allows uploads after successful redaction, so normal failure diagnostics are unchanged.
- `redaction_upload_gating_test.go` checks compiled conditions and ordering for AWF and no-AWF configurations, including failed, skipped, cancelled, and missing outcome values. These are compiler/expression checks, not live-run verification or proof that successful redaction removes every secret.
- This extends the fail-closed publication principle described for the separate detection job in [ADR-69520](69520-gate-detection-publication-on-host-redaction.md) to these agent-job uploads; it does not change detection-job behavior.

---

*Proposed for maintainer review with this pull request.*
