---
"gh-aw": patch
---

Use the unified `agent-artifacts` artifact for downloads and remove duplicate
artifact downloads. Updated tests to expect `agent-artifacts` and removed
dead/unused artifact upload helpers.

This is an internal fix that consolidates artifact downloads used by
safe_outputs and threat detection jobs.

