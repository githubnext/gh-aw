---
"githubnext/gh-aw": minor
---

Add support for common GitHub URL formats in workflow specifications

Users can now use GitHub URLs directly in workflow imports:
- GitHub /files/ path format: `owner/repo/files/REF/path.md`
- raw.githubusercontent.com URLs with refs/heads/, refs/tags/, or commit SHA
- Automatically extracts the ref from GitHub UI copy-paste paths
