---
"gh-aw": patch
---

Fix fleet-wide smoke-test outage caused by a Docker image tag mismatch: `download_docker_images.sh` now aliases every pulled image under the mutable `:latest` tag in addition to its version-pinned tag, so `docker compose up -d --pull never` can resolve `ghcr.io/github/gh-aw-firewall/cli-proxy` (and other AWF images) regardless of which tag they are referenced by.
