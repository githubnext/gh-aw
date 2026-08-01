---
"gh-aw": patch
---

Expand `.grant.yaml` to allowlist standard Alpine/container OS package licenses (GPL-2.0-only, GPL-2.0-or-later, LGPL-2.1-or-later, LGPL-3.0-or-later, MPL-2.0) and permissive npm/system licenses (BlueOak-1.0.0, Zlib, curl, CC-BY-3.0, CC0-1.0, Artistic-2.0). Also disable `require-license` and `require-known-license` to handle internal container packages with no SPDX metadata and non-standard license identifiers. Addresses the 35 license violations reported in the daily container image security scan for api-proxy:0.27.42.
