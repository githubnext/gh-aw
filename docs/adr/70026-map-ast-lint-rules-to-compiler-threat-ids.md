# ADR-70026: Map AST Lint Rules to Compiler Threat IDs and Share Compiler Policy Detectors

**Date**: 2026-05-15
**Status**: Draft
**Deciders**: pelikhan, Unknown (review required)

---

### Context

`gh aw lint` runs an in-process registry of AST lint rules (see ADR-69813). The rules check authored AW Markdown and parsed Actions YAML without recompiling. The Compiler Threat Detection Specification (`specs/compiler-threat-detection-spec.md`) lists numbered threat rules (`CTR-NNN`) that the compiler enforces, but before this PR the linter had no checks for several of them. These included privileged `pull_request_target` role admission (CTR-008), shell expansion in safe-output steps (CTR-009), and SSL inspection for firewall URL filters (CTR-011). Lint findings also gave no way to trace a diagnostic back to the threat rule it relates to. The compiler already had a private lexical detector for unsafe shell expansion in `safe_outputs_steps_shell_expansion_validation.go`. Reimplementing that detector in the linter would let the two drift apart.

### Decision

We will add an optional `ThreatRuleID` field to `lint.Rule` and `lint.Diagnostic`. `lint.New` validates it against `^CTR-[0-9]{3}$`. The CLI shows it in rule listings and in text and JSON diagnostics. Every built-in rule carries its known CTR mapping, and extension rules may leave it out. A mapping means the rule covers the observable checks described for that threat rule. It does not claim full conformance, so a clean lint result is not proof of runtime conformance. We will add three AW-only rules: `pull-request-target-roles` (CTR-008), `safe-output-shell-expansion` (CTR-009), and `firewall-ssl-bump` (CTR-011). The firewall rule inspects authored `sandbox.agent.args` for URL-filter flags, not schema-rejected `network.firewall` fields. The compiler's shell-expansion regex moves into a new shared package, `pkg/workflow/shellpattern`, which both the compiler validation and the lint rule call, so they apply one lexical policy. The rules stay read-only AST checks: they do not load imports, execute scripts, or infer AW policy for native Actions inputs.

### Alternatives Considered

#### Alternative 1: Keep lint rules independent of the threat specification (no CTR IDs)

The three rules could have been added without linking findings to threat rules. That is simpler and avoids keeping a mapping up to date. It was rejected because reviewers and tooling could not tell which threat rule a finding relates to. Checking linter coverage against the specification would also stay manual. The PR states that all 21 rule listings were checked against the specification mappings, and that check depends on these IDs.

#### Alternative 2: Duplicate the shell-expansion pattern inside the lint package

The `safe-output-shell-expansion` rule could have used its own copy of the regex. That avoids adding a package and changing compiler code. It was rejected because the compiler and linter could then disagree on the same input, giving false "clean" lint results or false lint failures. One `shellpattern.FirstUnsafeExpansion` function keeps both paths consistent.

#### Alternative 3: Have linters reuse full compiler validation (e.g., compile and inspect errors)

The linter could have called compiler validation directly to get CTR coverage. It was not chosen because lint is meant to be a fast check on authored source that does not recompile, resolve imports, or produce outputs, as described in the Workflow Linters reference. Running full compilation would break that boundary and couple lint to compiler internals.

### Consequences

#### Positive
- Diagnostics and `--list-rules` output name the related CTR rule, so findings can be traced to the threat specification.
- The compiler and linter use the same shell-expansion detection logic, so they cannot drift on CTR-009.
- Three more threat rules have authored-source coverage, bringing the registry to 21 rules.
- Format validation of `ThreatRuleID` in `lint.New` rejects malformed IDs when the registry is built.

#### Negative
- Built-in rule IDs, the threat spec, and the Workflow Linters reference page now have to be kept in sync by hand. Renumbering a CTR rule touches all three.
- A visible CTR ID may suggest that a rule fully enforces the threat rule, even though the docs say mappings cover only observable checks.
- The lexical detector deliberately matches inside quoted text and comments. Its false positives now show up in lint as well as in compilation.
- The new rules only see authored source. `firewall-ssl-bump` checks literal `--allow-urls` and `--ssl-bump` tokens in `sandbox.agent.args`, an internal override accepted only in non-strict mode. It reports schema-invalid legacy `network.firewall` blocks as requiring migration, without claiming runtime enforcement or resolving imported policy.

#### Neutral
- The JSON diagnostic schema gains a `threat_rule_id` field, omitted when empty. It is an additive change.
- The detailed rule catalog moved from the CLI docs page to a dedicated `reference/workflow-linters.md` page, and the specification and changelog versions were bumped to 1.0.61.
- `pull-request-target-roles` keeps the compiler's strict and non-strict behavior. It honors the `dangerously-allow-roles-read-and-pull-request-target` feature flag only at warning severity.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
