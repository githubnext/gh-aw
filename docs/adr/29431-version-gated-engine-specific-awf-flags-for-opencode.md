# ADR-29431: Version-Gated Engine-Specific AWF Flags for OpenCode

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: lpcox, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` compiler translates workflow definitions into AWF (Agentic Workflow Firewall) CLI invocations. Different engines require different AWF flags and open network ports. The OpenCode engine introduces a dynamic provider routing proxy that listens on port 10004 and is activated by the AWF flag `--enable-opencode`. AWF itself did not support this flag before v0.25.30; passing an unrecognized flag to an older AWF version causes the run to fail at startup with an "unknown flag" error. An existing version-gating mechanism (`awfSupportsAllowHostPorts`) already handles the same problem for `--allow-host-ports`, establishing a pattern the codebase can follow.

### Decision

We will emit `--enable-opencode` and append port 10004 to `--allow-host-ports` in `BuildAWFArgs()` when (a) the workflow engine is `opencode` and (b) the effective AWF version is ≥ v0.25.30. Both conditions must be true simultaneously, enforced through a dedicated version-gate function `awfSupportsEnableOpenCode`. The `AWFEnableOpenCodeMinVersion` constant is introduced in `pkg/constants/version_constants.go` to make the threshold explicit and testable. Because `AWFEnableOpenCodeMinVersion` (v0.25.30) is greater than `AWFAllowHostPortsMinVersion` (v0.25.24), when the OpenCode flag is enabled, `--allow-host-ports` support is always already guaranteed.

### Alternatives Considered

#### Alternative 1: Unconditional flag emission (no version gate)

Emit `--enable-opencode` and port 10004 for every OpenCode workflow regardless of AWF version. This is simpler and eliminates the version-gate bookkeeping. It was rejected because OpenCode workflows that pin an older AWF image version (e.g., in a workflow's `firewall.version` field) would receive an unknown flag and fail immediately at AWF startup — a hard runtime error with no graceful fallback.

#### Alternative 2: Dedicated firewall config field for OpenCode ports

Add a first-class field (e.g., `firewall.opencodePorts`) to the workflow's sandbox configuration so operators can declare the port list explicitly rather than deriving it from the engine name. This was considered because it separates the network port decision from the engine-name string, making each workflow's intent explicit. It was rejected because no existing workflow uses such a field, the engine name is already the canonical signal of intent in this codebase, and adding a new config field would require schema changes and documentation with no benefit for the current use case.

### Consequences

#### Positive
- OpenCode workflows on supported AWF versions correctly activate the API proxy listener on port 10004, enabling dynamic provider routing.
- The solution follows the established version-gating pattern (`awfSupportsAllowHostPorts`), making the codebase consistent and the pattern easy to discover for future contributors.
- The `AWFEnableOpenCodeMinVersion` constant gives operators a single place to look up the minimum AWF version required for OpenCode support.

#### Negative
- Each new AWF feature flag now requires a minimum-version constant, a version-gate function, and associated tests. The pattern scales linearly with the number of features, adding maintenance surface.
- The `opencodeEnabled` pre-computation in `BuildAWFArgs` couples engine-name logic with firewall-version logic in one function, making that function responsible for more decisions than before.

#### Neutral
- The `"latest"` AWF version string is treated as always meeting the minimum version requirement, which is a conservative assumption already established by the existing pattern.
- Non-semver version strings (e.g., branch names used in development) are treated as too old (returns false), erring on the side of not emitting the flag.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Flag Emission

1. Implementations **MUST** emit `--enable-opencode` in the AWF argument list when the workflow engine is `opencode` and the effective AWF version is greater than or equal to `AWFEnableOpenCodeMinVersion` (v0.25.30).
2. Implementations **MUST NOT** emit `--enable-opencode` when the workflow engine is not `opencode`.
3. Implementations **MUST NOT** emit `--enable-opencode` when the effective AWF version is less than `AWFEnableOpenCodeMinVersion`, even if the engine is `opencode`.
4. Implementations **MUST** treat the string `"latest"` as satisfying the minimum version requirement for `--enable-opencode`.
5. Implementations **MUST** treat non-semver version strings (e.g., branch names) as not satisfying the minimum version requirement (conservative default).

### Port Allowance

1. Implementations **MUST** append port 10004 to the `--allow-host-ports` argument if and only if `--enable-opencode` would also be emitted (same condition: engine is `opencode` and AWF version ≥ v0.25.30).
2. Implementations **MUST NOT** include port 10004 in `--allow-host-ports` for any engine other than `opencode`.
3. Implementations **MUST NOT** include port 10004 in `--allow-host-ports` for `opencode` workflows using an AWF version below `AWFEnableOpenCodeMinVersion`.
4. Implementations **SHOULD** compute the `opencodeEnabled` boolean once (pre-computation) rather than repeating the engine-name and version checks at each emission site within `BuildAWFArgs`.

### Version Constant Management

1. Implementations **MUST** define the minimum AWF version threshold for `--enable-opencode` as the named constant `AWFEnableOpenCodeMinVersion` in `pkg/constants/version_constants.go`.
2. Implementations **MUST NOT** embed the version string `"v0.25.30"` as a magic literal at call sites; the named constant **MUST** be used instead.
3. Implementations **SHOULD** document the reason the minimum version was introduced (the AWF PR or issue that added the flag) in the constant's comment.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25201347429) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
