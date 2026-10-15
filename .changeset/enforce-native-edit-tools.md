---
"gh-aw": patch
---

Enforce explicit `tools.edit: true` and `tools.edit: false` through shared native-edit engine capabilities. Fix Gemini's disabled editors and Copilot's wildcard permission handling, and keep Copilot SDK process input tools separate from native editing.

Codex and Agy now reject `edit: false` because they cannot enforce it. Custom engine integrations must declare their native editing capabilities and enforce the `GH_AW_TOOLS_EDIT` runtime policy to support explicit booleans. Omitted settings and legacy `edit:` / `edit: {}` forms retain existing defaults.
