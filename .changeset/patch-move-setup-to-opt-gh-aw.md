---
"gh-aw": patch
---

Move action setup and compiler paths from `/tmp/gh-aw` to `/opt/gh-aw` so agent access is
read-only; updates setup action, compiler constants, tests, and AWF mounts.

This is an internal/tooling change and does not change the public API.

