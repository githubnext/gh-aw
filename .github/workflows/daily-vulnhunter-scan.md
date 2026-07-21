---
private: true
emoji: "🛡️"
name: Daily VulnHunter Scan
description: Daily Claude Code workflow that clones Capital One VulnHunter and runs its vulnhunt methodology inside the sandbox against this repository
on:
  schedule: daily
  workflow_dispatch:
permissions:
  actions: read
  contents: read
  issues: read
engine:
  id: claude
  model: claude-sonnet-4.6
jobs:
  vulnhunter_bundle:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    outputs:
      artifact_name: ${{ steps.artifact_name.outputs.value }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v7.0.0
        with:
          persist-credentials: false
      - name: Compute artifact name
        id: artifact_name
        run: |
          echo "value=vulnhunter-bundle-${{ github.run_id }}" >> "$GITHUB_OUTPUT"
      - name: Prepare VulnHunter bundle
        run: |
          set -euo pipefail
          BUNDLE_ROOT="$RUNNER_TEMP/vulnhunter-bundle"
          REPO_ROOT="$BUNDLE_ROOT/repo"
          SKILL_ROOT="$BUNDLE_ROOT/vulnhunter"

          rm -rf "$BUNDLE_ROOT"
          mkdir -p "$REPO_ROOT" "$SKILL_ROOT" "$BUNDLE_ROOT/out"

          git archive --format=tar HEAD | tar -xf - -C "$REPO_ROOT"
          curl -fsSL https://codeload.github.com/capitalone/VulnHunter/tar.gz/refs/heads/main \
            | tar -xz --strip-components=1 -C "$SKILL_ROOT"

          cat > "$BUNDLE_ROOT/README.md" <<'EOF'
          # VulnHunter bundle

          - `repo/` contains a clean snapshot of the target repository.
          - `vulnhunter/` contains the downloaded Capital One VulnHunter source tree.
          - `out/` is the writable directory for scan notes and structured findings.
          EOF
      - name: Upload VulnHunter bundle artifact
        uses: actions/upload-artifact@v7.0.1
        with:
          name: ${{ steps.artifact_name.outputs.value }}
          path: ${{ runner.temp }}/vulnhunter-bundle
          if-no-files-found: error
          retention-days: 7
sandbox:
  agent:
    sudo: false
steps:
  - name: Download VulnHunter bundle artifact
    uses: actions/download-artifact@v8.0.1
    with:
      name: ${{ needs.vulnhunter_bundle.outputs.artifact_name }}
      path: /tmp/gh-aw/agent/vulnhunter
tools:
  bash:
    - "*"
safe-outputs:
  create-issue:
    title-prefix: "[vulnhunter] "
    labels: [security, vulnhunter]
    close-older-issues: true
    max: 1
  noop:
timeout-minutes: 60
strict: true
network:
  allowed:
    - defaults
    - github
imports:
  - shared/otlp.md
evals:
  - id: scan_completed
    question: Did the agent download the prepared VulnHunter bundle artifact, load its vulnhunt skill instructions, and complete a repository scan?
  - id: issue_created_or_noop
    question: Was a security issue created for verified exploitable findings, or was noop used when VulnHunter found nothing actionable?
---

# Daily VulnHunter Scan

Run Capital One's [VulnHunter](https://github.com/capitalone/VulnHunter) methodology inside the sandbox against the repository snapshot packaged by the `vulnhunter_bundle` job.

## Task

1. Read `/tmp/gh-aw/agent/vulnhunter/README.md` for the prepared bundle layout.
2. Read the extracted scanner instructions before analyzing the repository snapshot:
   - `/tmp/gh-aw/agent/vulnhunter/vulnhunter/README.md`
   - `/tmp/gh-aw/agent/vulnhunter/vulnhunter/vulnhunt/README.md`
   - `/tmp/gh-aw/agent/vulnhunter/vulnhunter/vulnhunt/SKILL.md`
   - every file under `/tmp/gh-aw/agent/vulnhunter/vulnhunter/vulnhunt/phases/`
3. Follow the extracted `vulnhunt` instructions as your operating playbook and scan `/tmp/gh-aw/agent/vulnhunter/repo` for verified, exploitable vulnerabilities.
4. Save your intermediate notes and any machine-readable findings under `/tmp/gh-aw/agent/vulnhunter/out/`.

## Reporting Rules

- Only report findings that survive VulnHunter's falsification/disproof process.
- Do not report speculative, low-confidence, or test-only issues.
- If there are no verified exploitable findings, call `noop` with a short explanation.
- If there are verified findings, create exactly one issue summarizing up to the 5 highest-confidence vulnerabilities.

## Issue Format

Use the title `VulnHunter findings in ${{ github.repository }}`.

For each reported finding include:
- affected file(s) and function or component
- vulnerability type and severity
- attacker path or exploit preconditions
- why the finding is credible after falsification
- concrete remediation guidance

Keep the issue concise and evidence-backed.
