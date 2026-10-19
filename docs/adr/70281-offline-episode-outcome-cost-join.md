# ADR-70281: Offline Join of Episode Usage with Recorded Outcome Observations

**Date**: 2026-10-19
**Status**: Draft
**Deciders**: pelikhan [TODO: verify]

---

### Context

`gh aw outcomes` already classifies the safe-output actions of individual workflow runs and appends them to JSONL, and `gh aw logs --verbose --json` already groups runs into episodes and exposes downloaded usage artifacts (including detection usage) priced in AIC. There was no way to answer "what did an episode cost per accepted outcome?" because outcomes, membership, and cost lived in separate artifacts with no shared item identity. Existing outcome JSONL records could not distinguish two indistinguishable actions in one run, so repeated evaluations of the same run risked double-counting accepted outcomes. ADR-70205 had already moved outcome evaluation to a CLI-only model and removed a scheduled collector, so any new reporting had to avoid reintroducing an evaluator, collector, or lifecycle service.

### Decision

We will add an offline `gh aw outcomes episodes` command that joins existing artifacts — logs episode membership, outcome JSONL, and downloaded usage — by source `run_id`, keeps only the latest observation per action identity, and reports accepted outcomes, observed AIC, and `cost_per_accepted_outcome`. New outcome JSONL records gain a manifest-position `item_id` and `run_item_count` so deduplication is by action identity (never by acceptance) and evidence completeness can be checked. The report is explicitly a selected-run snapshot (`membership_scope: selected_runs`): any gap — missing outcomes, missing/unpriced/malformed usage, incomplete item counts, partial membership, or legacy records lacking `item_id` — is surfaced as a qualification and suppresses the efficiency figure (set to `null`) rather than producing a misleading number. The output contract is published as a checked-in JSON schema (`gh aw json-schema outcomes-episodes`).

### Alternatives Considered

#### Alternative 1: Online/live episode accounting service

Query live outcome state and discover full episode membership at report time (or via a scheduled collector), so reports reflect final acceptance. This would give more complete answers but reintroduces the scheduled collector and lifecycle state that ADR-70205 removed, adds GitHub API load and credentials to a reporting path, and makes results non-reproducible. Rejected in favour of a deterministic offline join over artifacts the user already has.

#### Alternative 2: Deduplicate on legacy content keys only (no new `item_id`)

Keep the JSONL format unchanged and coalesce observations by type/repo/object URL/number/created-at. This avoids a schema change, but indistinguishable actions in one run (e.g. several comments without object URLs) collapse into one, undercounting outcomes, and there is no way to know whether a run's evidence is complete. Retained only as a conservative fallback for legacy records, which disables efficiency and asks the user to reevaluate.

#### Alternative 3: Best-effort totals without qualification

Always compute `observed_aic / accepted` from whatever data is available. Simpler output, but silently presents partial cost or partial outcomes as episode totals, inflating or deflating efficiency. Rejected because the metric is used for economic decisions and must not imply completeness it does not have.

### Consequences

#### Positive
- Cost-per-accepted-outcome becomes reportable with no new evaluator, collector, or service; the command is fully offline and reproducible.
- Latest-observation-per-item semantics prevent repeated evaluations from inflating accepted counts, and later rejection correctly supersedes earlier acceptance.
- Explicit nullable fields, coverage lists, and qualifications make incompleteness machine-readable for downstream consumers via the published schema.

#### Negative
- Results are a snapshot of recorded observations for the selected runs only; the command cannot prove full-episode membership or final acceptance.
- Legacy JSONL without `item_id` can be conservatively undercounted and yields no efficiency figure until runs are reevaluated.
- Adds ~1,000 lines of new code, a new schema, and new JSONL fields (`item_id`, `run_item_count`) that must be maintained for compatibility.

#### Neutral
- Identical-timestamp conflicting observations for the same item are treated as an error requiring reevaluation rather than resolved heuristically.
- Optional lineage/collection coverage from logs is copied as `membership_coverage` without depending on the episode producer's internal coverage implementation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
