---
"gh-aw": patch
---

Add human-friendly schedule format parser, schema updates, docs, and tests.

This change introduces a deterministic parser that converts simplified
natural-language schedule expressions into valid GitHub Actions cron
syntax, updates the workflow schema to accept shorthand and array formats,
adds fuzz and unit tests, and enhances documentation with usage examples.

Non-breaking: this is an internal feature addition and documentation update.

