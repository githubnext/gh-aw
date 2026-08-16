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
  mcp-mode: cli
  bash:
    - "cat /tmp/gh-aw/agent/image-scan/compile-output.txt"
safe-outputs:
  create-issue:
    title-prefix: "[container-image-scan] "
    labels: [cookie, security]
    max: 25
    deduplicate-by-title: true
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
2. For each image with vulnerabilities or license violations, create one issue.
   Use the title `Container findings for <image-name>` so repeated findings for
   the same image are deduplicated. If the scan step failed to produce output,
   create a `Container scan operational failure` issue.
3. If there are no findings and no operational errors, call `noop`.
4. Follow the Output Format section for each image issue.
5. In each image issue, include:
   - the image name and pinned reference;
   - every vulnerability with severity, CVE ID, package, installed version, and
     fixed versions;
   - every rejected or unknown license and the affected package;
   - actionable remediation guidance.
6. Order the findings in every issue by severity, Critical first, then High,
   Medium, Low, and Unknown, so the Critical backlog is triaged first.
7. When any image has Critical or High findings, also create one consolidated
   burn-down issue titled `Container CVE burn-down` that links every per-image
   issue you created in this run and lists, per image, the Critical and High
   counts plus the total. Keep it to a single summary table so it stays the
   parent tracking item instead of duplicating per-image detail.
8. In the burn-down issue, state the remediation SLA cadence: Critical findings
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