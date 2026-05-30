# ADR-31614: Auto-Detect ARC/DinD and Emit AWF `--docker-host-path-prefix` at Runtime

**Date**: 2026-05-12
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

AWF-backed workflows running on Actions Runner Controller (ARC) with the Docker-in-Docker (DinD) sidecar pattern have a split filesystem: the runner container and the Docker daemon container do not share `/tmp` and other staging paths, so bind-mounts that AWF constructs from the runner's perspective fail inside the daemon. AWF supports a `--docker-host-path-prefix` flag to bridge this split (`v0.25.43`+), but until now `gh-aw`'s compiler did not emit it. Users on ARC DinD had to author per-workflow `sandbox.agent.args` hacks or pass the flag manually, even though the topology is detectable at runtime from a single environment variable (`DOCKER_HOST`). The fix needs to be safe for non-DinD runners and for pinned AWF versions that do not understand the flag.

### Decision

We will inject a generated bash probe into every AWF command that inspects `DOCKER_HOST` at runtime, matches it against `^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$`, and conditionally appends `--docker-host-path-prefix /tmp/gh-aw` to the `awf` invocation when the topology looks like ARC DinD. The probe is gated by a new version constant `AWFDockerHostPathPrefixMinVersion = v0.25.43` and the helper `awfSupportsDockerHostPathPrefix(...)`, so workflows pinned to older AWF versions never receive the unsupported flag. The bash regex, the hardcoded prefix path, and the shell variable name are centralized as package-level constants in `pkg/workflow/awf_helpers.go` to keep the generated script deterministic and grep-able.

### Alternatives Considered

#### Alternative 1: Require users to opt in via workflow frontmatter

We could add a YAML field such as `runtime.arc-dind: true` that explicitly tells the compiler to emit `--docker-host-path-prefix`. This would keep the generated script smaller for the common GitHub-hosted-runner case, but it pushes platform topology knowledge onto every workflow author, contradicts the goal of "first-class" ARC DinD support, and means a misconfigured runner produces a confusing failure at execution time rather than a self-correcting probe. We rejected this because the topology is observable from `DOCKER_HOST` with no false positives expected for typical GitHub-hosted runners (which do not export a localhost TCP `DOCKER_HOST`).

#### Alternative 2: Detect DinD at compile time from runner labels

The compiler could inspect `runs-on:` and emit the flag only when the workflow targets an ARC runner. This would avoid generating runtime shell on hosted runners entirely. However, ARC runner labels are arbitrary user strings (`self-hosted`, `arc-runners`, custom pool names), there is no reliable way to distinguish "ARC with DinD sidecar" from "ARC with rootless Docker" or "self-hosted bare metal" at compile time, and the detection would need to be re-implemented every time a new ARC topology emerged. The runtime probe relies on a single facts-based check (`DOCKER_HOST` shape) that is invariant across ARC versions.

#### Alternative 3: Push the detection into AWF itself

AWF could implement the same `DOCKER_HOST` regex internally and self-apply the path prefix when needed, removing the responsibility from `gh-aw`. This is the cleanest long-term design but requires an AWF release, coordinated version pinning across the ecosystem, and a fallback path in `gh-aw` for older AWF versions anyway. Doing the detection in `gh-aw` ships the fix immediately and is reversible: once a future AWF version self-detects, the compiler can stop emitting the probe behind a higher version gate.

### Consequences

#### Positive
- AWF-backed workflows on ARC DinD now work out of the box without user-authored `sandbox.agent.args` or manual `--docker-host-path-prefix` flags.
- The probe is a no-op on GitHub-hosted runners and any environment where `DOCKER_HOST` is unset or points at a Unix socket, so existing users see no behavioral change.
- The version gate prevents breakage for workflows pinned to AWF `< v0.25.43`, which would otherwise fail with an unknown-flag error.
- Detection logic lives next to the AWF command builder where reviewers can audit it together with related flag emission (`--allow-host-ports`, `--cli-proxy`, etc.).

#### Negative
- Every generated AWF command on a supported version now includes ~4 extra lines of shell, even on runners that will never match the regex. This adds noise to the compiled `.lock.yml` output.
- The bash regex must stay synchronized with whatever `DOCKER_HOST` shapes ARC DinD actually exports; if a future ARC release uses a non-localhost TCP host or a Unix socket forwarded over a different scheme, the probe will silently miss it.
- The path prefix `/tmp/gh-aw` is hardcoded; workflows that customize the runner temp directory will not see it overridden by this probe (though no such customization is currently supported in `gh-aw`).

#### Neutral
- A new minimum-version constant (`AWFDockerHostPathPrefixMinVersion`) joins the existing family of AWF version gates (`AWFCliProxyMinVersion`, `AWFAllowHostPortsMinVersion`), continuing the established pattern.
- Lock files generated against `latest` or `v0.25.43+` will now contain the probe; downstream consumers comparing pre/post-change lock diffs should expect this new block.
- The probe's three-line structure (init → conditional → flag assignment) is asserted by a test (`TestFirewallArgsInCopilotEngine`) that pins the ordering, making accidental refactors detectable.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Runtime ARC/DinD Probe Emission

1. When `awfSupportsDockerHostPathPrefix(firewallConfig)` returns true, `BuildAWFCommand` **MUST** prepend a bash probe block to the generated AWF command that initializes the variable `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS` to the empty string, tests `DOCKER_HOST` against the regex `^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$`, and on match assigns the variable to `"--docker-host-path-prefix /tmp/gh-aw"`.
2. The generated `awf` invocation **MUST** reference the probe variable via `${GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS}` (unquoted) so the flag expands to zero arguments when the probe does not match.
3. The probe block **MUST** appear before the `awf` invocation in the generated script so that the variable is defined by the time the command is executed.
4. The probe's three textual fragments (variable initialization, `if [[ ... =~ ... ]]; then`, flag assignment) **MUST** appear in that order in the generated output; reordering breaks the `TestFirewallArgsInCopilotEngine` conformance test.
5. The bash regex used in the probe **MUST** remain compatible with bash `[[ ... =~ ... ]]` matching and **MUST NOT** be rewritten to a different shell's regex dialect (e.g. POSIX BRE).

### Version Gating

1. The constant `AWFDockerHostPathPrefixMinVersion` **MUST** be the single source of truth for the minimum AWF version that supports `--docker-host-path-prefix`.
2. When `firewallConfig` is `nil` or its `Version` field is empty, `awfSupportsDockerHostPathPrefix` **MUST** fall back to `constants.DefaultFirewallVersion` and gate accordingly.
3. When the configured AWF version is the literal string `"latest"` (case-insensitive), `awfSupportsDockerHostPathPrefix` **MUST** return `true`.
4. When the configured AWF version is less than `AWFDockerHostPathPrefixMinVersion` per semver comparison, `BuildAWFCommand` **MUST NOT** emit the probe block, the variable reference, or the `--docker-host-path-prefix` flag.
5. New version-gated AWF flags **SHOULD** follow the same three-part pattern: (a) a `AWF<Flag>MinVersion` constant in `pkg/constants/version_constants.go`, (b) a `awfSupports<Flag>(firewallConfig)` helper in `pkg/workflow/awf_helpers.go`, (c) conditional emission inside `BuildAWFCommand` or `BuildAWFArgs`.

### Hardcoded Constants

1. The shell variable name `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS`, the regex `^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$`, and the flag value `--docker-host-path-prefix /tmp/gh-aw` **MUST** be defined as package-level constants in `pkg/workflow/awf_helpers.go` and **MUST NOT** be duplicated as string literals elsewhere in the codebase.
2. The path prefix `/tmp/gh-aw` **MUST** match the workspace root that `gh-aw` uses for staging on the runner; changing one **MUST** be accompanied by changing the other.

### Test Coverage

1. Tests **MUST** verify that the probe block is emitted (in the correct order) for the default AWF version and for `latest`.
2. Tests **MUST** verify that the probe block is not emitted when the configured AWF version is below `AWFDockerHostPathPrefixMinVersion` (e.g. `v0.25.42`).
3. Tests **MUST** verify the exact regex string and flag assignment text so that accidental rewording of the generated probe is caught at compile time.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
