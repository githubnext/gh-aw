---
"gh-aw": minor
---

Add `sandbox.agent.runtime: docker-sbx` frontmatter field for Docker sbx microVM runtime support.

When set to `docker-sbx`, the compiler:

1. Emits fail-fast steps that check KVM availability and Docker Hub secrets (`DOCKER_PAT`, `DOCKER_USERNAME`) before any installation begins
2. Emits a `docker-sbx` installation step that adds the Docker apt repository and installs the `docker-sbx` package
3. Emits an auth and daemon step that starts the `sbx` daemon, authenticates with Docker Hub, resets and re-initialises the allow-all policy, and pre-pulls the `docker/sandbox-templates:shell-docker` image
4. Emits a pre-flight smoke test that creates a throwaway sandbox, runs `uname -a`, and cleans up — confirming the sbx stack is functional before committing to the expensive AWF setup
5. Passes `--container-runtime sbx` to the AWF CLI invocation
6. Adds `host.docker.internal` to `network.allowDomains` so the sbx microVM can reach the api-proxy, MCP gateway, and Squid proxy via the Docker bridge
7. Binds the MCP gateway to `0.0.0.0` (instead of `127.0.0.1`) so the microVM can reach it via `host.docker.internal`
8. Sets `MCP_GATEWAY_HOST_DOMAIN=host.docker.internal` so CLI wrapper scripts generated inside the microVM point to the correct gateway URL

Compile-time validation rejects incompatible combinations:

- `sandbox.agent.runtime: docker-sbx` + `runner.topology: arc-dind` — sbx requires KVM; ARC DinD runners typically lack nested virtualisation
- `sandbox.agent.runtime: docker-sbx` without `sandbox.agent.sudo: true` — the sbx install step requires root access
- `sandbox.agent.runtime: docker-sbx` with an effective AWF version older than `v0.28.0` — `awf --container-runtime sbx` requires AWF support

The `sudo: true` deprecation warning/error is suppressed when `runtime: docker-sbx` is set because sbx fundamentally requires root for installation. Despite requiring `sudo: true`, network isolation is always enabled (`network.isolation: true`) — the sudo flag is only for the install steps, not for network enforcement.
