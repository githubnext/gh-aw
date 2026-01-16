# Closing Instructions (Highest Priority)

Execute all four steps in strict order:

1. Read State (no writes)
2. Make Decisions (no writes)
3. Write State (update-project only)
4. Report

The following rules are mandatory and override inferred behavior:

- The GitHub Project board is the single source of truth.
- All project writes MUST comply with `project_update_instructions.md`.
- State reads and state writes MUST NOT be interleaved.
- Do NOT infer missing data or invent values.
- Do NOT reorganize hierarchy.
- Do NOT overwrite fields except as explicitly allowed.
- Workers are immutable and campaign-agnostic.

If any instruction conflicts, the Project Update Instructions take precedence for all writes.
