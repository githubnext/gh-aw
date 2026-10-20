# ADR-70549: Authenticate Bootstrap Downloads with Release-Bound Digests in the Setup Action

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Pending gh-aw maintainer review in [PR #70549](https://github.com/github/gh-aw/pull/70549); no named maintainer approval is recorded here.

---

### Context

Before this change, the AWF and Copilot CLI bootstrap installers (`install_awf_binary.sh` and `install_copilot_cli.sh`) downloaded executables over HTTPS and checked them against a checksum manifest (`checksums.txt` / `SHA256SUMS.txt`) fetched from the same release over the same channel. The generated Docker Compose step and `install_gh_cli.sh` also depend on HTTPS bootstrap downloads. If a certificate authority trusted by the runner is compromised, an attacker who can intercept traffic can impersonate an allowed HTTPS host. That attacker can then replace both an artifact and its checksum file, so the checksum check provides no independent authentication. Domain allowlisting decides *where* a client may connect. It does not prove *who* answered or *what* bytes arrived. The compiler and generated Go runtime also need a toolchain that implements the relevant upstream distrust behavior.

The incident motivating this change is corroborated by [Mozilla's root-program announcement](https://groups.google.com/a/mozilla.org/g/dev-security-policy/c/0R3XLJL6FuE), which reports compromised eMudhra intermediate CA private keys and describes Mozilla's blocking response. The [Go team's security release announcement](https://groups.google.com/g/golang-announce/c/66pojueaQek) identifies CVE-2026-111943 and the intermediate-certificate filtering fix in Go 1.26.10 and 1.27.3. These are primary sources for their respective responses, not evidence that every runner trust store is patched. The [incident website](https://emudhra.we-have-a-problem.com) is a secondary report, not an official eMudhra or CA/Browser Forum notice.

### Decision

The proposed decision is to authenticate bootstrap artifacts against SHA256 digests bound to specific releases and supplied independently of network-served checksums. A new `actions/setup/sh/release_integrity.sh` holds a reviewed pin table with entries of the form `(repository, version, asset, sha256)`. AWF and Copilot CLI installers must verify their downloaded checksum manifest against this table or a literal workflow `checksums-sha256` override before using it. Overrides belong to `sandbox.agent` for AWF and `engine` for Copilot, require an explicit semantic release version, and propagate to matching detection/eval installations. Explicit Copilot pins bypass the toolcache and verify both host and requested musl archives. Docker Compose bytes and the GitHub CLI apt signing key are pinned directly; the generated Compose step must keep its verifier inside the trusted compiled step rather than sourcing a helper from a writable runtime directory. If a release has neither a built-in nor a supplied pin, the download fails closed; `latest` cannot be paired with a supplied pin. An AWF bundle that fails verification no longer falls back to the standalone binary. The hardened AWF, Copilot CLI, generated Docker Compose, and GitHub CLI installer `curl` calls are limited to HTTPS for both the initial request and any redirects (`--proto '=https' --proto-redir '=https'`); this is not a guarantee for every bootstrap download path. Release builds require Go 1.26.10 or newer on the 1.26 branch, 1.27.3 or newer on the 1.27 branch, or a newer supported stable release; each native binary's embedded toolchain is recorded. Together, these steps separate destination authorization, transport authentication, and artifact integrity as distinct security boundaries.

### Alternatives Considered

#### Alternative 1: Rely on TLS validation plus network-served checksums (status quo)

Keep downloading `checksums.txt` / `SHA256SUMS.txt` alongside the artifact and rely on ordinary hostname and chain validation. This option needs no maintenance and supports any upstream version, including `latest`. We rejected it because a compromised CA defeats both checks together. An attacker who can impersonate the host can serve a matching artifact and checksum pair.

#### Alternative 2: Verify upstream cryptographic signatures or attestations (e.g., Sigstore / GitHub artifact attestations)

Verify publisher-signed provenance instead of maintaining a digest table. With an enforced publisher and signer-workflow policy, signatures or attestations could reduce manual digest maintenance. This still requires trusted verifier tooling and trust roots, release/tag binding, and authenticated coverage for each selected asset; signature validity alone does not establish the intended publisher or release.

The bounded evaluation for this ADR inspected the bootstrap installers and existing guest attestation paths and queried GitHub's attestation API for the bundled AWF v0.28.62 `checksums.txt` digest and Copilot CLI v1.0.98 `SHA256SUMS.txt` digest on 2026-10-20. Both queries returned HTTP 404. These results do not prove that attestations or signatures are absent: visibility, API access, other assets, and separately published bundles were not exhaustively evaluated. This review therefore does not establish a verified signature/attestation replacement covering the selected bootstrap manifests and Compose assets. Release-bound pins provide the bounded control in this PR; signature-based verification remains a possible future enhancement, not a rejected mechanism. Existing Cloud Hypervisor/NVX guest attestation requirements remain in force and must not fall back to digest-only verification.

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
- The inline Maven bootstrap remains outside the hardened installer paths. This decision does not establish independent artifact authentication or HTTPS-only redirect enforcement for Maven.
- The security architecture spec gains §6.8, "Transport Trust and Bootstrap Integrity" (TT-01..TT-04), and an emergency distrust procedure. Broader enforcement for package-manager downloads is recorded as a candidate rule, not an active `CTR-*` rule.
- Existing threat-detect asset pins and Cloud Hypervisor/NVX attestation checks are unchanged. Trusted runner toolcache selection is unchanged when no explicit Copilot manifest pin is supplied.

---

*Draft prepared by the adr-writer agent and updated with source and implementation evidence. Maintainer acceptance remains pending; completion of this text does not change its status to Accepted.*
