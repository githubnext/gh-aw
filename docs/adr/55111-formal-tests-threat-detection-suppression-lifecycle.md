# ADR-55111: Formal Test Suite for Threat-Detection Suppression and Deprecation Lifecycle

**Date**: 2026-08-23
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The formal spec for compiler threat detection compliance (`specs/compiler-threat-detection-compliance/README.md`) defines precise behavioral norms for two lifecycle areas: the suppression lifecycle (§6.4 False-Positive Handling, T-CTR-024/025/029) and the rule deprecation lifecycle (§5.4 Deprecation Policy). The existing implementation in `pkg/workflow/threat_detection_suppression.go` already conforms to these norms at the code level, but the formal spec verifier flagged that no test cases were explicitly traced to the named spec requirements. For §5.4, no runtime deprecation registry exists yet — it is currently a documentation/process obligation rather than a code construct. The team needs tests that make conformance verifiable and detectable if the implementation drifts from the spec.

### Decision

We will introduce formal test files whose test functions are explicitly named and documented against their governing spec section and requirement identifiers. For §6.4 (T-CTR-024/025/029), `pkg/workflow/threat_detection_suppression_lifecycle_formal_test.go` exercises the existing production implementation directly, and asserts the audit fields on the serialized `gh-aw-manifest` emitted into the lock file rather than only on the parsed struct. For §5.4, `pkg/workflow/threat_detection_deprecation_policy_formal_test.go` parses the real specification artifacts (Section 5.1 catalog, Section 7.1 mapping, Section 8.1 test catalog, Section 10 change log, and the compliance crosswalk) and verifies each deprecation obligation against them; because no rule is deprecated today, fixtures exercising a deprecated row (compliant and non-conforming variants) prove the verifier detects violations.

### Alternatives Considered

#### Alternative 1: Add Conventional Unit Tests Without Spec Tracing

Add standard Go unit tests for suppression validation and expiry behaviour without referencing specific spec sections or requirement IDs in the test names or comments. This satisfies code coverage metrics but provides no explicit link between tests and spec obligations. Future readers cannot easily determine which spec norms are covered, making compliance drift harder to detect. Rejected because the spec verifier's concern is precisely about traceability, not just coverage.

#### Alternative 2: Model §5.4 With a Test-Local Deprecation State Machine

Add a stub `deprecationRegistry` type inside the test file to model the required transition semantics until a runtime registry exists. This documents the expected state machine, but the assertions cannot fail when the real specification, catalog, compliance gate, or change log violates §5.4, so it provides no compliance signal. Rejected in favour of verifying the specification artifacts themselves, which are the actual normative surface for §5.4.

#### Alternative 3: Implement a Full Runtime Deprecation Registry Before Testing

Build a complete, production-grade `DeprecationRegistry` type in `pkg/workflow` (not test-only) before writing any spec tests for §5.4. This blocks any spec test progress until the registry design is agreed upon, which could take multiple PRs, and §5.4 obligations are expressed in specification documents rather than compiler behaviour, so a runtime registry would not verify them anyway.

### Consequences

#### Positive
- Spec norms (§6.4, §5.4) are explicitly traced to named test cases, making future implementation drift detectable by the spec verifier.
- Suppression lifecycle edge cases (expiry at UTC day boundary, audit field retention, rule-ID isolation) are now concretely tested against the production implementation.
- §5.4 obligations (catalog row retention, cleared Section 7.1 mapping, `[DEPRECATED]` test IDs excluded from the required gate, change-log entry) are checked against the real specification artifacts, so a non-conforming future deprecation fails CI.

#### Negative
- The §5.4 verifier depends on the annotation conventions now documented in the compliance README; a deprecation written in a different form is reported as a violation until either the spec or the conventions are updated.
- Spec section references embedded in test comments (e.g., `§5.4`, `§6.4`, `T-CTR-024`) can become stale if the spec is renumbered or reorganised without a corresponding test update.

#### Neutral
- No changes to production source files are required by this PR; the additions are confined to test files and the compliance README, which documents the deprecation annotation conventions.
- The `//go:build !integration` tag ensures these formal tests run with the standard unit test suite and are excluded from integration-only builds.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
