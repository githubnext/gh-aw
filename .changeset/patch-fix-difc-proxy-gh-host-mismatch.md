---
"gh-aw": patch
---

Fix GH_HOST mismatch when DIFC proxy is active with user-defined setup steps: inject a "Derive GH_HOST for setup steps" step immediately after `start_difc_proxy.sh` and before user-defined `steps:`. The proxy sets `GH_HOST=localhost:18443` in `GITHUB_ENV`, which broke `gh` CLI calls in custom steps because the host didn't match the checkout remote. The new step re-derives `GH_HOST` from `GITHUB_SERVER_URL` (GHEC-safe), restoring the correct value for all subsequent user-defined steps while API calls continue to route through the proxy via `GITHUB_API_URL`/`GITHUB_GRAPHQL_URL`.

Also remove the hardcoded `GH_HOST: github.com` step-level env override from the `Install GitHub Copilot CLI` step. The override was unnecessary (the install script uses hardcoded `curl` URLs and does not use `gh` CLI) and caused issues on GHEC where the correct host should be derived from `GITHUB_SERVER_URL`.
