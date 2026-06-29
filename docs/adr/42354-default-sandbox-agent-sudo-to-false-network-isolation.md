# ADR-42354: Default sandbox.agent.sudo to False (Network Isolation)

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `sandbox.agent` configuration controls how the AWF (Agentic Workflow Firewall) process is launched when running AI agents in GitHub Actions workflows. Previously, omitting the `sudo` field in `sandbox.agent` frontmatter was equivalent to `sudo: true`, causing AWF to be invoked as `sudo -E awf` — granting the agent elevated host-level access by default. This "permissive by default" posture conflicted with the security principle of least privilege. Any workflow that did not explicitly configure `sudo: false` unknowingly ran in a more privileged mode. The goal of this change is to make network isolation (rootless mode) the safe default, requiring explicit opt-in for elevated access.

### Decision

We will change the global default for `sandbox.agent.sudo` from `true` (host-access/sudo mode) to `false` (network isolation/rootless mode). When `sudo` is omitted from the frontmatter, `NetworkIsolation=true` will be set and AWF will run without `sudo`. Explicitly setting `sudo: true` will still work but will emit a compile-time error in strict mode and a warning in non-strict mode, signaling that the field is deprecated and its use should be intentional.

### Alternatives Considered

#### Alternative 1: Keep sudo: true as the Default (Status Quo)

The existing behavior could be retained, requiring operators to explicitly set `sudo: false` to enable network isolation. This was rejected because security-by-default is strongly preferable: most workflows do not require host-level access, and relying on operators to opt into a safer mode leaves a large surface area exposed by inaction or oversight.

#### Alternative 2: Remove the sudo Option Entirely and Always Use Network Isolation

The `sudo` field could be removed from the schema so that all workflows unconditionally run in rootless/network-isolation mode. This was rejected because some legitimate workflows may currently depend on `sudo: true` for reasons not yet eliminated. A hard removal without a deprecation path would be a breaking change with no escape hatch; the warning/error feedback mechanism preserves discoverability while signaling the direction of travel.

### Consequences

#### Positive
- Workflows that omit `sudo` now default to the more secure rootless network-isolation mode, reducing the default attack surface for AI agents.
- Explicit `sudo: true` usage is surfaced at compile time (error in strict mode, warning otherwise), giving operators visibility into elevated-privilege configurations.
- Aligns the sandbox defaults with the security principle of least privilege.

#### Negative
- Existing workflows that omit `sudo` and relied on the old default (`sudo -E awf`) will silently switch to rootless mode, which may break workflows that require host-level access or sudo networking.
- The `SudoExplicitlyEnabled` sentinel field adds complexity to `AgentSandboxConfig`, requiring callers and test code to distinguish between "sudo not set" and "sudo set to false."
- All golden files and tests that previously asserted `sudo -E awf` as the default output must be updated, increasing the scope of a seemingly small default change.

#### Neutral
- The change does not alter the YAML serialization format; `sudo: true` and `sudo: false` remain valid frontmatter values.
- The deprecation path for `sudo: true` (strict error vs. non-strict warning) introduces two distinct enforcement modes whose behavior differences may need to be documented for operators.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
