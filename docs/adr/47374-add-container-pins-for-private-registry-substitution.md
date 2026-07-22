# ADR-47374: Add `container_pins` to `aw.json` for Private-Registry Container Image Substitution

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw compiler supports `action_pins` in `aw.json` to redirect GitHub Action references to internal mirrors, enabling private-cloud and air-gapped enterprise environments to compile workflows without public GitHub access. However, container images referenced in job containers, MCP gateway containers, and custom MCP server containers still resolve exclusively against public registries (e.g., `ghcr.io`, Docker Hub). In environments where these public registries are unreachable, compilation produces output that cannot be executed at runtime. Enterprises need a symmetric mechanism to redirect container image references to internal mirrors at compile time.

### Decision

We will add a `container_pins` key to `aw.json` that maps source container image references (e.g., `ghcr.io/owner/image:tag`) to SHA-256 digest-pinned replacement image objects. Each mapping value is an object with separate `image` (ref name without digest) and `digest` (`sha256:<64-hex>`) fields, so that each component is independently schema-validated. Together they form the immutable pinned reference `image@digest`. The substitution is applied at compile time, before digest-pin resolution, so the compiled `.lock.yml` files bake in the mapped registry. The feature mirrors the existing `action_pins` design: repository-level configuration, exact key matching only, informational console messages per applied mapping (deduplicated), and a defensive copy of the combined mapping attached to `WorkflowData` at compile time.

Example `aw.json`:

```json
{
  "container_pins": {
    "ghcr.io/owner/image:tag": {
      "image": "registry.acme.com/image:tag",
      "digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    }
  }
}
```

### Alternatives Considered

#### Alternative 1: Runtime registry mirroring via Docker daemon configuration

Enterprise users could configure registry mirrors at the Docker/container-runtime layer (e.g., `registry-mirrors` in `daemon.json`) without any changes to the compiler. This delegates the redirect entirely to infrastructure configuration.

This was not chosen because it requires coordination outside the compilation toolchain, does not integrate with the compiler's digest-pinning guarantees, and provides no visibility into which images were redirected during compilation. It also doesn't cover the MCP gateway command-line image reference, which is rendered as a literal string in the compiled output.

#### Alternative 2: Wildcard or prefix-based container image mapping

Instead of requiring exact source image references as keys, the mapping could support prefix patterns (e.g., `ghcr.io/github/*` → `registry.acme.com/github/*`) or glob wildcards, reducing the number of entries users must maintain.

This was not chosen because wildcard matching is harder to validate at schema load time, introduces ambiguity when multiple patterns match a single image, and could silently redirect images the user did not intend to map. Exact matching is consistent with the `action_pins` design and makes the substitution table auditable.

### Consequences

#### Positive
- Enterprise users in private-cloud and air-gapped environments can now compile workflows that reference container images without requiring public registry access at runtime.
- Replacement values are required to be SHA-256 digest-pinned, enforcing immutable image references and providing a supply-chain security guarantee equivalent to action pinning.
- The substitution is visible in the compiled `.lock.yml` output and announced via informational console messages during compilation, making redirects auditable.

#### Negative
- Each source image version must be mapped individually — wildcard and prefix matching are not supported — so users must add a new `container_pins` entry each time a container image version changes.
- Users must re-run `gh aw compile` after modifying `container_pins` in `aw.json`; stale compiled lock files will continue to reference the old (possibly unmapped) image until recompiled.

#### Neutral
- `container_pins` is a repository-level setting in `aw.json` and is not supported in individual workflow frontmatter, consistent with the `action_pins` precedent.
- The substitution is applied at three distinct resolution sites — job container images (`resolveContainerImage`), Docker pre-download manifests (`applyContainerPins`), and MCP container commands (`applyContainerPinMappingFromData`) — requiring callers at each site to be updated when the feature is extended.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
