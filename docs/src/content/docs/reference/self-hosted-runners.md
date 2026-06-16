---
title: Self-Hosted Runners
description: How to configure and run agentic workflows on self-hosted runners, ARC/Kubernetes, and GHES environments.
sidebar:
  order: 810
---

Use the `runs-on` frontmatter field to target a self-hosted runner instead of the default `ubuntu-latest`.

Runners must be Linux with Docker support. macOS and Windows are not supported.

Self-hosted runners must allow `sudo` for agentic workflows. This is a requirement to allow all GH-AW security features to be enabled. Specific technical needs are:

- AWF (Agentic Workflow Firewall) applies host-level `iptables` rules to the Linux kernel `DOCKER-USER` chain to enforce network egress filtering for all agent containers on the AWF bridge network. This outer security boundary requires root UID.

- Container-level `iptables`, Squid proxy ACLs, and capability drops add additional defense in depth, but they do not replace host-level filtering.

For these reasons, a non-sudo mode is not supported, including ARC configurations with `allowPrivilegeEscalation: false`.

## ARC with Docker-in-Docker (DinD)

Actions Runner Controller (ARC) deployments that use a Docker-in-Docker sidecar split the runner container and the Docker daemon container across separate filesystems, so bind mounts constructed from the runner's perspective fail inside the daemon.

`gh aw compile` emits a runtime probe in generated workflows that inspects `DOCKER_HOST` and appends `--docker-host-path-prefix /tmp/gh-aw` to the AWF invocation when the value matches `tcp://localhost:<port>` or `tcp://127.0.0.1:<port>`. No workflow-level configuration is required.

The probe is gated on AWF `v0.25.43` or newer. Workflows pinned to an older AWF version, or running on GitHub-hosted runners (where `DOCKER_HOST` is unset or points at a Unix socket), are unaffected.

## runs-on formats

**String** — single runner label:

```aw
---
on: issues
runs-on: self-hosted
---
```

**Array** — runner must have *all* listed labels (logical AND):

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
---
```

**Object** — named runner group, optionally filtered by labels:

```aw
---
on: issues
runs-on:
  group: my-runner-group
  labels: [linux, x64]
---
```

## Sharing configuration via imports

`runs-on` must be set in each workflow — it is not merged from imports. Other settings like `network` and `tools` can be shared:

```aw title=".github/workflows/shared/runner-config.md"
---
network:
  allowed:
    - defaults
    - private-registry.example.com
tools:
  bash: {}
---
```

```aw
---
on: issues
imports:
  - shared/runner-config.md
runs-on: [self-hosted, linux, x64]
---

Triage this issue.
```

## Configuring the detection job runner

When [threat detection](/gh-aw/reference/threat-detection/) is enabled, the detection job runs on the agent job's runner by default. Override it with `safe-outputs.threat-detection.runs-on`:

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
safe-outputs:
  create-issue: {}
  threat-detection:
    runs-on: ubuntu-latest
---
```

This is useful when your self-hosted runner lacks outbound internet access for AI detection, or when you want to run the detection job on a cheaper runner.

## Configuring the framework job runner

Framework jobs — activation, pre-activation, safe-outputs, unlock, APM, update_cache_memory, and push_repo_memory — default to `ubuntu-slim`. Use `runs-on-slim:` to override all of them at once:

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
runs-on-slim: self-hosted
safe-outputs:
  create-issue: {}
---
```

> [!NOTE]
> `runs-on` controls only the main agent job. `runs-on-slim` controls all framework/generated jobs. `safe-outputs.runs-on` still takes precedence over `runs-on-slim` for safe-output jobs specifically.
> `runs-on-slim` accepts the same string, array, or runner-group object forms as `runs-on`.

## Configuring the maintenance workflow runner

The generated `agentics-maintenance.yml` workflow defaults to `ubuntu-slim` for all its jobs. To use a self-hosted runner for maintenance jobs, set `runs_on` in `.github/workflows/aw.json`:

**Single label:**

```json
{
  "maintenance": {
    "runs_on": "self-hosted"
  }
}
```

**Multiple labels** (runner must match all):

```json
{
  "maintenance": {
    "runs_on": ["self-hosted", "linux", "x64"]
  }
}
```

This setting applies to every job in `agentics-maintenance.yml` (close-expired-entities, cleanup-cache-memory, run_operation, apply_safe_outputs, create_labels, validate_workflows, and activity_report). Re-run `gh aw compile` after changing `aw.json` to regenerate the workflow.

> [!NOTE]
> `aw.json` is separate from individual workflow frontmatter. It provides repository-level settings for generated infrastructure workflows.

## Related documentation

- [Frontmatter](/gh-aw/reference/frontmatter/#run-configuration-run-name-runs-on-runs-on-slim-timeout-minutes) — `runs-on` and `runs-on-slim` syntax reference
- [Imports](/gh-aw/reference/imports/) — importable fields and merge semantics
- [Threat Detection](/gh-aw/reference/threat-detection/) — detection job configuration
- [Network Access](/gh-aw/reference/network/) — configuring outbound network permissions
- [Sandbox](/gh-aw/reference/sandbox/) — container and Docker requirements
- [Ephemerals](/gh-aw/reference/ephemerals/#maintenance-configuration) — full `aw.json` maintenance configuration reference
- [Enterprise Configuration](/gh-aw/reference/enterprise-configuration/) — custom API endpoints for GHEC/GHES

## Runner environment requirements

Self-hosted runners must meet these requirements for agentic workflows to run reliably.

### Docker

A working Docker daemon is required. The MCP gateway and sandbox run as containers.

- **Unix socket**: Docker must be accessible via a Unix socket (typically `/var/run/docker.sock`). If `DOCKER_HOST` is unset, the gateway mounts `/var/run/docker.sock`. If `DOCKER_HOST` is `unix://...` or a bare absolute path, the gateway mounts that socket path. Other schemes (for example `tcp://...`) are ignored for mounts and default back to `/var/run/docker.sock`.
- **Docker group**: The runner user must be in the `docker` group, or the socket must be world-readable.
- **ARC/Kubernetes**: If using [actions-runner-controller](https://github.com/actions/actions-runner-controller) with Docker-in-Docker (dind), the dind sidecar must share the Docker socket via an `emptyDir` volume. The gateway will retry the socket check for up to 10 seconds to handle startup race conditions.

### Filesystem

- **Use `RUNNER_TEMP` for transient state.** Put sandbox state, tool downloads, and intermediate outputs in `$RUNNER_TEMP`, which is cleaned between jobs. On shared runners, avoid writing arbitrary workflow data to `/tmp` because it can persist across jobs. The `/tmp/gh-aw` prefix is reserved for gh-aw/AWF ARC DinD path rewriting. `actions/setup` resets `/tmp/gh-aw` at job start, and your normal runner `/tmp` cleanup policy should handle stale data from interrupted jobs.
- **No root or sudo assumption.** The runner user may not have root or `sudo` access (except for the initial iptables setup, which requires `sudo`). Tool installs, file operations, and sandbox setup should work as the unprivileged runner user.
- **No global installs.** Do not install packages to `/usr/local/`, `/opt/hostedtoolcache/`, or other system-wide paths. These may be read-only, shared across runners, or bind-mounted read-only inside the sandbox. Use job-scoped writable locations instead.
- **No hardcoded `HOME` paths.** The runner's home directory may not be `/home/runner`. Use `$HOME` or `$RUNNER_TEMP` instead of hardcoded paths.

### Post-job cleanup

Self-hosted runners persist between jobs. Agentic workflows should clean up after themselves:

- Files written to `$RUNNER_TEMP` are automatically cleaned.
- Docker containers on the `awf-net` bridge are stopped and removed by the sandbox teardown.
- If your workflow creates files outside `$RUNNER_TEMP` (e.g. in `$GITHUB_WORKSPACE`), the runner's built-in workspace cleanup handles this.

### Network

Self-hosted runners need outbound HTTPS access to:

- `api.githubcopilot.com` (or your enterprise Copilot endpoint)
- `github.com` (or your GHES instance)
- `ghcr.io` (to pull the MCP gateway container image)
- Any domains listed in your workflow's `network.allowed` configuration

## GHES (GitHub Enterprise Server)

Agentic workflows can run on GHES with some additional configuration.

### Artifact compatibility

GHES does not support the `@actions/artifact` v2.0.0+ backend used by `upload-artifact@v4+` and `download-artifact@v4+`. Compiled workflows use the latest artifact action versions by default, which fail on GHES with `GHESNotSupportedError`.

Enable GHES compatibility mode in `.github/workflows/aw.json` to use compatible v3.x artifact actions:

```json
{
  "ghes": true
}
```

Or compile with `--ghes` for one-off workflow generation:

```bash
gh aw compile --ghes my-workflow.md
```

This makes the compiler emit `upload-artifact@v3.2.2` and `download-artifact@v3.1.0` instead of the latest versions, which are compatible with all GHES versions.

### API endpoint

GHES instances need the `api-target` engine configuration. See [Enterprise Configuration](/gh-aw/reference/enterprise-configuration/) for full setup instructions.

```aw
---
engine:
  id: copilot
  api-target: api.enterprise.githubcopilot.com
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
---
```

## ARC (Actions Runner Controller)

When running on [ARC](https://github.com/actions/actions-runner-controller) with Kubernetes:

### Docker-in-Docker (dind) sidecar

The standard ARC dind pattern with a shared `emptyDir` for the Docker socket is supported. The MCP gateway:

1. Resolves the Docker socket path from `DOCKER_HOST` (supports `unix://` paths and bare absolute paths)
2. Auto-detects the socket's group ID for correct permissions
3. Retries the socket check for up to 10 seconds to handle the race condition where the gateway starts before `dockerd`

### Pod security

The runner pod requires `privileged: true` on both the dind sidecar and the runner container. This is needed for:

- `dockerd` in the dind sidecar
- `iptables` rules for the agentic workflow firewall
- Chroot/sandbox setup in the runner container
