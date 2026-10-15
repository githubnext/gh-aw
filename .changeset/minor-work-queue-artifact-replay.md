---
"gh-aw": minor
---

Add read-only work-queue replay from an exact Actions activation artifact or a downloaded snapshot, with captured scheduling predictions. Leave dispatch API permission enforcement to GitHub instead of inferring token access from repository-role metadata, and retain durable grants when launch processing fails. Queue diagnostics and replay errors explain the problem and safe recovery steps without exposing credentials or task data.
