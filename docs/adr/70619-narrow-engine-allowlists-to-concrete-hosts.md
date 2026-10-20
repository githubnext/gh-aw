# ADR-70619: Narrow Engine Allowlists to Concrete Hosts and Guard Effective Suffix/Wildcard Coverage

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify additional reviewers]

---

### Context

Engine domain sets (`engine-defaults` in `pkg/workflow/data/ecosystem_domains.json`) are named allowlist bundles that workflows opt into through `network.allowed`. The AWF firewall treats a bare domain such as `openai.com` as a suffix rule, and `*.googleapis.com` as a wildcard. As a result, the Codex bundle (`openai.com`, `chatgpt.com`) and the Gemini bundle (`*.googleapis.com`) granted far more than the endpoints those engines actually need. For example, `*.googleapis.com` also covers `storage.googleapis.com`, which belongs to the Dart and Chrome package ecosystems. Audit findings (#70225, #70226, #70227, clustered in #70462) reported this. The existing guard test, `TestEngineDefaultDomainsDoNotOverlapEcosystems`, compared literal strings only, so it could not detect suffix or wildcard overlap with package ecosystems that the workflow did not opt into.

### Decision

We will limit engine domain bundles to concrete transport hosts. Codex keeps `api.openai.com`, gateway hosts, and GitHub hosts. Gemini keeps `generativelanguage.googleapis.com` and GitHub hosts. Optional vendor authentication and telemetry hosts move to a new explicit opt-in set named `codex-vendor` (`auth.openai.com`, `ab.chatgpt.com`). The `codex-vendor` set follows the existing `copilot-vendor` pattern and is preserved through inline and external threat detection. GitHub-hosted Codex inference, including detection, no longer gets direct `api.openai.com` access. We will also replace the literal-only guard with an AWF-semantics overlap check (`awfDomainRulesOverlap`). This check fails when any engine rule's effective suffix or wildcard coverage overlaps a package ecosystem: Node, Python, Go, Dart, Java, .NET, or Ruby. `github.com` is the only documented exception, because AWF has no exact-host-only spelling and repository access requires it.

### Alternatives Considered

#### Alternative 1: Keep the broad parent domains and rely on documentation

We could leave `openai.com`, `chatgpt.com`, and `*.googleapis.com` in place and document what they cover. This option has no compatibility risk for Codex login flows or Gemini auxiliary APIs. We rejected it because opting into an engine would still silently grant access to unrelated hosts, including package download hosts. That contradicts the documented rule that package ecosystems require explicit opt-in.

#### Alternative 2: Add an exact-host matching mode to the firewall

We could add an exact-host-only syntax to AWF. Then `github.com` could be allowed without also allowing `*.pkg.github.com`, and there would be no exception. This is the most precise fix. We did not choose it because it requires a firewall feature that is outside this repository and this change. The PR uses a documented `network.blocked: [pkg.github.com]` workaround instead. [TODO: verify whether this is tracked upstream]

#### Alternative 3: Fold vendor hosts into the engine bundle as exact subdomains

We could replace `openai.com` and `chatgpt.com` with `auth.openai.com` and `ab.chatgpt.com` inside the `codex` bundle itself. This would keep interactive authentication and telemetry working by default. We rejected it to match the `copilot-vendor` precedent: model transport stays separate from optional vendor authentication and telemetry, and the latter requires an explicit opt-in.

### Consequences

#### Positive
- Engine opt-ins grant only the concrete hosts the engine needs for transport. This removes implicit access to unrelated OpenAI, ChatGPT, and googleapis subdomains.
- The guard test now models AWF suffix and wildcard semantics. A future broad or wildcard entry that overlaps a package ecosystem fails the build instead of passing a literal comparison.
- GitHub-hosted Codex detection no longer reaches `api.openai.com`, which reduces the detection job's egress surface.
- Vendor opt-ins are modeled consistently (`copilot-vendor`, `codex-vendor`).

#### Negative
- Workflows that relied on Codex interactive login or telemetry hosts, or on other `*.googleapis.com` endpoints through the Gemini bundle, will now be blocked until they add `codex-vendor` or the matching ecosystem opt-in. This is a potentially breaking behavior change.
- `github.com` remains a documented exception: it still permits GitHub Packages hosts, so the invariant is not absolute.
- The overlap helper is test-only, so its AWF matching semantics must stay in sync with the firewall's real behavior.

#### Neutral
- 32 compiled `.lock.yml` workflows were regenerated. Their only change is the removal of the two Codex parent domains.
- `docs/src/content/docs/reference/network.md` was updated to describe the narrowed bundles, the `codex-vendor` set, and the GitHub Packages exception.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
