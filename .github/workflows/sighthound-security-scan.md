---
private: true
emoji: "🛡️"
name: Sighthound Security Scan
description: Run Sighthound in a separate job, upload findings as an artifact, and run an agent only when findings are detected
on:
  pull_request:
    types: [opened, synchronize, reopened]
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  issues: read
  actions: read
strict: true
if: needs.sighthound_scan.outputs.findings_detected == 'true'
jobs:
  sighthound_scan:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      findings_detected: ${{ steps.scan.outputs.findings_detected }}
      findings_count: ${{ steps.scan.outputs.findings_count }}
      artifact_name: ${{ steps.artifact_name.outputs.value }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.0
        with:
          persist-credentials: false

      - name: Install Sighthound
        run: |
          cargo install --locked --git https://github.com/Corgea/Sighthound --bin sighthound
          echo "$HOME/.cargo/bin" >> "$GITHUB_PATH"
          sighthound --help >/dev/null

      - name: Run Sighthound scan
        id: scan
        run: |
          set -euo pipefail
          RESULTS_DIR="/tmp/gh-aw/agent/sighthound"
          RESULTS_JSON="$RESULTS_DIR/results.json"
          mkdir -p "$RESULTS_DIR"

          set +e
          sighthound --output-format json . > "$RESULTS_JSON"
          SCAN_EXIT=$?
          set -e

          if [ ! -s "$RESULTS_JSON" ]; then
            echo "[]" > "$RESULTS_JSON"
          fi

          FINDINGS_COUNT="$(jq 'if type=="array" then length else 0 end' "$RESULTS_JSON" 2>/dev/null || echo 0)"
          echo "findings_count=$FINDINGS_COUNT" >> "$GITHUB_OUTPUT"

          if [ "$FINDINGS_COUNT" -gt 0 ]; then
            echo "findings_detected=true" >> "$GITHUB_OUTPUT"
          else
            echo "findings_detected=false" >> "$GITHUB_OUTPUT"
          fi

          {
            echo "# Sighthound scan summary"
            echo ""
            echo "- Exit code: $SCAN_EXIT"
            echo "- Findings count: $FINDINGS_COUNT"
          } > "$RESULTS_DIR/summary.md"

      - name: Compute artifact name
        id: artifact_name
        run: |
          echo "value=sighthound-results-${{ github.run_id }}" >> "$GITHUB_OUTPUT"

      - name: Upload Sighthound results artifact
        uses: actions/upload-artifact@v7.0.1
        with:
          name: ${{ steps.artifact_name.outputs.value }}
          path: /tmp/gh-aw/agent/sighthound
          if-no-files-found: error
          retention-days: 7

steps:
  - name: Download Sighthound artifact
    uses: actions/download-artifact@v8.0.1
    with:
      name: ${{ needs.sighthound_scan.outputs.artifact_name }}
      path: /tmp/gh-aw/agent/sighthound

safe-outputs:
  add-comment:
    max: 1
    target: "*"
  create-issue:
    max: 1
    labels: [security, sighthound]
  noop:
---

# Sighthound Security Scan Triage

The `sighthound_scan` job already scanned this repository using Sighthound and uploaded results to `/tmp/gh-aw/agent/sighthound/results.json`.

## Task

1. Read `/tmp/gh-aw/agent/sighthound/results.json` and `/tmp/gh-aw/agent/sighthound/summary.md`.
2. Confirm whether findings are valid and actionable in this repository.
3. If no actionable findings remain after review, call `noop` with a short explanation.
4. If actionable findings exist and this run was triggered by a pull request, call `add_comment` (without `item_number`) with:
   - total findings count
   - top findings grouped by severity
   - concrete remediation guidance
5. If actionable findings exist and this run was not triggered by a pull request, call `create_issue` with:
   - title: `Sighthound findings in ${{ github.repository }} (run ${{ github.run_id }})`
   - a concise summary
   - key findings and remediation guidance

Keep output concise and only report real findings from the artifact.
