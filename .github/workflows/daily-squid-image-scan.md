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
      "$GITHUB_WORKSPACE/gh-aw" compile --syft --grype --grant 2>&1 | tee "$output_dir/compile-output.txt" || true
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
sandbox:
  agent:
    sudo: false
timeout-minutes: 90
imports:
  - shared/reporting.md
---

### Daily Container Image Security Scan

Review the Syft SBOM, Grype vulnerability, and Grant license scan results in
`/tmp/gh-aw/agent/image-scan/compile-output.txt`.

1. Read `compile-output.txt`.
2. For each image with vulnerabilities or license violations, create one issue.
   Use the title `Container findings for <image-name>` so repeated findings for
   the same image are deduplicated. If the scan step failed to produce output,
   create a `Container scan operational failure` issue.
3. If there are no findings and no operational errors, call `noop`.
4. Format each image issue with `###` and `####` headings only. Keep a visible
   `### Summary` section at the top, and wrap verbose per-vulnerability or
   per-license breakdowns in `<details><summary>...</summary>` blocks.
5. In each image issue, include:
   - the image name and pinned reference;
   - every vulnerability with severity, CVE ID, package, installed version, and
     fixed versions;
   - every rejected or unknown license and the affected package;
   - actionable remediation guidance.
6. Keep the report factual and compact. Never omit lower-severity
   vulnerabilities.

The configured `create-issue` safe output is the only allowed write operation.
