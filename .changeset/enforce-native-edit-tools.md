---
"gh-aw": patch
---

Enforce native editing policies through shared engine capabilities. `tools.edit: true`, `tools.edit:` and `tools.edit: {}` are equivalent enables. Fix Gemini's disabled editors and Copilot's wildcard permission handling, and keep Copilot SDK process input tools separate from native editing. Gemini's explicit empty `tools.bash: []` allowlist now disables the shell tool instead of granting unrestricted shell access.

Codex and Agy now reject `edit: false` because they cannot enforce it. Custom engine integrations retain native editing support by default, can opt out with `native-edit: false`, and must explicitly declare `native-edit-disable: true` to support a denial. Configured settings are exported through the `GH_AW_TOOLS_EDIT` runtime policy. Omitted edit settings retain existing defaults.
