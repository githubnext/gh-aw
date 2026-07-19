# Refactoring task: reduce function-length lint findings in pkg/cli

You are refactoring Go functions in `pkg/cli/` (package `cli`) that exceed 60 lines.
Goal: reduce the `largefunc` lint findings by extracting focused helper functions.

## Hard rules
1. **Pure refactoring — preserve ALL existing behavior.** No logic, output, or control-flow changes.
2. Extract logical sub-sections of a long function into new helper functions so the
   original function drops to **≤ 60 lines** (counting the `func` signature line through the
   closing `}` inclusive). Each new helper should also be ≤ 60 lines.
3. **Keep helpers in the SAME file** as the function you extracted them from.
4. **Naming (critical — avoid collisions):** every NEW helper function MUST be named by
   using the original function's name as a prefix, then a descriptive suffix.
   Example: extracting from `buildAuditData` → `buildAuditDataMetrics`, `buildAuditDataFirewall`.
   For extracted anonymous/`func literal` findings (e.g. cobra `RunE:` closures), create a
   named package function derived from the command/context, e.g. `runLogsCommand`,
   and reference it from the closure: `RunE: func(cmd *cobra.Command, args []string) error { return runLogsCommand(cmd, args, &opts) }`.
   Do NOT reuse a generic name that another file might also use.
5. **Do NOT modify `_test.go` files.**
6. Do not change exported function signatures or public API. Helpers should be unexported.
7. Preserve comments; move them with the code they describe.
8. Keep imports correct. Do not add new external dependencies.

## Technique
- Read the full function first. Identify natural blocks (often separated by blank lines or
  comments, or handling distinct phases: validation, data gathering, API calls, rendering).
- Extract each block into a helper that takes exactly the inputs it needs and returns exactly
  the outputs the caller needs. Pass pointers to structs when the block mutates shared state.
- If a block references many local variables, consider passing a small params struct OR
  passing the needed values; prefer passing the existing struct/receiver when available.
- Be careful with `return`, `continue`, `break`, named return values, `defer`, and closures
  capturing loop variables — extracting these requires returning values/errors to preserve flow.
- For methods (receiver funcs), helpers can be methods on the same receiver.

## After editing each file
- Run `gofmt -w` on the file to ensure it is syntactically valid and formatted.
  If gofmt reports a syntax error, fix it.

## Do NOT
- Do NOT run `go build ./pkg/cli/...` or the package-wide linter — other agents are editing
  the same package concurrently and it will show spurious errors. Only use `gofmt` per-file.
- Do NOT touch files outside your assigned list.
