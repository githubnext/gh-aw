---
"gh-aw": patch
---

Fix `buildEngineFailureContext` to filter AWF infrastructure lines (container lifecycle, firewall wrapper) from the fallback "last agent output" display. When the agent-stdio.log contains only infrastructure lines — a pattern characteristic of transient startup failures (service unavailable, rate limiting) — a dedicated message is now shown instead of confusing container shutdown noise.
