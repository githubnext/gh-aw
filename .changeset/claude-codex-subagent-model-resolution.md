---
"gh-aw": patch
---

Claude inline sub-agents are now registered by Claude Code: their generated agent files get the required `name`. Claude and Codex resolve and record each declared sub-agent model before the engine starts. A workflow-declared Claude sub-agent whose model is unavailable stops the run before Claude starts. Codex warns that it does not load the Markdown sub-agents. `aw_info.sub_agent_models` now lists Claude and Codex declarations.
