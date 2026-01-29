---
title: Container mount security
description: Threat model and best practices for mounting host paths into agent containers.
---

# Container mount security

Mounting host paths into agent containers can be powerful, but
it expands what the agent can read and write on the runner.
Use mounts only when you cannot achieve the same outcome by
installing tools inside the container or by passing data
through workflow inputs.

## Threat model

Agentic workflows run in containers, but mounts pierce the
filesystem boundary. Consider three main threats:

- **Malicious workflow author** attempts to read or alter
  host files by adding broad mounts.
- **Compromised dependencies** try to access host secrets or
  runner configuration through mounted paths.
- **Accidental exposure** of sensitive data (tokens, SSH keys,
  config files) when mounting user directories or system paths.

## Current security controls

gh-aw limits the default mount set and applies defense in depth:

- **Read-only mounts for system tools.** Only specific binaries
  such as `gh` and `yq` are mounted, not entire system paths.
- **Principle of least privilege.** Default mounts are narrow
  and purpose-built for agent execution.
- **No Docker socket access.** The agent sandbox does not mount
  `/var/run/docker.sock`, and custom mounts cannot add it.
- **No privileged containers.** The agent container runs without
  `--privileged` and does not grant kernel-level capabilities.

For the full default mount list, see
[Sandbox Configuration](/gh-aw/reference/sandbox/).

## Risks by host path

Some host paths create outsized risk even when mounted read-only:

- **System binaries and libraries** (`/usr/bin`, `/lib`,
  `/usr/local/bin`): broad visibility into the host runtime and
  access to tooling you may not intend to expose.
- **Configuration and secrets** (`/etc`, `/home/runner`,
  `/root`, `/var/lib`): likely to contain credentials,
  SSH keys, or package manager state.
- **Process and device paths** (`/proc`, `/sys`, `/dev`,
  `/var/run`): can expose kernel interfaces or sockets.
- **Workspace root** (`${GITHUB_WORKSPACE}`): already mounted
  by default. Additional mounts that overlap it can create
  confusing precedence or unreviewed write access.

> [!WARNING]
> Read-only mounts still allow disclosure.
> Treat any readable host path as sensitive data that the
> workflow can exfiltrate through allowed tools.

## Best practices for workflow authors

Follow these guidelines whenever you add `sandbox.agent.mounts`:

1. **Prefer installing tools inside the container.** Use
   setup steps or MCP tools instead of mounting host binaries.
2. **Mount the smallest possible path.** Mount a single file or
   dedicated directory instead of a whole tree.
3. **Default to `ro`.** Use `rw` only for explicit output paths
   such as `/tmp` caches or scratch directories.
4. **Avoid system and credential paths.** Do not mount
   `/usr`, `/lib`, `/etc`, `/home`, `/root`, `/proc`, `/sys`,
   `/dev`, `/var/run`, or `/var/lib/docker`.
5. **Document the purpose.** Treat mounts as security-sensitive
   configuration that must be reviewed.

## Safe and unsafe examples

✅ Safe, narrow, read-only mounts:

```yaml wrap
sandbox:
  agent:
    id: awf
    mounts:
      - "/opt/tools/custom-cli:/usr/local/bin/custom-cli:ro"
      - "/opt/data/input:/data/input:ro"
      - "/tmp/gh-aw-cache:/cache:rw"
```

❌ Unsafe, overly broad mounts:

```yaml wrap
sandbox:
  agent:
    id: awf
    mounts:
      - "/usr:/usr:ro"
      - "/:/host:ro"
      - "/var/run/docker.sock:/var/run/docker.sock:rw"
```

## Checklist for adding new default mounts

Use this checklist when proposing new default mounts:

- [ ] Mount is required for core agent functionality.
- [ ] No safer alternative exists (install in container or MCP).
- [ ] Path is as narrow as possible.
- [ ] Access mode is `ro` unless write access is unavoidable.
- [ ] Path does not expose system configuration or secrets.
- [ ] Impact is documented in the security guide and sandbox
      reference.
- [ ] Threat model review completed with security reviewers.

## See also

- [Sandbox Configuration](/gh-aw/reference/sandbox/)
- [Security Best Practices](/gh-aw/guides/security/)
- [Network Configuration](/gh-aw/reference/network/)


