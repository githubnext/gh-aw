# ADR-70549: Authenticate Bootstrap Downloads with Release-Bound Digests in the Setup Action

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan, gh-aw maintainers [TODO: verify]

---

### Context

The setup action's bootstrap installers (`install_awf_binary.sh`, `install_copilot_cli.sh`, the generated Docker Compose step, and `install_gh_cli.sh`) download executables over HTTPS and check them against a checksum manifest (`checksums.txt` / `SHA256SUMS.txt`) fetched from the same release over the same channel. If a certificate authority trusted by the runner is compromised, an attacker who can intercept traffic can impersonate an allowed HTTPS host. That attacker can then replace both the artifact and its checksum file, so the checksum check provides no independent authentication. Domain allowlisting decides *where* a client may connect. It does not prove *who* answered or *what* bytes arrived. The compiler and generated Go runtime also need a toolchain that implements the relevant upstream distrust behavior.

### Decision

We will authenticate bootstrap artifacts against SHA256 digests bound to specific releases and supplied independently of network-served checksums. A new `actions/setup/sh/release_integrity.sh` holds a reviewed pin table with entries of the form `(repository, version, asset, sha256)`. AWF and Copilot CLI installers must verify their downloaded checksum manifest against this table or a literal workflow `checksums-sha256` override before using it. Overrides belong to `sandbox.agent` for AWF and `engine` for Copilot, require an explicit semantic release version, and propagate to matching detection/eval installations. Explicit Copilot pins bypass the toolcache and verify both host and requested musl archives. Docker Compose bytes and the GitHub CLI apt signing key are pinned directly. If a release has neither a built-in nor a supplied pin, the download fails closed; `latest` cannot be paired with a supplied pin. An AWF bundle that fails verification no longer falls back to the standalone binary. All bootstrap `curl` calls are limited to HTTPS for both the initial request and any redirects (`--proto '=https' --proto-redir '=https'`). Release builds require Go 1.26.10 or newer on the 1.26 branch, 1.27.3 or newer on the 1.27 branch, or a newer supported stable release; each native binary's embedded toolchain is recorded. Together, these steps separate destination authorization, transport authentication, and artifact integrity as distinct security boundaries.

### Alternatives Considered

#### Alternative 1: Rely on TLS validation plus network-served checksums (status quo)

Keep downloading `checksums.txt` / `SHA256SUMS.txt` alongside the artifact and rely on ordinary hostname and chain validation. This option needs no maintenance and supports any upstream version, including `latest`. We rejected it because a compromised CA defeats both checks together. An attacker who can impersonate the host can serve a matching artifact and checksum pair.

#### Alternative 2: Verify upstream cryptographic signatures or attestations (e.g., Sigstore / GitHub artifact attestations)

Verify publisher-signed provenance instead of maintaining a digest table. This would remove the manual pin updates and scale to new releases automatically. We did not choose it here because it needs signature or attestation support from every upstream (AWF, Copilot CLI, Docker Compose), plus verifier tooling and trust roots on the runner. Those trust roots would themselves need bootstrapping. The PR does not show that these prerequisites are in place. [TODO: verify whether this was evaluated]

#### Alternative 3: Pin only the final artifact digests in the compiler (per architecture)

Embed per-architecture artifact digests in compiled lock files, as threat-detect and Compose already do. This avoids trusting the manifest at all. For AWF and Copilot, which ship many platform and libc variants, pinning one manifest digest per release covers every variant with a single reviewed entry. For that reason, manifest pinning was chosen for those two, and direct pins were kept where they already existed.

### Consequences

#### Positive
- Replacing both an artifact and its checksum manifest over an intercepted TLS session no longer bypasses verification. Integrity is anchored in the trusted setup action or a literal, independently provisioned workflow pin.
- HTTPS-only redirects block downgrade-to-HTTP attacks during bootstrap. Tests use a real local HTTPS-to-HTTP redirect to confirm the rejection.
- Fail-closed behavior, with no AWF binary fallback after a bundle integrity failure, removes a silent path around verification.
- Release toolchain checks and recorded evidence make it auditable which Go `crypto/x509` behavior each shipped binary contains.

#### Negative
- **Breaking change**: AWF/Copilot downloads without a built-in or supplied pin and uncached `latest` downloads now fail. Users must select a reviewed built-in release or supply an explicit version and independently trusted checksum-manifest pin.
- New built-in releases need maintainer-reviewed pin updates in `release_integrity.sh`. Workflow authors can select other releases without waiting for that update, but must obtain their manifest digests through an independently authenticated channel.
- The trust anchor moves to the setup action and the compiler. A compromised setup action, runner image, or toolcache is out of scope and is not mitigated.
- Requiring a newer Go version (module go directive 1.26.10 and a preferred 1.27.3 toolchain) may break contributors or build environments using older toolchains.

#### Neutral
- Container images, MCP servers, Node/Python clients, and proxy trust stores are not patched or attested. `runtime_trust_inventory.sh` only logs diagnostic version evidence.
- The security architecture spec gains §6.8, "Transport Trust and Bootstrap Integrity" (TT-01..TT-04), and an emergency distrust procedure. Broader enforcement for package-manager downloads is recorded as a candidate rule, not an active `CTR-*` rule.
- Existing threat-detect asset pins and Cloud Hypervisor/NVX attestation checks are unchanged. Trusted runner toolcache selection is unchanged when no explicit Copilot manifest pin is supplied.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
