# ADR-70875: Require Successful Redaction for All Agent Artifact Uploads

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The compiled agent job uploads two artifacts after the engine runs: the unified `agent` artifact and the `agent-output-fallback` artifact. Before an upload runs, the job runs the built-in secret redactor (`redact_secrets`) and any custom `secret-masking` steps. Before this PR, only some uploads had to wait for redaction to succeed. The unified upload required it only when there were custom safe-job artifact paths, when the work queue was enabled, or when the paths included the AWF resolved config file. The fallback upload required it only when the agent firewall (AWF) was enabled. In every other case, both uploads ran under a plain `if: always()`. That meant a workflow without AWF could upload agent artifacts even when redaction had failed, been skipped, or been cancelled. For example, run 39025245757 ran out of JavaScript heap during redaction, and run 38987303689 hit read-permission failures. A custom hook with `continue-on-error: true` could also hide a masking failure.

### Decision

We will gate both the `agent` and `agent-output-fallback` uploads on the same rule every time: `always() && secretRedactionSuccessCondition(data)`. This rule requires `steps.redact_secrets.outcome == 'success'` and a `success` outcome for every configured custom secret-masking step. We will remove the `requireSuccessfulRedaction` parameter from `generateUnifiedArtifactUpload`, the special case for the AWF resolved config path, and the `isFirewallEnabled` branch in `generateAgentOutputFallbackUpload`. The main reason is safety by default: no artifact leaves the runner without successful redaction, whatever the sandbox, artifact paths, or features are. We will check `outcome` rather than `conclusion`, so a custom hook marked `continue-on-error` still blocks uploads when it fails. An engine failure on its own still allows diagnostic uploads, as long as redaction succeeded.

### Alternatives Considered

#### Alternative 1: Keep conditional gating and add more trigger cases

We could keep the existing approach, where gating turns on when AWF is enabled, a sensitive path is present, or safe-job or work-queue artifacts exist, and then add the cases that are missing today. This would keep diagnostic uploads available in more failure modes for low-risk configurations. We rejected it because every new artifact path or feature would need someone to remember to add a gate. The gap this PR fixes is exactly that kind of omission, in the no-AWF path.

#### Alternative 2: Upload unredacted artifacts to a restricted location, or drop only the unredacted files, when redaction fails

We could keep uploading on redaction failure but quarantine or strip the files that might hold secrets. This would keep more debugging data available. We rejected it because it adds complexity, and because a partly failed redaction pass gives no reliable signal about which files are safe. Blocking the whole upload is simpler and fails closed.

### Consequences

#### Positive
- Every workflow configuration fails closed: no `agent` or `agent-output-fallback` upload happens unless built-in and custom redaction both succeed.
- The upload code is simpler: there is one shared condition, and path- and firewall-specific branches are gone.
- Custom masking hooks with `continue-on-error: true` can no longer hide a failure from the upload gate.

#### Negative
- When redaction fails (for example, from heap exhaustion or permission errors), no-AWF workflows lose the diagnostic artifacts they used to get, which makes those failures harder to debug.
- Upload availability now depends on how reliable the redactor is. The PR explicitly does not fix the redactor's underlying memory and ownership problems.

#### Neutral
- An engine failure alone still allows uploads after successful redaction, so normal failure diagnostics are unchanged.
- The regression matrix in `redaction_upload_gating_test.go` now covers both AWF and no-AWF configurations for every custom-hook outcome, and checks that exactly one tracked redaction step runs before the uploads.
- This builds on the detection-side policy in [ADR-69520](69520-gate-detection-publication-on-host-redaction.md), which applies the same fail-closed approach to agent-job uploads.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
