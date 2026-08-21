---
"gh-aw": patch
---

Restore two properties of the threat-detection job on the external `threat-detect` path. The detection execution step is again bounded by a step-level `timeout-minutes` (aligned with the `GH_AW_TIMEOUT_MINUTES` value it already exported), so a stall before the binary reaches its own timeout logic no longer runs up to the 360 minute GitHub default. The detection job's AWF binary install now passes `--rootless` when its own default Docker/rootless profile invokes `awf` that way, matching the detection invocation rather than the main agent job.
