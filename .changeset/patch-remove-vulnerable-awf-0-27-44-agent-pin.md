---
"gh-aw": patch
---

Remove vulnerable `ghcr.io/github/gh-aw-firewall/agent:0.27.44` container image pin from shared action lock data. The image digest `sha256:0d727725c737b58c7bdf51f640cffb928385ec46517e0917c7f1a02f1bada8b4` is flagged by daily container scanning for High severity Node.js CVEs (CVE-2026-56846, CVE-2026-56848, CVE-2026-58043) and npm dependency advisories in `brace-expansion` and `ip-address`. The default firewall version is already `v0.28.1`; removing this historical agent pin prevents the vulnerable digest from being resolved from embedded lock metadata.
