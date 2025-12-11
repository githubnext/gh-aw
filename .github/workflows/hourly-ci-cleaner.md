---
description: Hourly CI cleaner that fixes format, lint, and test issues when CI fails on main branch
on:
  schedule:
    - cron: "0 * * * *"  # Every hour
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: hourly-ci-cleaner
engine: copilot
tools:
  bash: ["*"]
  edit:
steps:
  - name: Check last CI workflow run status on main branch
    id: ci_check
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Get the last CI workflow run on main branch, excluding pending and cancelled runs
      LAST_RUN=$(gh run list --workflow=ci.yml --branch=main --limit 50 --json conclusion,status,databaseId \
        | jq -r '[.[] | select(.status == "completed" and (.conclusion == "success" or .conclusion == "failure"))] | .[0]')
      
      CONCLUSION=$(echo "$LAST_RUN" | jq -r '.conclusion')
      RUN_ID=$(echo "$LAST_RUN" | jq -r '.databaseId')
      
      echo "Last CI run conclusion: ${CONCLUSION}"
      echo "Run ID: ${RUN_ID}"
      
      # Write to environment and step summary
      echo "CI_STATUS=${CONCLUSION}" >> "$GITHUB_ENV"
      echo "CI_RUN_ID=${RUN_ID}" >> "$GITHUB_ENV"
      
      if [ "$CONCLUSION" = "success" ]; then
        echo "✅ CI is passing on main branch - no action needed" >> "$GITHUB_STEP_SUMMARY"
        echo "CI_NEEDS_FIX=false" >> "$GITHUB_ENV"
        exit 1
      else
        echo "❌ CI is failing on main branch - agent will attempt to fix" >> "$GITHUB_STEP_SUMMARY"
        echo "Run ID: ${RUN_ID}" >> "$GITHUB_STEP_SUMMARY"
        echo "CI_NEEDS_FIX=true" >> "$GITHUB_ENV"
      fi
safe-outputs:
  create-pull-request:
    title-prefix: "[ca] "
timeout-minutes: 45
imports:
  - ../agents/ci-cleaner.agent.md
---

# Hourly CI Cleaner

You are an automated CI cleaner that runs hourly to fix CI failures on the main branch.

## Mission

When CI fails on the main branch, automatically diagnose and fix the issues by:
1. Formatting code
2. Running and fixing linters
3. Running and fixing tests
4. Recompiling workflows

## Context

- **Repository**: ${{ github.repository }}
- **Run Number**: #${{ github.run_number }}
- **CI Status**: ${{ env.CI_STATUS }}

## Your Task

The CI workflow has failed on the main branch. Follow the instructions from the ci-cleaner agent to:

1. **Format sources** - Run `make fmt` to format all code
2. **Run linters** - Run `make lint` and fix any issues
3. **Run tests** - Run `make test-unit` and fix failures
4. **Recompile workflows** - Run `make recompile` to update lock files

## Execution Guidelines

- Work through each step systematically
- Fix issues as you encounter them
- Re-run checks after fixes to verify
- Only proceed to next step when current step passes
- Create a pull request with all fixes

## Pull Request Guidelines

Your pull request should:
- Have a clear title describing what was fixed (e.g., "[ca] Fix formatting and linting issues")
- Include a description of:
  - What CI failures were found
  - What fixes were applied
  - Confirmation that all checks now pass
- Be ready for review and merge

Begin by checking out the main branch and running the CI cleaner steps.
