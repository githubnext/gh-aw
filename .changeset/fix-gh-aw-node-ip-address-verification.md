---
"gh-aw": patch
---

Fix the `gh-aw-node` image build so the `ip-address` patch (GHSA-mwp4-54f8-5fhr) actually ships. The previous verification step ran `node -e "require('ip-address')"`, which always failed with `MODULE_NOT_FOUND` because npm's global tree is not on Node's default resolution path; the build aborted and the published image kept the vulnerable `ip-address` 10.2.0. Patched packages are now replaced (instead of overlaid on stale files) and each one is loaded and version-printed by absolute path. Also extend the `.grant.yaml` license policy exception to the remaining upstream base-image packages (Alpine `git`/`libgcc`/`libstdc++`/`libcurl`/`libidn2`/`libunistring`/`zstd-libs` and the Node/npm runtime with npm's bundled dependencies), which clears the container license findings for `ghcr.io/github/gh-aw-node`.
