# Fix "too many parameters (limit 8)" findings in pkg/cli

Some helper functions were recently extracted with more than 8 parameters, which trips
a linter. Your job: refactor each listed function so it has **≤ 8 parameters**, WITHOUT
changing behavior.

## Preferred technique: options/params struct
For each offending function `fooBar(a, b, c, ... many ...)`:
1. Define a struct in the SAME file, named `fooBarParams`, with fields for the parameters
   that logically group together (or all of them). Use exported-style Go field names
   (e.g. field `Verbose bool`), but the struct itself stays unexported (`fooBarParams`).
2. Change the function signature to accept the struct, e.g.
   `func fooBar(p fooBarParams) ... {` and update the body to reference `p.FieldName`.
   Keep a small number of separate params only if it reads better, but the TOTAL parameter
   count MUST be ≤ 8. Simplest is to move ALL params into one struct (1 param total).
3. Update EVERY call site (usually just one — the function that this helper was extracted
   from). Build the struct literal with the same values in the same order/meaning.

## Alternative technique
If several params are already fields of an existing struct that the caller has in scope,
you may pass that struct/pointer instead of individual fields.

## Hard rules
- Preserve behavior EXACTLY. Same values passed, same order of operations.
- Do NOT modify `_test.go` files.
- Keep the new struct and function in the same file.
- Struct field types must match the original parameter types exactly (watch pointers vs values).
- Run `gofmt -w <file>` on each edited file. Fix any syntax errors gofmt reports.
- Do NOT run the package-wide `go build`/linter (other agents edit the same package).
- Only edit the files assigned to you.

## Verifying parameter count
After refactor, the function signature should list ≤ 8 comma-separated parameters
(a single struct param counts as 1). Double-check nested/variadic params.
