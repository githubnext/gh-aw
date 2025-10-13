---
"gh-aw": patch
---

Extract shared engine installation and permission error helpers

Refactors engine-specific implementations to eliminate ~165 lines of duplicated code by extracting shared installation scaffolding and permission error handling into reusable helper functions. Creates `BuildStandardNpmEngineInstallSteps()` and permission error detection helpers, maintaining backward compatibility with no breaking changes.
