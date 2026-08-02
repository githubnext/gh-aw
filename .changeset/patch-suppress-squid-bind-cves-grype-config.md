---
"gh-aw": patch
---

Add `.grype.yaml` with ignore rules for High-severity bind CVEs (CVE-2026-11331, CVE-2026-11605, CVE-2026-11622, CVE-2026-11721, CVE-2026-12617, CVE-2026-13204, CVE-2026-13321) affecting `bind-libs` and `bind-tools` in `ghcr.io/github/gh-aw-firewall/squid:0.27.43`. These packages are not required by squid and will be removed in a future `gh-aw-firewall` release. The grype scanner (`--grype` flag) now mounts this config file into the Docker container so the ignore rules are respected during scans. Tracking: #49507.
