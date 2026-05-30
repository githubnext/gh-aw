---
"gh-aw": patch
---

Fix `malformed version:` error for `gh --repo` commands in user-defined steps when the DIFC proxy is active.

**Root cause**: `proxyEnvVars()` injected `GH_HOST=localhost:18443` as step-level env on every custom step. The `gh` CLI treats any host that is not `github.com` or `*.ghe.com` as GitHub Enterprise Server (GHES) and performs a `/meta` version check before each `--repo` call. The DIFC proxy forwards the check to the upstream (github.com), which returns `/meta` without `installed_version`. The `gh` CLI rejects the empty version string with `malformed version: ` and aborts.

**Fix**: Change `GH_HOST` in `proxyEnvVars()` from `localhost:18443` to `${{ env.GH_HOST || 'github.com' }}`. This uses the identity host written to `GITHUB_ENV` by the preceding `configure_gh_for_ghe.sh` step, with a `github.com` fallback:

- **github.com / GHEC (`*.ghe.com`)**: `GH_HOST` resolves to `github.com` or the GHEC tenant hostname. The `gh` CLI skips the GHES version check entirely. All API traffic still routes through the proxy via `GITHUB_API_URL`.
- **GHES**: `GH_HOST` resolves to the real GHES hostname. The `gh` CLI performs the GHES version check via `GITHUB_API_URL` (the proxy), which forwards the `/meta` request to the real GHES upstream. The GHES response includes `installed_version`, so the check passes.
