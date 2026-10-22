# ADR-70281: Offline Join of Episode Usage with Recorded Outcome Observations

**Date**: 2026-10-19
**Status**: Superseded proposal; see [ADR-71218](71218-move-outcome-evaluation-into-logs-command.md) for the in-memory logs join that replaces the separate command and schema.
**Deciders**: Not recorded

---

### Context

`gh aw outcomes` already classifies the safe-output actions of individual workflow runs and writes per-item observations to JSONL, and `gh aw logs --verbose --json` already groups runs into episodes and exposes downloaded usage artifacts (including detection usage) priced in AIC. There was no way to answer "what did an episode cost per accepted outcome?" because outcomes, membership, and cost lived in separate artifacts with no shared item identity. Existing outcome JSONL records could not distinguish two indistinguishable actions in one run, so retaining snapshots of repeated evaluations in additional files risked double-counting accepted outcomes. The [revised scope of ADR-70205](70205-durable-pending-backlog-for-outcome-evaluation.md) moved outcome evaluation to a CLI-only model and removed a scheduled collector, so this proposal must avoid reintroducing an evaluator, collector, or lifecycle service.

### Decision

The proposed offline `gh aw outcomes episodes` command joins existing artifacts — logs episode membership, outcome JSONL, and downloaded usage — by source `run_id`, keeps only the latest observation per action identity, and reports accepted outcomes, observed AIC, and `cost_per_accepted_outcome`. New outcome JSONL records gain a content-derived `item_id` and `run_item_count` so deduplication is by action identity (never by acceptance) and evidence completeness can be checked. IDs combine a SHA-256 digest of the source record with an occurrence number for indistinguishable records, preserving distinct actions without depending on other items' order. Modern item identities supersede matching legacy observations, including earlier positional IDs, only when all matching precise observations are at least as recent; retained legacy evidence is qualified and disables efficiency.

The report is explicitly a selected-run snapshot (`membership_scope: selected_runs`): missing outcomes, missing/unpriced/malformed usage, incomplete item counts, partial membership, or retained records lacking stable action identities suppress the efficiency figure (set to `null`), as does having no accepted outcomes. Qualifications and coverage fields expose evidence gaps, and the checked-in JSON schema (`gh aw json-schema outcomes-episodes`) defines the output contract.

The command reuses the AIC usage reader for both cost and completeness diagnostics, including detection usage, rather than maintaining a second scanner with potentially different parsing rules. Successful JSON and text reports use the configured Cobra stdout writer, and warnings use its stderr writer, so callers can capture reports independently of diagnostics.

### Alternatives Considered

#### Alternative 1: Online/live episode accounting service

Query live outcome state and discover full episode membership at report time (or via a scheduled collector), so reports reflect final acceptance. This would give more complete answers but reintroduces the scheduled collector and lifecycle state that ADR-70205 removed, adds GitHub API load and credentials to a reporting path, and makes results non-reproducible. Rejected in favour of a deterministic offline join over artifacts the user already has.

#### Alternative 2: Deduplicate on legacy content keys only (no new `item_id`)

Keep the JSONL format unchanged and coalesce observations by type/repo/object URL/number/created-at. This avoids a schema change, but indistinguishable actions in one run (e.g. several comments without object URLs) collapse into one, undercounting outcomes, and there is no way to know whether a run's evidence is complete. Retained only as a conservative fallback where modern identities are absent, which disables efficiency and asks the user to reevaluate; matching legacy observations must not collapse distinct modern items or override their observations.

#### Alternative 3: Best-effort totals without qualification

Always compute `observed_aic / accepted` from whatever data is available. Simpler output, but silently presents partial cost or partial outcomes as episode totals, inflating or deflating efficiency. Rejected because the metric is used for economic decisions and must not imply completeness it does not have.

### Consequences

#### Positive
- Cost-per-accepted-outcome becomes reportable with no new evaluator, collector, or service; the command is fully offline and reproducible.
- Latest-observation-per-item semantics prevent repeated evaluations from inflating accepted counts, and later rejection correctly supersedes earlier acceptance.
- Explicit nullable fields, coverage lists, and qualifications make incompleteness machine-readable for downstream consumers via the published schema.

#### Negative
- Results are a snapshot of recorded observations for the selected runs only; the command cannot prove full-episode membership or final acceptance.
- Legacy-only JSONL without `item_id` can be conservatively undercounted and yields no efficiency figure until runs are reevaluated.
- A new command, schema, and JSONL fields (`item_id`, `run_item_count`) must be maintained for compatibility.

#### Neutral
- Conflicting evaluation fields at an item's latest timestamp, including evidence strength and summary inputs, are treated as an error requiring reevaluation rather than resolved heuristically. Newer observations supersede older conflicts consistently regardless of input order.
- A newer ambiguous legacy snapshot causes an error until every matching precise item's observation is refreshed to at least that timestamp.
- Optional lineage/collection coverage from logs is copied as `membership_coverage` without depending on the episode producer's internal coverage implementation.

---

*Proposed in [PR #70281](https://github.com/github/gh-aw/pull/70281); this record does not establish maintainer approval or publication.*
