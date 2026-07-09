# ADR-44446: Robust Docker Socket GID Resolution for ARC/DinD and Split-Daemon Topologies

**Date**: 2026-07-09
**Status**: Accepted
**Deciders**: pelikhan (author)

---

### Context

The MCP gateway setup shell script emitted by `generateMCPGatewaySetup` determines the Docker socket path and group ID (GID) so it can pass `--group-add <GID>` when launching the gateway container. On standard GitHub-hosted runners, this works because `/var/run/docker.sock` exists and `stat -c '%g'` returns a usable GID.

On ARC (Actions Runner Controller) runners with a Docker-in-Docker (DinD) sidecar, `DOCKER_HOST` is a TCP address (e.g., `tcp://localhost:2375`). The socket is exposed by the sidecar container, not as a Unix socket on the runner filesystem. The previous code fell back unconditionally to `/var/run/docker.sock` when `DOCKER_HOST` was not a `unix://` or absolute path, and then silently substituted GID `0` (root) when `stat` failed. This produced a non-root container with no Docker socket access and a confusing "Docker daemon is not accessible" error with no indication of the root cause.

Two additional bugs compounded the problem: `stat -c '%g'` reads the symlink's own GID rather than following it to the real socket, and `|| echo '0'` made every failure silent by substituting the root GID.

### Decision

We will extend the MCP gateway setup shell snippet to support two explicit operator overrides — `GH_AW_DOCKER_SOCK_PATH` and `GH_AW_DOCKER_SOCK_GID` — and will replace the silent `--group-add 0` fallback with a hard `exit 1` accompanied by an actionable `::error::` message naming the path that was tried and both override variables. In addition, `stat -c '%g'` is replaced with `stat -Lc '%g'` to correctly resolve symlinked sockets. The automatic detection from `DOCKER_HOST` is retained as a best-effort default when neither override is set.

### Alternatives Considered

#### Alternative 1: Require Explicit Socket Configuration — Remove Auto-Detection

Remove automatic socket path and GID detection entirely. Operators must always supply `GH_AW_DOCKER_SOCK_PATH` and `GH_AW_DOCKER_SOCK_GID`. This is the simplest possible shell logic and eliminates all inference ambiguity.

Not chosen because it would break all existing configurations on standard GitHub-hosted runners where `/var/run/docker.sock` and `stat` work correctly and operators have no need to set overrides. The migration cost is too high relative to the benefit.

#### Alternative 2: Detect TCP Hosts and Interrogate the Sidecar Daemon Directly

When `DOCKER_HOST` is a TCP address, use `docker info` or a direct API call against the daemon to retrieve the socket GID of the container running the daemon. This would make ARC/DinD work without any operator configuration.

Not chosen because it requires the Docker CLI to be available and authenticated before the gateway container starts, introduces a circular dependency (we're trying to set up Docker access), and is fragile across DinD image variants that expose different API surface. The env-var override achieves the same operator outcome with far less complexity.

### Consequences

#### Positive
- Operators on ARC/DinD and other split-daemon topologies can configure Docker socket access reliably without symlink hacks on the host.
- When resolution fails, the runner job fails immediately with an actionable error message naming the socket path tried and both override variables — debugging time drops from hours to minutes.
- `stat -Lc '%g'` correctly handles symlinked sockets, removing the need for `chown -h` workarounds.
- The existing zero-configuration experience on standard runners is preserved.

#### Negative
- Operators who previously relied on the silent `--group-add 0` fallback — even if their workflows were silently broken — will now see an explicit job failure where they saw none before. This is a breaking change in observable behavior.
- ARC/DinD operators must set two new environment variables (`GH_AW_DOCKER_SOCK_PATH`, `GH_AW_DOCKER_SOCK_GID`) on their runner; the correct values are topology-specific and are not documented in any automatic way.

#### Neutral
- All `.lock.yml` files and wasm golden files are regenerated as part of this change, producing a large but mechanical diff across the repository.
- The override check order (explicit env var → `DOCKER_HOST` inference → default `/var/run/docker.sock`) establishes a precedence convention that future detection logic should follow.

---
