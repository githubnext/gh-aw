## Flagged Items for Monitoring (2025-12-19)

- Data freshness gap: Daily firewall report (6943) used cached Dec 16 data because gh auth and visualization deps were missing; need dependable telemetry path.
- Persistent error noise: Recent runs still show large error counts despite success (e.g., [§20373702961](https://github.com/githubnext/gh-aw/actions/runs/20373702961) with 122 errors) and one recent failure ([§20373671376](https://github.com/githubnext/gh-aw/actions/runs/20373671376)); missing tools recorded twice.
- Doc testing blocker: Documentation noob test (6941) cannot install Playwright in restricted runners (403 from cdn.playwright.dev); alternative validation path required.
- Type safety backlog: Typist recommends typing frontmatter/tool/step handling to replace widespread `map[string]any`; high-effort refactor but central to future reliability.
