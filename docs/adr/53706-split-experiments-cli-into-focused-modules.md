# ADR-53706: Split Experiments CLI into Focused Modules

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/experiments_command.go` grew to approximately 1,120 lines by combining command wiring, local and remote data fetching, experiment state parsing (JSON and JSONL formats), git argument validation, human-readable rendering, and paginated JSON decoding in a single file. This made it difficult to navigate to a specific concern, reason about each responsibility in isolation, and maintain the file without risk of touching unrelated logic. The file also could not be unit-tested at the concern level because all functions were interleaved.

### Decision

We will decompose `experiments_command.go` into six files within the same `cli` package, each owning a single concern:

- `experiments_command.go` — Cobra command construction and list/analyze orchestration
- `experiments_fetch.go` — local and remote branch discovery, workflow frontmatter loading, and metric-evaluation retrieval
- `experiments_state.go` — experiment state models, JSON/JSONL parsing, aggregation, and branch ref lookup helpers
- `experiments_git_safety.go` — validation of `git show ref:path` arguments (ref safety, tree path safety)
- `experiments_render.go` — human-readable detail output to stderr
- `experiments_json_utils.go` — paginated JSON-array decoding shared by fetch logic

The existing CLI API and all runtime behavior are preserved exactly; this is a structural refactor only.

### Alternatives Considered

#### Alternative 1: Keep the monolith, add section comments

Add named comment blocks (`// --- Fetching ---`, `// --- Rendering ---`) to partition the single file visually without moving any code. This avoids a larger diff and leaves the package topology unchanged.

This was not chosen because comment-based partitioning does not enforce separation — functions in any section can freely call functions in any other, and editors/grep still return results from one enormous file. Navigation and isolated review remain hard.

#### Alternative 2: Extract to a dedicated sub-package (`pkg/cli/experiments/`)

Move all experiment logic into a `experiments` sub-package, exporting only the types and functions needed by the `cli` package, and importing them in `experiments_command.go`.

This was not chosen for this PR because it would require renaming types and adding explicit exports across a wider surface, increasing the diff size and review burden. It also risks breaking other callers in the `cli` package. A file-level split within the same package achieves meaningful separation with minimal risk and can be followed by a package extraction in a future PR if warranted.

### Consequences

#### Positive
- Each file has a single, named purpose, making it easy to locate code by concern.
- Future additions (new fetching strategies, alternative rendering modes) have a natural home without growing `experiments_command.go`.
- Reviewers can audit security-sensitive logic (git argument validation) in isolation in `experiments_git_safety.go`.
- Smaller files are easier to hold in working memory during review.

#### Negative
- Functions in the same package still share package scope; the file split does not create an enforced interface boundary. Cross-concern coupling can silently re-emerge over time.
- Readers must discover which file owns which concern; there is no directory-level signal (unlike a sub-package) to guide them.

#### Neutral
- The CLI public API and all data formats (state.jsonl, state.json) are unchanged, so no downstream callers or tests need updates.
- The total line count across all six files equals the original single-file line count; no logic was added or removed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
