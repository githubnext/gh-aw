---
"gh-aw": major
---

Replace `safe-outputs.create-pull-request.pre-create` with `safe-outputs.steer`, which creates a run-scoped issue for user feedback and reuses that issue for agent failure reporting.

**⚠️ Breaking Change**: Workflows that still set `safe-outputs.create-pull-request.pre-create` now fail validation.

**Migration guide:** Remove `pre-create: true`, add `steer: true` directly under `safe-outputs`, and grant top-level `issues: read`. Steering now uses the global safe-output credential, so move or duplicate any `create-pull-request.github-token` or `create-pull-request.github-app` configuration under `safe-outputs.github-token` or `safe-outputs.github-app`.
