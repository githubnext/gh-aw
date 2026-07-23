# ADR-47515: Add Syft SBOM Scanning to Compile Pipeline

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

Compiled GitHub Actions workflows reference container images recorded in each `.lock.yml` file under a `gh-aw-manifest` header. The compile pipeline already offers opt-in vulnerability scanning via `--grype` (ADR-47474), but generates no Software Bill of Materials (SBOM) data for those images. SBOM generation is an increasingly mandatory supply-chain security control (NIST SSDF, Executive Order 14028), and surfacing it at compile time — before workflows are deployed — gives developers earlier visibility than a separate CI step. The existing Docker-based scanner model (used by `--runner-guard`, `--poutine`, `--grype`) provides a consistent integration pattern that avoids native binary prerequisites.

### Decision

We will integrate `anchore/syft` as an opt-in post-compilation step (`--syft`) that generates SBOM data for container images extracted from `gh-aw-manifest` headers in compiled `.lock.yml` files. Syft will be invoked via Docker (`docker run --rm anchore/syft:latest <imageRef> -o syft-json`) so no native install is required. The implementation reuses the existing `collectContainerImagesFromLockFiles` function (shared with Grype) and the `runBatchLockFileTool` dispatch pattern. In strict mode (`--strict`), any scan error causes a non-zero exit.

### Alternatives Considered

#### Alternative 1: Use Trivy for SBOM generation instead of Syft

Trivy (`aquasecurity/trivy`) supports SBOM output formats (CycloneDX, SPDX) and has a compatible Docker-based invocation model. It was not chosen because Syft is already the de-facto SBOM tool from Anchore — the same vendor as Grype — making the two tools natural companions in the compile pipeline. Using Syft maintains vendor consistency and leverages developer familiarity already established by Grype adoption.

#### Alternative 2: Extract SBOM data from Grype output instead of a separate tool

Grype can produce CycloneDX-format SBOM alongside vulnerability findings using `--add-cpes-if-none` and related flags. Reusing Grype would avoid adding a new Docker image. It was not chosen because SBOM generation is a distinct concern from vulnerability scanning — Grype's primary output is a vulnerability report, not an SBOM — and conflating the two would make the `--grype` flag ambiguous. Syft is purpose-built for SBOM generation and produces richer package metadata.

#### Alternative 3: Implement SBOM scanning as a separate CI step, outside `gh aw compile`

A dedicated CI job or reusable workflow action could scan images listed in lock-file manifests after compilation, keeping the compile pipeline lighter. This was not chosen because tight coupling of the SBOM scan to compilation (same invocation, same strict-mode gate, same output format) gives developers immediate local feedback via `gh aw compile --syft` without requiring CI configuration changes, consistent with how `--grype`, `--zizmor`, and `--runner-guard` are used today.

### Consequences

#### Positive
- No native Syft installation required; Docker is the only prerequisite, already mandated by `--runner-guard`, `--poutine`, and `--grype`.
- Reuses the existing `collectContainerImagesFromLockFiles` and `runBatchLockFileTool` infrastructure, minimising new code surface area.
- Developers receive SBOM package counts per image at compile time, enabling early supply-chain visibility without CI changes.
- The `--syft` flag follows the same opt-in pattern as all other post-compile tools; existing compile invocations are unaffected.

#### Negative
- Docker must be running; users without Docker (or where the daemon is unavailable) silently skip Syft scanning rather than receiving a hard error, which may create a false sense of security.
- `anchore/syft:latest` is pinned to a mutable tag. A breaking upstream release could cause non-deterministic failures across environments; a pinned digest would be safer.
- Each unique image requires a `docker run` invocation; cold-start Docker image pull adds significant latency on first use per machine.
- Syft SBOM output is currently printed as a package count only; the full SBOM artifact is not persisted to disk or attached to compilation output, limiting downstream use of the generated data.

#### Neutral
- The MCP tool interface (`mcp_tools_readonly.go`) exposes `syft` as a JSON schema field, making it available to AI-assisted compile invocations alongside `grype` and other scanners.
- Strict mode integration mirrors the `--grype` implementation: any scan error returns a non-zero exit when `--strict` is set, otherwise a warning is printed.
- The implementation parses `syft-json` output format to extract artifact counts but does not otherwise post-process or filter findings — SBOM evaluation is left to the consumer.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
