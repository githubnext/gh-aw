---
"gh-aw": patch
---

Replaced hardcoded `/opt/gh-aw` paths with `${{ runner.temp }}/gh-aw` (and `${RUNNER_TEMP}/gh-aw` in shell contexts) so compiled workflows and setup scripts run correctly on self-hosted runners without `/opt` write access.
