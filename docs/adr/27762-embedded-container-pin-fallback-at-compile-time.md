# ADR-27762: Embedded Container Pin Fallback at Compile Time

**Date**: 2026-04-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler emits container image references when producing `.lock.yml` output and manifest metadata. Before this change, image references were only pinned to a digest when a matching entry existed in the repo-local cache file (`.github/aw/actions-lock.json`). When this cache was absent or incomplete, builtin container images were emitted as mutable tags (e.g., `node:lts-alpine` rather than `node:lts-alpine@sha256:...`), undermining reproducibility and creating a supply chain security risk. The `pkg/actionpins` package already maintained an embedded JSON data file of action pins; the question was whether to extend this embedded store to cover container image pins as well.

### Decision

We will extend the embedded pin data model (`pkg/actionpins/data/action_pins.json`) with a `containers` section and expose a `GetContainerPin(image)` accessor. The workflow compiler will resolve container image pins using a two-level fallback: (1) repo-local cache, (2) embedded defaults. This ensures that builtin images are always emitted with a pinned digest at compile time, even when the local cache is missing or incomplete.

### Alternatives Considered

#### Alternative 1: Require a Complete Local Cache

Require automation to always produce a fully populated `.github/aw/actions-lock.json` before compilation. This would keep the compiler logic simple and avoid a second data source. It was rejected because it creates fragility in CI environments where the cache file may be missing (fresh clone, deleted, or not yet committed), requiring every user to maintain a complete set of builtin image pins manually.

#### Alternative 2: Fail the Build on Missing Pins

Return a compile error when a builtin container image has no pin entry in the local cache. This would make the security property explicit and fail loudly on gaps. It was rejected because it would break existing workflows that compile successfully today with partial or no cache, requiring a coordinated migration across all callers before the safety property could be enforced.

#### Alternative 3: Fetch Pins at Compile Time from a Remote Registry

Resolve the digest by querying the container registry live during compilation. This would always produce the freshest pin. It was rejected because it introduces a network dependency at compile time, increases latency, requires authentication credentials in the compilation environment, and would make builds non-reproducible across time.

### Consequences

#### Positive
- Builtin container images are always emitted with a sha256 digest in compiled output, eliminating mutable-tag references for known images.
- Compilation succeeds and produces pinned output even when the local cache file is absent or incomplete, reducing operational toil.

#### Negative
- Embedded pins must be kept up-to-date in the repository; a stale embedded pin will pin to an outdated (potentially vulnerable) image digest until the data file is refreshed.
- The pin resolution path now has two data sources (local cache + embedded defaults), increasing the complexity of the resolution logic and the surface area for subtle ordering bugs.

#### Neutral
- The `pkg/actionpins` data structure gains a new `containers` map field alongside the existing `entries` map; JSON files that omit `containers` are still valid (field is `omitempty`).
- Tests for `applyContainerPins` must use image references not present in the embedded data when asserting "no pin applied" behavior, since embedded pins now act as an implicit baseline.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Pin Resolution Order

1. The compiler **MUST** apply container image pins in the following priority order: (1) repo-local cache (`.github/aw/actions-lock.json`), then (2) embedded defaults (`pkg/actionpins/data/action_pins.json` `containers` section).
2. The compiler **MUST NOT** emit a mutable tag reference for any container image that has a matching entry in either the local cache or the embedded defaults.
3. The compiler **SHOULD** log which source (local cache or embedded defaults) was used to resolve each pin to aid debugging.

### Embedded Pin Data Model

1. The embedded pin data file **MUST** include a top-level `containers` object mapping image reference strings to objects with `image`, `digest`, and `pinned_image` fields.
2. The `containers` field **MAY** be absent from the JSON file; absence **MUST** be treated as an empty map (not an error).
3. Embedded container pin entries **MUST** be updated whenever the canonical builtin image tags resolve to new digests.
4. Implementations **MUST NOT** use the embedded pin store as the sole source of truth when a local cache entry exists for the same image.

### Accessor API

1. The `GetContainerPin(image string)` function **MUST** return the `ContainerPin` for the given image reference if one exists in the embedded data, or `(ContainerPin{}, false)` if not.
2. The accessor **MUST** be safe to call from multiple goroutines (the underlying load **MUST** use `sync.Once` or equivalent).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
