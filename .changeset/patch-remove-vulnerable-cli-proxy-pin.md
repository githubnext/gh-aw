---
"gh-aw": patch
---

Remove the vulnerable embedded digest pin for `ghcr.io/github/gh-aw-firewall/cli-proxy:0.27.43` from the shared action lock data. This keeps the default `cli-proxy` image reference available by tag while preventing workflows from resolving the known-vulnerable digest from embedded pin metadata.
