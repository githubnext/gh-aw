---
"gh-aw": patch
---

Remove vulnerable `ghcr.io/github/gh-aw-firewall/api-proxy:0.27.44` container image pin from shared action lock data. The image digest `sha256:b50fbadba138f6e9aba94aca09711335c489bb3b15861220cb66f6092e042dc7` is flagged by daily container scanning for Node.js and npm dependency CVEs. The default firewall version is already `v0.28.1`; removing this historical api-proxy pin prevents the vulnerable digest from being resolved from embedded lock metadata.
