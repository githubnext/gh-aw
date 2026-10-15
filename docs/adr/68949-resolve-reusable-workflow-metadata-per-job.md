# ADR-68949: Resolve Reusable-Workflow Metadata Inside Each Consuming Job

**Date**: 2026-10-14
**Status**: Accepted
**Deciders**: @pelikhan (PR author). Maintainer review pending; this record does not imply formal architecture approval.

---

### Context

Compiled workflows that expose a `workflow_call` trigger need an artifact-name prefix and host/platform repository coordinates at runtime. The prefix separates artifacts for invocations with different inputs or run attempts. The repository coordinates support sparse `.github`/`.agents` checkouts, GitHub App token scoping, and dispatch configuration; the caller and the repository hosting the reusable workflow can differ.

Previously, downstream jobs obtained this metadata through activation or agent job outputs, including `needs.activation.outputs.artifact_prefix`, `needs.agent.outputs.artifact_prefix`, and `needs.activation.outputs.target_repo`. GitHub Actions can omit job outputs that it detects as containing secrets. A metadata value that matches a masked secret can therefore arrive empty downstream, breaking artifact naming or repository targeting ([issue #38345](https://github.com/github/gh-aw/issues/38345)). This is a transport problem, not a reason to encode or expose secrets.

The artifact-prefix step also depended on `${RUNNER_TEMP}/gh-aw/actions/compute_artifact_prefix.sh`, which is only staged by Setup Scripts. Jobs without that setup need an independent way to compute the same prefix.

### Decision

Compute reusable-workflow metadata in each consuming job, using local step outputs rather than cross-job outputs. Emit exactly one producer step for each metadata type needed by a job, before its first consumer. This applies to shared job builders and dedicated jobs, including SARIF upload, coverage upload, and upload-asset paths.

For artifact names, `artifactPrefixExprForDownstreamJob` and `artifactPrefixExprForAgentDownstreamJob` reference `steps.artifact-prefix.outputs.prefix`. Embed the canonical `actions/setup/sh/compute_artifact_prefix.sh` in the compiler through `actions/setup/artifact_prefix.go` and inline its executable lines in generated steps. This permits computation without Setup Scripts while retaining the existing inputs-and-run-attempt hash.

For host metadata, use `steps.resolve-host-repo.outputs.*` in the same job. Non-inlined workflows resolve coordinates from `job.workflow_*`; inlined-import workflows use the corresponding `github.*` context as selected in `compiler_activation_job.go`. Keep the checkout SHA (`target_checkout_ref`) distinct from the dispatch-compatible branch/tag (`target_ref`). Repository/ref consumers and generated GitHub App token owner/repository scope must use consistent coordinates, rather than mixing a locally resolved host repository with a caller-repository default. This does not change which repository a safe output is intended to modify.

Reject missing or malformed repository coordinates in `resolve_host_repo.cjs` instead of silently falling back to the caller repository. An unparseable dispatch ref still produces a warning and an empty `target_ref`; this decision does not introduce complete ref validation or make SHA-pinned calls dispatchable.

### Alternatives Considered

#### Alternative 1: Keep job outputs and obfuscate/encode the transported values

Continue transporting encoded values through `needs.*.outputs` and decode them in each consumer. This keeps a single computation site but adds an encoding contract to every producer/consumer pair. Encoding does not provide a reliable guarantee against secret detection and is not an appropriate way to bypass masking.

#### Alternative 2: Always run the Setup Scripts step in every job

Materialize `compute_artifact_prefix.sh` in every consuming job and invoke the staged file. This avoids inlining but adds setup dependencies to otherwise minimal jobs such as scriptless custom safe-output, SARIF, and coverage jobs. Embedding only the required computation avoids requiring the full setup path.

#### Alternative 3: Fall back to the caller repository when target metadata is absent

Preserve the fallback from missing host metadata to `github.repository`. Rejected because, in a cross-repository call, it can silently select the caller instead of the host for checkout or token scoping. Explicitly selecting `github.*` for inlined imports is a separate supported mode, not a fallback for invalid host metadata.

### Consequences

#### Positive
- Local consumers avoid the cross-job output suppression failure for this metadata. This is not a guarantee about all GitHub Actions masking behavior.
- Jobs without Setup Scripts can compute the same artifact prefix without staging the full script bundle.
- Invalid repository coordinates fail explicitly instead of silently selecting the caller.

#### Negative
- Inlined computation increases generated lock-file size and diff noise. Metadata resolution repeats work and adds potential step failures in each consuming job.
- Shared and dedicated job builders must preserve producer ordering and uniqueness. A local expression alone is insufficient if the job omits its producer step or injects duplicate step IDs.
- The staged and embedded consumption modes share one script source, but script changes require rebuilding the compiler and regenerating affected lock files.
- Rejecting invalid repository metadata can fail workflows that previously continued through the caller fallback. Ref parsing remains a separate limitation.
- The pinned `github/codeql-action/upload-sarif` and `actions/upload-code-coverage` actions publish to the caller repository without a repository override. The dedicated jobs therefore explicitly reject cross-repository reusable-workflow publication rather than silently publishing to the caller. Successful cross-repository SARIF or coverage upload is not implemented by this PR.

#### Neutral
- The `agent` job's `artifact_prefix` output is retained for compatibility; external consumers of that output remain subject to job-output suppression.
- Naming retains the first eight hexadecimal characters of SHA256 over `INPUTS_JSON` plus `::attempt=` and `GITHUB_RUN_ATTEMPT`, followed by `-`. Identical inputs in the same run attempt still share a prefix, and truncated hashes can collide; this change does not guarantee invocation uniqueness.
- Inputs remain environment-variable data rather than shell-source interpolation.
- Resolving a host repository locally does not grant access to it. Token permissions, installation access, and cross-repository checkout constraints still apply.
- The pinned coverage action also derives its revision from the caller event, using the PR head for pull requests. Non-inlined reusable coverage jobs reject missing or mismatched target revisions instead of publishing against that caller revision; supporting other revisions requires an uploader that accepts explicit coordinates.
- Compiler assertions in `pkg/workflow/workflow_call_local_metadata_test.go` and resolver tests in `actions/setup/js/resolve_host_repo.test.cjs` provide regression coverage, not evidence of a successful enterprise runner execution or every credential configuration.

---

Implementation: [PR #68949](https://github.com/github/gh-aw/pull/68949).
