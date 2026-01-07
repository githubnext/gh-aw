---
"gh-aw": patch
---

Update Copilot CLI to `0.0.375` and Codex to `0.79.0`.

This patch updates the bundled CLI version constants, test expectations, and
regenerates workflow lock files. It also adds support for generating
conversation markdown via the Copilot `--share` flag (used when available),
fixes a path double-slash bug for `conversation.md`, and addresses an
async/await bug in the log parser.

These changes are backward-compatible and affect tooling, tests, and
workflow compilation only.

