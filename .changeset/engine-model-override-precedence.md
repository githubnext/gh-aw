---
"gh-aw": major
---

`engine.model` is no longer deprecated and no longer emits a compile-time warning. A nested `engine.model` (e.g. `safe-outputs.threat-detection.engine.model`) now takes precedence over the top-level `model` field for that engine instance, so split-engine workflows (different models for the main agent and threat detection) can set both without any warning. The `gh aw fix` codemod that migrated `engine.model` to the top-level `model` field has been removed, since automatic migration is unsafe when different engine instances need different models.

**Breaking change:** Workflows that set both fields now use `engine.model`. To retain the former top-level precedence, remove `engine.model`; alternatively, set both fields to the same model.
