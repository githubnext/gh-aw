---
"gh-aw": patch
---

Fixed threat detection workflow compilation by skipping the detection job when `threat-detection.engine` is disabled with no custom steps, and by always including `aw-*.patch` files in agent artifacts when threat detection is enabled.
