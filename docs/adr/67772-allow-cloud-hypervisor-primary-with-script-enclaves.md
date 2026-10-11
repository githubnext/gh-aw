# ADR-67772: Allow a Cloud Hypervisor Primary Agent with Static Script Enclaves

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: lpcox (PR author), gh-aw maintainers

---

### Context

The gh-aw compiler historically treated `sandbox.agent.runtime: cloud-hypervisor` and the `enclaves:` subsystem as mutually exclusive: `validateSandboxConfig` (`pkg/workflow/sandbox_validation.go`) rejected any workflow that declared enclaves alongside a Cloud Hypervisor primary, and `validateCloudHypervisorEnclavePrimaryRuntime` (`pkg/workflow/enclaves_runtime.go`) grouped `cloud-hypervisor` with the unsupported `sbx`/`nvx` primaries. That rule forced compiler-owned workflows that target AWF's unified workload backend to choose either a microVM primary agent or microVM enclaves, but never both, even though AWF `v0.28.47+` ships both from the same release. The combination is still a preview: compiling it must not be read as acceptance of Linux/KVM isolation or as a production-readiness signal. Agent enclaves additionally depend on a scoped executor bearer handoff that is not yet available under a Cloud Hypervisor primary.

### Decision

We will permit a Cloud Hypervisor primary agent to be combined with Cloud Hypervisor **static script** enclaves, and enforce that the primary and enclave backends match rather than banning the combination outright. Concretely: the blanket `cloud-hypervisor is incompatible with enclaves` check is removed from `validateSandboxConfig`; `cloud-hypervisor` is dropped from the rejected primary runtimes in `validateCloudHypervisorEnclavePrimaryRuntime`; a new `validateCloudHypervisorEnclaveAWFArgs` helper rejects any explicit `--container-runtime` value that disagrees with the resolved primary runtime; and agent enclaves under a Cloud Hypervisor primary are rejected with a directed error pointing at static script enclaves. The primary driver is unblocking same-backend workflows while keeping the failure mode closed and explicit for every configuration we have not validated.

### Alternatives Considered

#### Alternative 1: Keep the blanket rejection and require a Docker primary

Leave `validateSandboxConfig` as-is and continue to require `sandbox.agent.runtime: docker` whenever `enclaves[].runtime: cloud-hypervisor` is set. This is the lowest-risk option — it preserves the only configuration that has been exercised end to end — and was a genuine close call given the preview status of the feature. It was rejected because it permanently blocks the compiler-owned workflows that need a single microVM backend for both the primary agent and its enclaves, and because it forces a Docker dependency onto hosts selected precisely to avoid one.

#### Alternative 2: Allow all enclave kinds (including agent enclaves) under a Cloud Hypervisor primary

Remove the primary-runtime restriction entirely and let `enclaves[].agent` run under a Cloud Hypervisor primary as well. This would have been simpler — one deletion instead of a new targeted check — and avoids a second special case in the validator. It was rejected because agent enclaves require the API proxy and a scoped executor bearer handoff that is not implemented for a microVM primary; allowing them would fail opaquely at runtime instead of at compile time, which contradicts the enclave subsystem's fail-closed contract.

#### Alternative 3: Gate the combination behind a separate opt-in experimental flag

Introduce a dedicated frontmatter or CLI flag that must be set before the primary+enclave combination compiles. Rejected because `enclaves[].runtime: cloud-hypervisor` is already documented as an experimental preview with an explicit warning; adding a second gate duplicates the signal and adds a schema surface we would then have to deprecate when the feature stabilizes.

### Consequences

#### Positive
- Workflows can now run a Cloud Hypervisor primary and Cloud Hypervisor static script enclaves from the same AWF release tag, so artifact identities stay coordinated (verified by `TestCompileCloudHypervisorPrimaryWithScriptEnclave`, which pins bundle → enclave-artifact → gateway → `awf` step ordering and the `GH_AW_AWF_VERSION` propagation).
- Backend mismatches are now a *specific* compile-time error (`requires --container-runtime <runtime> to match sandbox.agent.runtime`) rather than a generic "incompatible with enclaves" rejection, which is more actionable for workflow authors.
- Validation for AWF arguments is factored into `validateCloudHypervisorEnclaveAWFArgs`, shortening `validateCloudHypervisorEnclaves` and making the primary-runtime parameter explicit.

#### Negative
- The compiler now accepts a configuration that has not been validated for Linux/KVM isolation acceptance or production readiness; successful compilation becomes a weaker signal than it was, and the guard against misreading it lives only in documentation prose.
- The valid/invalid matrix grows: `cloud-hypervisor` is rejected as a primary for agent enclaves but accepted for script enclaves, so the runtime-compatibility rules are no longer expressible as a single list of banned primaries.
- Removing the `validateSandboxConfig` check concentrates enforcement in the enclave validator; any future code path that reaches sandbox validation without running enclave validation would silently allow unsupported combinations.

#### Neutral
- `docker-sudo-iptables`, `sbx`, and `nvx` primaries, `dynamic` enclaves, enclave `image` overrides, `--enable-dind`, `--docker-host-path-prefix`, and `runner.topology: arc-dind` remain rejected — the change is scoped to the Cloud Hypervisor primary case only.
- An existing test was inverted rather than removed (`TestCloudHypervisorValidationRejectsEnclaves` → `TestCloudHypervisorValidationAllowsEnclaves`), preserving the original coverage point with its new expected outcome.
- The experimental enclaves documentation (`docs/src/content/docs/experimental/enclaves.md`) now states the AWF `v0.28.47+` requirement, the same-release artifact provisioning behaviour, and the acceptance boundary explicitly.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
