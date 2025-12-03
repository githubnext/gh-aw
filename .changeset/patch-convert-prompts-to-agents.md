---
"gh-aw": patch
---

Convert `.prompt.md` templates to `.agent.md` format and move them
to `.github/agents/`. Update the CLI, tests, workflows, and Makefile
to reference the new agent files and remove the old prompt files.

This change is internal/tooling only and does not change the public API.

