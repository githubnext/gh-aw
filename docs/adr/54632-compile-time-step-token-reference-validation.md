# ADR-54632: Compile-Time Validation of Same-Job Step-Output Token References

**Date**: 2026-08-21
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

`safe-outputs.github-token` accepts expressions of the form `${{ steps.<id>.outputs.<name> }}`, which reference the output of a step that runs in the same job. GitHub Actions step outputs are strictly job-scoped: a step declared in job A cannot produce an output consumed by job B. The gh-aw workflow compiler previously did not enforce this constraint — it emitted a lock file containing the unresolvable reference, which then failed `actionlint` and produced an empty token at runtime. A workflow that minted the token via `safe-outputs.steps` (which runs *after* the `safe_outputs` job's checkout) or that omitted the minting step in the `conclusion` job compiled without error, then silently failed at runtime. There was no mechanism to surface the misconfiguration until the generated workflow actually executed.

### Decision

We add a compile-time validation pass (`validateSafeOutputStepTokenReferences` in `pkg/workflow/safe_outputs_step_token_validation.go`) as the final step of `buildJobs`. After all jobs are assembled, the compiler collects every step id referenced by a `steps.<id>.outputs.*` expression in any `github-token` field (global and per-output), then checks each consuming job to ensure that (a) the step with that id exists in the job and (b) it is declared before the first step that consumes the token. If either condition fails, compilation returns an error naming the job and the `pre-steps` frontmatter snippet needed to fix it. The check is scoped to YAML mapping-value positions so that references inside run scripts or prompt text do not trigger false positives.

### Alternatives Considered

#### Alternative 1: Auto-propagate the minting step into every consuming job

The compiler could detect that a job needs the minting step and inject it automatically, mirroring the existing `applyBuiltinJobPreSteps` pattern. This keeps author-facing configuration minimal. It was rejected because it would silently side-effect every consuming job with whatever the minting action does (OIDC token issuance, network calls, permissions) without the author explicitly opting in. It also hides a real configuration gap: if the author deploys the workflow to a context where the minting action is unavailable, the error would surface at runtime rather than at compile time.

#### Alternative 2: Emit a warning or annotation instead of a hard error

The compiler could allow the lock file to be emitted and annotate the affected lines or print a warning to stderr. This was rejected because the runtime consequence is a critical failure — `actionlint` rejects the generated lock file and the token is empty, causing downstream safe-output tool calls to fail. A warning does not prevent the broken lock file from being committed or deployed. A hard compile error ensures the configuration is fixed before any generated artifact is written.

### Consequences

#### Positive
- Authors receive an actionable compile-time error that names the exact job and provides the `pre-steps` snippet required to resolve the misconfiguration, eliminating a class of silent runtime failures.
- All 286 existing repository workflows continue to compile unchanged, confirming the validation is purely additive for correct configurations.

#### Negative
- Configurations that previously compiled (and silently failed at runtime) now fail at compile time, requiring authors to migrate `safe-outputs.steps` token-minting to `pre-steps` under the appropriate job.
- The validation is restricted to YAML mapping-value positions in the rendered job YAML; step references embedded in `run:` scripts or free-form text are not detected as consumers, which is a conservative scope that may miss some edge cases.

#### Neutral
- The new file introduces two index-based helper functions (`jobStepIDDeclarationIndex`, `jobStepOutputConsumptionIndex`) that operate on rendered job YAML strings; these are deliberately scoped to this validation pass rather than integrated into the broader step-ordering infrastructure.
- Documentation in `reference/safe-outputs.md` was updated to show a keyless OIDC minting example covering all three consuming jobs (`agent`, `safe_outputs`, `conclusion`), replacing a simpler single-job example that implied that pattern was sufficient.
