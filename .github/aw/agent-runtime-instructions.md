---
description: Choose and configure agent runtimes for GitHub Agentic Workflows.
disable-model-invocation: true
---

# Agent Runtime Instructions

Use these instructions when creating or updating workflows that mention Docker, gVisor, Docker sbx, Cloud Hypervisor, ARC DinD, self-hosted runners, or `sandbox.agent.runtime-install`.

## Runtime fields

- Omit `sandbox.agent.runtime` for the default Docker agent runtime.
- Set `sandbox.agent.runtime: gvisor` only when the runner has a local Docker daemon and can install or already has `runsc`.
- Set `sandbox.agent.runtime: docker-sbx` only when the runner supports KVM-backed microVMs.
- Set `sandbox.agent.runtime: cloud-hypervisor` only for the preview microVM runtime on a GitHub-hosted Ubuntu x86_64 runner with `/dev/kvm`; prefer `docker-sbx` or `gvisor` when those host constraints are not guaranteed.
- Do not set `sandbox.agent.runtime: docker`; Docker is selected by omitting the field.
- Do not set `sandbox.agent.runtime: sbx`; `sbx` is not a valid `sandbox.agent.runtime` value.
- Set `runner.topology: arc-dind` for ARC or equivalent Kubernetes runners that use a Docker-in-Docker sidecar. This is a runner topology, not an agent runtime.

## Compatibility

- Do not combine `runner.topology: arc-dind` with `sandbox.agent.runtime: gvisor`, `sandbox.agent.runtime: docker-sbx`, or `sandbox.agent.runtime: cloud-hypervisor`.
- ARC DinD workflows must be rootless: do not add `sudo`, `apt-get install`, or other host package bootstrap steps.
- Docker sbx requires KVM and normally does not work on ARC DinD because the sbx daemon must run on the runner host.
- Cloud Hypervisor requires `RUNNER_ENVIRONMENT=github-hosted`, Ubuntu Linux x86_64, and `/dev/kvm`; it is not supported on self-hosted or ARC DinD runners.

## `runtime-install`

- `sandbox.agent.runtime-install` defaults to `true` for gVisor and Docker sbx provisioning.
- Set `runtime-install: false` only when the runner image or pod is pre-provisioned with the runtime and required daemon or policy.
- When any imported workflow sets `runtime-install: false`, false wins during import merging.
- With `runtime-install: false`, gh-aw skips generated runtime checks and setup, so the runner must already satisfy those prerequisites.

## gVisor guidance

- gVisor uses `runsc` for the agent container while AWF infrastructure containers continue to use Docker.
- The generated gVisor installer may use host `sudo`; the compiler derives that from `runtime: gvisor`. There is no `sandbox.agent.sudo` field.
- Use gVisor when stronger kernel isolation is needed and the workload is compatible with gVisor syscall behavior.

## Docker sbx guidance

- Docker sbx runs the agent in a KVM-backed microVM and requires a KVM-capable Linux runner.
- With runtime installation enabled, gh-aw installs `docker-sbx`, adjusts `/dev/kvm`, starts the sbx daemon, authenticates CLIs, pulls the template, and runs a smoke test. The compiler derives the required host privileges from `runtime: docker-sbx`.
- Docker sbx requires both `DOCKER_USERNAME` and `DOCKER_PAT` Actions secrets. `DOCKER_PAT` must be a Docker Hub personal access token that can authenticate Docker Hub pulls for the sandbox template.
- `DOCKER_USERNAME` and `DOCKER_PAT` remain required even with `runtime-install: false`, because compiled workflows refresh sbx credentials immediately before agent execution.
- Do not use Docker sbx for workflows triggered from untrusted forks unless the trigger and credential model safely provide those secrets.

## Cloud Hypervisor guidance (preview)

- Preview scope is narrow: GitHub-hosted runners only, Ubuntu Linux x86_64 only, and `/dev/kvm` must be present.
- The compiler emits host preflight and release-asset provisioning steps that download and checksum-verify the pinned Cloud Hypervisor binary, `virtiofsd`, kernel, rootfs, and supervisor from the `gh-aw-firewall` release before AWF starts, and grants only the runner user scoped read/write access to `/dev/kvm`.
- AWF launches with the host privileges required to create the VM but keeps strict network isolation; the guest defaults to 2 vCPUs and 4096 MiB, and its trusted topology attachment is limited to the MCP gateway on TCP 8080 (no CLI proxy).
- Not supported under Cloud Hypervisor: `tools.github.mode: gh-proxy`, the `integrity-reactions` feature, `sandbox.agent.allow-host-ports`, GitHub Actions `services:` with published ports, and `enclaves:` configuration.
- Do not recommend this runtime for self-hosted, non-Ubuntu, or non-x86_64 runners; use `docker-sbx` or `gvisor` instead.

## ARC DinD guidance

- Use `runner.topology: arc-dind` when `DOCKER_HOST` points to a DinD sidecar such as `tcp://localhost:2375` or `tcp://dind:2375`.
- Ensure the runner container and DinD sidecar share `/home/runner/_work`.
- Use a daemon-visible tool cache path such as `/tmp/gh-aw/tool-cache`, not `/opt/hostedtoolcache`.
- If the Docker socket is bind-mounted at a nonstandard path, set `GH_AW_DOCKER_SOCK_PATH`. Set `GH_AW_DOCKER_SOCK_GID` only when group detection with `stat` fails.
