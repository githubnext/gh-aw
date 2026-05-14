---
"gh-aw": patch
---

Token steering (`apiProxy.enableTokenSteering`) is now enabled by default; the `firewall.effective-token-steering` frontmatter key has been removed. Set `max-effective-tokens` to a negative value to disable both budget enforcement and token steering.
