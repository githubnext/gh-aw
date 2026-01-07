---
"gh-aw": patch
---

Use `awf logs summary` to generate the CI firewall report and print it to the GitHub Actions step summary.

- Adds `continue-on-error: true` to the "Firewall summary" step so CI does not fail when generating reports.
- Recompiles workflow lock files and merges `main` to pick up latest changes.
- Fixes githubnext/gh-aw#9041

