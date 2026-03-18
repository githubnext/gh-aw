---
"gh-aw": patch
---

Fix `configure_gh_for_ghe.sh` failing when `GH_TOKEN` is set. Previously the script always ran `gh auth login --with-token`, which the `gh` CLI rejects when `GH_TOKEN` is already in the environment. Now, when `GH_TOKEN` is present, the script skips `gh auth login` and only exports `GH_HOST` to `GITHUB_ENV` — the token already handles authentication and `GH_HOST` is all that is needed to point `gh` at the correct GitHub Enterprise host.
