---
title: Self-Hosted Runners
description: How to configure and run agentic workflows on self-hosted runners, ARC/Kubernetes, and GHES environments.
sidebar:
  order: 810
---

Use the `runs-on` frontmatter field to target a self-hosted runner instead of the default `ubuntu-latest`.

Runners must be Linux with Docker support. macOS and Windows are not supported.

Self-hosted runners may require `sudo` depending on the selected engine and configuration. For the default GitHub Copilot engine, there are two distinct sudo considerations:

- **AWF (Agentic Workflow Firewall)**: Runs rootless in the default network-isolation mode. Egress is enforced via Docker network topology — an internal Docker network (`awf-net`) with no internet route and a dual-homed Squid proxy as the sole egress path. No `sudo` and no `NET_ADMIN` are required on the runner for AWF in this mode. Container-level `iptables`, Squid proxy ACLs, and capability drops provide defense in depth, all managed inside the Docker daemon's domain.

- **Copilot CLI install**: The `install_copilot_cli.sh` script runs as the runner user but escalates via `sudo` for specific file operations (fixing `.copilot` directory ownership, cleaning stale chroot directories, and installing the Copilot binary). ARC pods with `allowPrivilegeEscalation: false` will fail at this step with `sudo: The "no new privileges" flag is set`.

## ARC with Docker-in-Docker (DinD)

For a complete ARC DinD setup walkthrough for GitHub Copilot coding agent, see [How to run GitHub Copilot coding agent on ARC with Docker-in-Docker](/gh-aw/guides/arc-dind-copilot-agent/).

Actions Runner Controller (ARC) deployments that use a Docker-in-Docker sidecar split the runner container and the Docker daemon container across separate filesystems.

Set `runner.topology: arc-dind` in workflow frontmatter for this environment.
Compiled workflows emit a runtime probe that inspects `DOCKER_HOST`.
Any `tcp://` endpoint (for example `tcp://localhost:2375`, `tcp://dind:2375`, or `tcp://172.30.0.5:2375`) is treated as ARC DinD, so ensure `DOCKER_HOST` points to the DinD daemon for that runner pod.

With ARC DinD handling enabled, AWF receives `--docker-host`, shared-work sysroot staging is applied, and chroot config patching is enabled. The runtime no longer uses `--docker-host-path-prefix`.

### Docker socket override for split-daemon topologies

When `DOCKER_HOST` is a TCP address (e.g., `tcp://localhost:2375`) and the Docker socket is mounted via a bind mount at a non-standard path, the MCP gateway needs explicit configuration to find the socket and determine its group ID.

Set these environment variables at the runner level (e.g., in your ARC runner pod spec or Kubernetes deployment):

- **`GH_AW_DOCKER_SOCK_PATH`** — absolute path to the bind-mounted Docker socket (e.g., `/dind-sock/docker.sock`)
- **`GH_AW_DOCKER_SOCK_GID`** — numeric group ID of the Docker socket (e.g., `999`)

**Example runner pod configuration:**

```yaml
env:
  - name: GH_AW_DOCKER_SOCK_PATH
    value: /dind-sock/docker.sock
  - name: GH_AW_DOCKER_SOCK_GID
    value: "999"
```

When both overrides are set, the MCP gateway will mount the specified socket path and add the specified group to the container without attempting automatic detection. If either variable is omitted, the gateway falls back to auto-detection from `DOCKER_HOST` and `stat`, and will fail with an actionable error if resolution fails.

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
    runs-on: [self-hosted, linux, x64]
---
```

This is useful when your self-hosted runner lacks outbound internet access for AI detection, or when you want to run the detection job on a cheaper runner.

## Configuring the framework job runner

Framework jobs — activation, pre-activation, safe-outputs, unlock, APM, update_cache_memory, and push_repo_memory — default to `ubuntu-slim`. Use `runs-on-slim:` to override all of them at once:

```aw
---
on: issues
runs-on: [self-hosted, linux, x64]
runs-on-slim: [self-hosted, linux, x64]
safe-outputs:
  runs-on: [self-hosted, linux, x64]
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
- **ARC/Kubernetes**: Docker-in-Docker (DinD) is **required** for ARC. Set `containerMode.type="dind"` in your ARC Helm configuration. The `containerMode.type="kubernetes"` mode is not supported. The dind sidecar must share the Docker socket via an `emptyDir` volume, and the gateway retries the socket check for up to 10 seconds to handle startup race conditions. See [How to run GitHub Copilot coding agent on ARC with Docker-in-Docker](/gh-aw/guides/arc-dind-copilot-agent/) for the complete setup guide, and [ARC (Actions Runner Controller)](#arc-actions-runner-controller) below for pod security details.
- **Split-daemon override**: On ARC or other split-daemon topologies where the socket path or group ID cannot be auto-detected, set `GH_AW_DOCKER_SOCK_PATH` and `GH_AW_DOCKER_SOCK_GID` environment variables at the runner level. See [Docker socket override for split-daemon topologies](#docker-socket-override-for-split-daemon-topologies) for details.

### Filesystem

- **Use `RUNNER_TEMP` for transient state.** Put sandbox state, tool downloads, and intermediate outputs in `$RUNNER_TEMP`, which is cleaned between jobs. On shared runners, avoid writing arbitrary workflow data to `/tmp` because it can persist across jobs.
- **No root assumption.** Tool installs, file operations, and sandbox setup should run as the unprivileged runner user. The Copilot CLI install script escalates via `sudo` for specific operations; see the sudo requirements above.
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

### GHES compatibility mode

GHES does not support the `@actions/artifact` v2.0.0+ backend used by `upload-artifact@v4+` and `download-artifact@v4+`. Compiled workflows use the latest artifact action versions by default, which fail on GHES with `GHESNotSupportedError`.

Enable GHES compatibility mode in `.github/workflows/aw.json` to compile with GHES mode explicitly enabled:

```json
{
  "ghes": true
}
```

Or compile with `--ghes` for one-off workflow generation:

```bash
gh aw compile --ghes my-workflow.md
```

Artifact actions continue using the latest non-v3 pins because v3 artifact actions are deprecated.

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

GitHub Copilot coding agent **requires** Docker-in-Docker (DinD) mode on ARC. Set `containerMode.type="dind"` in your ARC Helm configuration. The `containerMode.type="kubernetes"` mode is not supported.

Set `runner.topology: arc-dind` in workflow frontmatter to enable ARC DinD split-filesystem handling. See the [ARC with Docker-in-Docker (DinD)](#arc-with-docker-in-docker-dind) section above and the [ARC DinD setup guide](/gh-aw/guides/arc-dind-copilot-agent/) for a complete walkthrough.

### Docker-in-Docker (dind) sidecar

The MCP gateway:

1. Resolves the Docker socket path from `DOCKER_HOST` (supports `unix://` paths and bare absolute paths)
2. Auto-detects the socket's group ID for correct permissions
3. Retries the socket check for up to 10 seconds to handle the race condition where the gateway starts before `dockerd`

### Pod security

The dind sidecar requires `privileged: true` so `dockerd` can run. The runner container does **not** need `privileged: true` or `NET_ADMIN`.

In network-isolation mode (the default for `topology: arc-dind`), AWF enforces egress via Docker network topology — an internal Docker network with no internet route and a dual-homed Squid proxy. All network enforcement happens inside the Docker daemon's domain (the dind sidecar). The runner container only issues Docker API commands via the socket; it never manipulates host `iptables` or network namespaces.

> [!NOTE]
> If your cluster enforces `allowPrivilegeEscalation: false` or `no-new-privileges` on the runner container, the Copilot CLI install script will fail. See [Known limitations](/gh-aw/guides/arc-dind-copilot-agent/#known-limitations) in the ARC DinD guide.
