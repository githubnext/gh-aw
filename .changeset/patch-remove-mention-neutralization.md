---
"gh-aw": patch
---

Remove mention neutralization/sanitization code from `compute_text.cjs`.

Switched from `sanitizeContent` to `sanitizeIncomingText`, removed the
dead `knownAuthors` collection and the `isPayloadUserBot` import. Tests
were updated to match the simplified sanitization behavior.

