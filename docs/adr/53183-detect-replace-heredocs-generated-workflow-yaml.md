# ADR-53183: Detect and Replace Heredocs in Generated Workflow YAML

**Date**: 2026-08-16
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflow shell steps that generate files via heredocs embed content in shell-evaluated strings. When the workflow YAML itself is generated, this creates a shell injection vector: content flowing through `GH_AW_SAFE_OUTPUTS_CONFIG` or similar environment variables can undergo command, parameter, or backtick expansion. The codebase had accumulated a number of such patterns with no systematic mechanism to prevent new ones.

The change must preserve arbitrary file content byte for byte, avoid base64 as an indirect shell transport, keep generated files inside the runner-owned directory, and permit existing heredocs to be migrated incrementally without weakening enforcement for new code.

### Decision

We will enforce a two-part strategy:

1. Add a Go static-analysis linter (`generatedyamlheredoc`) that detects heredoc operators inside string literals used to generate workflow shell and fails the build on new violations. Existing sites are explicitly suppressed to record migration debt.
2. Introduce a JavaScript file renderer (`create_files.cjs`) that accepts a typed JSON file manifest and writes environment-provided content without shell evaluation or base64 encoding. The renderer confines paths to the configured runner directory, rejects traversal and symlinked-parent escapes, requires `O_NOFOLLOW`, and writes files with `0600` permissions.

The safe-outputs configuration is the first migrated caller. Remaining suppressed sites will be migrated separately so this change can establish enforcement without combining every heredoc migration into one review.

### Alternatives Considered

#### Alternative 1: Base64-encode heredoc content

Heredoc payloads could be base64-encoded before injection, eliminating shell-special-character expansion. This would be a smaller change, requiring only an encode/decode wrapper around existing heredoc patterns. It was rejected because it still routes content through a shell heredoc (adding decode complexity, increasing command-line length, and leaving the heredoc pattern in place as a future footgun), and it does not address the structural problem of using heredocs for file generation at all.

#### Alternative 2: Accept heredocs with input sanitization only

Existing heredoc sites could be left in place, with the environment variable values sanitized at injection time (escaping shell metacharacters). This was rejected because sanitization rules are fragile—they must track every shell-special character and every quoting context—and do not compose well with multi-step pipelines. A missing escape in any one place reintroduces the vulnerability. The JavaScript renderer eliminates the category of risk by never invoking shell evaluation on the content at all.

#### Alternative 3: Migrate every existing heredoc before adding enforcement

All existing sites could be migrated in one change and the linter enabled without suppressions. This was rejected because it would combine many independent workflow-generation paths into a single high-risk review. Explicit suppressions make the debt visible while ensuring new heredocs cannot be added unnoticed.

### Consequences

#### Positive
- Generated YAML content is no longer evaluated by the shell, eliminating heredoc-based injection as a vulnerability class in generated workflow files.
- The JavaScript renderer (`create_files.cjs`) enforces output-path constraints (runner-directory confinement, traversal rejection, symlink escaping), improving defense in depth.
- The linter blocks regression: no new heredoc patterns can be introduced in generated workflow shell without an explicit suppression comment that captures the migration debt.

#### Negative
- Existing heredoc sites must be explicitly suppressed, creating a tracked but unresolved migration backlog that must be addressed in follow-on PRs.
- The JavaScript renderer introduces a new runtime dependency on Node.js being available in the workflow runner environment; environments without Node.js are unsupported.
- File rendering fails closed on platforms without `O_NOFOLLOW` rather than silently writing without symlink protection.

#### Neutral
- Compiled workflow locks must be regenerated when the workflow YAML changes; this is a normal part of the workflow authoring cycle and is handled by the existing regeneration process.
- The analyzer is registered with the Go analyzer framework; adding it follows the same pattern as existing analyzers in the codebase.
