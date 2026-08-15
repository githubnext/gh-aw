---
"gh-aw": patch
---

Remove vulnerable `ghcr.io/github/gh-aw-firewall/cli-proxy:0.27.44` container image pin from shared action lock data. The image digest `sha256:c064d15974f7c933ec7d3f7b4038f4fd203547b3154bdc821afd379144887eff` is flagged by daily container scanning for Node.js and npm dependency CVEs. The default firewall version is already `v0.28.1`; removing this historical cli-proxy pin prevents the vulnerable digest from being resolved from embedded lock metadata.
