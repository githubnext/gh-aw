---
"gh-aw": patch
---

Add support for configuring default agent and detection models through GitHub Actions variables.

This change introduces environment variables for agent execution and threat detection (e.g.
`GH_AW_MODEL_AGENT_COPILOT`, `GH_AW_MODEL_DETECTION_COPILOT`, etc.), updates workflow
YAML generation to inject those variables, and ensures explicit frontmatter configuration
still takes precedence. No breaking CLI changes.
