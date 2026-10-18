---
"gh-aw": major
---

Run graders as pure computation over the redacted in-memory unified session in
conclusion, with ten fast built-ins enabled by default.

**Breaking change:** Custom JavaScript and Bash graders, including
`operational-value`, `script`, and `run`, are rejected for security. Remove
these entries and use registered built-ins with data-only configuration.
Expensive trajectory analyses now require explicit opt-in.

Grader artifacts use schema version 2 and live in `usage/graders/`, not the
agent artifact. Recompile workflows to adopt conclusion-only grading. For
`gh aw graders run`, replace legacy trace payloads with unified-session JSONL
or a JSON event array; run-ID replay downloads the `usage` session.
