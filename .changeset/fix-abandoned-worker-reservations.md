---
"gh-aw": patch
---

Recover never-started work-queue worker reservations after their exact originating run attempt finishes. Preserve reservations for active or unverifiable origins and possibly started launches, allowing cancelled ESLint factory dispatches to stop blocking subsequent scheduling.
