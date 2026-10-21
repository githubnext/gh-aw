# ADR-70868: Use Actions and AW ASTs to Anchor Scanner Finding Identities

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan [TODO: verify]

---

### Context

During compilation, the differential scanner (`pkg/cli/compile_scanner_differential.go`) scans the existing lock and workflow files, recompiles, scans again, and reports only findings that are new. To tell old findings from new ones, each finding needs a stable identity that survives line shifts. Before this change, the identity came from walking the raw goccy YAML AST to the nearest *preceding* node on the finding's line. It combined that node with the trimmed raw line text and a step key, built from the first non-empty `id`, `name`, `uses`, or `run` field. That approach had three problems. It could attach findings to unrelated neighbouring fields. It treated distinct steps that shared a name or action reference as the same step. It reported harmless formatting changes, such as quoting, comments, or alias renames, as new findings. It also ignored AW markdown frontmatter (`.md`) entirely.

### Decision

We will compute scanner finding identities from the semantic Actions AST (`pkg/workflow/actions`) and AW AST (`pkg/workflow/aw`). Each AST value will carry exact source-position ranges, including AW frontmatter offsets and resolved YAML aliases. The identity will no longer come from nearest-preceding raw YAML nodes or raw line text. Steps will be matched by `id` or by a unique `name`. When neither exists, a step will be matched by a digest of its full contents. Parsed sources will be cached in a `scannerSources` map scoped to one scanner result. When a finding cannot be mapped to an exact AST location, or the workflow cannot be parsed, the identity will fall back conservatively to coordinates plus a file-content digest. It will never guess a neighbouring field. The main driver is precision: fewer false "new finding" reports, and no hidden real regressions.

### Alternatives Considered

#### Alternative 1: Keep the raw YAML nearest-node visitor and tune its heuristics

We could have kept `scannerLocationVisitor` and patched specific problems, for example by adding more step-key fields or normalizing quotes in line text. This would have been a smaller change. It was rejected because the root problem is that raw line text and preceding-node lookup do not represent the semantic value. Aliases, flow mappings, quoted dotted keys, and comments would each need their own special case. It would also still not support AW frontmatter.

#### Alternative 2: Match findings only by coordinates and file fingerprint

We could have treated any finding whose file changed as new, keyed by line, column, and file digest. This is simple and never conflates findings. It was rejected because every line shift or unrelated edit would re-report all existing findings, which defeats the purpose of differential scanning. The PR keeps this approach only as the conservative fallback.

#### Alternative 3: Identify steps by positional index

We could have keyed steps by their `steps[N]` index. That is stable for unchanged files but breaks as soon as steps are inserted or reordered. The previous implementation already tried to avoid it. It was rejected for the same reason.

### Consequences

#### Positive
- Findings are tied to the exact field value and logical step. Quoting, comments, indentation, alias renames, and reordering of steps with an ID or unique name no longer show up as new findings.
- Steps that share a name or action reference are no longer treated as the same step, so real regressions in "duplicate" steps are no longer hidden.
- AW markdown frontmatter (steps, post-steps, safe-jobs) gets the same structural matching as lock YAML.
- ShellCheck identities keep the diagnostic text, and invalid (non-positive) script coordinates are rejected.

#### Negative
- The differential scanner now depends on the internal Actions and AW AST packages. Changes to those parsers' positions or value models can change finding identities.
- Anonymous steps are matched by a digest of their contents, so editing any field in such a step changes the identity of every finding in it. That can produce new-finding reports that a human would see as the same finding.
- The code is larger and more complex: a new 365-line `compile_scanner_source.go` with location indexing, digests, and caching.

#### Neutral
- Source caching is deliberately scoped to one scanner result (`scannerSources`), so it never goes stale across before/after compilation. The cost is re-parsing per result.
- Unparseable workflows and unmapped locations still produce findings, identified by coordinates plus a file digest.
- The user-facing reference docs (`compilation-process.md`) describe the new matching behaviour.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
