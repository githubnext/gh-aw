---
"gh-aw": patch
---

Add a second pass for JavaScript formatting that runs `terser` on `.cjs` files after `prettier` to reduce file size while preserving readability and TypeScript compatibility.

This change adds the `terser` dependency and integrates it into the `format:cjs` pipeline (prettier → terser → prettier). Files that are TypeScript-checked or contain top-level/dynamic `await` are excluded from terser processing to avoid breaking behavior.

This is an internal tooling change only (formatting/minification) and does not change runtime behavior or public APIs.

Summary of impact:
- 14,784 lines removed across 65 files (~49% reduction) due to minification and formatting
- TypeScript type checking preserved
- All tests remain passing

