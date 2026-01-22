---
"gh-aw": patch
---

Add safe-inputs gh CLI testing to smoke workflows.

This patch adds validation to the smoke workflows to exercise the GitHub CLI
integration via the `safeinputs-gh` tool. It also updates `shared/gh.md`
to remove the `network.allowed` restriction so the `safeinputs-gh` tool can
query PRs using the provided `GITHUB_TOKEN`.

