# ADR-70859: Allowlist Non-Sensitive Gemini engine.env Names Through Shell Environment Redaction

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The Gemini CLI strictly filters the environment passed to shell commands when it runs in GitHub Actions. During an AW health investigation, a workflow lost an environment setting that the author had explicitly set in `engine.env`, so the setting never reached the agent's shell commands. The compiler already generates `.gemini/settings.json` (`generateGeminiSettingsStep` in `pkg/workflow/gemini_tools.go`) and already knows which credentials to keep out of the agent (`ComputeAWFExcludeEnvVarNames`, `GetRequiredSecretNames`). Any fix must keep secrets, credential-like names, and runtime-expanded values (`${{ ... }}`) out of the agent's shell. The same PR also adds Node.js 24 to the shared Serena Go import and clarifies the concurrency ledger payload, but those are configuration and prompt fixes rather than architectural changes.

### Decision

The compiler will add a `security.environmentVariableRedaction` block (`enabled: true`, plus an `allowed` list) to the generated Gemini `settings.json`. The list contains only those `engine.env` variable names that meet all three rules: the name is not in the AWF credential exclude list, it does not match a sensitive-name pattern (`token|secret|password|passwd|key|auth|credential|creds|private|cert`, case-insensitive), and its value is a literal with no `${{` runtime expression. Only names are written to the settings file, never values, and the list is sorted so compiled output is deterministic. We chose to keep Gemini's redaction turned on and allowlist explicit, safe names. Turning redaction off, or forwarding every configured variable, would restore the setting but would also widen what could leak.

### Alternatives Considered

#### Alternative 1: Disable Gemini environment variable redaction

Set `environmentVariableRedaction.enabled: false` so the shell receives the full environment. This is the simplest fix and would cover every missing variable at once. We rejected it because it removes a defence-in-depth control and could expose credentials or other runner variables to agent-run shell commands.

#### Alternative 2: Allowlist every `engine.env` name without filtering

Forward every name the author configured in `engine.env`. This respects author intent and needs no heuristics. We rejected it because authors often put secret-backed values (`${{ secrets.X }}`) or credential-like names in `engine.env`, and allowlisting them would undo the existing credential exclusions.

#### Alternative 3: Require an explicit per-workflow Gemini allowlist field

Add a new frontmatter field so authors list the variables to preserve. This is more explicit, but it adds schema surface and duplicates `engine.env`, which already states intent. Not chosen: the PR treats an explicit `engine.env` entry as sufficient opt-in.

### Consequences

#### Positive
- Settings that authors explicitly configure in `engine.env` now reach Gemini shell commands, which fixes the observed failure.
- Redaction stays on, and credential exclusions, sensitive-name filtering, and runtime-expression filtering still apply.
- Only variable names appear in the generated settings, never values, and sorted output keeps lock files stable.

#### Negative
- The sensitive-name regex is a heuristic. It can wrongly exclude harmless names (for example, anything containing `key` or `auth`), and it can miss secrets stored under neutral names with literal values.
- Gemini now applies one more environment policy that the other engines do not share, so the behaviour per engine diverges and needs maintaining.
- This depends on Gemini CLI's `security.environmentVariableRedaction` settings schema. If upstream changes that schema, this code must change too.

#### Neutral
- Workflows without `engine.env`, or whose entries are all filtered out, emit no `security` block, so their settings output does not change.
- As part of this change, the bash-to-`run_shell_command` mapping was moved into `computeGeminiShellTools` with no change in behaviour.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
