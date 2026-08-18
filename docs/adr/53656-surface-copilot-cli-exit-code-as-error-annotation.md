# ADR-53656: Surface Copilot CLI Exit Code as a GitHub Actions Error Annotation

**Date**: 2026-08-18
**Status**: Accepted
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

The generated `Execute GitHub Copilot CLI` step embeds a shell `EXIT` trap that already captures and persists the Copilot CLI's exit code to `/tmp/gh-aw/agent_execution_exit_code.txt` for downstream OTLP conclusion spans. However, when the Copilot CLI exits non-zero, the job log gives no visible signal: the agent stream and its post-processing (`rpc-messages.jsonl`, `gateway.md`, `token-usage.jsonl`, AI Credits) complete normally, no `##[error]` line is emitted, and the step appears indistinguishable from a successful run. This masked at least three affected workflow runs and blocked diagnosis of the root cause. The existing trap structure (single EXIT handler, single-quoted body for deferred `$HOME` expansion) is already the shared path for both the direct and AWF/firewall invocation modes.

### Decision

We will extend `buildCopilotSettingsCleanupAndExitCodeTrap` to emit a `::error title=Execute GitHub Copilot CLI::` GitHub Actions workflow command inside the EXIT trap whenever `$gh_aw_exit_code` is non-zero. A shared `copilotExecutionStepName` constant provides the step label used in both the generated `name:` field and the annotation title, preventing drift. Successful runs (exit 0) remain silent. The annotation fires on the shared invocation path and therefore covers all Copilot workflows automatically without per-workflow changes.

### Alternatives Considered

#### Alternative 1: Detect failure in a subsequent step using the persisted exit-code file

A downstream step (e.g., the OTLP conclusion span step) could read `agent_execution_exit_code.txt` and emit the annotation or fail explicitly there. This avoids modifying the EXIT trap.

Not chosen because it introduces a temporal gap: the annotation would appear several steps after the actual failure, making log correlation harder. It also requires each downstream step that cares about the exit code to independently re-read the file.

#### Alternative 2: Propagate the non-zero exit code directly and let GitHub Actions native step failure handle visibility

The script could re-exit with the captured non-zero code so the step itself shows as failed in the Actions UI, providing native visibility.

Not chosen because the step's post-processing (settings cleanup, exit-code file write) must run even on failure. Re-exiting non-zero from the trap body would abort cleanup. The existing design deliberately separates "step appears to succeed" from "exit code is recorded for observability," and this PR keeps that separation intact.

### Consequences

#### Positive
- Non-zero Copilot CLI exits now appear as explicit `::error::` annotations in the job log, making failures immediately visible without consulting the exit-code file.
- The fix is applied on the single shared code path (`buildCopilotSettingsCleanupAndExitCodeTrap`), so all generated Copilot workflow files (direct mode, AWF/firewall mode) receive the annotation automatically — no per-workflow logic is needed.

#### Negative
- Any non-zero exit — including transient or expected ones — will produce a visible error annotation. Operators seeing the annotation still need to correlate with the agent stream to understand the root cause; the annotation conveys the exit code but no higher-level diagnosis.
- The annotation appears inside the EXIT trap body, not as a GitHub Actions step conclusion, so the step's overall status in the Actions UI remains green while an error annotation appears in the log — a potentially confusing combination for users unfamiliar with `::error::` annotations.

#### Neutral
- Regenerated `.lock.yml` workflow files and wasm golden fixtures are the bulk of the diff; the functional change is confined to `pkg/workflow/copilot_engine_execution.go` and its tests.
- The `copilotExecutionStepName` constant was introduced alongside this change, tightening the coupling between step naming and annotation text as a side effect.
