---
"gh-aw": patch
---

Fixed Copilot assignment behavior for `create-issue` and `create-pull-request` by making assignment state handling concurrency-safe and assigning Copilot inline during issue creation paths. Also updated compiler safe-outputs environment/token wiring so Copilot assignment receives the required token when configured.
