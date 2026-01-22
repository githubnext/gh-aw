---
"gh-aw": patch
---

Add safe-inputs gh CLI testing to smoke workflows; updates `shared/gh.md` to remove the `network.allowed` restriction and validate GitHub CLI access using `GITHUB_TOKEN`.

This changeset accompanies the PR that adds `safeinputs-gh` testing to all smoke workflows (smoke-copilot.md, smoke-claude.md, smoke-codex.md, smoke-opencode.md) and adjusts `shared/gh.md` accordingly.

