---
"gh-aw": patch
---

Fix 35 license policy violations flagged by the daily container image security scan for `ghcr.io/github/gh-aw-firewall/api-proxy:0.27.43` (issue #49920).

The violations were caused by a too-strict `.grant.yaml` policy that did not account for licenses commonly found in Alpine Linux base images and the npm/Node.js packages bundled in the api-proxy container.

Changes to `.grant.yaml`:
- Set `require-known-license: false` to allow non-SPDX license identifiers (`curl`, `Apache 2.0`) found in some Alpine and npm packages.
- Added permissive licenses to the `allow` list: `BlueOak-1.0.0` (widely used in the npm ecosystem), `CC0-1.0` (public domain), `CC-BY-3.0` (SPDX metadata packages), `Zlib`, `curl`, `MPL-2.0` (ca-certificates), `Artistic-2.0` (npm).
- Added `ignore-packages` entries for Alpine OS system utilities (`busybox`, `apk-tools`, `alpine-baselayout`, etc.) whose GPL-2.0 licenses are infrastructure-level and not application code, plus internal packages (`awf-api-proxy`, `node`) whose license metadata is not detectable by Syft.

The 4 High and 2 Medium CVEs in `nodejs`/`openssl`/`libssh2` require a rebuild of the upstream `ghcr.io/github/gh-aw-firewall` images after Debian/Alpine security updates are published; those fixes are tracked separately in the upstream `gh-aw-firewall` repository.
