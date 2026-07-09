---
title: How to run GitHub Copilot coding agent on ARC with Docker-in-Docker
description: Configure Actions Runner Controller with Docker-in-Docker so GitHub Copilot coding agent can run on self-hosted Kubernetes runners.
sidebar:
  order: 440
---

Use this guide to run GitHub Copilot coding agent on an [Actions Runner Controller (ARC)](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners-with-actions-runner-controller/about-actions-runner-controller) runner scale set with Docker-in-Docker (DinD).

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
    "command": ["/home/runner/run.sh"],
    "securityContext": {
      "capabilities": {
        "add": ["NET_ADMIN"]
      }
    }
  }]' \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set
```

`NET_ADMIN` is required on the **runner container** so AWF can apply host-level `iptables` rules to the `DOCKER-USER` chain for egress filtering.

When `containerMode.type="dind"` is enabled, ARC configures the DinD sidecar in privileged mode by default so the Docker daemon can run. If you use a custom pod template, ensure you do not remove that privileged setting.

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

After editing the frontmatter, recompile the lock file:

```bash
gh aw compile
```

Commit both the `.md` workflow file and the generated `.lock.yml` file.

## 6. How it works

When compiled workflows detect a `tcp://` value in `DOCKER_HOST` (set automatically by ARC DinD), a runtime probe activates ARC DinD handling:

- **Sysroot staging** — system binaries (`/usr`, `/lib`, `/bin`, `/sbin`) are copied into a Docker named volume so the Docker daemon can provide them to the agent container without bind-mounting the runner's filesystem.
- **Workspace mount** — the checked-out repository at `GITHUB_WORKSPACE` is explicitly mounted into the agent container. Both runner and daemon can see it because ARC DinD shares the `/home/runner/_work/` volume.
- **Chroot identity** — the runner's UID/GID and home directory are patched into the AWF config so the agent runs with the correct identity inside the chroot.
- **Artifact consolidation** — agent output files are consolidated under `${{ runner.temp }}/gh-aw/` before upload so downstream jobs (detection, safe-outputs) can find them.

## 7. Required versions

Use versions at or above these minimums:

| Component | Minimum version | Why |
| --- | --- | --- |
| `gh-aw` | `v0.82.5` | Includes ARC DinD workspace and detection fixes. |
| AWF (`agentic-workflow-firewall`) | `v0.27.22` | Includes DinD squid log permission fixes. |

## Required and optional configuration

| Item | Required? | Notes |
| --- | --- | --- |
| DinD container mode | **Yes** | GitHub Copilot coding agent needs a Docker daemon in the runner pod. |
| `NET_ADMIN` capability | **Yes** | Required on the runner container so AWF can manage host-level `DOCKER-USER` `iptables` rules. |
| `ghcr.io/actions/actions-runner:latest` | Recommended | Use the official runner image, or a compatible custom image with equivalent runner requirements. |
| Runner user | **Yes** | Non-root runner users are supported, but `sudo` must remain available on the runner host for AWF setup operations. |
| DinD sidecar privilege | **Yes** | ARC DinD mode configures a privileged sidecar for Docker daemon operation. |
| Shared work volume (`/home/runner/_work`) | **Yes** | Runner and Docker daemon share this volume in ARC DinD mode, so workspace mounts work without host path translation. |
| Specific Kubernetes distribution | **No** | Any conformant cluster works (for example minikube, EKS, AKS, or GKE). |
| Specific namespace names | **No** | `arc-system` and `arc-runners` are conventions only. |

## Upgrading from manual workarounds

If you previously used custom bootstrap actions, copilot shims, `/etc` pre-seeding, XDG environment overrides, or manual `DOCKER_HOST` / `MCP_GATEWAY_DOMAIN` settings to run on ARC DinD, remove them when adopting `runner.topology: arc-dind`. The compiler now handles all of these automatically. Leftover workarounds may conflict with the generated workflow steps.

To migrate:

1. Remove any `pre-agent-steps`, `resources`, or `safe-outputs.threat-detection.steps` blocks that were workarounds for ARC DinD.
2. Remove manual `engine.env` overrides for `XDG_CACHE_HOME`, `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `MCP_GATEWAY_DOMAIN`, `MCP_GATEWAY_PORT`, and `DOCKER_HOST`.
3. Remove `sandbox.agent.mounts` entries that staged files for the DinD daemon.
4. Add `runner.topology: arc-dind` to frontmatter.
5. Run `gh aw compile` and commit the updated lock file.

## Known limitations

- **`allowPrivilegeEscalation: false` is not supported.** The Copilot CLI install script uses `sudo`. Clusters that enforce `no-new-privileges` via PodSecurity Admission or OPA policies will fail at the install step.
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

The runner pod's security context has `allowPrivilegeEscalation: false`. Remove that constraint or adjust your PodSecurity policy to allow privilege escalation in the runner container.

### `Docker daemon is not accessible` in MCP gateway

The MCP gateway can't reach the Docker socket. Ensure a Unix socket is available at `/var/run/docker.sock` on the runner container. For DinD setups where the daemon only exposes a TCP endpoint, share the sidecar's socket file via a volume:

```yaml
# In your custom runner pod template
volumes:
  - name: dind-sock
    emptyDir: {}
# Mount in both runner and DinD sidecar containers
volumeMounts:
  - name: dind-sock
    mountPath: /var/run
```

## Related documentation

- [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/)
- [ARC Helm charts](https://github.com/actions/actions-runner-controller/tree/master/charts)
