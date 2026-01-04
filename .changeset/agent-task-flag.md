---
"gh-aw": patch
---

Add --agent-task flag to logs command to include GitHub Copilot agent task runs in addition to agentic workflows. By default, the logs command now only downloads agentic workflows created by `gh aw compile`. Use the new `--agent-task` flag to also include workflows from `gh agent task`.
