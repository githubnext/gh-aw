---
graders:
  # Fraction of process-model checks satisfied by ordered IR tool calls.
  # Higher is better. Configure the model under graders.process-alignment-fitness.config.model.
  # Built-in grader implemented in actions/setup/js/trajectory_graders.cjs.
  # Built-in graders run on every trace; importing this fragment is optional.
  process-alignment-fitness: {}
---

<!--
process-alignment-fitness compares toolCalls[].name ordered by eventIndex with
config.model: { start: "search", end: "edit",
transitions: { search: ["read"], read: ["edit"] } }. start and end are
tool names, and transitions maps each tool name to allowed successor names.
Each tool call after the first is checked against its predecessor; the first
and last calls are also checked against start and end. The score is satisfied
checks divided by tool-call count plus one (the two boundary checks replace
one transition check). A single-call process can use the same start and end.

No configured model is not applicable. Malformed models, absent or malformed
indexed tool calls, and empty tool-call sequences are unavailable.
-->
