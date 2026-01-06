---
"gh-aw": patch
---

Bump Codex CLI default version to 0.78.0.

This updates the repository to reference `@openai/codex@0.78.0` (used by workflows),
and aligns the `DefaultCodexVersion` constant and related tests/docs with the new
version. Changes include security hardening, reliability fixes, and UX improvements.

Files affected in the PR: constants, tests, docs, and recompiled workflow lock files.

Fixes: githubnext/gh-aw#9159

