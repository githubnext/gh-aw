---
"gh-aw": patch
---

Add support for configuring default agent and detection models via GitHub Actions
variables. This exposes the following variables for workflows:

- `GH_AW_MODEL_AGENT_COPILOT`, `GH_AW_MODEL_AGENT_CLAUDE`, `GH_AW_MODEL_AGENT_CODEX`
- `GH_AW_MODEL_DETECTION_COPILOT`, `GH_AW_MODEL_DETECTION_CLAUDE`, `GH_AW_MODEL_DETECTION_CODEX`

These variables provide configurable defaults for agent execution and threat
detection models without changing workflow frontmatter.

