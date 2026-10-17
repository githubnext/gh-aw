# ADR-69777: Gate Cloud Hypervisor Agent Enclaves Behind a Narrow Acceptance Fixture

**Date**: 2026-10-17
**Status**: Draft
**Deciders**: lpcox (PR author), [TODO: verify other reviewers]

---

### Context

[ADR-67772](67772-allow-cloud-hypervisor-primary-with-script-enclaves.md) let a Cloud Hypervisor primary agent run with static **script** enclaves. It also rejected **agent** enclaves under a Cloud Hypervisor primary, because the scoped executor bearer handoff and API-proxy path were not validated for a microVM primary. The companion AWF change (github/gh-aw-firewall#10171) now needs to run one end-to-end acceptance workflow, `smoke-enclave-cloud-hypervisor-copilot`, that combines a Cloud Hypervisor primary with a Cloud Hypervisor Copilot agent enclave. That workflow cannot compile while the blanket rejection in `validateCloudHypervisorEnclaves` (`pkg/workflow/enclaves_runtime.go`) is in place. The combination is still experimental and must not be presented as general or production support. Every existing fail-closed check must stay in force: no mixed runtimes, no image overrides, no dynamic enclaves, no GitHub tools, the API-proxy/provider route, and attested artifacts.

### Decision

We will keep rejecting Cloud Hypervisor agent enclaves under a Cloud Hypervisor primary. The only exception is a single acceptance fixture, enforced by the new `validateCloudHypervisorAgentAcceptance` validator. The exception applies only when every one of these conditions holds:

- **Opt-in:** The workflow sets the checked-in boolean `features.cloud-hypervisor-agent-acceptance: true`. An ambient `GH_AW_FEATURES` override does not count.
- **Workflow identity:** The workflow ID is exactly `smoke-enclave-cloud-hypervisor-copilot`.
- **Repository and branch:** The `if:` condition is exactly `github.repository == 'github/gh-aw-firewall' && github.ref_type == 'branch'`.
- **Trigger:** `workflow_dispatch` is the only trigger.
- **Enclave:** There is exactly one enclave, scoped to the static public repository `github/gh-aw-firewall`.
- **Engine and model:** Both the primary and the enclave use Copilot, the enclave model is `claude-sonnet-5`, and there is no profile override.
- **Limits:** The timeout, output, invocation, task, model-request and model-token limits must match exact values.
- **AWF version:** AWF is pinned to `v0.28.62`.

The main reason for this design is to unblock the one acceptance run while keeping every other configuration rejected at compile time with a clear error.

### Alternatives Considered

#### Alternative 1: Keep the blanket rejection and build the acceptance workflow with a private or forked compiler

The rule from ADR-67772 would stay as it is, and the AWF repository would compile the fixture with an out-of-tree patched compiler. This keeps the main compiler's surface unchanged. We rejected it because the patched compiler would not be versioned, reviewed or tested in gh-aw, so its gate could quietly drift from the checks the upstream compiler enforces. The PR still describes a source-built targeted backport, but the gate logic and its tests now live upstream.

#### Alternative 2: Generally allow agent enclaves under a Cloud Hypervisor primary

The restriction would be removed for all workflows, which matches ADR-67772's Alternative 2. This is simpler and avoids a fixture-specific special case. We rejected it because the scoped executor bearer handoff is still not production-ready, and the model request/token limits are only advisory in the pinned guest. General admission would let unvalidated configurations compile and then fail, or run with weaker guarantees, at runtime.

#### Alternative 3: Use a generic experimental feature flag without workflow, repository and parameter pinning

Any workflow that sets the feature flag would be admitted. This is less brittle than hard-coding exact values. We rejected it because the flag would effectively become general opt-in support, which is what ADR-67772's Alternative 3 warned against. Pinning the workflow ID, repository, trigger and limits keeps the blast radius to one manually dispatched workflow in one repository.

### Consequences

#### Positive
- The AWF acceptance workflow can compile with an upstream, reviewed and tested compiler gate (`pkg/workflow/enclaves_agent_acceptance_test.go`), and every other configuration still fails closed.
- Ambient `GH_AW_FEATURES` cannot enable the path. The opt-in must be committed in workflow frontmatter, which leaves an auditable trail.
- All existing Cloud Hypervisor enclave restrictions (mixed runtimes, image overrides, DinD, GitHub tools, API-proxy routing, attested artifacts) still apply.

#### Negative
- The compiler now contains hard-coded, fixture-specific values: a workflow ID, a repository, a model name, numeric limits and an AWF version. Any change on the AWF side means a coordinated gh-aw code change.
- This partly revisits ADR-67772's choice not to add a separate opt-in flag, so the compiler gains a new feature key that it must later remove or generalize.
- The model request/token settings are only advisory in the pinned guest, so the fixture's limits are not all enforced.

#### Neutral
- Successful compilation is explicitly not evidence of successful inference, Linux/KVM isolation acceptance or production readiness. The documentation in `docs/src/content/docs/experimental/enclaves.md` says so.
- The acceptance path depends on an AWF host built from a branch, paired with the `v0.28.62` attested guests. This compiler change does not provision or verify that host.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
