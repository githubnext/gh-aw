---
"gh-aw": patch
---

Fixed the misleading error message in the `push_repo_memory` MCP pre-check. The previous error text incorrectly stated "20% overhead for git diff format" — but the pre-check measures total raw file sizes in the memory directory, not a git diff. The comment claiming it "mirrors the same calculation in push_repo_memory.cjs" was also inaccurate. Both the error message and the internal comments now correctly describe that the check sums raw file sizes and compares them against `max_patch_size × 1.2` (the same factor used by the push gate, so a full rewrite of all memory content stays within the push limit).
