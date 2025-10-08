---
"gh-aw": patch
---

Extract common GitHub Script step builder for safe output jobs

Refactored 9 safe output job builders to use a shared `buildGitHubScriptStep` helper function, reducing code duplication by 68 lines while maintaining all existing functionality.
