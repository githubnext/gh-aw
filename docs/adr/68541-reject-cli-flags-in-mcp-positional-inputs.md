# ADR-68541: Reject CLI Flag-Shaped Values in MCP Positional Inputs

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown (drafted by the Design Decision Gate from PR #68541 evidence)

---

### Context

`gh aw mcp-server` exposes CLI commands (`add`, `update`, `fix`, `compile`, `mcp-inspect`, `logs`, `audit`, `audit diff`) as MCP tools. Each handler builds an `exec.Cmd` argument vector by appending client-supplied positional values — workflow names, workflow specifications, run IDs, and run URLs — directly onto a fixed command prefix. Because Go's flag/Cobra parsing treats any argument beginning with `-` as an option, a client could smuggle options such as `--force`, `--disable-security-scanner`, `--create-pull-request`, `--repo=owner/private`, or `--write` through a positional parameter and change command behaviour beyond what the tool schema declares. The tool schemas already expose every supported option as a named parameter. However, workflow display names accepted by `logs` are free-form and can legitimately begin with `-`. The constraint is that command options must remain server-controlled without breaking valid workflow display names, workflow specs, run URLs with fragments, or local files whose names begin with `-`.

### Decision

Named MCP parameters are the only supported way to select command options. A single shared helper, `validateMCPPositionalArgs(field string, values ...string)` in `pkg/cli/mcp_positional_args.go`, rejects values beginning with `-` before launching the subprocess and returns a `jsonrpc.CodeInvalidParams` error naming the offending field. It protects `workflows`, `workflow_file`, `run_ids_or_urls`, the normalized `run_id` / `run_id_or_url` aliases, and `base_run_id` / `compare_run_ids`. Users who need a leading-dash local filename can pass an explicit relative path such as `./-workflow.md`.

For `logs`, preserve the free-form workflow display name unchanged. Construct all supported and server-controlled options first, then append `--` followed by the workflow name. This prevents the CLI parser from interpreting the name as an option while preserving names such as `-nightly`. Existing workflow-existence validation remains in place.

### Alternatives Considered

#### Alternative 1: Insert a `--` end-of-options separator before positionals

Appending `--` before positional arguments makes the CLI parser treat every following value as a positional. This approach is selected for `logs` because dash-prefixed workflow display names have no path-style escape hatch. Applying it universally would also require moving every server-controlled trailing option before the separator, including compile's manifest baseline and audit's timeout. The other tools retain early rejection to provide explicit invalid-parameters errors with minimal command-construction changes.

#### Alternative 2: Per-parameter allow-list validation (regex for workflow specs, run IDs, and URLs)

Instead of denying a `-` prefix, each field could be validated positively against its expected shape — `owner/repo/name@ref` for workflow specs, digits for run IDs, a GitHub Actions run URL pattern for URLs. This is stricter and would catch a broader class of malformed input. It was rejected for this change because the accepted shapes vary widely per field (optional refs, relative paths, URLs with `#step:` fragments, bare local names), so a correct allow-list is substantially larger and riskier to get right, and any gap in it becomes a false rejection of valid user input. The narrow deny rule closes the specific injection vector now; an allow-list remains a possible follow-up.

#### Alternative 3: Validate centrally in the `execCmd` factory

A single choke point wrapping subprocess creation could scan the whole argument vector for unexpected flags. It was rejected because at that point the handler-constructed, legitimate flags (`--major`, `--force`, `--json`, `-o`) are indistinguishable from injected ones without reconstructing per-tool knowledge, and the resulting error could not name the offending MCP parameter.

### Consequences

#### Positive

- Flag injection through MCP positional parameters is blocked for all eight affected tools, including the normalized `audit` aliases. Rejected requests never launch a subprocess; `logs` passes display names only after an end-of-options boundary.
- Rejection errors are returned as `jsonrpc.CodeInvalidParams` with the offending field name, so MCP clients get actionable diagnostics instead of surprising CLI behaviour.
- Dash-prefixed workflow display names remain usable through `logs` without changing the name.
- The rule is one small function, so adding a new MCP tool that forwards positionals is a one-line call rather than a bespoke validation design.

#### Negative

- Enforcement is opt-in per call site: a future handler must validate its positionals or place them after an end-of-options separator. Nothing in the type system enforces this for new tools.
- Legitimate values that begin with `-` — most notably a local workflow file literally named `-workflow.md` — are rejected and must be rewritten as `./-workflow.md`, a behavioural break for any existing client relying on the bare form.
- The check is purely prefix-based, so it offers no protection against other positional-based abuse (path traversal, unexpected repository references) that does not start with `-`.

#### Neutral

- Supported named options are unchanged. The `logs` command places all options before its positional boundary; other command construction remains unchanged.
- The reference documentation (`gh-aw-as-mcp-server.md`) now states the named-parameters-only contract and the `./` escape hatch.
- Shared-validator and handler regression tests assert rejection before subprocess execution and preservation of valid inputs. Logs-specific tests verify that dash-prefixed display names remain positional while server-controlled options retain their values.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
