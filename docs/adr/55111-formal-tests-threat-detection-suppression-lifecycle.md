# ADR-55111: Formal Test Suite for Threat-Detection Suppression and Deprecation Lifecycle

**Date**: 2026-08-23
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The formal spec for compiler threat detection compliance (`specs/compiler-threat-detection-compliance/README.md`) defines precise behavioral norms for two lifecycle areas: the suppression lifecycle (§6.4 False-Positive Handling, T-CTR-024/025/029) and the rule deprecation lifecycle (§5.4 Deprecation Policy). The existing implementation in `pkg/workflow/threat_detection_suppression.go` already conforms to these norms at the code level, but the formal spec verifier flagged that no test cases were explicitly traced to the named spec requirements. For §5.4, no runtime deprecation registry exists yet — it is currently a documentation/process obligation rather than a code construct. The team needs tests that make conformance verifiable and detectable if the implementation drifts from the spec.

### Decision

We will introduce a formal test file (`pkg/workflow/threat_detection_suppression_lifecycle_formal_test.go`) whose test functions are explicitly named and documented against their governing spec section and requirement identifiers (T-CTR-024/025/029, §5.4). For §6.4, these tests exercise the existing production implementation directly. For §5.4, we will add a stub `deprecationRegistry` state machine within the test file to model the required transition semantics until a concrete runtime implementation is built; at that point the stub tests will be replaced with direct assertions against the real implementation.

### Alternatives Considered

#### Alternative 1: Add Conventional Unit Tests Without Spec Tracing

Add standard Go unit tests for suppression validation and expiry behaviour without referencing specific spec sections or requirement IDs in the test names or comments. This satisfies code coverage metrics but provides no explicit link between tests and spec obligations. Future readers cannot easily determine which spec norms are covered, making compliance drift harder to detect. Rejected because the spec verifier's concern is precisely about traceability, not just coverage.

#### Alternative 2: Implement a Full Runtime Deprecation Registry Before Testing

Build a complete, production-grade `DeprecationRegistry` type in `pkg/workflow` (not test-only) before writing any spec tests for §5.4. This avoids test-only stub code, but blocks any spec test progress until the registry design is agreed upon, which could take multiple PRs. Rejected because §5.4 is currently a documentation obligation only — deferring tests until a runtime implementation exists leaves a spec compliance gap that the verifier will continue to flag. The stub approach closes the traceability gap immediately while signalling that a real implementation is needed.

### Consequences

#### Positive
- Spec norms (§6.4, §5.4) are explicitly traced to named test cases, making future implementation drift detectable by the spec verifier.
- Suppression lifecycle edge cases (expiry at UTC day boundary, audit field retention, rule-ID isolation) are now concretely tested against the production implementation.
- The stub deprecation registry clearly documents the expected state-machine semantics and serves as a design contract for the eventual runtime implementation.

#### Negative
- The stub `deprecationRegistry` and its tests live in test-only code and do not exercise any production logic for §5.4; they must be manually replaced once a real implementation lands.
- Spec section references embedded in test comments (e.g., `§5.4`, `§6.4`, `T-CTR-024`) can become stale if the spec is renumbered or reorganised without a corresponding test update.

#### Neutral
- No changes to production source files are required by this PR; all additions are confined to the test file.
- The `//go:build !integration` tag ensures these formal tests run with the standard unit test suite and are excluded from integration-only builds.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
