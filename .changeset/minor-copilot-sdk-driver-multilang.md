---
"gh-aw": minor
---

Extend `engine.copilot-sdk-driver` to support Python (`.py`), TypeScript (`.ts`, `.mts`), Ruby (`.rb`), and arbitrary commands (no extension) in addition to the existing JavaScript (`.js`, `.cjs`, `.mjs`) support. The SDK install step automatically selects the correct language package manager based on the driver extension.
