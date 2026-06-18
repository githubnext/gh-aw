---
"gh-aw": patch
---

Fix `create_pull_request` and `push_to_pull_request_branch` failing with `Repository '<owner>/<repo>' not found in workspace` after the safe-outputs MCP server moved into a container. The checkout manifest is now written to and read from `$RUNNER_TEMP/gh-aw/safeoutputs/checkout-manifest.json` instead of `$RUNNER_TEMP/gh-aw/checkout-manifest.json`, so it lives inside the `safeoutputs/` directory that is bind-mounted into the containerized safe-outputs MCP server. Previously the manifest was a sibling of the mounted directory and therefore invisible inside the container, causing manifest-first checkout resolution to fail.
