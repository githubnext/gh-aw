---
"gh-aw": patch
---

Forward recorded pull request head baseline metadata through the MCP gateway and safe-outputs containers so `push_to_pull_request_branch` can generate incremental updates for fork PRs. Reject unconfigured contributor-fork pushes during the agent's safe-output call with guidance to report the proposed change to maintainers.
