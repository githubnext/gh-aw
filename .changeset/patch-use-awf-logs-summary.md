---
"gh-aw": patch
---

Use `awf logs summary` to generate CI firewall reports and print them to the GitHub Actions step summary. Adds `continue-on-error: true` to the "Firewall summary" step so CI does not fail when generating reports. Recompiled workflow lock files and merged `main` to pick up latest changes.

Fixes githubnext/gh-aw#9041

---

Crafted by Changeset Generator

