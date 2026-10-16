# ADR-69497: Layered, Overridable Defaults for Threat-Detection Enforcement Mode

**Date**: 2026-10-16
**Status**: Accepted
**Deciders**: pelikhan

---

### Context

Threat detection runs in warning mode by default (`safe-outputs.threat-detection.continue-on-error: true`). Administrators at enterprise, organization, and repository scope need an overridable warning or blocking default without editing every workflow. Previously, enforcement was configured per workflow or import, including runtime expressions. Existing enterprise defaults already use Actions variables, and repository configuration uses `aw.json`. The new control must reuse those mechanisms, keep the built-in warning fallback, and leave detection enablement unchanged.

### Decision

Detection enforcement uses a fixed precedence chain: (1) explicit workflow/import `continue-on-error` (literal or expression), then (2) the boolean compile-time `aw.json` key `default_detection_continue_on_error`, then (3) the runtime GitHub Actions variable `GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR`, then (4) built-in `true`. Existing import precedence is preserved.

When neither explicit layer is set, the compiler emits a `detection_mode` step (`resolve_detection_continue_on_error.cjs`). It accepts case-insensitive `true`/`false` with surrounding whitespace ignored; unset or empty means warning mode, and other values fail explicitly. Overridden variables are not read or validated. The same compiler resolution determines whether to emit this step and which value every enforcement consumer receives, preventing precedence drift. Runtime defaults are resolved once and shared through `steps.detection_mode.outputs.continue_on_error`; explicit settings bypass this step.

The external and inline detector paths use the effective value for setup, conclusion classification, and mode-dependent step failure handling. Blocking mode rejects threats and incomplete assessments, including failed or cancelled execution that left a clean-looking verdict. Existing downstream write gates remain tied to the detection job result. Expression-provided `False`/`FALSE` values are treated consistently as blocking. These controls are overridable defaults, not mandatory policy or a switch that enables detection.

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
- A shared compiler resolution and, when needed, one runtime `detection_mode` output keep external and inline enforcement consumers consistent.
- Explicit workflow settings, including `false` and expressions, are still respected, which keeps backward compatibility.
- Blocking mode now fails closed when detection execution fails or is cancelled.

#### Negative
- Workflows compiled before this change do not honor the variable until they are recompiled.
- It adds an extra `actions/github-script` step to detection jobs that do not set the mode explicitly or through `aw.json`, which costs a little runtime and lock-file churn.
- An invalid effective variable value fails the detection job at runtime, so a misconfiguration at org/enterprise scope can affect repositories that have no explicit override. Intentionally skipped detection does not validate this unused default.
- Three configuration layers (workflow, `aw.json`, Actions variable) make it harder to work out which value is in effect.

#### Neutral
- `resolveThreatDetectionContinueOnError` becomes a `Compiler` method because it now reads repository config through `loadRepoConfig()`.
- The `gh aw env` defaults file accepts the matching string-valued key, while `aw.json` uses a boolean.
- Detection enablement, skips, reporting, models, budgets, and compiler strict mode are unchanged.

---

This record captures the user-approved configuration contract implemented in PR #69497. It does not assert additional reviewer approval.
