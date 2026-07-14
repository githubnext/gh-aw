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
if: needs.sighthound_scan.outputs.actionable_findings_detected == 'true'
jobs:
  sighthound_scan:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    env:
      SCAN_ROOT: /tmp/gh-aw/sighthound/repo
    outputs:
      findings_detected: ${{ steps.scan.outputs.findings_detected }}
      findings_count: ${{ steps.scan.outputs.findings_count }}
      actionable_findings_detected: ${{ steps.prefilter.outputs.actionable_findings_detected }}
      artifact_name: ${{ steps.artifact_name.outputs.value }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.0
        with:
          persist-credentials: false

      - name: Restore Sighthound binary cache
        id: cache-sighthound
        uses: actions/cache@v6
        with:
          path: ~/.cargo/bin/sighthound
          key: sighthound-${{ runner.os }}-v1
          # Bump the 'v1' suffix in the key above to force a rebuild
          # (e.g., when a newer Sighthound release is needed).
          restore-keys: sighthound-${{ runner.os }}-

      - name: Add cargo bin to PATH
        run: echo "$HOME/.cargo/bin" >> "$GITHUB_PATH"

      - name: Install Sighthound
        if: steps.cache-sighthound.outputs.cache-hit != 'true'
        run: |
          cargo install --locked --git https://github.com/Corgea/Sighthound --bin sighthound
          sighthound --help >/dev/null

      - name: Prepare clean scan root
        run: |
          set -euo pipefail
          rm -rf "$SCAN_ROOT"
          mkdir -p "$SCAN_ROOT"
          git archive --format=tar HEAD | tar -xf - -C "$SCAN_ROOT"

      - name: Run Sighthound scan
        id: scan
        run: |
          set -euo pipefail
          RESULTS_DIR="/tmp/gh-aw/agent/sighthound"
          RESULTS_JSON="$RESULTS_DIR/results.json"
          mkdir -p "$RESULTS_DIR"

          set +e
          sighthound --output-format json "$SCAN_ROOT" > "$RESULTS_JSON"
          SCAN_EXIT=$?
          set -e

          if [ ! -s "$RESULTS_JSON" ]; then
            echo "[]" > "$RESULTS_JSON"
          fi

          FINDINGS_COUNT="$(jq 'if type=="array" then length else 0 end' "$RESULTS_JSON" 2>/dev/null || echo 0)"
          echo "findings_count=$FINDINGS_COUNT" >> "$GITHUB_OUTPUT"

          if [ "$SCAN_EXIT" -ne 0 ] && [ "$FINDINGS_COUNT" -eq 0 ]; then
            echo "::error::Sighthound failed with exit code $SCAN_EXIT before producing any findings."
            exit "$SCAN_EXIT"
          fi

          if [ "$FINDINGS_COUNT" -gt 0 ]; then
            echo "findings_detected=true" >> "$GITHUB_OUTPUT"
          else
            echo "findings_detected=false" >> "$GITHUB_OUTPUT"
          fi

          {
            echo "# Sighthound scan summary"
            echo ""
            echo "- Exit code: $SCAN_EXIT"
            echo "- Scan root: $SCAN_ROOT"
            echo "- Findings count: $FINDINGS_COUNT"
          } > "$RESULTS_DIR/summary.md"

      - name: Pre-filter findings
        id: prefilter
        run: |
          set -euo pipefail
          RESULTS_JSON="/tmp/gh-aw/agent/sighthound/results.json"
          ACTIONABLE_JSON="/tmp/gh-aw/agent/sighthound/actionable.json"

          # Remove findings whose file path is under a known test-only directory.
          # Sighthound uses a 'file' field for the path; fall back to 'path' if absent.
          jq '[.[] | select(
            ((.file // .path // "") | test("testdata/|/testdata$|_test\\.go")) | not
          )]' "$RESULTS_JSON" > "$ACTIONABLE_JSON"

          ACTIONABLE_COUNT="$(jq 'length' "$ACTIONABLE_JSON" 2>/dev/null || echo 0)"
          echo "actionable_count=$ACTIONABLE_COUNT" >> "$GITHUB_OUTPUT"
          if [ "$ACTIONABLE_COUNT" -gt 0 ]; then
            echo "actionable_findings_detected=true" >> "$GITHUB_OUTPUT"
          else
            echo "actionable_findings_detected=false" >> "$GITHUB_OUTPUT"
          fi
          echo "Pre-filter: $(jq 'length' "$RESULTS_JSON") total → $ACTIONABLE_COUNT potentially actionable"

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

The `sighthound_scan` job ran Sighthound and pre-filtered findings. Read the files:
- `/tmp/gh-aw/agent/sighthound/actionable.json` — findings outside test/testdata paths (primary input)
- `/tmp/gh-aw/agent/sighthound/summary.md` — scan summary

## Task

1. Read `/tmp/gh-aw/agent/sighthound/actionable.json` and `/tmp/gh-aw/agent/sighthound/summary.md`.
2. If this is triggered by a pull request, call `add_comment` (no `item_number`) with: total count, top findings by severity, and remediation guidance.
3. If not triggered by a pull request, call `create_issue` with:
   - title: `Sighthound findings in ${{ github.repository }} (run ${{ github.run_id }})`
   - concise summary, key findings, and remediation guidance.

Keep output concise. Only report findings from `actionable.json`.
