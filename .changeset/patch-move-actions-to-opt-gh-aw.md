---
"gh-aw": patch
---

Move setup action files and compiler paths from `/tmp/gh-aw` to `/opt/gh-aw` so they are readonly to the agent.

This updates the setup action, compiler constants, tests, and AWF mounts to use `/opt/gh-aw`.

