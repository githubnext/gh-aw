---
"gh-aw": patch
---

Fix `gh aw upgrade` falsely reporting "already up to date" in GHE-authenticated environments.

In mixed-host setups where the active `GH_HOST` points at a GitHub Enterprise Server instance, the extension self-upgrade check was querying the GHE host for `github/gh-aw` release metadata. Since that repository does not exist on GHE, the API call failed silently and the upgrade was skipped.

Two changes fix this:
1. `getLatestRelease` now creates the REST client with `Host: "github.com"` so release metadata is always fetched from the canonical registry.
2. All `gh extension upgrade/install/remove` subprocess invocations now set `GH_HOST=github.com` in the child process environment, ensuring the `gh` CLI targets github.com regardless of the ambient auth context.
