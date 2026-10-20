# ADR-70619: Narrow Engine Allowlists to Concrete Hosts and Guard Effective Suffix/Wildcard Coverage

**Date**: 2026-10-20
**Status**: Proposed (pending maintainer acceptance)
**Proposed decider**: pelikhan (PR author). No additional human decider or ADR acceptance is recorded in the PR reviews as of 2026-10-20; automated reviews do not establish design approval.

---

### Context

Engine domain sets (`engine-defaults` in `pkg/workflow/data/ecosystem_domains.json`) are named allowlist bundles that workflows opt into through `network.allowed`. The AWF firewall treats a bare domain such as `openai.com` as a suffix rule, and `*.googleapis.com` as a wildcard. As a result, the Codex bundle (`openai.com`, `chatgpt.com`) and the Gemini bundle (`*.googleapis.com`) granted far more than the endpoints those engines actually need. For example, `*.googleapis.com` also covers `storage.googleapis.com`, which belongs to the Dart and Chrome package ecosystems. Audit findings (#70225, #70226, #70227, clustered in #70462) reported this. The existing guard test, `TestEngineDefaultDomainsDoNotOverlapEcosystems`, compared literal strings only, so it could not detect suffix or wildcard overlap with package ecosystems that the workflow did not opt into.

### Decision

We will limit engine domain bundles to concrete transport hosts. Codex keeps `api.openai.com`, gateway hosts, and GitHub hosts. Gemini keeps `generativelanguage.googleapis.com` and GitHub hosts. Optional vendor authentication and telemetry hosts move to a new explicit opt-in set named `codex-vendor` (`auth.openai.com`, `ab.chatgpt.com`). The `codex-vendor` set follows the existing `copilot-vendor` pattern and is preserved through inline and external threat detection. GitHub-hosted Codex inference, including detection, no longer gets direct `api.openai.com` access. We will also replace the literal-only guard with an AWF-semantics overlap check (`awfDomainRulesOverlap`). This check fails when any engine rule's effective suffix or wildcard coverage overlaps a package ecosystem: Node, Python, Go, Dart, Java, .NET, or Ruby. `github.com` is the only documented exception, because AWF has no exact-host-only spelling and repository access requires it.

### Alternatives Considered

#### Alternative 1: Keep the broad parent domains and rely on documentation

We could leave `openai.com`, `chatgpt.com`, and `*.googleapis.com` in place and document what they cover. This option has no compatibility risk for Codex login flows or Gemini auxiliary APIs. We rejected it because opting into an engine would still silently grant access to unrelated hosts, including package download hosts. That contradicts the documented rule that package ecosystems require explicit opt-in.

#### Alternative 2: Add an exact-host matching mode to the firewall

We could add an exact-host-only syntax to AWF. Then `github.com` could be allowed without also allowing `*.pkg.github.com`, and there would be no exception. This is the most precise fix. We did not choose it because it requires a firewall feature that is outside this repository and this change. The PR uses a documented `network.blocked: [pkg.github.com]` workaround instead. No dedicated upstream exact-host-only tracking issue was verified in the 2026-10-20 review: searches of open and closed `github/gh-aw-firewall` issues for `"exact-host"` and `"exact domain"` did not identify one. This records the verification limit, not a claim that no related work exists; the decision does not depend on an upstream feature or commitment.

#### Alternative 3: Fold vendor hosts into the engine bundle as exact subdomains

We could replace `openai.com` and `chatgpt.com` with `auth.openai.com` and `ab.chatgpt.com` inside the `codex` bundle itself. This would keep interactive authentication and telemetry working by default. We rejected it to match the `copilot-vendor` precedent: model transport stays separate from optional vendor authentication and telemetry, and the latter requires an explicit opt-in.

### Consequences

#### Positive
- Engine opt-ins name concrete transport hosts rather than vendor-wide parent domains. AWF still applies suffix matching to those hosts; this removes implicit access to sibling OpenAI, ChatGPT, and googleapis subdomains, not to descendants of each retained host.
- The guard test now models AWF suffix and wildcard semantics. A future broad or wildcard entry that overlaps a package ecosystem fails the build instead of passing a literal comparison.
- GitHub-hosted Codex detection no longer reaches `api.openai.com`, which reduces the detection job's egress surface.
- Vendor opt-ins are modeled consistently (`copilot-vendor`, `codex-vendor`).

#### Negative
- Workflows that relied on hosts removed from the Codex or Gemini bundles must explicitly opt into the required hosts or ecosystems. This is a potentially breaking behavior change; `codex-vendor` restores only its two listed hosts, not the former vendor-wide suffixes.
- `github.com` remains a documented exception: it still permits GitHub Packages hosts, so the invariant is not absolute.
- The overlap helper is test-only, so its AWF matching semantics must stay in sync with the firewall's real behavior.

#### Neutral
- 32 compiled `.lock.yml` workflows were regenerated. Their only change is the removal of the two Codex parent domains.
- `docs/src/content/docs/reference/network.md` was updated to describe the narrowed bundles, the `codex-vendor` set, and the GitHub Packages exception.

#### Compatibility migration

For Codex authentication or telemetry at `auth.openai.com` and `ab.chatgpt.com`, add `codex-vendor` to the existing `network.allowed` list. Keep existing entries: this is an additional opt-in, not a replacement allowlist. Any other host formerly reachable through `openai.com` or `chatgpt.com` needs its own explicit domain.

For Gemini workflows that used other Google API hosts, add the required concrete domain or a matching ecosystem. For example, `dart` grants `storage.googleapis.com` for Dart/Flutter downloads; `codex-vendor` does not restore Google API access.

GitHub-hosted Codex inference and detection no longer inherit direct `api.openai.com` access. A workflow intentionally making separate direct OpenAI requests must explicitly allow `api.openai.com`; the normal GitHub-hosted transport does not need it. Explicit `codex-vendor` access is preserved through inline and external detection.

---

*Reviewed for PR #70619 on 2026-10-20. Maintainer acceptance remains pending; linking this record does not itself signify approval.*
