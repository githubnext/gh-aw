---
"gh-aw": patch
---

Lockdown validation now runs inside the activation job's "Generate agentic run info" step so the required env vars are injected and failures surface before the agent job starts.
