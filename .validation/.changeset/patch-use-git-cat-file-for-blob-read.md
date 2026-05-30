---
"gh-aw": patch
---

Use `git cat-file blob <hash>` instead of `git show <sha>:<path>` when reading destination blobs in signed commit push operations, simplifying blob lookup and improving robustness for renamed or copied files.
