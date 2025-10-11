---
"gh-aw": patch
---

Update documentation unbloater workflow with cache-memory and PR checking

Enhanced the unbloat-docs workflow to improve coordination and avoid duplicate work:
- Added cache-memory tool for persistent storage of cleanup notes across runs
- Added search_pull_requests GitHub API tool to check for conflicting PRs
- Updated workflow instructions to check cache and open PRs before selecting files to clean
