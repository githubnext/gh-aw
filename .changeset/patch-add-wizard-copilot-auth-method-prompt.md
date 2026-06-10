---
"gh-aw": patch
---

add-wizard now prompts Copilot users to choose between copilot-requests (org billing, no PAT) and a Personal Access Token. When copilot-requests is selected, `permissions.copilot-requests: write` is injected into the workflow and `COPILOT_GITHUB_TOKEN` setup is skipped.
