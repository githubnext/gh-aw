# ADR-53183: Detect and Replace Heredocs in Generated Workflow YAML

**Date**: 2026-08-16
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflow shell steps that generate YAML content via heredocs embed that content in shell-evaluated strings. When the workflow YAML itself is generated (rather than hand-authored), this creates a shell injection vector: content flowing through `GH_AW_SAFE_OUTPUTS_CONFIG` or similar environment variables is evaluated by the shell as part of the heredoc, allowing unexpected expansion or injection. The codebase had accumulated a number of such heredoc patterns in generated workflow files, and there was no systematic mechanism to prevent new ones from being introduced.

### Decision

We will enforce a two-part strategy: (1) add a Go static-analysis linter (`generatedyamlheredoc`) that detects heredocs inside generated workflow shell steps and fails the build on new violations, and (2) introduce a JavaScript file renderer (`create_files.cjs`) that writes environment-provided content directly to disk without shell evaluation or base64 encoding. Existing heredoc sites are explicitly suppressed to capture migration debt. New sites are blocked at lint time.

### Alternatives Considered

#### Alternative 1: Base64-encode heredoc content

Heredoc payloads could be base64-encoded before injection, eliminating shell-special-character expansion. This would be a smaller change, requiring only an encode/decode wrapper around existing heredoc patterns. It was rejected because it still routes content through a shell heredoc (adding decode complexity, increasing command-line length, and leaving the heredoc pattern in place as a future footgun), and it does not address the structural problem of using heredocs for file generation at all.

#### Alternative 2: Accept heredocs with input sanitization only

Existing heredoc sites could be left in place, with the environment variable values sanitized at injection time (escaping shell metacharacters). This was rejected because sanitization rules are fragile—they must track every shell-special character and every quoting context—and do not compose well with multi-step pipelines. A missing escape in any one place reintroduces the vulnerability. The JavaScript renderer eliminates the category of risk by never invoking shell evaluation on the content at all.

### Consequences

#### Positive
- Generated YAML content is no longer evaluated by the shell, eliminating heredoc-based injection as a vulnerability class in generated workflow files.
- The JavaScript renderer (`create_files.cjs`) enforces output-path constraints (runner-directory confinement, traversal rejection, symlink escaping), improving defense in depth.
- The linter blocks regression: no new heredoc patterns can be introduced in generated workflow shell without an explicit suppression comment that captures the migration debt.

#### Negative
- Existing heredoc sites must be explicitly suppressed, creating a tracked but unresolved migration backlog that must be addressed in follow-on PRs.
- The JavaScript renderer introduces a new runtime dependency on Node.js being available in the workflow runner environment; environments without Node.js are unsupported.

#### Neutral
- Compiled workflow locks must be regenerated when the workflow YAML changes; this is a normal part of the workflow authoring cycle and is handled by the existing regeneration process.
- The analyzer is registered with the Go analyzer framework; adding it follows the same pattern as existing analyzers in the codebase.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
