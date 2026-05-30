---
"gh-aw": patch
---

Fix copilot-driver: detect null-type tool_call 400 error and restart fresh instead of retrying with `--continue`. A malformed tool call with `type: null` poisons the conversation history; retrying via `--continue` re-injects the same broken state and fails identically on every attempt. This change restarts fresh to discard the poisoned history and permanently disables `--continue` for the remainder of the run so the corrupt state can never be reloaded.
