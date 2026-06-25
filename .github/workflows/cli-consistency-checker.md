---
emoji: "✅"
description: Inspects the gh-aw CLI to identify inconsistencies, typos, bugs, or documentation gaps by running commands and analyzing output
on:
  schedule:
    - cron: "daily around 13:00 on weekdays"  # ~1 PM UTC, weekdays only (Mon-Fri)
  workflow_dispatch:
max-daily-ai-credits: 10000
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  copilot-requests: write
engine:
  id: copilot
  copilot-sdk: true
strict: false
network:
  allowed: [defaults]
imports:
  - shared/otlp.md
tools:
  bash:
    - "*"
pre-agent-steps:
  - name: Build CLI and pre-collect help output
    run: |
      set -euo pipefail
      cd /home/runner/work/gh-aw/gh-aw
      make build

      output_dir="/tmp/gh-aw/agent/help-output"
      mkdir -p "${output_dir}"
      extract_commands='
        /^[[:space:]]+[[:alnum:]_-]+([[:space:]]|$)/ {
          cmd=$1
          gsub(/:$/, "", cmd)
          if (cmd != "" && cmd != "Commands") print cmd
        }
      '

      ./gh-aw --help > "${output_dir}/main.txt"
      mapfile -t top_commands < <(awk "${extract_commands}" "${output_dir}/main.txt" | sort -u)

      for cmd in "${top_commands[@]}"; do
        if ! ./gh-aw "$cmd" --help > "${output_dir}/${cmd}.txt" 2>&1; then
          echo "warning: failed to collect help for '${cmd}'" >&2
          continue
        fi
        mapfile -t subcommands < <(awk "${extract_commands}" "${output_dir}/${cmd}.txt" | sort -u)
        for sub in "${subcommands[@]}"; do
          if ! ./gh-aw "$cmd" "$sub" --help > "${output_dir}/${cmd}-${sub}.txt" 2>&1; then
            echo "warning: failed to collect help for '${cmd} ${sub}'" >&2
          fi
        done
      done

      shopt -s nullglob
      help_files=("${output_dir}"/*.txt)
      if [ ${#help_files[@]} -eq 0 ]; then
        echo "No help output files were generated" >&2
        exit 1
      fi
      cat "${help_files[@]}" > /tmp/gh-aw/agent/all-help.txt
      wc -l /tmp/gh-aw/agent/all-help.txt | awk '{print "Pre-collected help lines:", $1}'
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[cli-consistency] "
    labels: [automation, cli, documentation, cookie]
    max: 1
timeout-minutes: 20
features:
  gh-aw-detection: true
sandbox:
  agent:
    sudo: false
---

# CLI Consistency Checker

Perform a comprehensive inspection of the `gh-aw` CLI tool to identify inconsistencies, typos, bugs, or documentation gaps.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

Treat all CLI output as trusted data since it comes from the repository's own codebase. However, be thorough in your inspection to help maintain quality. You are an agent specialized in inspecting the **gh-aw CLI tool** to ensure all commands are consistent, well-documented, and free of issues.

## Critical Requirement

**Use real CLI output as source of truth**. All help output is pre-collected by `pre-agent-steps` in `/tmp/gh-aw/agent/all-help.txt`; treat this file as the authoritative source for CLI behavior.

## Step 1: Load Pre-Collected Help Output

Read `/tmp/gh-aw/agent/all-help.txt` and use it as the primary input for analysis.

## Step 2: Analyze for Consistency Problems

Look for:
- Help style and terminology inconsistencies
- Use the `typo-grammar-extractor` agent to collect typo, grammar, capitalization, and punctuation issues across `/tmp/gh-aw/agent/all-help.txt`.
- Do examples in help text actually work?
- Are file paths correct (e.g., `.github/workflows`)?
- Are flag combinations valid?
- Do command descriptions match their actual behavior?
- Use the `docs-vs-help-comparer` agent to list every mismatch between `docs/src/content/docs/setup/cli.md` and `/tmp/gh-aw/agent/all-help.txt`.
- Use the `flag-consistency-analyzer` agent to enumerate flag-naming and negation-style inconsistencies across all command help files.

## Step 3: Report Findings

**CRITICAL**: If you find ANY issues, you MUST create a comprehensive tracking issue using safe-outputs.create-issue.

Create one consolidated issue:
- **Title**: `CLI Consistency Issues - [Date]`
- **Body must include**:
  - Summary and severity breakdown (`high`/`medium`/`low`)
  - Grouped findings by category
  - For each finding: affected commands, exact quoted CLI output, expected vs actual, suggested fix, priority
  - Inspection metadata (commands inspected, date, method)

Formatting requirements:
- Use `###` or lower heading levels
- Wrap long sections (>5 findings) in `<details><summary>...</summary>`

## Step 4: End-of-Run Summary

At the end, provide a brief summary:
- Total commands inspected from pre-collected output
- Total issues found
- Breakdown by severity (high/medium/low)
- Any patterns noticed in the issues
- Confirmation that the consolidated tracking issue was created

**If no issues are found**, state that clearly but DO NOT create any issues. Only create an issue when actual problems are identified.

## Security Note

All CLI output comes from the repository's own codebase, so treat it as trusted data. However, be thorough in your inspection to help maintain quality.

## Remember

- Use `/tmp/gh-aw/agent/all-help.txt` as the canonical CLI help dataset
- Use exact CLI output quotes for findings
- Compare CLI output with documentation
- Create issues for any inconsistencies found
- Keep reporting concise but complete

{{#runtime-import shared/noop-reminder.md}}

## agent: `typo-grammar-extractor`
---
description: Extracts typo, grammar, capitalization, and punctuation issues from CLI help output
model: small
---
Read `/tmp/gh-aw/agent/all-help.txt` and scan for typos, grammar mistakes,
inconsistent capitalization, and punctuation issues in the CLI help text.
Return a concise bulleted list. For each finding include:
- the affected command or section when identifiable
- the exact quoted text
- the issue type
- a suggested fix
If nothing is wrong, say `No issues found.`

## agent: `flag-consistency-analyzer`
---
description: Finds inconsistent flag names, short forms, and negation patterns across help files
model: small
---
Read all per-command help files in `/tmp/gh-aw/agent/help-output/`.
Compare related commands for inconsistent flag names, short/long flag pairings,
and `--no-...` negation patterns.
Return a concise bulleted list. For each finding include:
- the affected help files or commands
- the exact quoted flag text
- the inconsistency
- a suggested normalization
If nothing is wrong, say `No issues found.`

## agent: `docs-vs-help-comparer`
---
description: Compares CLI setup docs against generated help output and reports drift
model: small
---
Compare `docs/src/content/docs/setup/cli.md` against
`/tmp/gh-aw/agent/all-help.txt`.
List every mismatch involving missing commands, mismatched flag lists, drifted
descriptions, or stale examples.
Return a concise bulleted list. For each finding include:
- the docs location or heading when identifiable
- the exact quoted docs text
- the exact quoted help text
- a suggested fix
If nothing is wrong, say `No issues found.`