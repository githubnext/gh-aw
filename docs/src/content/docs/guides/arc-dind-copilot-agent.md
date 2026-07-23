---
title: How to run GitHub Copilot coding agent on ARC with Docker-in-Docker
description: Configure Actions Runner Controller with Docker-in-Docker so GitHub Copilot coding agent can run on self-hosted Kubernetes runners.
sidebar:
  order: 440
---

Use this guide to run GitHub Copilot coding agent on an [Actions Runner Controller (ARC)](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners-with-actions-runner-controller/about-actions-runner-controller) runner scale set with Docker-in-Docker (DinD).

For the full self-hosted runner reference (including `runs-on` formats, framework job runners, Docker socket overrides, and GHES), see [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/).

> [!NOTE]
> **Not using ARC?** If you run a custom Kubernetes operator with a DinD sidecar pattern (not ARC), the same principles apply: your pod needs a privileged DinD sidecar, a shared work volume, and `DOCKER_HOST` set to a `tcp://` endpoint. Set `runner.topology: arc-dind` in workflow frontmatter — the topology applies to any DinD sidecar setup, not just ARC specifically. If your Docker socket is at a non-standard path, see [Docker socket override](/gh-aw/reference/self-hosted-runners/#docker-socket-override-for-split-daemon-topologies).

## Prerequisites

Before starting, confirm you have a Kubernetes cluster, `helm` and `kubectl` installed, and credentials for runner registration (a GitHub PAT or GitHub App credentials).

> [!IMPORTANT]
> DinD (`containerMode.type="dind"`) is required for GitHub Copilot coding agent on ARC. Kubernetes mode (`containerMode.type="kubernetes"`) is not supported for this setup.

## 1. Install the ARC controller

```bash
helm install arc \
  --namespace "arc-system" --create-namespace \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set-controller
```

## 2. Create the runner namespace and auth secret

Create the namespace and a Kubernetes secret with your runner registration credentials. You can use either a GitHub PAT or GitHub App credentials:

```bash
kubectl create ns arc-runners

# Option A: Personal access token
kubectl create secret generic arc-runner-secret \
  --namespace=arc-runners \
  --from-literal=github_token=<YOUR_PAT>

# Option B: GitHub App (recommended for production)
kubectl create secret generic arc-runner-secret \
  --namespace=arc-runners \
  --from-literal=github_app_id=<APP_ID> \
  --from-literal=github_app_installation_id=<INSTALL_ID> \
  --from-literal=github_app_private_key=<PRIVATE_KEY>
```

See [Authenticating to the GitHub API](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners-with-actions-runner-controller/authenticating-to-the-github-api) for details on each option.

## 3. Install a runner scale set in DinD mode

```bash
helm install "arc-runner-set" \
  --namespace "arc-runners" --create-namespace \
  --set githubConfigUrl="https://github.com/<OWNER>/<REPO>" \
  --set githubConfigSecret="arc-runner-secret" \
  --set containerMode.type="dind" \
  --set-json 'template.spec.containers=[{
    "name": "runner",
    "image": "ghcr.io/actions/actions-runner:latest",
    "command": ["/home/runner/run.sh"]
  }]' \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set
```

When `containerMode.type="dind"` is enabled, ARC configures the DinD sidecar in privileged mode by default so the Docker daemon can run. The runner container does not require `privileged: true` or `NET_ADMIN`. If you use a custom pod template, ensure you do not remove the privileged setting on the DinD sidecar.

## 4. Verify the runner is online

Open `https://github.com/<OWNER>/<REPO>/settings/actions/runners` (or the organization-level runners page) and confirm the `arc-runner-set` runner is online.

## 5. Target the runner set from a workflow

Set your workflow frontmatter to use the runner scale set label and ARC DinD topology:

```aw
---
on: issues
runs-on: arc-runner-set
runner:
  topology: arc-dind
---
```

`runner.topology: arc-dind` is required so compiled workflows enable ARC DinD split-filesystem handling (a shared runner/daemon workspace root, Docker-daemon-visible mount paths, and ARC-specific sandbox setup). No other sandbox or network settings are needed — the defaults handle everything else.

> [!TIP]
> `runner.topology: arc-dind` enables sysroot staging and tool-cache warnings at compile time. Other ARC-specific behaviors (network isolation, chroot identity, `--docker-host` passthrough) are activated at **runtime** when the compiled workflow detects a `tcp://` value in `DOCKER_HOST`. You do not need to configure these separately.

After editing the frontmatter, recompile the lock file:

```bash
gh aw compile
```

Commit both the `.md` workflow file and the generated `.lock.yml` file.

## 6. How it works

When compiled workflows detect a `tcp://` value in `DOCKER_HOST` (set automatically by ARC DinD), a runtime probe activates ARC DinD handling. It stages a sysroot volume for system binaries, mounts `GITHUB_WORKSPACE` from the shared `/home/runner/_work/` volume so both runner and daemon see the same files, patches the runner UID/GID and home directory into the AWF config, consolidates agent output under `${{ runner.temp }}/gh-aw/` for downstream jobs, and enforces egress through an internal Docker network (`awf-net`) plus a dual-homed Squid proxy. The runner container only talks to the DinD sidecar's Docker API; the daemon creates the networks and applies traffic controls, so no host `iptables` rules are needed.

## Network requirements

ARC DinD runners need outbound HTTPS (port 443) access from the runner pod. If your cluster uses Kubernetes NetworkPolicies or a service mesh, ensure the runner pod can reach:

| Destination | Purpose |
| --- | --- |
| `github.com` (or your GHES instance) | Git clone, API calls, Actions runtime |
| `api.githubcopilot.com` (or your enterprise Copilot endpoint) | Copilot engine communication |
| `ghcr.io` and `pkg-containers.githubusercontent.com` | Pull MCP gateway and AWF container images |
| Domains in your workflow's `network.allowed` list | Agent egress (npm registries, PyPI, etc.) |

### Intra-pod communication

The runner container communicates with the DinD sidecar over `DOCKER_HOST` (typically `tcp://localhost:2375`). This is intra-pod traffic over the loopback interface — no NetworkPolicy is needed for it.

AWF creates Docker networks inside the DinD daemon for sandbox isolation. All agent egress is routed through a Squid proxy container on the `awf-net` Docker network. This is internal to the DinD daemon and invisible to Kubernetes networking.

### What is NOT required

You do not need `NET_ADMIN`, the `iptables` binary, host networking, or a privileged runner container. AWF enforces egress through Docker network topology inside the DinD sidecar, and only that sidecar needs `privileged: true`.

> [!NOTE]
> If you see `iptables`-related output in workflow logs, it does not mean `iptables` is required. In network-isolation mode (the default for `topology: arc-dind`), AWF logs this as informational context but does not execute any `iptables` commands from the runner container.

## 7. Required versions

Use versions at or above these minimums:

| Component | Minimum version | Why |
| --- | --- | --- |
| `gh-aw` | `v0.82.8` | Includes ARC DinD workspace/detection fixes and the MCP gateway Docker socket access fix. |
| AWF (`agentic-workflow-firewall`) | `v0.27.22` | Includes DinD squid log permission fixes. |

If you're on `gh-aw` `v0.82.5`–`v0.82.7`, upgrade and recompile before using this guide:

```bash
gh aw upgrade
gh aw compile
```

## Required and optional configuration

| Item | Required? | Notes |
| --- | --- | --- |
| DinD container mode | **Yes** | GitHub Copilot coding agent needs a Docker daemon in the runner pod. |
| `NET_ADMIN` capability | **No** | Not required. AWF enforces egress via Docker network topology (network isolation mode), not host `iptables`. The DinD sidecar daemon manages all network enforcement internally. |
| `ghcr.io/actions/actions-runner:latest` | Recommended | Use the official runner image, or a compatible custom image with equivalent runner requirements. |
| Runner user | **Yes** | Non-root runner users are supported. By default, `sudo` must be available on the runner container for the Copilot CLI install script. If your cluster enforces `allowPrivilegeEscalation: false`, use the `--rootless` flag — see [Pod security and rootless install](#pod-security-and-rootless-install) below. |
| DinD sidecar privilege | **Yes** | ARC DinD mode configures a privileged sidecar for Docker daemon operation. |
| Shared work volume (`/home/runner/_work`) | **Yes** | Runner and Docker daemon share this volume in ARC DinD mode, so workspace mounts work without host path translation. |
| Specific Kubernetes distribution | **No** | Any conformant cluster works (for example minikube, EKS, AKS, or GKE). |
| Specific namespace names | **No** | `arc-system` and `arc-runners` are conventions only. |

## Tool cache redirection

If your runner image uses the default `RUNNER_TOOL_CACHE` location (`/opt/hostedtoolcache`), tools installed by `setup-*` actions (for example `setup-node`, `setup-python`) will be invisible to the DinD daemon because `/opt` is not on a shared volume.

Redirect the tool cache to a shared path by setting `RUNNER_TOOL_CACHE` in your pod template:

```yaml
# In your runner container spec
env:
  - name: RUNNER_TOOL_CACHE
    value: /tmp/gh-aw/tool-cache
```

The compiled workflow emits a warning at runtime if it detects `RUNNER_TOOL_CACHE` under `/opt`. If you see this warning, apply the redirect and re-run.

## Finding AWF logs

On ARC DinD runners, sandbox logs are written to `$RUNNER_TEMP/gh-aw/sandbox/firewall/logs/`, **not** `/tmp/gh-aw/`. This is because `$RUNNER_TEMP` (typically `/home/runner/_work/_temp`) is on the shared work volume, while `/tmp` may not be.

Key log files:

| Log | Path | Contains |
| --- | --- | --- |
| CLI proxy | `$RUNNER_TEMP/gh-aw/sandbox/firewall/logs/cli-proxy.log` | DIFC proxy connection attempts, DNS resolution errors |
| Squid access | `$RUNNER_TEMP/gh-aw/sandbox/firewall/logs/squid-access.log` | Egress requests (allowed/denied) |
| AWF startup | `$RUNNER_TEMP/gh-aw/sandbox/firewall/logs/awf.log` | Sandbox setup, network isolation, container creation |

> [!NOTE]
> On rootless or non-privileged runner containers, the post-job log permission repair (`chmod -R a+rX`) may fail with `Operation not permitted`. If this happens, the logs are still written but may only be readable inside the container. AWF v0.27.22+ automatically repairs log directory ownership between runs on persistent ARC runners.

## Upgrading from manual workarounds

If you previously added custom bootstrap actions, Copilot shims, `/etc` pre-seeding, XDG overrides, or manual `DOCKER_HOST` / `MCP_GATEWAY_DOMAIN` settings for ARC DinD, remove them before adopting `runner.topology: arc-dind`; the compiler now handles them automatically, and leftover workarounds can conflict with generated steps.

To migrate, remove any ARC-specific `pre-agent-steps`, `resources`, `safe-outputs.threat-detection.steps`, `engine.env` overrides (`XDG_CACHE_HOME`, `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `MCP_GATEWAY_DOMAIN`, `MCP_GATEWAY_PORT`, `DOCKER_HOST`), and `sandbox.agent.mounts` entries that staged files for the DinD daemon. Then add `runner.topology: arc-dind`, run `gh aw compile`, and commit the updated lock file.

## Pod security and rootless install

Clusters that enforce `allowPrivilegeEscalation: false` (via PodSecurity Admission or OPA policies) prevent the default Copilot CLI install script from using `sudo`.

Pass `--rootless` to `install_copilot_cli.sh` in your `copilot-setup-steps` to install to `~/.local/bin` without any `sudo` calls. The script adds that directory to `$GITHUB_PATH` so subsequent steps find the binary:

```yaml
copilot-setup-steps:
  runs-on: arc-runner-set
  steps:
    - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
      with:
        repository: github/gh-aw
        path: gh-aw
    - name: Install Copilot CLI (rootless)
      run: bash "${GITHUB_WORKSPACE}/gh-aw/actions/setup/sh/install_copilot_cli.sh" --rootless
      env:
        GH_HOST: github.com
```

In rootless mode, the binary is installed to `~/.local/bin/copilot`, `~/.local/bin` is appended to `$GITHUB_PATH` for later steps, and ownership or cleanup operations that previously used `sudo` run as the current user instead.

## Known limitations

- **MCP gateway Docker socket access** — on runners where `DOCKER_HOST` is a TCP endpoint and no Unix socket exists at `/var/run/docker.sock`, the MCP gateway may fail to connect to the Docker daemon (`Docker daemon is not accessible`). As a workaround, expose the DinD sidecar's Unix socket on the runner container at `/var/run/docker.sock` via a shared volume or symlink. See [#44251](https://github.com/github/gh-aw/issues/44251) for tracking.

## Troubleshooting

### Agent reports empty workspace

The agent sees no files and exits with a no-op message. This was fixed in gh-aw v0.82.5. Upgrade and recompile:

```bash
gh aw upgrade
gh aw compile
```

### Detection job fails with `spawn /usr/local/bin/copilot ENOENT`

The threat-detection job can't find the Copilot binary. This was fixed in gh-aw v0.82.5 ([#44445](https://github.com/github/gh-aw/pull/44445)). The fix is the same — upgrade and recompile.

### `sudo: The "no new privileges" flag is set`

The runner pod's security context has `allowPrivilegeEscalation: false`. Pass `--rootless` to the Copilot CLI install script so it installs without `sudo` — see [Pod security and rootless install](#pod-security-and-rootless-install) above.

### `awf-cli-proxy could not connect to the external DIFC proxy`

The AWF firewall's CLI proxy cannot reach the DIFC proxy at startup. Check the cli-proxy log at `$RUNNER_TEMP/gh-aw/sandbox/firewall/logs/cli-proxy.log` for details.

**If the log shows `getaddrinfo EAI_AGAIN <hostname>`:** The proxy hostname is a Kubernetes service name (for example `awmg-cli-proxy`) that DinD-spawned containers cannot resolve. Docker containers created by the DinD daemon run on Docker's internal network, which does not have access to Kubernetes cluster DNS. To fix this, ensure the proxy is reachable by IP address or configure DNS forwarding from the DinD Docker network to the Kubernetes DNS resolver.

**If the log shows connection timeouts:** The DinD daemon may not be ready when AWF starts. AWF retries probes up to 10 times with 2-second delays. If this is insufficient, check that the DinD sidecar starts before the runner job begins and that no network policies block communication between the runner container and the DinD sidecar.

### `RUNNER_TOOL_CACHE is under /opt` warning

The compiled workflow detected `RUNNER_TOOL_CACHE` at `/opt/hostedtoolcache`, which is not on a volume shared with the DinD daemon. See [Tool cache redirection](#tool-cache-redirection) to fix this.

### `Docker daemon is not accessible` in MCP gateway

The MCP gateway cannot connect to the Docker daemon. On ARC DinD, `DOCKER_HOST` is typically a TCP endpoint (`tcp://localhost:2375`) and no Unix socket exists at `/var/run/docker.sock`.

Set `GH_AW_DOCKER_SOCK_PATH` and `GH_AW_DOCKER_SOCK_GID` in your runner pod spec to tell the MCP gateway where to find the Docker socket. See [Docker socket override for split-daemon topologies](/gh-aw/reference/self-hosted-runners/#docker-socket-override-for-split-daemon-topologies) for details and a YAML example.

### `chmod: Operation not permitted` on log or audit directories

The post-job cleanup tries to make sandbox logs world-readable but fails on non-root containers. This does not affect workflow execution — the agent has already finished. The logs are still present but may require container-level access to read. AWF v0.27.22+ automatically repairs ownership on persistent ARC runners at the start of each run.

## Related documentation

- [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/) — `runs-on` formats, Docker socket overrides, framework job runners, and GHES compatibility
- [Docker socket override for split-daemon topologies](/gh-aw/reference/self-hosted-runners/#docker-socket-override-for-split-daemon-topologies) — `GH_AW_DOCKER_SOCK_PATH` and `GH_AW_DOCKER_SOCK_GID`
- [ARC Helm charts](https://github.com/actions/actions-runner-controller/tree/master/charts)
- [Rootless Copilot CLI install tracking](https://github.com/github/gh-aw/issues/46046)
