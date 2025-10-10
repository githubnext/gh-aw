---
"gh-aw": minor
---

Add support for GitHub toolsets configuration in agentic workflows

Adds a new `toolset` field to the `tools.github` frontmatter configuration, allowing workflows to specify which groups of GitHub API functionalities to enable. The field accepts an array of toolset names (e.g., `repos`, `issues`, `pull_requests`, `actions`, `code_security`, etc.) or the special `all` toolset.

All three agentic engines (Claude, Copilot, Codex) now properly render the GITHUB_TOOLSETS environment variable in local mode. The schema validates toolset names using enum constraints and only accepts array format for consistent configuration across workflows.
