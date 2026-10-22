# ADR-69520: Gate Detection Publication on Host-Side Secret Redaction

**Date**: 2026-10-17
**Status**: Proposed
**Deciders**: pelikhan, gh-aw maintainers

---

### Context

The threat-detection job produces its own files after the agent job finishes: detector verdicts (`detection_result.json`), detection token-usage/accounting files, the detection AWF run's firewall proxy/audit logs, and anything written by custom detection post-steps. The agent job's existing secret redaction cannot cover these files because they are created later, on a different runner. Before this PR, the detection job relied on GitHub Actions console masking (for example, echoing the step summary) and uploaded artifacts and published the detection conclusion under `if: always()`. Console masking does not sanitize downloadable artifact contents, and the existing redactor only matched a fixed extension allowlist, so rotated or extensionless firewall logs (for example, `access.log.1`) were not covered. The PR describes this as a coverage gap, not as an observed secret leak.

### Decision

Run the existing `redact_secrets.cjs` redactor on the host twice inside the detection job, for both the inline and external detector paths. The first pass (`redact_detection_logs`) runs after detector results and firewall logs are collected and after custom post-steps finish. The second pass (`redact_detection_artifacts`) runs after token-usage accounting so that newly generated usage files are also sanitized. Token-usage parsing requires the first pass; log rendering, artifact upload, and consumption of detection results require both passes. Failed, skipped, or cancelled redaction blocks publication for an active detection run, even in warn mode (`continue-on-error`).

When the detection guard explicitly reports that no analysis is needed, skip redaction and report the existing successful skipped conclusion without consuming detector output. This keeps unrelated temporary files from turning a legitimate no-output run into a CI failure. Restrict exact-value redaction secrets to the detection job's own steps and inherited environment; do not forward unrelated main-provider or safe-output credentials to a separate detection runner.

Collect runtime `::add-mask::` values from `detection.log` before mutating any output. Retain those values between redaction steps through runner-private `GITHUB_ENV`, outside uploaded artifact trees, so the second pass also protects late-created accounting files. Recursively uploaded firewall files must be valid UTF-8 text no larger than 16 MiB. Remove unsupported binary, compressed, or oversized byproducts rather than rewriting or uploading unsanitized bytes; a removal failure blocks publication.

### Alternatives Considered

#### Alternative 1: Keep relying on GitHub Actions console masking

Echoing files to the runner log lets Actions mask registered secrets in the console output, and this was the existing approach. We rejected it because masking only affects the rendered log. Uploaded artifacts and files read by later steps keep the raw values, so it does not protect downloadable detection artifacts.

#### Alternative 2: Run a single redaction pass immediately before upload

One pass just before artifact upload would be simpler and would add fewer steps. We rejected it because accounting parses and renders log-derived fields into the step summary before upload, so unsanitized inputs would already be rendered. The accounting step also creates new files that need their own pass, which a pass placed earlier would miss.

#### Alternative 3: Make publication best-effort (`always()`) and only warn on redaction failure

Keeping `if: always()` on consumers would preserve the current warn-mode behaviour of always publishing a conclusion. We rejected it because a failed or skipped redaction would then still allow unsanitized files to be uploaded or rendered. The PR chooses fail-closed publication instead.

### Consequences

#### Positive
- Detector verdicts, usage/accounting files, custom post-step output, and supported copied firewall logs (including rotated and extensionless text) are sanitized before they are rendered or uploaded.
- The inline and external detector paths now share one step ordering (collect → post-steps → redact → accounting → redact → render/upload/conclude), which reduces divergence between the two compiler branches.
- Redaction secrets are collected from effective detection inputs and passed to host redaction steps through `GH_AW_SECRET_NAMES`/`SECRET_*`. Unrelated workflow credentials are not added to the detection job.

#### Negative
- A redaction failure now blocks detection artifact upload and conclusion, even in warn mode. As a result, detection publication can now fail in cases where it previously succeeded under `always()`.
- Each detection job gains two extra `actions/github-script` steps, and every compiled workflow lock file changes (305 locks were regenerated), which adds runtime and review churn.
- A legitimate no-output run must bypass file-consuming publication steps while still reporting a skipped conclusion. Merely skipping redaction without preserving that conclusion would block downstream safe outputs.
- Binary, compressed, and oversized firewall byproducts are omitted from the diagnostic artifact. Generated runtime-mask values remain available to later steps in the detection job through runner-private environment state, but are never stored as artifact state.

#### Neutral
- The existing secret redactor remains hard-failing, and this ADR does not change proxy diagnostics, credential selection, or MCP enforcement.

---

*Proposed for maintainer review with this pull request.*
