---
"gh-aw": patch
---

Fix single-file artifact directory nesting in logs command

When downloading artifacts with a single file, the file is now moved to the parent directory and the unnecessary nested folder is removed. This implements the "artifact unfold rule" which simplifies artifact access by removing unnecessary nesting for single-file artifacts while preserving multi-file artifact directories.
