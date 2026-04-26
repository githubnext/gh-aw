---
"gh-aw": patch
---
Harden AWF binary installation against CDN availability blips:

- Cache the AWF binary in GitHub Actions cache keyed by version+os+arch, eliminating
  CDN dependency on every run after the first download.
- Add stale-version fallback: if the CDN returns a non-2xx on the checksums download,
  the script now uses any previously cached AWF binary instead of failing outright.
- Increase curl retry budget from 5×10s (≈60s) to 6×15s with a 300s max-time,
  giving the GitHub releases CDN more time to recover from transient outages.
