---
graders:
  # Fraction of distinct read/inspected files touched by the reference patch.
  # Built-in grader implemented in actions/setup/js/trajectory_graders.cjs.
  # Built-in graders run on every trace; importing this fragment is optional.
  code-read-precision: {}
---

<!--
code-read-precision compares reference.patch.files[] (strings or { path }
records) with distinct actions[].target paths for actions whose type is exactly
read or inspect and whose consequential flag is false. Searches, listings and
mutations do not count. Paths are trimmed and a leading "./" or "/" is stripped,
matching code-search-recall. Targets must be file paths, not symbols or resource
IDs; the IR has no symbol records. Repeated reads count once.

The score is relevant distinct read files / all distinct read files. Missing or
empty reference data, malformed required fields and runs without file reads
report unavailable with a reason, not a fabricated score.
-->
