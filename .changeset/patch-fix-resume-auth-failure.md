---
"gh-aw": patch
---

Fix copilot-driver `--continue` auth failure: when "No authentication information found" occurs on a `--continue` retry attempt (session credential may be corrupted by mid-stream exit), fall back to a fresh run instead of bailing immediately, giving the job a recovery path via env-var auth.
