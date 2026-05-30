---
"gh-aw": patch
---

Add gh CLI configuration script for GitHub Enterprise support. Workflows can now source `configure_gh_for_ghe.sh` before running `gh` commands to automatically detect and configure the correct GitHub Enterprise host from environment variables (`GITHUB_SERVER_URL`, `GITHUB_ENTERPRISE_HOST`, `GITHUB_HOST`, or `GH_HOST`). This fixes the "none of the git remotes configured for this repository point to a known GitHub host" error when running workflows like repo-assist on GHE domains.
