# ADR-51413: sandbox.agent.runtime-install Field with Shell Script Extraction

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflows that use a sandbox runtime (docker-sbx or gVisor) currently always generate installation steps (KVM check, secrets check, install, daemon startup, pre-flight smoke test) as part of the agent setup. For organizations running pre-baked runner images with the sandbox runtime already installed, these steps are redundant and add unnecessary setup time. There is no mechanism to skip them while retaining other runtime-related steps (e.g. credential refresh).

Additionally, all setup shell logic was previously inlined in Go step-generator functions, making the scripts hard to read, test independently, or reuse outside Go code.

### Decision

We will introduce a `sandbox.agent.runtime-install` boolean field (default `true`) on `AgentSandboxConfig`. When set to `false`, the engine skips generating the KVM check, secrets check, install, daemon, and pre-flight steps for docker-sbx, and the install step for gVisor, while still emitting the credential-refresh step unconditionally. The field follows **false-wins** merge semantics across workflow imports: if any imported workflow sets `runtime-install: false`, the accumulated value becomes `false` regardless of other imports.

We also extract all previously-inlined setup shell code into standalone scripts under `actions/setup/sh/`, using a `sudo_` prefix convention for scripts that require elevated privileges.

### Alternatives Considered

#### Alternative 1: Automatic Runtime Detection

Auto-detect at generation time whether the sandbox runtime is already installed on the runner (e.g. by probing the binary path or checking a known file). Skip the install steps if detected.

Automatic detection is unreliable in heterogeneous runner fleets where the same workflow YAML targets both fresh and pre-baked runners. It also adds detection logic that can silently suppress necessary installation, making failures harder to diagnose. Explicit opt-out (`runtime-install: false`) gives operators deliberate control and keeps intent visible in the workflow configuration.

#### Alternative 2: Separate Reusable Install Workflow / Action

Move sandbox installation into a standalone reusable workflow or composite action that callers invoke only when needed, removing the embedded install step generation from the engine entirely.

This would require all existing callers to explicitly add an `uses: …/sandbox-install` step, a breaking change to the current model. It also removes the ability for shared agent workflows to centrally declare the installation policy and propagate it through imports. The new field preserves the existing workflow shape while giving callers opt-out control.

### Consequences

#### Positive
- Organizations with pre-baked sandbox runners avoid redundant installation time on every run.
- Shell scripts under `actions/setup/sh/` are independently readable, testable, and callable outside the Go step generator.
- The `sudo_` filename prefix makes privilege requirements visible without reading the script body.
- The credential-refresh step always runs regardless of the field, keeping Docker Hub tokens fresh.

#### Negative
- The `*bool` pointer type for `RuntimeInstall` adds indirection; callers must distinguish nil (unset), `true`, and `false`.
- False-wins semantics mean a single transitively-imported shared workflow can suppress installation for all consumers, which may be unexpected in complex import graphs.
- The new field extends the public schema surface of `sandbox.agent`, which must be maintained and documented.

#### Neutral
- `MergedSandboxAgentRuntimeInstall` is added to `ImportsResult` alongside the existing `MergedSandboxAgentMounts` accumulator, following the established pattern.
- The `isRuntimeInstallEnabled()` predicate returns `true` when no runtime is specified (noop path), preserving existing behaviour for workflows that do not use a sandbox runtime.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
