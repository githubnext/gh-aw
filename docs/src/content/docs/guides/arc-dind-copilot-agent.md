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

```bash
kubectl create ns arc-runners
kubectl create secret generic arc-runner-secret \
  --namespace=arc-runners \
  --from-literal=github_token=<YOUR_PAT>
```

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

`NET_ADMIN` is required so the runner can support sandbox network behavior in DinD mode.

## 4. Verify the runner is online

Open `https://github.com/<OWNER>/<REPO>/settings/actions/runners` (or the organization-level runners page) and confirm the `arc-runner-set` runner is online.

## 5. Target the runner set from a workflow

Set your workflow frontmatter to use the runner scale set label:

```aw
---
on: issues
runs-on: arc-runner-set
---
```

## Required and optional configuration

| Item | Required? | Notes |
| --- | --- | --- |
| DinD container mode | **Yes** | GitHub Copilot coding agent needs a Docker daemon in the runner pod. |
| `NET_ADMIN` capability | **Yes** | Required for network operations in the DinD topology. |
| `ghcr.io/actions/actions-runner:latest` | Recommended | Use the official runner image, or a compatible custom image with equivalent runner requirements. |
| Rootless execution | Assumed | The official runner image default is rootless. |
| Specific Kubernetes distribution | **No** | Any conformant cluster works (for example minikube, EKS, AKS, or GKE). |
| Specific namespace names | **No** | `arc-system` and `arc-runners` are conventions only. |

## Related documentation

- [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/)
- [ARC Helm charts](https://github.com/actions/actions-runner-controller/tree/master/charts)
