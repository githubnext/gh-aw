# ADR-53369: Extend `.grant.yaml` with an `ignore-images` Glob List to Skip Entire Images from License Scanning

**Date**: 2026-08-17
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The daily container scan reports 677 license-policy violations for `ghcr.io/oraios/serena:latest` — 566 cargo crates that ship no SPDX license metadata and 111 named-license flags from its Debian/GCC/Perl toolchain. All violations are unactionable: gh-aw runs Serena as a third-party MCP server and never links or redistributes any of its contents, so the findings cannot be resolved by upgrading packages or changing licensing. Grant's existing policy schema only supports per-package suppression via `ignore-packages`. Adding individual entries for ~700 packages would silently weaken the allowlist for every other scanned image and create a maintenance burden as the image evolves.

### Decision

We will add an `ignore-images` key to `.grant.yaml` that accepts a list of glob patterns matched against both the tagged and pinned (`@sha256:`) references of each container image. When an image matches, `gh aw compile --grant` skips it for license scanning entirely. The key is gh-aw–specific; grant unmarshals its policy non-strictly and ignores unknown keys, so `.grant.yaml` remains a valid grant config. Vulnerability scanning (`--grype`) is unaffected and still covers every image including excluded ones.

### Alternatives Considered

#### Alternative 1: Add ~700 `ignore-packages` entries for all offending Serena packages

Per-package exceptions are already the standard mechanism for known-good third-party packages (Alpine, Debian base OS, Node.js runtime). Adding entries for each Serena cargo crate would suppress the violations without schema changes. However, `ignore-packages` applies globally across all scanned images, so each entry silently reduces coverage for every other image. It also requires re-entry whenever Serena's dependency tree changes, creating an unbounded maintenance tail.

#### Alternative 2: Exclude the Serena image from the images collected for scanning (no code change to grant.go)

The images to scan are assembled from lock manifests before grant runs. Filtering at the collection layer rather than the scanning layer would also suppress violations but would be harder to discover and harder to document inline in the policy file. It would not benefit from grant's policy-file lifecycle (the exclusion would live in Go code, not in the auditable `.grant.yaml`).

### Consequences

#### Positive
- Eliminates 677 unactionable license violation reports without polluting the `ignore-packages` allowlist.
- The exclusion is explicit and self-documenting: the `.grant.yaml` comment and CONTRIBUTING.md entry explain exactly why the image is skipped.
- Glob patterns allow one entry to cover all tags of an image, reducing future maintenance.
- Vulnerability coverage via `--grype` is preserved for excluded images.

#### Negative
- `ignore-images` is a non-standard key not part of grant's schema; anyone unfamiliar with gh-aw's extension may not realize license scanning is being skipped for that image.
- The mechanism creates a precedent for excluding images from license review entirely; future uses require careful justification to avoid eroding license coverage.

#### Neutral
- Grant's non-strict unmarshaling means the policy file is technically valid for both grant and gh-aw, but the two tools parse overlapping YAML differently (grant ignores `ignore-images`; gh-aw ignores grant-native keys beyond `ignore-images`).
- The `ignore-images` filtering logic lives in `pkg/cli/grant.go` alongside existing grant integration code and is covered by new unit and integration tests.

---
