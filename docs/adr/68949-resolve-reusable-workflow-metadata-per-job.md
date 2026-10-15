# ADR-68949: Resolve Reusable-Workflow Metadata Inside Each Consuming Job

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: @pelikhan (PR author). Maintainer review pending; this record does not imply formal architecture approval.

---

### Context

Compiled workflows that expose a `workflow_call` trigger need two pieces of reusable-workflow metadata at runtime: an artifact-name prefix (so concurrent invocations of the same `.lock.yml` in one run do not clash) and the host/platform repository coordinates used for sparse `.github`/`.agents` checkout, GitHub App token scoping, and dispatch configuration. Until now both values were computed once in the activation (or agent) job and transported to downstream jobs through job outputs — `needs.activation.outputs.artifact_prefix`, `needs.agent.outputs.artifact_prefix`, and `needs.activation.outputs.target_repo`. GitHub Actions omits a job output whose value matches a masked secret, so a prefix or repository string that collides with masking is delivered to downstream jobs as an empty string, breaking artifact naming and checkout targeting (issue #38345). In addition, the artifact-prefix step depended on `${RUNNER_TEMP}/gh-aw/actions/compute_artifact_prefix.sh`, which only exists in jobs that run the Setup Scripts step, so jobs without setup could not compute the value locally.

### Decision

We will compute reusable-workflow metadata locally in every job that needs it, instead of propagating it across jobs through outputs. `artifactPrefixExprForDownstreamJob` and `artifactPrefixExprForAgentDownstreamJob` now return the same local step expression used by the activation job (`pkg/workflow/compiler_workflow_call.go`), and `CheckoutManager` consumes `steps.resolve-host-repo.outputs.target_repo` in every job rather than `needs.activation.outputs.target_repo` (`pkg/workflow/checkout_manager.go`). To make this possible in jobs that do not run Setup Scripts, the canonical `actions/setup/sh/compute_artifact_prefix.sh` is embedded into the compiler (`actions/setup/artifact_prefix.go`) and inlined into the generated `Compute artifact prefix` step, preserving the existing hash-based naming semantics. Host-repository resolution distinguishes inlined from non-inlined imports (`job.workflow_*` versus `github.*` context in `compiler_activation_job.go`), and `resolve_host_repo.cjs` now fails explicitly on missing or malformed target-repository metadata instead of silently falling back to the caller repository. The primary driver is correctness under secret masking; keeping a single embedded script is what preserves naming compatibility.

### Alternatives Considered

#### Alternative 1: Keep job outputs and obfuscate/encode the transported values

Continue transporting the prefix and target repository through `needs.*.outputs` but encode them (for example base64 or a sentinel wrapper) so they never match a masked secret, decoding in each consumer. This keeps a single computation site, but it adds an encoding contract to every producer/consumer pair, is still only a heuristic against masking, and leaves empty-output failures silent when it does not hold. Rejected because the failure mode is data-dependent and hard to test.

#### Alternative 2: Always run the Setup Scripts step in every job

Make every generated job materialize `compute_artifact_prefix.sh` in `RUNNER_TEMP` so it can be invoked the same way everywhere, keeping the script on disk rather than embedding it. This was a close call, but it adds setup cost and dependencies (checkout/script staging) to otherwise minimal jobs such as scriptless custom safe-output, SARIF, and coverage jobs. Embedding the script keeps those jobs lightweight while still using one canonical implementation.

#### Alternative 3: Fall back to the caller repository when target metadata is absent

Preserve the previous permissive behaviour where unresolved target-repository metadata degrades to `github.repository`. Rejected because the fallback silently checks out and scopes tokens against the wrong repository for cross-repo reusable workflows — exactly the class of bug this PR fixes. Failing explicitly surfaces the misconfiguration at run time.

### Consequences

#### Positive
- Artifact prefixes and host-repository targeting no longer depend on cross-job outputs, so secret masking can no longer blank them (#38345).
- Jobs without Setup Scripts (scriptless custom safe-output, SARIF, coverage jobs) can compute the prefix, removing an implicit ordering dependency on setup.
- Misconfigured or malformed target-repository metadata fails loudly instead of silently resolving to the caller repository.
- Regression coverage was added at both layers: `pkg/workflow/workflow_call_local_metadata_test.go` and `actions/setup/js/resolve_host_repo.test.cjs`.

#### Negative
- The artifact-prefix script is duplicated inline into multiple jobs of each generated lock file, increasing lock-file size and diff noise (five lock files regenerated here, +~1,000 lines).
- Metadata resolution now runs once per consuming job rather than once per workflow run, adding a small amount of repeated work and one more step that can fail per job.
- The embedded-script path creates a second consumption mode for `compute_artifact_prefix.sh` (staged file and `go:embed`), which must be kept in sync; changing the script now requires recompiling all lock files.
- Explicit failure on malformed metadata is a behaviour change that can break workflows that previously ran (incorrectly) via the caller-repository fallback.

#### Neutral
- The public `agent` job `artifact_prefix` output is retained for compatibility even though internal consumers no longer read it.
- Artifact naming semantics (sha256 of `INPUTS_JSON` + `GITHUB_RUN_ATTEMPT`, first 8 hex chars) are unchanged, so existing artifact names are preserved.
- Inputs continue to be passed through an environment variable rather than interpolated into the shell script, so the template-injection posture is unchanged.
- Documentation was updated in `docs/src/content/docs/reference/artifacts.md` and a patch changeset was added.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*

Implementation: [PR #68949](https://github.com/github/gh-aw/pull/68949).
