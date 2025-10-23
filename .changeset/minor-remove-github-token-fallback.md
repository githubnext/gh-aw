---
"gh-aw": minor
---

Remove GITHUB_TOKEN fallback for Copilot operations

This is a breaking change. The default `secrets.GITHUB_TOKEN` fallback has been removed from Copilot-related operations (create-agent-task, assigning Copilot to issues, and adding Copilot as PR reviewer) because it lacks the required permissions, causing silent failures.

Users must now configure a Personal Access Token (PAT) as either `GH_AW_COPILOT_TOKEN` or `GH_AW_GITHUB_TOKEN` secret to use these features. Enhanced error messages now guide users to proper configuration when authentication or permission errors occur.
