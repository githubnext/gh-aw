---
"gh-aw": patch
---

Include Copilot session-state file in agent output artifacts. The Copilot CLI session-state at `~/.copilot/session-state` is copied to `/tmp/gh-aw/sandbox/agent/session-state.json` and uploaded as part of the `agent_outputs` artifact to improve debugging and state inspection.

