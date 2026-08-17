---
"gh-aw": patch
---

Support an `ignore-images` glob list in `.grant.yaml` to exclude an image from container license scanning, and use it for `ghcr.io/oraios/serena` whose bundled Rust/Python/Node toolchain reports hundreds of unactionable license findings.
