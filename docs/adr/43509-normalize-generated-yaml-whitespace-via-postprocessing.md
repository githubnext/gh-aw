# ADR-43509: Normalize Generated YAML Whitespace via Post-Processing Pass

**Date**: 2026-07-05
**Status**: Accepted
**Deciders**: pelikhan, Copilot

---

### Context

The gh-aw YAML compiler generates lock files from workflow definitions. These files embed multi-line
content (heredoc prompt bodies, embedded GitHub-script and shell snippets, indented sub-documents,
and ASCII logo comment lines) that consistently emit blank lines carrying the surrounding indentation
as pure whitespace. yamllint flags these as `trailing-spaces` and `empty-lines`, producing ~4,340
warnings per compilation run — 3,518 `trailing-spaces`, 258 `empty-lines`, 333 `comments-indentation`,
and 231 `indentation`. The whitespace noise degrades linter signal-to-noise and makes it harder to
spot genuine formatting problems. Fixing each individual generator component would require changes
across many locations in the compiler, carrying a high churn risk and potential for missed sources.

### Decision

We will add a `normalizeBlankLines` post-processing pass applied once at the end of
`compiler_yaml.go`'s `generateYAML` function. This pass rewrites every whitespace-only line to an
empty string and ensures the file ends with exactly one trailing newline. Additionally, the ASCII logo
emitter in `header.go` will right-trim each logo line before writing it as a comment. A whitespace-only
line is semantically a blank line inside YAML block scalars, so clearing its whitespace does not change
the parsed content. These two targeted changes address ~85% of yamllint warnings (3,724 of 4,340)
with minimal code surface.

### Alternatives Considered

#### Alternative 1: Fix Each Generator Source Individually

Identify and modify every code path that emits trailing whitespace: heredoc template rendering,
embedded script blocks, sub-document indentation logic, and logo line emission. Each fix would be
local and precise. This approach was not chosen because the sources are spread across many generator
components and templates; enumeration is incomplete without exhaustive testing, and each source fix
risks introducing regressions in content-bearing lines. The total change surface would be large
relative to the benefit, and the fix would need to be maintained as new content sources are added.

#### Alternative 2: Configure yamllint to Suppress the Warnings

Disable the `trailing-spaces` and `empty-lines` rules in the yamllint configuration for generated
files, or add per-file `# yamllint disable` markers. This approach was not chosen because it
addresses the symptom (noisy reports) rather than the root cause (generated output quality). It
would permanently hide genuine trailing-space bugs that might be introduced in content-bearing
lines, and the warnings that remain (`comments-indentation`, `indentation`) would still be mixed
with suppressed categories, making the overall linter output harder to interpret.

#### Alternative 3: Prevent Whitespace at Template/Render Time

Modify the template rendering engine or block scalar emitter to strip trailing whitespace from blank
lines as part of the rendering pipeline. This is architecturally cleaner but requires changes at a
lower level than the current PR scope, carries higher risk of altering content in block-scalar
strings, and is a larger refactor that defers the immediate noise reduction. This may be the correct
long-term approach once the templating layer is better understood; the post-processing pass chosen
here is an interim measure that can be replaced once a lower-level solution is validated.

### Consequences

#### Positive
- 85% reduction in yamllint warnings (3,724 eliminated from 4,340 total), dramatically improving
  linter signal quality for generated lock files.
- Minimal code surface: two small functions (`normalizeBlankLines`, updated logo loop) and one
  call site, making the change easy to understand and revert.
- No semantic change to parsed YAML: whitespace-only lines in YAML block scalars are semantically
  empty lines regardless of whitespace content.
- Golden test fixtures are updated in the same PR, keeping the test suite green and making the
  output change visible and reviewable.

#### Negative
- 51 `trailing-spaces` warnings remain on content-bearing lines (markdown intentional trailing
  spaces in prompt bodies), because the pass deliberately leaves non-blank lines untouched.
- 564 warnings remain unaddressed (`comments-indentation`: 333, `indentation`: 231); these require
  different fixes (user-authored frontmatter comments and sequence-indentation style respectively)
  that could not be automated without risking user content.
- Adds one linear post-processing pass over the assembled YAML string. For typical workflow YAML
  this cost is negligible, but it is an additional allocation and scan on every compile.

#### Neutral
- The `normalizeBlankLines` function must be maintained if the generated YAML format changes in
  ways that require trailing whitespace to be semantically significant (currently no such case
  exists in YAML 1.1/1.2).
- Generated `.lock.yml` files are not included in this PR; they will reflect the change after
  `gh aw compile` is re-run post-merge.

---

*ADR created by [adr-writer agent] and finalized by [copilot-swe-agent].*
