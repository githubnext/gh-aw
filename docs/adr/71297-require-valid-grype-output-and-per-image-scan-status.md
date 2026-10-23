# ADR-71297: Require Valid Grype Output and Per-Image Scan Status

**Date**: 2026-10-23
**Status**: Draft
**Deciders**: pelikhan, Unknown (PR #71297 reviewers)

---

### Context

`gh aw compile` runs Grype (via Docker) against container images in lock-file manifests. In `pkg/cli/grype.go`, `grypeRunOnImage` parsed JSON only if stdout started with `{`. As a result, empty output, non-JSON output, or output without a `matches` array could be cached and reported as a successful scan with zero findings. The `static-analysis-report` workflow checked only for the `Running grype` marker. It could not tell "every image was scanned and was clean" from "some scans never completed". Issue #71044 found this gap. Downstream reports could present incomplete scans as confirmed zero-CVE results.

### Decision

We will treat Grype output as a strict contract. A scan succeeds only when stdout is valid JSON with a non-null `matches` array. An empty `matches: []` is still a valid result. Any other output is a scan error. That error is cached and cannot become a zero-finding result. We will also make scan coverage machine-checkable: the CLI logs `Grype scan planned for <ref>`, `Grype scan completed for <ref>: N vulnerability finding(s)`, and `Grype scan failed for <ref>` lines, using the pinned reference when one exists. The `static-analysis-report` workflow adds an "Assert Grype image scan completeness" step. That step fails the job when the planned and completed counts differ or when any scan failed. The main driver is reporting integrity. "Zero findings" must mean that a scan finished and found nothing, not that no output was produced.

### Alternatives Considered

#### Alternative 1: Keep lenient parsing and add only reporting caveats

We could keep the existing parse logic, where non-`{` output is ignored, and only tell the reporting agent to be careful. This change would be smaller and could not break existing compiles. We rejected it because the CLI would still cache and return zero-finding results for broken scans. That undermines every later consumer, not just one workflow.

#### Alternative 2: Emit a structured JSON status artifact instead of log lines

The CLI could write a per-image status file in JSON, and the workflow could validate it. That would be sturdier than regex matching on human-readable stderr. It was not chosen because it would add a new output file and plumbing to the compile command and the workflow. The log-line approach reuses the existing `compile-output.txt` capture and the existing tool-marker pattern in the workflow. [TODO: verify whether a structured artifact is planned as a follow-up]

#### Alternative 3: Fail compilation on any Grype error, regardless of strict mode

Any invalid scan could stop compilation. This was rejected because non-strict mode intentionally lets compilation continue when the scanner fails, and the PR keeps that behavior. The workflow-level gate enforces completeness only where it matters, in the reporting pipeline.

### Consequences

#### Positive
- Malformed, empty, or partial Grype output can no longer be cached or reported as a clean scan.
- Each image gets an explicit completed or failed status, keyed by pinned digest. That makes coverage auditable and lets reports tell "unknown/incomplete" apart from "zero findings".
- The completeness gate is tested by running the script itself (`grype_workflow_contract_test.go`), and the parser is tested across exit codes 0, 1, and 2.

#### Negative
- Grype or Docker output that used to pass silently now fails. In strict mode it also fails compilation, and in the static-analysis-report workflow it fails the job. This may surface new CI failures.
- The workflow gate depends on the exact wording of stderr log messages, which creates an implicit contract between `grype.go` and a Python regex in `static-analysis-report.md`. Rewording a message can break the gate without any warning.
- The contract test parses workflow markdown frontmatter and rewrites a hard-coded log path, which couples Go tests to the workflow file layout.

#### Neutral
- Existing exit-code semantics are kept: exit 1 with valid JSON means findings were reported, and other non-zero exit codes are errors.
- The change does not explain past Serena scan dropouts and does not claim that any CVEs were fixed.
- `static-analysis-report.lock.yml` was regenerated to include the new step.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
