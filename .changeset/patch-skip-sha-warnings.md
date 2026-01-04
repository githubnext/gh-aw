---
"gh-aw": patch
---

Skip dynamic resolution warnings for actions pinned to full SHAs

Actions pinned to full 40-character commit SHAs should not emit dynamic
resolution warnings. This change updates `GetActionPinWithData()` to
detect SHA-based versions, suppress dynamic resolution warnings for
SHA-pinned actions, and preserve known SHA->version annotations when
available. Tests were added to cover SHA-pinned behavior.

