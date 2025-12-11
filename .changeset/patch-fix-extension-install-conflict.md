---
---

# Fix gh extension installation conflict

Fixed workflow failure when another extension provides the `aw` command. The `shared/mcp/gh-aw.md` workflow now properly detects and removes any extension providing the `aw` command before installing `gh-aw`, preventing installation conflicts.

**Before:** Only checked for `githubnext/gh-aw` specifically
**After:** Checks for any extension where the COMMAND column is 'aw' using `awk '$2 == "aw"`

This resolves workflow run failures like https://github.com/githubnext/gh-aw/actions/runs/20126107581/job/57756021754
