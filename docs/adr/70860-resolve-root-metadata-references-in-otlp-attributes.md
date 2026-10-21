# ADR-70860: Resolve Root Workflow Metadata References in OTLP Attributes at Compile Time

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan, Copilot coding agent

---

### Context

Workflows can declare a top-level `metadata` map, such as `version`, `category`, and `owner`. They can also declare custom OTLP span attributes (`observability.otlp.attributes`) and resource attributes (`observability.otlp.resource-attributes`). These OTLP maps are often defined once in a shared import, for example `shared/otlp.md`, and reused by many root workflows. Before this change, there was no way for those shared OTLP maps to pick up the importing workflow's metadata (issue #70377). Authors had to copy values by hand or pass them through import inputs. Separately, the parser's import merge (`mergeObservabilityConfigs`) kept span `attributes` but silently dropped imported `resource-attributes`. Any solution has to keep metadata from being exported unless the author opts in, and it must not open a path for GitHub Actions expression injection.

### Decision

We will support whole-value `${{ github.aw.metadata.<key> }}` references in both OTLP custom attribute maps. The compiler resolves them in `resolveOTLPMetadataReferences`, which runs right after imported observability configuration is merged. Values always come from the **root** workflow's top-level `metadata` map, never from the import's own metadata. Resolution is strict. The reference must be the entire attribute value and the key must be a simple identifier. Compilation fails if the key is missing, if the root workflow has no metadata, if the metadata value is not a string, or if it contains `${{` or `}}`. We will also merge imported `resource-attributes` across imports using the same first-occurrence-wins rule already used for span attributes, with the root workflow still taking precedence.

### Alternatives Considered

#### Alternative 1: Pass metadata values through import inputs

Shared OTLP imports could declare inputs that the root workflow fills with its metadata values. This approach uses existing machinery, which is why it was considered. It was rejected because every root workflow would have to repeat its metadata in an `imports` input block. That duplication is exactly the problem issue #70377 describes.

#### Alternative 2: Automatically export all metadata as OTLP attributes

The compiler could copy every `metadata` entry into the OTLP attribute maps without any references. This needs no new syntax. It was rejected because it would export data without anyone asking for it, and metadata may contain values that should not leave the workflow. The documentation added in this PR states that "Metadata is not exported automatically" and that "only explicitly selected values are exported".

#### Alternative 3: Resolve references at runtime as GitHub Actions expressions

The references could be passed through for GitHub Actions or the runtime to expand. This was rejected because `github.aw.metadata` is not a real Actions context. Resolving at runtime would also put metadata text into an expression-evaluation path. Resolving at compile time, and rejecting values that contain expression delimiters, keeps the compiled lock file static and closes off expression injection.

### Consequences

#### Positive
- Shared OTLP configuration can label telemetry with each importing workflow's own metadata, without duplicating values or passing them through import inputs.
- Strict validation (whole-value only, simple keys, string values, no `${{`/`}}` in values) turns mistakes into compile errors and prevents expression injection through metadata.
- Imported `resource-attributes` are no longer dropped silently. Span and resource attributes now share one merge helper (`mergeOTLPStringMapFirstWins`).

#### Negative
- The `${{ ... }}` syntax looks like a GitHub Actions expression but is a compiler-only pseudo-context. Authors may wrongly expect other Actions contexts or embedded interpolation to work there.
- Embedded references (for example `"v${{ github.aw.metadata.version }}"`), nested selectors, and non-string metadata are not supported. These limits may need to be relaxed later.
- The parser's attribute extraction is still a local copy of the workflow package's logic, kept separate to avoid a circular import. This PR widens that duplicated code to cover `resource-attributes`, so there is more to keep in sync.

#### Neutral
- Resolution always uses the root workflow's scope, even when the reference appears in an imported file. A `metadata` map inside the import itself is ignored for this purpose.
- Resolution runs inside `mergeImportedWorkflowConfiguration`, so it depends on running after imported observability is merged and before environment variables are built from the OTLP maps.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
