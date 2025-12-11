---
"gh-aw": patch
---

Display agent response text in action logs as a conversation-style summary.

This change updates the action log rendering so agent replies are shown
inline with tool calls, making logs read like a chat conversation. Agent
responses are prefixed with "Agent:", tool calls use ✓/✗, shell commands
are shown as `$ command`, and long outputs are truncated to keep logs
concise.

This is an internal, non-breaking improvement to log formatting.

