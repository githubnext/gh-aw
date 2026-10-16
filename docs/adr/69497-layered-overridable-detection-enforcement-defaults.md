# ADR-69497: Layered, Overridable Defaults for Threat-Detection Enforcement Mode

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify additional reviewers]

---

### Context

Threat detection runs in warning mode by default (`safe-outputs.threat-detection.continue-on-error: true`): detection failures produce warnings rather than blocking safe-output publication. Administrators at enterprise, organization, and repository scope want to choose warning or blocking mode as a default without editing every workflow. Before this change, the only control was the per-workflow (or imported) `continue-on-error` setting, resolved at compile time by a package-level helper in `pkg/workflow/threat_detection_steps.go`. Existing enterprise defaults (for example `GH_AW_DEFAULT_DETECTION_JOB_TIMEOUT_MINUTES`) already follow a pattern of runtime Actions variables plus `aw.json` repository config, so a new control has to fit that model, keep the built-in warning fallback, and not change whether detection is enabled.

### Decision

We will resolve the detection enforcement mode through a fixed precedence chain: (1) explicit workflow/import `continue-on-error` (literal or expression), then (2) the compile-time `aw.json` key `default_detection_continue_on_error`, then (3) the runtime GitHub Actions variable `GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR`, then (4) the built-in default `true`. When neither of the first two is set, the compiler emits a `detection_mode` step (`resolve_detection_continue_on_error.cjs`) that validates the variable and rejects values other than `true`/`false` (empty means warning mode). All detection steps then use the shared output `steps.detection_mode.outputs.continue_on_error` as their `continue-on-error` expression. In blocking mode, `conclude_threat_detection.sh` also blocks publication when detection execution failed or was cancelled, even if a clean-looking verdict exists. These are defaults that workflows can override, not mandatory policy.

### Alternatives Considered

#### Alternative 1: Compile-time-only default (aw.json or compiler environment)

Resolve the default entirely when compiling, as `GH_AW_DEFAULT_DETECTION_MODEL` does. This is simpler, adds no runtime step, and is partly kept as the `aw.json` layer. It was not enough on its own: changing the mode across an organization would require recompiling every workflow in every repository, which defeats the goal of central, scope-wide administration.

#### Alternative 2: Mandatory enterprise policy that workflows cannot override

Treat the organization or enterprise setting as an enforced policy that overrides workflow frontmatter. That would give stronger governance guarantees. It was rejected because it would silently break workflows that intentionally choose warning mode, would conflict with the existing expression-controlled `continue-on-error` (ADR-29257), and Actions variables are not a tamper-resistant policy channel anyway.

#### Alternative 3: Change the built-in default to blocking

Flip the fallback to `false` so that every workflow blocks by default. This is the simplest code change, but it would break existing deployments that depend on warning-mode resilience (ADR-29031) and would offer no per-scope choice.

### Consequences

#### Positive
- Administrators can switch enterprise, organization, or repository defaults by changing one Actions variable, with no recompilation once workflows have been regenerated.
- One resolution point (`detection_mode` output) keeps the external detector, inline detector, conclusion, and install steps consistent.
- Explicit workflow settings, including `false` and expressions, are still respected, which keeps backward compatibility.
- Blocking mode now fails closed when detection execution fails or is cancelled.

#### Negative
- Workflows compiled before this change do not honor the variable until they are recompiled.
- It adds an extra `actions/github-script` step to detection jobs that do not set the mode explicitly or through `aw.json`, which costs a little runtime and lock-file churn.
- An invalid variable value fails the detection job at runtime, so a misconfiguration at org/enterprise scope can affect many repositories at once.
- Three configuration layers (workflow, `aw.json`, Actions variable) make it harder to work out which value is in effect.

#### Neutral
- `resolveThreatDetectionContinueOnError` becomes a `Compiler` method because it now reads repository config through `loadRepoConfig()`.
- The `gh aw env` defaults file accepts the matching string-valued key, while `aw.json` uses a boolean.
- Detection enablement, skips, reporting, models, budgets, and compiler strict mode are unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
