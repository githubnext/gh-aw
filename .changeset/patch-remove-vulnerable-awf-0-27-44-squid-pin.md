---
"gh-aw": patch
---

Remove vulnerable `ghcr.io/github/gh-aw-firewall/squid:0.27.44` container image pin from shared action lock data. The image digest `sha256:83e48bbe12c634be8c228a576832fe45f66c529ac3659db92bddbcf2eeb6d627` is flagged by daily container scanning for High/Medium CVEs in Alpine packages. The default firewall version is already `v0.28.1`; removing this historical squid pin prevents the vulnerable digest from being resolved from embedded lock metadata.
