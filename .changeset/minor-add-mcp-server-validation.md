---
"gh-aw": minor
---

Add validation step to mcp-server command startup

The `mcp-server` command now validates configuration before starting the server. It runs `gh aw status` to verify that the gh CLI and gh-aw extension are properly installed, and that the working directory is a valid git repository with `.github/workflows`. This provides immediate, actionable feedback to users about configuration issues instead of cryptic errors when tools are invoked.
