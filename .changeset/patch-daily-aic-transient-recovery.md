---
"gh-aw": patch
---

Retry transient daily AI Credits accounting reads before conservatively estimating unavailable historical usage, preserve fail-closed quota and permission handling, and restore trusted default-branch issue-event scan snapshots. Report rate-limit reset deadlines only for quota stops. Add safe diagnostics for retry outcomes and snapshot validation, and verify daily accounting fallbacks with focused Vitest contract cases.

Close specification gaps in per-endpoint history retries and quota observation, malformed history rejection, interrupted artifact body recovery, zero-alias token pricing, and finite aggregate accounting with confirmed-zero run statistics.
