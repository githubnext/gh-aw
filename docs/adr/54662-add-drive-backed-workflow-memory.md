# ADR-54662: Add Drive-Backed Persistent Workflow Memory

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Agentic workflows need a way to persist state across runs. The existing options — `repo-memory` and `cache-memory` — both have meaningful drawbacks. `repo-memory` pollutes git history and requires write access to the repository. `cache-memory` entries are ephemeral: GitHub Actions cache is bounded in size and evicted after periods of disuse, so state is not guaranteed to survive long gaps between runs. A third backend is needed that offers durable, named, FUSE-mounted file storage scoped outside the repository's git history and accessible across workflow runs without expiry risk.

### Decision

We will introduce `tools.drive-memory` as an experimental third persistent-memory backend, backed by the `actions/gh-drives-preview` GitHub Drives service. The compiler mounts one or more named drives into agent, custom, and pre-activation jobs via FUSE, infers least-privilege permissions (`contents`, `id-token`, `drives`), validates and commits changes after successful runs, and threads drive contents through the existing threat-detection artifact handoff to preserve isolation guarantees. Users opt in with `tools.drive-memory: true` or a richer configuration object; the feature is explicitly marked experimental and gated on Linux runners enrolled in the GitHub Drives private preview.

### Alternatives Considered

#### Alternative 1: Continue using `cache-memory` only

`cache-memory` is already supported and requires no new external dependency. However, cache entries are evicted by GitHub after 7 days of disuse and are bounded by a per-repository quota. For workflows that run infrequently or accumulate large amounts of state, cache eviction silently discards agent memory, making it unsuitable as a durable long-term store.

#### Alternative 2: Continue using `repo-memory` only

`repo-memory` is durable — it commits state directly to the repository — but it pollutes git history with agent-generated commits, widens the required `contents: write` permission to the entire repository, and can conflict with concurrent workflows writing to the same branch. It also ties agent memory semantics tightly to branch history, which is undesirable for cross-branch or repository-wide state.

### Consequences

#### Positive
- Agents gain truly durable, named file storage that persists across runs without expiry risk and without adding commits to git history.
- Threat-detection isolation is preserved: when threat detection is enabled, drive contents are staged as an artifact and published only after detection succeeds, with concurrent-update detection to prevent silent overwrites.
- Multiple drives can be mounted per workflow, supporting both writable and restore-only modes, enabling scenarios such as a shared read-only reference drive alongside a writable notes drive.

#### Negative
- The feature takes a hard dependency on the `actions/gh-drives-preview` private preview, which has no versioned release and whose interface may change without notice; gh-aw pins a `main` SHA, creating maintenance overhead.
- Runner support is limited to GitHub-hosted `ubuntu-latest` with FUSE; self-hosted runners and job containers are not supported, narrowing the addressable user base.
- GitHub Drives allows only one active writer per drive, so workflows that run concurrently and write the same drive will contend for the writer lease, potentially causing failures.

#### Neutral
- The compiler automatically infers `id-token: write` and a new `drives` permission scope, which is added to the frontmatter schema alongside existing permission fields.
- Drive names are repository-wide and branch-aware according to the preview service, so naming conventions need documentation to avoid accidental cross-branch state sharing.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
