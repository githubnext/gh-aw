---
"gh-aw": patch
---

Fix `gh aw compile --use-samples` replay failing with `Failed to pin branch '...': ERR_SYSTEM: fatal: Needed a single revision` when a `create_pull_request` / `push_to_pull_request_branch` sample patch targets a cross-repo checkout placed in a subdirectory (`path: github`). `apply_samples.cjs` now stages the sample branch and commit inside the target repo's checkout subdirectory — resolved via the same checkout manifest the safe-outputs MCP handler uses — instead of always staging in the main workspace root, so the MCP server can find and pin the branch.
