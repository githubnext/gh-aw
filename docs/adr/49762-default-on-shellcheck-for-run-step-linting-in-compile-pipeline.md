# ADR-49762: Default-on Shellcheck for Run Step Linting in Compile Pipeline

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw compile` pipeline validates GitHub Actions workflow files and generates lock files, but it did not previously analyze the shell scripts inside `run:` steps for correctness. Bash/sh step scripts can contain subtle bugs (undefined variables, quoting issues, subshell errors) that are invisible to YAML-level validators and actionlint. Shellcheck is the standard static analysis tool for shell scripts and already handles the vast majority of common bash/sh issues. The existing external-tool integrations (yamllint, grype, zizmor, poutine, etc.) are all opt-in flags; this PR diverges from that pattern by making shellcheck default-on, with an escape hatch (`--no-shellcheck`) to disable it.

### Decision

We will integrate shellcheck as a **default-on** phase of the compile pipeline: shellcheck runs automatically after each lock file is emitted unless the user passes `--no-shellcheck`. The system binary is used (no Docker); the tool is silently skipped when not installed in normal mode, and emits a warning in `--validate` mode or an error in `--strict` mode. A curated set of GitHub Actions-specific false-positive codes (`SC2016`, `SC1090`, `SC1091`) is suppressed by default to avoid noise from `${{ }}` expression syntax and dynamic source paths.

The primary driver is that run step scripts are the most likely place for shell-specific bugs; making coverage automatic ensures users benefit without needing to know to opt in.

### Alternatives Considered

#### Alternative 1: Opt-in flag (consistent with existing tools like `--yamllint`, `--grype`)

Make shellcheck opt-in via a `--shellcheck` flag, matching the established pattern for all other post-compile linting tools. Users who want shellcheck checking would pass the flag explicitly.

Why not chosen: Opt-in guarantees low adoption — developers rarely add new flags to existing workflows. Shell script bugs in `run:` steps are universal enough that silent coverage by default provides substantially more value. The cost of a false positive is low (a warning or skipped check), while the cost of a missed shell bug can be significant.

#### Alternative 2: Docker-based shellcheck (consistent with yamllint/grype/syft)

Run shellcheck inside a Docker container, mirroring how yamllint and the security scanners are invoked. This would remove the "must install separately" dependency.

Why not chosen: Shellcheck is a lightweight, widely-available system binary with no complex runtime dependencies. Requiring Docker adds startup latency (~2–5 s per invocation), makes the feature unavailable in environments without Docker (CI runners, devcontainers without Docker-in-Docker), and increases operational complexity for a tool that most developers already have installed. The binary approach is faster and more portable.

#### Alternative 3: Delegate to actionlint for shell checking

Actionlint already performs some shell script analysis via an embedded shellcheck call when shellcheck is available. Relying on actionlint rather than adding a direct shellcheck phase would avoid duplication.

Why not chosen: Actionlint's shellcheck integration is opportunistic and only activates when actionlint is also run. Separately wiring shellcheck ensures coverage regardless of whether actionlint is in use, and gives users direct control over shellcheck's behaviour (strict mode, ignore codes, etc.).

### Consequences

#### Positive
- Run step shell scripts are automatically linted on every compile invocation without any developer opt-in, improving baseline shell code quality across all workflows.
- Known GitHub Actions expression false positives (`SC2016`, `SC1090`, `SC1091`) are suppressed by default, reducing noise without sacrificing coverage.
- The feature is transparent — silently skipped when shellcheck is absent — so existing setups without shellcheck installed are unaffected.

#### Negative
- Introduces an **implicit system dependency**: developers who encounter shellcheck findings must install shellcheck locally to understand and reproduce them; the dependency is not bundled or pinned.
- Default-on is a **breaking change in behaviour** for any existing integration tests or CI pipelines that run `gh aw compile` without expecting shellcheck output on stderr; tests that pass `--strict` and have shell issues in run steps will now fail until `--no-shellcheck` is added or findings are fixed.

#### Neutral
- The opt-out flag name (`--no-shellcheck`) deviates from the positive-flag naming convention used by all other tools (`--yamllint`, `--grype`, etc.), which reflects the inverted default but may surprise users scanning the flag list.
- The suppressed code list (`shellcheckDefaultIgnoreCodes`) is hardcoded; teams with unusual expression patterns may need to request additions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
