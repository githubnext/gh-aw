---
"gh-aw": minor
---

Add entrypointArgs field to container-type MCP configuration

This adds a new `entrypointArgs` field that allows specifying arguments to be added after the container image in Docker run commands. This provides greater flexibility when configuring containerized MCP servers, following the standard Docker CLI pattern where arguments can be placed before the image (via `args`) or after the image (via `entrypointArgs`).
