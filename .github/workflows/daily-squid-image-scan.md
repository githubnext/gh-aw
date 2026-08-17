---
name: Daily Container Image Security Scan
description: Scan container images used by compiled workflows for vulnerabilities, updates, and rejected licenses
emoji: "🛡️"
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  packages: read
  copilot-requests: write
strict: true
network:
  allowed:
    - defaults
    - go
tools:
  cli-proxy: true
  bash:
    - "cat /tmp/gh-aw/agent/image-scan/compile-output.txt"
safe-outputs:
  create-issue:
    title-prefix: "[container-image-scan] "
    labels: [cookie, security]
    assignees: [pelikhan]
    max: 1
    deduplicate-by-title: true
  update-issue:
    target: "52657"
    body: true
    max: 1
  assign-to-user:
    target: "52657"
    allowed: [pelikhan]
    max: 1
  close-issue:
    target: "*"
    required-title-prefix: "[container-image-scan] Container findings for "
    required-labels: [cookie, security]
    state-reason: duplicate
    max: 25
  noop:
    report-as-issue: false
steps:
  - name: Build gh-aw from source
    run: |
      set -e
      make build
      "$GITHUB_WORKSPACE/gh-aw" --version
  - name: Run compile with vulnerability scanners
    continue-on-error: true
    run: |
      set -uo pipefail
      output_dir="/tmp/gh-aw/agent/image-scan"
      mkdir -p "$output_dir"
      "$GITHUB_WORKSPACE/gh-aw" compile --force-refresh-container-pins --syft --grype --grant 2>&1 | tee "$output_dir/compile-output.txt" || true
post-steps:
  - name: Enforce critical vulnerability and license gates
    if: always()
    run: |
      output="/tmp/gh-aw/agent/image-scan/compile-output.txt"
      if [ ! -f "$output" ]; then
        echo "::error::Scan output not found. The compile step did not produce output."
        exit 1
      fi
      if grep -qE ': error: \[Critical\]' "$output"; then
        echo "::error::Critical vulnerabilities detected in container images."
        exit 1
      fi
      if grep -q ': error: license policy violation:' "$output"; then
        echo "::error::License policy violations detected in container images."
        exit 1
      fi
timeout-minutes: 90
evals:
  - id: container_images_scanned
    question: Did the agent analyze container images for vulnerabilities, updates, and rejected licenses?
  - id: findings_reported_or_noop
    question: Did the agent report actionable image findings, or use noop when no findings required action?
  - id: critical_burn_down_tracked
    question: When Critical or High findings existed, did the agent create a consolidated burn-down issue linking the per-image issues and stating the remediation SLA?
features:
  gh-aw-detection: true
---

# Daily Container Image Security Scan

Review the Syft SBOM, Grype vulnerability, and Grant license scan results in
`/tmp/gh-aw/agent/image-scan/compile-output.txt`.

1. Read `compile-output.txt`.
2. Treat [Container CVE burn-down](https://github.com/github/gh-aw/issues/52657)
     as the single tracker. Assign it to `pelikhan` if it is unassigned.
3. Do not create per-image finding issues. Update #52657 with the current scan's
     summary table and collapsible per-image details, including:
     - the image name and pinned reference;
     - every vulnerability with severity, CVE ID, package, installed version, and
     fixed versions;
     - every rejected or unknown license and the affected package;
     - actionable remediation guidance.
4. Close every open issue titled `Container findings for ...` as a duplicate of
     #52657. Include a short closure comment linking to #52657. Do not close
     operational-failure issues.
5. If the scan step failed to produce output, create one `Container scan
     operational failure` issue assigned to `pelikhan`.
6. If there are no findings and no operational errors, update #52657 to show
     the clean scan, then call `noop`.
7. Order the findings in #52657 by severity, Critical first, then High, Medium,
     Low, and Unknown, so the Critical backlog is triaged first.
8. In #52657, state the remediation SLA cadence: Critical findings
     are remediated or explicitly risk-accepted within 7 days, High within 30
     days, and every scanned image is rebuilt on a refreshed base image at least
     weekly (this workflow runs `gh aw compile --force-refresh-container-pins`
     daily, so a pin refresh PR is the default remediation step).
9. Keep the report factual and compact. Never omit lower-severity
     vulnerabilities.

### Output Format

- Use `###` (h3) or lower for all report headers; never use `#` or `##` inside the report body.
- Wrap long lists, tables, and detailed findings in `<details><summary><b>...</b></summary>...</details>` blocks to reduce scrolling.
- Structure reports as: overview → key metrics/issues → collapsible detail → next actions.

The configured `create-issue` safe output is the only allowed write operation.