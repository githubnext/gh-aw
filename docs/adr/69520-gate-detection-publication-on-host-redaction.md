# ADR-69520: Gate Detection Publication on Host-Side Secret Redaction

**Date**: 2026-10-17
**Status**: Draft
**Deciders**: pelikhan, gh-aw maintainers

---

### Context

The threat-detection job produces its own files after the agent job finishes: detector verdicts (`detection_result.json`), detection token-usage/accounting files, the detection AWF run's firewall proxy/audit logs, and anything written by custom detection post-steps. The agent job's existing secret redaction cannot cover these files because they are created later, on a different runner. Before this PR, the detection job relied on GitHub Actions console masking (for example, echoing the step summary) and uploaded artifacts and published the detection conclusion under `if: always()`. Console masking does not sanitize downloadable artifact contents, and the existing redactor only matched a fixed extension allowlist, so rotated or extensionless firewall logs (for example, `access.log.1`) were not covered. The PR describes this as a coverage gap, not as an observed secret leak.

### Decision

We will run the existing `redact_secrets.cjs` redactor on the host twice inside the detection job, for both the inline and external detector paths. The first pass (`redact_detection_logs`) runs after detector results and firewall logs are collected and after custom post-steps finish. The second pass (`redact_detection_artifacts`) runs after token-usage accounting so that newly generated usage files are also sanitized. Every downstream consumer must check that redaction succeeded before it runs. Token-usage parsing requires the first pass. Log rendering, artifact upload, and detection conclusion require both passes, using `always() && steps.<id>.outcome == 'success'`. If redaction fails, is skipped, or is cancelled, publication is blocked, even in warn mode (`continue-on-error`). The redactor also accepts `additionalDirectories` and scans every regular file in them, regardless of extension, so that every copied firewall log is redacted.

### Alternatives Considered

#### Alternative 1: Keep relying on GitHub Actions console masking

Echoing files to the runner log lets Actions mask registered secrets in the console output, and this was the existing approach. We rejected it because masking only affects the rendered log. Uploaded artifacts and files read by later steps keep the raw values, so it does not protect downloadable detection artifacts.

#### Alternative 2: Run a single redaction pass immediately before upload

One pass just before artifact upload would be simpler and would add fewer steps. We rejected it because accounting parses and renders log-derived fields into the step summary before upload, so unsanitized inputs would already be rendered. The accounting step also creates new files that need their own pass, which a pass placed earlier would miss.

#### Alternative 3: Make publication best-effort (`always()`) and only warn on redaction failure

Keeping `if: always()` on consumers would preserve the current warn-mode behaviour of always publishing a conclusion. We rejected it because a failed or skipped redaction would then still allow unsanitized files to be uploaded or rendered. The PR chooses fail-closed publication instead.

### Consequences

#### Positive
- Detector verdicts, usage/accounting files, custom post-step output, and all copied firewall logs (including rotated and extensionless ones) are sanitized before they are rendered or uploaded.
- The inline and external detector paths now share one step ordering (collect → post-steps → redact → accounting → redact → render/upload/conclude), which reduces divergence between the two compiler branches.
- Exact-value secrets are passed only to the host redaction steps through `GH_AW_SECRET_NAMES`/`SECRET_*`, not to the detector engine.

#### Negative
- A redaction failure now blocks detection artifact upload and conclusion, even in warn mode. As a result, detection publication can now fail in cases where it previously succeeded under `always()`.
- Each detection job gains two extra `actions/github-script` steps, and every compiled workflow lock file changes (305 locks were regenerated), which adds runtime and review churn.
- The redaction step builds secret references from the frontmatter, the markdown, imported markdown, and the step text it has generated so far. Secrets referenced only in steps added later would not be included.

#### Neutral
- Because `redact_secrets.cjs` now accepts `additionalDirectories` with an extension filter of `null` (all regular files), other callers could reuse it for full-directory scans.
- The existing secret redactor remains hard-failing, and this ADR does not change proxy diagnostics, credential selection, or MCP enforcement.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
