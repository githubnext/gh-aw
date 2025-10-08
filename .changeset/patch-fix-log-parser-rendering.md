---
"gh-aw": patch
---

Fix Codex and Claude log parsers to properly render messages in GitHub Actions

The Codex and Claude JavaScript log parsers were not rendering any output in GitHub Actions workflow runs. This occurred because the parsers called `main()` unconditionally at the end of the file, which prevented proper module import for testing. Additionally, the test harness broke `require.main` when mocking dependencies.

This fix:
- Updates both parsers to conditionally execute `main()` only when run directly
- Fixes the test harness to preserve `require.main` when mocking
- Adds comprehensive test coverage for the Codex parser
- Validates that parsers correctly use `core.*` functions for message rendering

The parsers now correctly render tool calls, commands, reasoning, and token usage in GitHub Actions workflow step summaries.
