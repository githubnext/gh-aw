# ADR-55148: `gh aw models` Command for Catalog Pricing, Aliases, and Observed Models

**Date**: 2026-08-23
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Choosing a model for a workflow currently requires reading three disconnected sources: the embedded models catalog (`pkg/cli/model_costs.go`, providing per-token pricing), the built-in alias map in `pkg/workflow` (which alias resolves to which ordered list of concrete model IDs), and downloaded run artifacts (`summary.json`, per-run token usage files, and `awf-reflect.json`) that show which models automation has actually used. Agents and maintainers had no single command that answers "what can I pick, what does it cost, and what is actually in use here?", so model selection relied on grepping the repository or reading raw JSON artifacts.

### Decision

We add a dedicated read-only `gh aw models` CLI command in the analysis command group (`pkg/cli/models_command.go`) that renders three sections — catalog pricing, alias resolution order, and models observed in local automation artifacts — with `--json` for machine consumption.

Observed-model discovery aggregates three artifact sources under a shared record keyed by normalized `provider/model`, merging provenance labels and occurrence counts. Because `summary.json` is generated from the sibling `run-*` directories, run IDs recorded in the summary are skipped when walking run directories, so a run's requests are counted once rather than twice.

Catalog membership is provider-scoped: an observation with a known provider is matched against `provider/model` catalog IDs only, and the bare model-name index is consulted only for observations with no provider, so an unrelated `other/gpt-5.4` is not reported as catalog-backed just because `gpt-5.4` exists under another provider.

For the optional artifact refresh (`--refresh-observed`, on by default), the command reuses `DownloadWorkflowLogs` with a new `SuppressRender` option. That option stops the logs orchestrator after artifacts and the summary file are written, so the refresh does not emit its own report onto stdout and `gh aw models --json` stays a single valid JSON document.

### Alternatives Considered

#### Alternative 1: Extend `gh aw logs` or `gh aw audit` With a Models View

Add a `--models` flag to an existing analysis command instead of introducing a new one. This avoids growing the command surface, but both commands are run-centric (they take run selectors, dates, and artifact filters), whereas catalog pricing and alias resolution are static repository data unrelated to any run. Overloading them would make the flag semantics conditional on unrelated options and would still require the same suppression work for JSON output. Rejected as a worse fit for the data being reported.

#### Alternative 2: Capture Rendered Output Instead of Adding `SuppressRender`

Redirect `os.Stdout` around the refresh call and discard whatever the logs orchestrator prints. This keeps the change local to the new command, but process-global stdout swapping is not concurrency-safe, hides genuine errors, and silently discards warnings that the orchestrator writes intentionally. Rejected in favour of separating downloading from rendering with an explicit option.

#### Alternative 3: Read Only `summary.json` for Observed Models

Restrict discovery to the aggregated summary file, avoiding both the run-directory walk and the de-duplication problem entirely. This loses models seen only in `awf-reflect.json` endpoint lists (models the sandbox exposed but the summary never attributed) and produces nothing at all when a logs directory contains run artifacts without a generated summary. Rejected because endpoint-level model availability is a primary reason to run the command.

### Consequences

#### Positive
- One command answers model selection questions that previously required reading three separate sources.
- `--json` output is a single valid document, so agents can pipe it directly into `jq`.
- `SuppressRender` gives any future caller a supported way to download artifacts without emitting a report, instead of each caller inventing its own stdout suppression.

#### Negative
- Observed-model results depend on artifact layout conventions (`run-<id>` directories, `summary.json` `run_id` fields, `awf-reflect.json` endpoint shape); a change to any of those degrades the observed section until the collectors are updated.
- The default refresh performs network calls, so the command is slower than a purely local report unless `--refresh-observed=false` is passed.

#### Neutral
- The command is read-only: it never writes workflow files and only writes artifacts through the existing logs download path.
- Provider-scoped catalog matching means observations from providers absent from the embedded catalog are reported as not-in-catalog rather than being matched by model name alone.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
