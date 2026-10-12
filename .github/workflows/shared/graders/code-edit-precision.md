---
graders:
  # Fraction of distinct edited files touched by the reference patch.
  # Built-in grader implemented in actions/setup/js/trajectory_graders.cjs.
  # Built-in graders run on every trace; importing this fragment is optional.
  code-edit-precision: {}
---

<!--
code-edit-precision compares reference.patch.files[] (strings or { path }
records) with distinct actions[].target paths for actions whose type is exactly
edit and whose consequential flag is true. Paths are trimmed and a leading
"./" or "/" is stripped, matching code-read-precision. Repeated edits count
once. The score is relevant distinct edited files / all distinct edited files;
the IR has no line or hunk records, so precision is file-level.

Missing or empty reference data, malformed required fields, and runs without
file edits report unavailable with a reason, not a fabricated score.
-->
