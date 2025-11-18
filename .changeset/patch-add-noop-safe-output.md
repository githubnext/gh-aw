---
"gh-aw": patch
---

Add noop safe output for transparent workflow completion

Agents need to emit human-visible artifacts even when no actions are required (e.g., "No issues found"). The noop safe output provides a fallback mechanism ensuring workflows never complete silently.
