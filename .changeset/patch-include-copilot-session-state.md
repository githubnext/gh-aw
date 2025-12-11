---
"gh-aw": patch
---

Include Copilot session-state file in agent output artifacts

The Copilot CLI session-state file at `~/.copilot/session-state` is now copied
to `/tmp/gh-aw/sandbox/agent/session-state.json` and included in the
`agent_outputs` artifact. This improves debugging and allows inspection of
Copilot session state alongside agent logs.

