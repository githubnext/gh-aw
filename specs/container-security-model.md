# Container Mount Security Model

This document defines the threat model and review checklist for
mounting host paths into agent containers. It complements the
user-facing guide in
`docs/src/content/docs/guides/security/container-mounts.md`.

## Scope

This model covers:

- AWF-based agent containers that mount host paths
- Custom mounts defined by `sandbox.agent.mounts`
- Default mounts required for agent operation

It does not cover:

- MCP gateway container mounts (see
  `docs/src/content/docs/reference/mcp-gateway.md`)
- Sandbox Runtime filesystem rules

## Threat model

### Adversaries

- **Malicious workflow author** with write access tries to
  expand mounts to access host secrets or runtime internals.
- **Compromised dependencies** (CLI tools, packages) attempt to
  read or alter host files exposed through mounts.
- **Accidental exposure** where legitimate workflows leak
  sensitive data via overly broad mounts.

### Assets at risk

- GitHub Actions runner configuration and tokens
- Repository secrets cached on disk
- Host system binaries and configuration
- Docker daemon and other privileged sockets

### Abuse paths

1. Mounting system directories (`/usr`, `/lib`, `/etc`) gives
   read access to host runtime details and secrets.
2. Mounting `/var/run` or `/dev` can expose control sockets or
   device interfaces.
3. Mounting home directories leaks cached tokens, SSH keys, or
   package manager credentials.

## Existing controls

- **Default mounts are narrow** and explicitly enumerated.
- **Read-only mounts for binaries** (`/usr/bin/gh`, `yq`, etc.).
- **No Docker socket mount**: `/var/run/docker.sock` is blocked.
- **No privileged container flags**: agent container runs
  without `--privileged` or extra capabilities.

## Required review questions

Use these questions in code review when adding mounts:

1. **Is the mount necessary?** Can the workflow install tools
   inside the container or use an MCP tool instead?
2. **Is the path minimal?** Prefer a single file or directory.
3. **Is the mode correct?** Use `ro` unless write access is
   essential.
4. **Is the path sensitive?** Reject mounts that expose system
   configuration, credentials, or device sockets.
5. **Is the purpose documented?** Add docs and release notes
   for any new default mount.

## Default mount change checklist

- [ ] Justification ties to core functionality.
- [ ] Alternative approaches documented and rejected.
- [ ] Path is the smallest possible surface.
- [ ] Access mode is `ro` unless output requires `rw`.
- [ ] No system, secrets, or socket paths.
- [ ] Security guide updated.
- [ ] Sandbox reference updated.

## References

- `docs/src/content/docs/reference/sandbox.md`
- `docs/src/content/docs/guides/security/container-mounts.md`
- `specs/github-actions-security-best-practices.md`
