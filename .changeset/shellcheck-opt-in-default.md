---
"gh-aw": major
---

`shellcheck` is now opt-in via `--shellcheck` (or `--validate`); the `--no-shellcheck` flag is deprecated.

**Breaking Change**: shellcheck previously ran by default on every `gh aw compile` invocation (opt-out via `--no-shellcheck`). It is now disabled by default, consistent with all other optional checkers (`--poutine`, `--zizmor`, `--actionlint`, etc.).

**Migration guide:**
- If you relied on shellcheck running by default, add `--shellcheck` to your `gh aw compile` invocations.
- Alternatively, use `--validate` to enable shellcheck together with schema, container-image, and action-SHA validation.
- The `--no-shellcheck` flag is retained as a deprecated no-op for script compatibility; it will be removed in a future release.
