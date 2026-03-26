---
"gh-aw": patch
---

Fixed a heredoc delimiter injection vulnerability in workflow compilation by randomizing generated delimiters and normalizing randomized tokens for stable lockfile skip-write comparisons.
