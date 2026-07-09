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

## 6. Required versions

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

> [!WARNING]
> ARC configurations that enforce `allowPrivilegeEscalation: false` (including `no-new-privileges` policy enforcement) are not currently supported for GitHub Copilot coding agent, because the setup flow requires `sudo`.

## Related documentation

- [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/)
- [ARC Helm charts](https://github.com/actions/actions-runner-controller/tree/master/charts)
