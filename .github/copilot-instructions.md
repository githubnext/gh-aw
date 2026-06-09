# Copilot Repository Instructions

## Upstream-managed workflow sources (read-only in this repo)

Workflows that declare a `source:` frontmatter entry (for example `source: githubnext/agentic-ops@<ref>`) are provenance-managed from an upstream bundle.

- Treat those workflow source files (for example `.github/workflows/agentic-token-audit.md` and `.github/workflows/agentic-token-optimizer.md`) as read-only in this repository.
- Do **not** manually edit their generated `.lock.yml` files.
- To change these workflows, use the approved update path:
  1. run `gh aw update` to refresh from source, and/or
  2. update the pinned source/version (`source: ...@...`), and/or
  3. make the change upstream first, then pull it in via `gh aw update`.
