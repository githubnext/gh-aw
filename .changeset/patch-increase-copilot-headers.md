---
"gh-aw": patch
---

Increase markdown header levels by 1 for Copilot `conversation.md` outputs
before writing them to GitHub Actions step summaries. This change adds a
JavaScript transformer (used in the Copilot log parser), associated tests,
and integration wiring. This is an internal tooling change and includes
comprehensive tests; it does not introduce breaking changes.

