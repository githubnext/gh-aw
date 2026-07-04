# ADR-43343: Two-Layer Regression Tests for gh-aw-firewall Digest Pinning

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Consumer-compiled lock files produced by gh-aw v0.82.2 emitted all four `gh-aw-firewall` images (`agent`, `api-proxy`, `cli-proxy`, `squid`) with tag-only references — no `digest` or `pinned_image` field. The embedded fallback pins in `action_pins.json` were already correct for the default version (`0.27.22`), but the existing regression tests only covered the older `0.27.0` version and did not cover `cli-proxy` (introduced in v0.82) at all. This coverage gap meant the regression shipped silently: no test failed, yet consumers received unpinned images. Without a test guard, any future `DefaultFirewallVersion` bump or sidecar addition carries the same silent-regression risk.

### Decision

We will add two regression tests tied to `constants.DefaultFirewallVersion` that together verify end-to-end digest pinning for all current firewall sidecars:

1. **Unit test** (`TestApplyContainerPins_DefaultFirewallVersion` in `docker_pin_test.go`) — asserts that each of the four images has a valid entry in the embedded pin table (non-empty `Digest` and `PinnedImage`), confirming `action_pins.json` coverage.
2. **End-to-end compile test** (`TestCompileWorkflow_FirewallImagesPinnedForDefaultVersion` in `docker_firewall_pin_compile_test.go`) — compiles a minimal workflow with `tools.github.mode: gh-proxy` (so `cli-proxy` is included) and asserts that the resulting lock file contains full `image`/`digest`/`pinned_image` metadata and correctly encoded `--image-tag` AWF config for all four images.

By binding both tests to `constants.DefaultFirewallVersion`, any version bump that does not refresh `action_pins.json` will immediately cause CI failures rather than silently shipping unpinned images.

### Alternatives Considered

#### Alternative 1: Fix action_pins.json without adding tests

Correct the embedded pin data for `0.27.22` and `cli-proxy` without introducing new tests. This removes the immediate regression but leaves the root cause — absence of a test guard — unaddressed. The next `DefaultFirewallVersion` bump or new sidecar addition would carry the same silent-regression risk. Rejected because it treats the symptom rather than the systemic gap.

#### Alternative 2: Unit-only test (skip the compile test)

Add the pin-table unit test (`TestApplyContainerPins_DefaultFirewallVersion`) but omit the compile test. This would catch missing `action_pins.json` entries but would not detect integration-level failures where the pin table is populated yet the compiler fails to emit `digest`/`pinned_image` fields into the lock file. The compile test provides a separate failure mode that the unit test cannot cover. Rejected because the historical regression was an integration-level failure (correct pin data, incorrect lock-file output), which only an end-to-end compile test would have caught.

### Consequences

#### Positive
- Future `DefaultFirewallVersion` bumps will fail CI immediately if `action_pins.json` is not updated with correct digests for all sidecars.
- The `cli-proxy` sidecar (new in v0.82) is now explicitly covered alongside the three legacy images, closing the coverage gap that allowed the v0.82.2 regression.
- Both tests use `constants.DefaultFirewallVersion` rather than a hardcoded version string, so they automatically track the canonical version constant.

#### Negative
- The compile test hardcodes specific digest SHA-256 values for the `0.27.22` release; when firewall images are rebuilt (e.g., base-image security patches), the test expected values must be updated even if the logic is unchanged.
- Two test layers (unit + compile) increase the test maintenance surface. Every time the compiler's lock-file schema changes, both tests may need updating.

#### Neutral
- The compile test requires `tools.github.mode: gh-proxy` in the workflow frontmatter; this is a test-only detail that does not affect production behavior.
- No production source files were changed in this PR — the change is confined to `_test.go` files, so there is no runtime impact on consumers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
