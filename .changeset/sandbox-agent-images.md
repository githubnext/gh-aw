---
"gh-aw": minor
---

Add `sandbox.agent.images` frontmatter for selecting digest-pinned AWF infrastructure images (AWF v0.28.4+). Values must be literal, registry-qualified references with both a tag and a SHA-256 digest, and the compiler emits them as the `container.images` manifest in the generated AWF configuration. The compiler rejects unknown roles, expressions or interpolation, incomplete manifests for the enabled feature set, and conflicting legacy image selectors.
