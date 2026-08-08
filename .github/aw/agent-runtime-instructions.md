---
description: Choose and configure agent runtimes for GitHub Agentic Workflows.
disable-model-invocation: true
---

# Agent Runtime Instructions

Use these instructions when creating or updating workflows that mention Docker, gVisor, Docker sbx, ARC DinD, self-hosted runners, or `sandbox.agent.runtime-install`.

## Runtime fields

- Omit `sandbox.agent.runtime` for the default Docker agent runtime.
- Set `sandbox.agent.runtime: gvisor` only when the runner has a local Docker daemon and can install or already has `runsc`.
- Set `sandbox.agent.runtime: docker-sbx` only when the runner supports KVM-backed microVMs.
- Do not set `sandbox.agent.runtime: docker`; Docker is selected by omitting the field.
- Do not set `sandbox.agent.runtime: sbx`; `sbx` is only a bounded-query runtime name.
- Set `runner.topology: arc-dind` for ARC or equivalent Kubernetes runners that use a Docker-in-Docker sidecar. This is a runner topology, not an agent runtime.

## Compatibility

- Do not combine `runner.topology: arc-dind` with `sandbox.agent.runtime: gvisor` or `sandbox.agent.runtime: docker-sbx`.
- ARC DinD workflows must be rootless: do not add `sudo`, `apt-get install`, or other host package bootstrap steps.
- Docker sbx requires KVM and normally does not work on ARC DinD because the sbx daemon must run on the runner host.

## `runtime-install`

- `sandbox.agent.runtime-install` defaults to `true` for gVisor and Docker sbx provisioning.
- Set `runtime-install: false` only when the runner image or pod is pre-provisioned with the runtime and required daemon or policy.
- When any imported workflow sets `runtime-install: false`, false wins during import merging.
- With `runtime-install: false`, gh-aw skips generated runtime checks and setup, so the runner must already satisfy those prerequisites.

## gVisor guidance

- gVisor uses `runsc` for the agent container while AWF infrastructure containers continue to use Docker.
- The generated gVisor installer may use `sudo`, but do not set `sandbox.agent.sudo: true` merely for gVisor.
- Use gVisor when stronger kernel isolation is needed and the workload is compatible with gVisor syscall behavior.

## Docker sbx guidance

- Docker sbx runs the agent in a KVM-backed microVM and requires a KVM-capable Linux runner.
- With runtime installation enabled, set `sandbox.agent.sudo: true` because gh-aw installs `docker-sbx`, adjusts `/dev/kvm`, starts the sbx daemon, authenticates CLIs, pulls the template, and runs a smoke test.
- Docker sbx requires both `DOCKER_USERNAME` and `DOCKER_PAT` Actions secrets. `DOCKER_PAT` must be a Docker Hub personal access token that can authenticate Docker Hub pulls for the sandbox template.
- `DOCKER_USERNAME` and `DOCKER_PAT` remain required even with `runtime-install: false`, because compiled workflows refresh sbx credentials immediately before agent execution.
- Do not use Docker sbx for workflows triggered from untrusted forks unless the trigger and credential model safely provide those secrets.

## ARC DinD guidance

- Use `runner.topology: arc-dind` when `DOCKER_HOST` points to a DinD sidecar such as `tcp://localhost:2375` or `tcp://dind:2375`.
- Ensure the runner container and DinD sidecar share `/home/runner/_work`.
- Use a daemon-visible tool cache path such as `/tmp/gh-aw/tool-cache`, not `/opt/hostedtoolcache`.
- If the Docker socket is bind-mounted at a nonstandard path, set `GH_AW_DOCKER_SOCK_PATH`. Set `GH_AW_DOCKER_SOCK_GID` only when group detection with `stat` fails.
