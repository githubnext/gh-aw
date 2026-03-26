---
"gh-aw": patch
---

Fixed `push_repo_memory` failing with `spawnSync git ENOBUFS` on large repositories (10K+ files).

The previous implementation called `git sparse-checkout disable` (forcing a full working-tree materialisation) and then `git rm -r -f --ignore-unmatch .` with `stdio: "pipe"` to clear the orphan branch index. On large repos, the `git rm` output (`rm 'path'` for every file) exceeded the pipe buffer, causing `ENOBUFS`.

The fix:
- Remove the `git sparse-checkout disable` call — it is not needed for orphan branch creation or for checking out a memory branch (which only contains a handful of small files).
- Replace `git rm -r -f --ignore-unmatch .` with `git read-tree --empty` (resets the index in O(1) with no output) followed by Node.js `fs.rmSync` for working-directory cleanup, which bypasses all pipe-buffer limits.
