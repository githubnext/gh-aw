---
---

# Fix GH_TOKEN reference in shared MCP workflow

Fixed workflow failures in agentic workflows that use the `shared/mcp/gh-aw.md` configuration. The GitHub CLI (`gh`) was complaining about missing GH_TOKEN even though it was set via `${{ secrets.GITHUB_TOKEN }}`.

**Root Cause:** The `gh` CLI in GitHub Actions context prefers `${{ github.token }}` over `${{ secrets.GITHUB_TOKEN }}` for authentication.

**Before:** Used `secrets.GITHUB_TOKEN` which could fail in some GitHub Actions contexts
**After:** Uses `github.token` which is the recommended pattern for GitHub Actions workflows

**Affected Workflows:** All workflows importing `shared/mcp/gh-aw.md`:
- audit-workflows
- cloclo
- daily-firewall-report
- deep-report
- dev-hawk
- mcp-inspector
- portfolio-analyst
- prompt-clustering-analysis
- q
- safe-output-health
- smoke-detector
- static-analysis-report

This resolves workflow run failures like https://github.com/githubnext/gh-aw/actions/runs/20222319080/job/58046531390#step:5:1
