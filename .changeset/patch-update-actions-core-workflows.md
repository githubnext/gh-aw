---
"gh-aw": patch
---

The `update` command now always bumps `actions/*` core actions to the latest major version and refreshes any `uses: actions/*` references or SHA pins inside workflow steps, so workflows stay aligned without manual edits.
