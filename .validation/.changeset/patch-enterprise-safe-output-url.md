---
"gh-aw": patch
---

Ensure the safe output handlers and helpers build URLs from the configured GitHub server (e.g., GITHUB_SERVER_URL) so enterprise hosts no longer hit https://github.com references that break authentication.
