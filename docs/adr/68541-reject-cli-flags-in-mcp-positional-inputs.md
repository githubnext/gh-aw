# ADR-68541: Reject CLI Flag-Shaped Values in MCP Positional Inputs

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown (drafted by the Design Decision Gate from PR #68541 evidence)

---

### Context

`gh aw mcp-server` exposes CLI commands (`add`, `update`, `fix`, `compile`, `mcp-inspect`, `logs`, `audit`, `audit diff`) as MCP tools. Each handler builds an `exec.Cmd` argument vector by appending client-supplied positional values — workflow names, workflow specifications, run IDs, and run URLs — directly onto a fixed command prefix. Because Go's flag/Cobra parsing treats any argument beginning with `-` as an option, a client could smuggle options such as `--force`, `--disable-security-scanner`, `--create-pull-request`, `--repo=owner/private`, or `--write` through a positional parameter and change command behaviour beyond what the tool schema declares. The tool schemas already expose every supported option as a named parameter, so flag-shaped positionals have no legitimate use. The constraint is that validation must happen *before* the subprocess is launched, and must not break valid inputs such as workflow specs, run URLs with fragments, or local files whose names begin with `-`.

### Decision

We will treat named MCP parameters as the only supported way to select command options, and reject any positional value beginning with `-` before constructing or launching the CLI subprocess. A single shared helper, `validateMCPPositionalArgs(field string, values ...string)` in `pkg/cli/mcp_positional_args.go`, returns a `jsonrpc.CodeInvalidParams` MCP error naming the offending field, and is called from every handler that forwards positionals (`workflows`, `workflow_file`, `workflow_name`, `run_ids_or_urls`, and the normalized `run_id` / `run_id_or_url` aliases, plus `base_run_id` / `compare_run_ids`). The primary driver is security: a deny-by-prefix check is simple, auditable, and fails closed. Users who genuinely need a leading-dash local filename can pass an explicit relative path such as `./-workflow.md`, which is documented in `docs/src/content/docs/reference/gh-aw-as-mcp-server.md`.

### Alternatives Considered

#### Alternative 1: Insert a `--` end-of-options separator before positionals

Appending `--` to each command vector before the positional arguments would make the underlying CLI parser treat every following value as a positional, neutralising flag injection without rejecting any input. It was considered because it preserves maximal input flexibility and requires no per-handler validation. It was rejected because correctness then depends on every current and future subcommand (including nested ones such as `audit diff` and `mcp inspect`) honouring `--` consistently, and because a value that merely *looks* like a flag is still almost certainly a client bug or an attack; failing loudly with `CodeInvalidParams` gives better diagnostics than silently passing `--force` through as a workflow name.

#### Alternative 2: Per-parameter allow-list validation (regex for workflow specs, run IDs, and URLs)

Instead of denying a `-` prefix, each field could be validated positively against its expected shape — `owner/repo/name@ref` for workflow specs, digits for run IDs, a GitHub Actions run URL pattern for URLs. This is stricter and would catch a broader class of malformed input. It was rejected for this change because the accepted shapes vary widely per field (optional refs, relative paths, URLs with `#step:` fragments, bare local names), so a correct allow-list is substantially larger and riskier to get right, and any gap in it becomes a false rejection of valid user input. The narrow deny rule closes the specific injection vector now; an allow-list remains a possible follow-up.

#### Alternative 3: Validate centrally in the `execCmd` factory

A single choke point wrapping subprocess creation could scan the whole argument vector for unexpected flags. It was rejected because at that point the handler-constructed, legitimate flags (`--major`, `--force`, `--json`, `-o`) are indistinguishable from injected ones without reconstructing per-tool knowledge, and the resulting error could not name the offending MCP parameter.

### Consequences

#### Positive

- Flag injection through MCP positional parameters is blocked for all eight affected tools, including the normalized `audit` aliases, and the subprocess is never launched for a rejected request.
- Errors are returned as `jsonrpc.CodeInvalidParams` with the offending field name, so MCP clients get actionable diagnostics instead of surprising CLI behaviour.
- The rule is one small function, so adding a new MCP tool that forwards positionals is a one-line call rather than a bespoke validation design.

#### Negative

- Enforcement is opt-in per call site: a future handler that forwards positionals without calling `validateMCPPositionalArgs` silently reopens the hole. Nothing in the type system or tests enforces the call for new tools.
- Legitimate values that begin with `-` — most notably a local workflow file literally named `-workflow.md` — are rejected and must be rewritten as `./-workflow.md`, a behavioural break for any existing client relying on the bare form.
- The check is purely prefix-based, so it offers no protection against other positional-based abuse (path traversal, unexpected repository references) that does not start with `-`.

#### Neutral

- Supported named options and the existing command-construction logic are unchanged; only a pre-flight guard is added.
- The reference documentation (`gh-aw-as-mcp-server.md`) now states the named-parameters-only contract and the `./` escape hatch.
- Regression coverage lives in `pkg/cli/mcp_positional_args_test.go` using in-memory MCP sessions with a counting `execCmd` factory, asserting both rejection (no subprocess) and preservation of valid inputs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
