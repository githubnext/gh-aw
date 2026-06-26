# ADR-41717: Non-TTY Fallback for the Interactive Workflow Wizard

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw new` interactive wizard uses the `huh` TUI library to present multi-select and single-select forms for configuring a new AI workflow (trigger, engine, tools, safe outputs, network, and intent). `huh` renders using terminal control sequences that require an interactive TTY; when the process is run in a piped, scripted, or CI context — where stderr is not a terminal — the forms either hang, emit garbled output, or fail silently with no actionable error. This makes it impossible to drive `gh aw new` from automation, integration tests, or environments that don't allocate a pseudo-TTY. The existing `huh.NewMultiSelect` fields already carried `WithTheme`/`WithAccessible` wiring, but no code path existed to bypass the form layer entirely when a TTY was absent.

### Decision

We will add a parallel non-TTY code path in `pkg/cli/interactive.go` that is activated by `tty.IsStderrTerminal()` returning `false`. When a TTY is absent, `promptForWorkflowName` and `promptForConfiguration` delegate to `promptForWorkflowNameFrom(io.Reader)` and `promptForConfigurationFrom(io.Reader)` respectively. These fallback functions render numbered-list single-select and comma-separated multi-select prompts to stderr, then read plain-text answers from a provided `io.Reader`. By accepting an `io.Reader` parameter rather than reading directly from `os.Stdin`, the functions are independently testable via `strings.NewReader` without any TTY mock.

### Alternatives Considered

#### Alternative 1: Error on non-TTY with a clear message

Detect the non-TTY condition at the `CreateWorkflowInteractively` entry point and return a descriptive error (e.g., `"interactive wizard requires a TTY; use --config flags instead"`). This is the simplest possible change and keeps a single code path. It was rejected because it still blocks legitimate automation use cases — CI pipelines, Copilot agents, and integration tests — that would benefit from a scripted wizard flow without requiring flag-based alternatives.

#### Alternative 2: Use `huh`'s built-in Accessible mode

`huh` exposes an `Accessible()` option that reduces forms to simpler line-based prompts. This was considered as a lower-effort path that would reuse the existing form definitions. It was rejected for two reasons: (a) `huh`'s accessible mode still relies on the terminal for cursor movement and may not tolerate fully non-interactive stdin in all versions; (b) the existing `huh.Option[string]` slices are defined inside functions that make them hard to reuse in tests, whereas a parallel `io.Reader`-based path enables injection-based unit testing at no extra complexity.

#### Alternative 3: Explicit `--non-interactive` / `--ci` flag

Introduce an explicit flag (e.g., `--non-interactive`) that callers must pass to get plain-text prompts, leaving the default TTY path unchanged for all contexts. This provides a clear opt-in API. It was rejected because the TTY state is an objective property of the environment — requiring callers to know about it and pass a flag creates accidental breakage for any script that forgets the flag, whereas automatic detection is the expected ergonomic for CLI tooling.

### Consequences

#### Positive
- `gh aw new` now works in piped, scripted, and CI contexts where no TTY is allocated.
- The `io.Reader`-injection design enables 13 new unit tests (`interactive_test.go`) that exercise the full prompt logic without any TTY or stdin mocking.
- Edge cases (EOF, out-of-range index, unknown value, deduplication) are now directly testable and tested.

#### Negative
- The trigger, engine, tool, network, and safe-output option lists are now duplicated between the `huh`-based path and the non-TTY path (as `[]struct{ label, value string }` slices). If new options are added to one path they must be added to the other, or they will silently diverge.
- The implicit TTY detection (`tty.IsStderrTerminal()`) means the wizard behaves differently in the same binary based on a runtime environment property, which can be surprising when debugging wizard output.

#### Neutral
- The non-TTY prompts write to `os.Stderr` (matching the TTY path, which also routes `huh` output to stderr) so stdout remains clean for programmatic consumers.
- No changes were made to the generated workflow format, validation logic, or the `InteractiveWorkflowBuilder` struct fields — only the input-collection code was extended.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
