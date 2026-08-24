---
"gh-aw": major
---

Replace `safe-outputs.create-pull-request.pre-create` with `steer`, which pre-creates a pull request and lets the run ingest user feedback from pull request comments.

**⚠️ Breaking Change**: Workflows that still set `safe-outputs.create-pull-request.pre-create` now fail validation.

**Migration guide:** Replace `pre-create: true` with `steer: true`.
