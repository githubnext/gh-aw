---
"gh-aw": patch
---

Normalize GitHub host detection and URL parsing so GitHub Enterprise deployments honor `GITHUB_SERVER_URL`, `GITHUB_ENTERPRISE_HOST`, `GITHUB_HOST`, and `GH_HOST`. Remote imports now use the normalized host when calling `git`/`gh`, and the parser handles enterprise-styled run/PR/file URLs.
