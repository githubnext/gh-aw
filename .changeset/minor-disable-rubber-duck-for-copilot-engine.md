---
"gh-aw": minor
---

Disable rubber-duck sub-agent when Copilot is the engine. Before running the Copilot CLI, write `{"builtInAgents":{"rubberDuck":false}}` to `~/.copilot/settings.json` so the CLI skips proactive plan-critique calls, reducing token overhead and latency for every Copilot-engine workflow run.
