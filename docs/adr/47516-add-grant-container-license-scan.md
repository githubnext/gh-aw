# ADR-47516: Add Grant Container Image License Scanning to Compile Pipeline

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

Compiled GitHub Actions workflows reference container images recorded in each `.lock.yml` file under a `gh-aw-manifest` header. The compile pipeline already enforces container image vulnerability policy via `--grype` (Anchore Grype), but has no mechanism to enforce software license compliance on those images. A repository-level license policy file (`.grant.yaml`) exists but is not applied at compile time. Developers and security teams need a way to detect denied-license packages in container images before workflows ship, without requiring a native scanner binary on every machine.

### Decision

We will integrate Anchore Grant as an opt-in post-compilation step (`--grant`) that scans container images extracted from `gh-aw-manifest` headers in compiled `.lock.yml` files. Grant will be invoked via Docker (`docker run --rm -v .grant.yaml:/tmp/policy.yaml:ro anchore/grant:latest --config /tmp/policy.yaml --output json check <image>`) so no native install is required. Images are collected from lock files using the same `collectContainerImagesFromLockFiles` helper used by Grype, deduplicated by pinned reference before scanning. Findings with `"decision": "deny"` are surfaced as `console.CompilerError` entries. In strict mode (`--strict`), any denied-license finding or scan error causes a non-zero exit.

### Alternatives Considered

#### Alternative 1: Extend Grype to report license findings alongside CVEs

Grype's primary focus is CVE detection; it does not evaluate license policy. Adding license output to Grype would require maintaining a fork or relying on unstable experimental features. Rejected because it conflates two distinct concerns (security vulnerabilities vs. license compliance) into a single tool, and Grant is purpose-built for license policy enforcement with first-class policy-file support.

#### Alternative 2: Use FOSSA or Snyk for license scanning

FOSSA and Snyk offer license scanning as a service with richer policy management and audit trails. Both require network access to a managed service and API credentials, making them unsuitable for offline or air-gapped compile runs. Grant runs fully locally via Docker and reads the existing `.grant.yaml` policy file, requiring no external service or credential management.

#### Alternative 3: Perform license scanning as a separate CI step outside the compile pipeline

A dedicated CI job could invoke grant on images listed in manifests after compilation. This keeps the compile pipeline lightweight and lets scanning scale independently. Rejected because coupling the scan to `gh aw compile --grant` gives developers immediate local feedback without requiring CI configuration changes, and aligns with the existing pattern established by `--grype`, `--zizmor`, `--poutine`, and `--runner-guard`.

### Consequences

#### Positive
- License compliance is enforced at compile time using the repository's own `.grant.yaml` policy, with no additional configuration needed.
- No native Grant installation required; Docker is the only prerequisite (already required by `--runner-guard`, `--poutine`, and `--grype`).
- Grant findings are surfaced through the existing `console.CompilerError` format, giving a unified developer experience alongside other scanner output.
- Strict mode integration (`--strict`) allows CI pipelines to gate on license violations without additional scripting.
- Follows the established opt-in scanner plugin pattern; existing compile invocations are unaffected.

#### Negative
- Docker must be running; users without a running Docker daemon silently skip grant scanning rather than receiving a hard error, which may create a false sense of license compliance.
- `anchore/grant:latest` is pinned to a mutable tag. Upstream releases may produce different results or break scanning non-deterministically across environments.
- Requires `.grant.yaml` to exist at the repository root; if absent, the scan fails rather than skipping gracefully, blocking the compile run.
- Each unique container image requires a separate `docker run` invocation; the first use per machine triggers a Docker image pull, adding significant latency.

#### Neutral
- The `--grant` flag follows the same opt-in model as `--zizmor`, `--poutine`, `--actionlint`, `--runner-guard`, and `--grype`.
- The MCP tool interface (`mcp_tools_readonly.go`) exposes `grant` as a JSON schema field, making it available to AI-assisted compile invocations.
- Grant's policy evaluation is determined by `.grant.yaml`; changes to that file change scan outcomes without code changes to this feature.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
