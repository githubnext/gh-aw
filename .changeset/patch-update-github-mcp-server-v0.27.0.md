---
"gh-aw": patch
---

Update the default GitHub MCP Server Docker image to `ghcr.io/github/github-mcp-server:v0.27.0`.

This updates the `DefaultGitHubMCPServerVersion` constant in `pkg/constants/constants.go`,
adjusts hardcoded version strings in tests and documentation, and recompiles workflow lock
files so workflows use the new MCP server image. The upstream release includes improvements
to `get_file_contents` (better error handling and default-branch fallback), `push_files`
(non-initialized repo support), and fixes for `get_job_logs`.

Note: The upstream `experiments` toolset was removed (experimental); this is unlikely to
affect production workflows.

