---
"gh-aw": patch
---

Fixed Copilot engine "terminated before producing output" failures on runners with a toolcache hit by always installing a `/usr/local/bin/copilot` wrapper when activating a cached Copilot CLI.
