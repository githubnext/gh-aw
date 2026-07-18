---
private: true
emoji: "🛡️"
name: Daily VulnHunter Scan
description: Daily Claude Code workflow that clones Capital One VulnHunter and runs its vulnhunt methodology inside the sandbox against this repository
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine:
  id: claude
  model: claude-opus-4.6
sandbox:
  agent:
    sudo: false
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
    question: Did the agent clone VulnHunter, load its vulnhunt skill instructions, and complete a repository scan?
  - id: issue_created_or_noop
    question: Was a security issue created for verified exploitable findings, or was noop used when VulnHunter found nothing actionable?
---

# Daily VulnHunter Scan

Run Capital One's [VulnHunter](https://github.com/capitalone/VulnHunter) methodology inside the sandbox against `${{ github.workspace }}`.

## Task

1. Create a fresh working directory under `/tmp/gh-aw/agent/vulnhunter`.
2. Download and extract the latest `capitalone/VulnHunter` source archive into that directory.
3. Read the extracted scanner instructions before analyzing the repository:
   - `README.md`
   - `vulnhunt/README.md`
   - `vulnhunt/SKILL.md`
   - every file under `vulnhunt/phases/`
4. Follow the extracted `vulnhunt` instructions as your operating playbook and scan `${{ github.workspace }}` for verified, exploitable vulnerabilities.
5. Save your intermediate notes and any machine-readable findings under `/tmp/gh-aw/agent/vulnhunter/out/`.

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
