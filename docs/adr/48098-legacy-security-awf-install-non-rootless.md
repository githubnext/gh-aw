# ADR-48098: Legacy-Security Mode Installs AWF to Non-Rootless Path

**Date**: 2026-07-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The AWF binary can be installed in two modes: **rootless** (`--rootless`), which places the binary in `~/.local/bin` to minimize privilege surface when `NetworkIsolation` is true; or **non-rootless**, which installs to `/usr/local/bin` for system-wide access. The `legacy-security: enable` sandbox option causes generated workflow steps to invoke AWF as `sudo -E awf`. The `sudo` command enforces its own `secure_path`, which includes `/usr/local/bin` but not `~/.local/bin`. When both `NetworkIsolation` and `LegacySecurity` were true, AWF was installed rootless (`~/.local/bin`) but invoked via `sudo -E awf`, causing `sudo: awf: command not found` at agent startup. The `print_firewall_logs.sh` step had the identical flaw: it conditionally passed `--rootless` based solely on `NetworkIsolation`, ignoring that legacy-security mode grants full sudo access.

### Decision

We will skip the `--rootless` install flag whenever `AgentSandboxConfig.LegacySecurity` is true, regardless of the `NetworkIsolation` setting. The condition in `generateAWFInstallationStep` becomes `NetworkIsolation && !Disabled && !LegacySecurity`, and `generateFirewallLogParsingStep` adds the same `!LegacySecurity` guard. This ensures AWF lands in `/usr/local/bin` (on `sudo`'s `secure_path`) in legacy-security mode, making `sudo -E awf` resolvable.

### Alternatives Considered

#### Alternative 1: Extend sudo's secure_path to include ~/.local/bin

Runner provisioning could be modified to add `~/.local/bin` to `sudo`'s `secure_path` via `/etc/sudoers.d/`. This would allow rootless installs to coexist with sudo invocations. However, this requires infrastructure changes outside this codebase's control, and broadening `secure_path` on shared runners increases the attack surface for privilege escalation via user-writable directories.

#### Alternative 2: Invoke AWF as `sudo -E env PATH=$PATH awf`

The generated sudo invocation could preserve the full user `PATH` using `sudo -E env PATH=$PATH awf`, avoiding the `secure_path` restriction without changing the install location. This would be a more invasive change touching the AWF invocation pattern across multiple generated steps, increasing regression risk for non-legacy-security paths that already work correctly.

### Consequences

#### Positive
- Agent startup no longer fails with `sudo: awf: command not found` in legacy-security mode with network isolation enabled.
- The firewall log parsing step now uses the correct sudo invocation mode, consistent with the AWF install path.

#### Negative
- AWF is installed to a root-owned directory (`/usr/local/bin`) in legacy-security mode, requiring the install script to run with elevated privileges (this is already the behavior for non-rootless installs).
- The condition `NetworkIsolation && !LegacySecurity` is a compound flag interaction; future maintainers changing either flag's semantics must account for this coupling.

#### Neutral
- The smoke-service-ports lock file is regenerated to drop the now-incorrect `--rootless` flags, keeping the file consistent with updated compiler behavior.
- Tests explicitly cover the `LegacySecurity=true` + `NetworkIsolation=true` combination for both the install step and the log parsing step, providing a regression guard.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
