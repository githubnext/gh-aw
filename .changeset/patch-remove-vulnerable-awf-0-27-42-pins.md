---
"gh-aw": patch
---

Remove vulnerable `ghcr.io/github/gh-aw-firewall` `0.27.42` container image pins (agent, api-proxy, squid) from the shared action lock data. These images contained High-severity CVEs (grpc GHSA-hrxh-6v49-42gf, brace-expansion GHSA-mh99-v99m-4gvg, golang.org/x/text GO-2026-5970). The default firewall version is already `v0.27.43` which addresses these findings; removing the `0.27.42` pins ensures these vulnerable digests are no longer resolvable from lock metadata.
