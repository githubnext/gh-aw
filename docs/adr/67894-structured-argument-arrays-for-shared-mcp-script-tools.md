# ADR-67894: Pass Structured Argument Arrays to the Shared Go and Make MCP Tools

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

The shared import `.github/workflows/shared/go-make.md` exposes two MCP script tools, `mcpscripts-go` and `mcpscripts-make`, that agents use to run Go commands and Make targets during a workflow run. Both tools accepted `args` as a single **string** and executed it from a generated Bash handler (`go.sh`, `make.sh`) as `go $INPUT_ARGS` / `make $INPUT_ARGS`. Because the expansion was unquoted, ShellCheck reported SC2086, the shell performed word splitting and pathname globbing on model-supplied text, and any argument containing spaces (for example `build -o "path with spaces/output"`) was split into multiple arguments. Quoting the variable was not a fix either: `"$INPUT_ARGS"` would collapse a multi-argument command into one literal argument. The tool contract itself — "one opaque command string interpreted by a shell" — was the defect, and it had to be replaced without introducing `eval`, a Python dependency, or a ShellCheck suppression.

### Decision

We will change the `args` input of both shared tools from `type: string` to `type: array` of strings, and replace the Bash `run:` handlers with JavaScript `script:` handlers that invoke Node's `execFile` directly (`.github/workflows/shared/go-make.md`, generating `go.cjs` and `make.cjs`). Each array element is passed to the child process as one literal argument, so no shell is involved and spaces, quotes, backslashes, empty strings, globs, variables, command substitutions, and shell operators are never expanded or interpreted. The handlers validate that `args` is an array of strings and fail before any command execution otherwise; command failures are surfaced through the existing shared error formatter (`buildEnhancedError`) so stdout, stderr, and exit-code diagnostics are preserved. The primary driver is correctness and safety of argument passing — eliminating shell interpretation of model-supplied input — rather than performance.

Authored callers and tool descriptions are updated to the array form, and both importing workflows (`daily-cli-performance`, `smoke-claude`) are recompiled. This is a breaking change to the tool contract: existing string calls must migrate to arrays, and shell quoting must not be added to array elements.

### Alternatives Considered

#### Alternative 1: Keep the string input and quote or split it in Bash

The minimal change would have been to keep `args: string` and either quote the expansion (`go "$INPUT_ARGS"`) or split it safely with `read -ra` / `xargs`. Quoting is wrong because it turns `test ./...` into a single argument that `go` rejects. Splitting with `read -ra` silences SC2086 but still re-implements shell word splitting over untrusted text and gives no way to express an argument that legitimately contains a space; globbing would still have to be disabled explicitly with `set -f`. This keeps an ambiguous contract and only hides the warning, so it was rejected.

#### Alternative 2: Accept an array but still execute through a shell

The `args` input could become an array while the handler stays in Bash, building a properly quoted command line (`printf %q`) and running it with `bash -c`. This preserves the existing shell-based handler machinery and shell features such as pipes, but it keeps a shell in the execution path and makes correctness depend on getting the quoting right for every element — exactly the class of bug being removed. Using `execFile` with no shell removes the escaping problem entirely, so the shell path was rejected. The cost is that shell features (pipes, redirection, chained commands) are no longer available through these tools; agents are directed to use `bash` for those cases.

#### Alternative 3: Add a ShellCheck suppression for SC2086

Annotating the generated handler with `# shellcheck disable=SC2086` would have made the compile-time `--shellcheck` run pass with no behavioural change. It was rejected because the warning is a true positive: the word splitting and globbing it flags are real, observable misbehaviour for paths with spaces, not a false alarm specific to this generator.

### Consequences

#### Positive
- Arguments are passed literally to `go`/`make`; paths with spaces, quotes, and backslashes work, and no shell expansion or globbing is applied to model-supplied input.
- The SC2086 warning is removed by construction rather than suppressed; `compile --strict --shellcheck` passes for the importing workflows without an exception.
- Invalid input (non-array, non-string elements) fails fast, before any process is spawned, with a clear error.
- The tool schema is now self-describing: an array type communicates the "one element per argument" contract that a free-form string could not.

#### Negative
- Breaking change for every existing caller: prompts, docs, and any downstream repository using `shared/go-make.md` with `args: "test ./..."` must migrate to `args: ["test", "./..."]`. Old-style string calls now fail validation instead of degrading gracefully.
- Shell features (pipes, redirection, `&&`, environment prefixes, glob expansion) are no longer usable through these tools; such invocations must move to `bash` steps.
- The handlers move from Bash to JavaScript, so the runtime now requires Node on the runner (a `Setup Node.js` step is added to the generated locks) and the handler logic is duplicated per tool in the shared import.

#### Neutral
- The MCP result shape, 60-second timeouts, 10 MB output buffer, and the shared `buildEnhancedError` diagnostics are unchanged, so consumer-side error handling is unaffected.
- Generated `.lock.yml` files for `daily-cli-performance` and `smoke-claude` change (new handler files, new tool descriptions, new content hashes) without adding or removing secrets, actions, or container pins.
- The `echo "go $INPUT_ARGS"` command-echo line is dropped; the executed command is no longer echoed into stdout ahead of the output.
- Regression coverage is added in `pkg/workflow/shared_go_make_test.go`, which executes the generated JavaScript handlers against multiple arguments, spaces, literal quotes and backslashes, empty arguments, glob/shell syntax, rejected inputs, command failures, and missing executables.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
