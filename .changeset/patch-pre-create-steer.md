---
"gh-aw": major
---

Replace `safe-outputs.create-pull-request.pre-create` with `safe-outputs.steer`, which creates a run-scoped issue for user feedback and reuses that issue for agent failure reporting.

**⚠️ Breaking Change**: Workflows that still set `safe-outputs.create-pull-request.pre-create` now fail validation.

**Migration guide:** Remove `pre-create: true`, add `steer: true` directly under `safe-outputs`, and grant top-level `issues: read`.
