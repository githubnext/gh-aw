---
"gh-aw": patch
---

Add a Copilot CLI driver wrapper that retries partial-session failures with `--resume`, improving reliability when transient mid-session errors (including CAPIError 400) occur after output has already been produced.
