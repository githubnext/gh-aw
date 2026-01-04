---
"gh-aw": patch
---

Migrate detection job artifacts to the unified `/tmp/gh-aw/artifacts` path and add validations.

- Update artifact download paths used by detection jobs to `/tmp/gh-aw/artifacts`.
- Fail fast when `prompt.txt` or `agent_output.json` are missing.
- Fail when `aw.patch` is expected but not present.

This is an internal tooling fix and non-breaking (patch).

