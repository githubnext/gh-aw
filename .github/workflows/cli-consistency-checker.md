---
on:
  schedule:
    - cron: "0 13 * * 1-5"  # Daily at 1 PM UTC, weekdays only (Mon-Fri)
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
network:
  allowed: [defaults, node, "api.github.com"]
tools:
  edit:
  web-fetch:
  bash:
    - "*"
safe-outputs:
  create-issue:
    title-prefix: "[cli-consistency] "
    labels: [automation, cli, documentation]
    max: 5
timeout-minutes: 20
---

# CLI Consistency Checker

Inspect the gh-aw CLI to ensure all commands are consistent, well-documented, and free of issues.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

## Your Task

Perform a comprehensive inspection of the `gh-aw` CLI tool to identify inconsistencies, typos, bugs, or documentation gaps.

### Step 1: Build and Test the CLI

1. Build the CLI binary:
   ```bash
   cd /home/runner/work/gh-aw/gh-aw
   make build
   ```

2. Verify the build was successful and the binary exists at `./gh-aw`

### Step 2: Explore All Commands

Run `./gh-aw --help` and explore EVERY available command and subcommand:

For each command discovered:
- Run `./gh-aw <command> --help` to get detailed help
- For commands with subcommands, run `./gh-aw <command> <subcommand> --help`
- Document the command structure, flags, and descriptions

Commands to inspect include (but are not limited to):
- `gh aw compile`
- `gh aw new`
- `gh aw add`
- `gh aw remove`
- `gh aw enable`
- `gh aw disable`
- `gh aw status`
- `gh aw logs`
- `gh aw audit`
- `gh aw mcp` (and all subcommands: list, list-tools, inspect, add)
- `gh aw pr`
- `gh aw run`
- `gh aw trial`
- `gh aw update`
- `gh aw init`
- `gh aw version`
- `gh aw mcp-server`

### Step 3: Check for Consistency Issues

Look for these types of problems:

**Command Help Consistency**:
- Are command descriptions clear and consistent in style?
- Do all commands have proper examples?
- Are flag names and descriptions consistent across commands?
- Are there duplicate command names or aliases?
- Check for inconsistent terminology (e.g., "workflow" vs "workflow file")

**Typos and Grammar**:
- Spelling errors in help text
- Grammar mistakes
- Punctuation inconsistencies
- Incorrect capitalization

**Technical Accuracy**:
- Do examples in help text actually work?
- Are file paths correct (e.g., `.github/workflows`)?
- Are flag combinations valid?
- Do command descriptions match their actual behavior?

**Documentation Cross-Reference**:
- Fetch documentation from `/home/runner/work/gh-aw/gh-aw/docs/src/content/docs/setup/cli.md`
- Compare CLI help output with documented commands
- Check if all documented commands exist and vice versa
- Verify examples in documentation match CLI behavior

**Flag Consistency**:
- Are verbose flags (`-v`, `--verbose`) available consistently?
- Are help flags (`-h`, `--help`) documented everywhere?
- Do similar commands use similar flag names?
- Check for missing commonly expected flags

### Step 4: Report Findings

**CRITICAL**: If you find ANY issues, you MUST create issues using safe-outputs.create-issue.

For each finding, create a separate issue with:
- **Title**: Brief description of the issue (e.g., "Typo in compile command help", "Missing example in logs command")
- **Body**: Include:
  - The command/subcommand affected
  - The specific issue found (with exact quotes)
  - The expected vs actual behavior
  - Suggested fix if applicable
  - Priority level: `high` (breaks functionality), `medium` (confusing/misleading), `low` (minor inconsistency)

**Example Issue Format**:
```markdown
## Issue Description

**Command**: `gh aw compile`
**Type**: Typo in help text
**Priority**: Low

### Current Text
"Compile markdown to YAML workflows"

### Issue
The word "markdown" should be capitalized in the help text for consistency with other commands.

### Suggested Fix
"Compile Markdown to YAML workflows"
```

### Step 5: Summary

At the end, provide a brief summary:
- Total commands inspected
- Total issues found
- Breakdown by severity (high/medium/low)
- Any patterns noticed in the issues

**If no issues are found**, state that clearly but DO NOT create an issue. Only create issues when actual problems are identified.

## Security Note

Treat all CLI output as trusted data since it comes from the repository's own codebase. However, be thorough in your inspection to help maintain quality.
