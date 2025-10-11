---
"gh-aw": patch
---

Security Fix: Allocation Size Overflow in Bash Tool Merging (Alert #7)

Fixed a potential allocation size overflow vulnerability (CWE-190) in the workflow compiler's bash tool merging logic. The fix implements input validation, overflow detection, and reasonable limits to prevent integer overflow when computing capacity for merged command arrays. This is a preventive security fix that maintains backward compatibility with no breaking changes.
