---
"gh-aw": patch
---

Fixed cached GitHub Copilot CLI activation to always install the `/usr/local/bin/copilot` wrapper for toolcache hits, including GitHub Actions runs that only export `GITHUB_PATH`.
