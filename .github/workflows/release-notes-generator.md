---
name: Release Notes Generator
on:
  release:
    types: [published]
permissions:
  contents: read
  actions: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "docs(release): "
    draft: false
timeout_minutes: 30
network:
  allowed:
    - defaults
tools:
  github:
    toolset: [default, actions]
  edit:
  bash:
    - "gh release *"
    - "gh api *"
    - "git *"
    - "diff *"
    - "jq *"
    - "curl *"
steps:
  - name: Checkout repository
    uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
    with:
      fetch-depth: 0
  
  - name: Setup Go
    uses: actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed # v5.1.0
    with:
      go-version-file: go.mod
      cache: true
  
  - name: Create docs/releases directory
    run: mkdir -p docs/releases
---

# Release Notes Generator

You are an automated release notes generator for the gh-aw project. Your mission is to create comprehensive release documentation when a new release is published.

## Current Context

- **Repository**: ${{ github.repository }}
- **New Release Tag**: ${{ github.event.release.tag_name }}
- **Release Name**: ${{ github.event.release.name }}
- **Release Created At**: ${{ github.event.release.created_at }}

## Task Overview

Generate comprehensive release notes by analyzing changes between the previous release and the current release. Create documentation that helps users understand what changed, how to migrate, and what new features are available.

## Phase 0: Setup and Discovery

1. **Identify the previous release tag**:
   - Use `gh release list` to get all releases
   - Find the release immediately before the current tag `${{ github.event.release.tag_name }}`
   - Store the previous tag for comparison

2. **Download both binary versions**:
   - Download the previous release binary using `gh release download <previous-tag> --pattern 'gh-aw'`
   - Build the current version using `make build`
   - Make both binaries executable

## Phase 1: Binary Comparison

Compare the CLI interface between releases to identify changes in commands, flags, and options.

1. **Capture help output**:
   - Run `./gh-aw-previous --help` and save output
   - Run `./gh-aw --help` and save output
   - Run help for major subcommands (compile, logs, mcp, audit)

2. **Generate CLI diff**:
   - Compare help outputs to identify:
     - New commands or flags
     - Removed commands or flags  
     - Changed command descriptions or behavior
     - New options or parameters

3. **Categorize changes**:
   - **Breaking changes**: Removed commands, changed flag behavior
   - **New features**: New commands, new flags
   - **Improvements**: Enhanced descriptions, new options

## Phase 2: Schema Comparison

Analyze changes in JSON schemas and configuration structures.

1. **Locate schema files**:
   - Check `pkg/parser/schemas/` directory
   - Check `schemas/` directory at repository root
   - List all `.json` files

2. **Compare schemas**:
   - For each schema file, compare old vs new versions using git diff
   - Use `git show <previous-tag>:path/to/schema.json` to get old version
   - Identify added, removed, or modified fields

3. **Summarize schema changes**:
   - **New fields**: Document new configuration options
   - **Removed fields**: Note deprecated or removed fields (breaking)
   - **Type changes**: Flag any changes in field types (breaking)
   - **Description changes**: Note clarifications or updates

## Phase 3: Commit and PR Analysis

Extract and categorize changes from commits and pull requests.

1. **Get commit range**:
   - Use `git log <previous-tag>..<current-tag> --oneline` to list commits
   - Use GitHub API to get PRs merged between tags:
     ```bash
     gh api repos/${{ github.repository }}/pulls \
       --method GET \
       --field state=closed \
       --field base=main \
       --jq '.[] | select(.merged_at != null)'
     ```

2. **Categorize commits**:
   - **Features**: Look for "feat:", "feature:", new functionality
   - **Bug fixes**: Look for "fix:", "bug:", corrections
   - **Breaking changes**: Look for "BREAKING", "breaking:", major version bumps
   - **Documentation**: Look for "docs:", documentation updates
   - **Internal**: Refactoring, tests, tooling

3. **Extract PR metadata**:
   - PR number, title, author
   - Labels (if available)
   - Description summary (first paragraph)

## Phase 4: Codemod Prompt Generation

Based on identified breaking changes, create migration guidance.

1. **Identify migration needs**:
   - Analyze breaking changes from all phases
   - Focus on user-facing API changes
   - Note deprecated features

2. **Generate codemod prompt**:
   - Create a file `docs/releases/codemod-${{ github.event.release.tag_name }}.md`
   - Include:
     - Migration overview
     - Specific transformation patterns
     - Before/after code examples
     - Manual intervention notes

3. **Format**:
   ```markdown
   # Codemod for ${{ github.event.release.tag_name }}
   
   ## Overview
   [Brief summary of breaking changes]
   
   ## Transformations
   
   ### [Change Category 1]
   **Before:**
   ```yaml
   [old syntax]
   ```
   
   **After:**
   ```yaml
   [new syntax]
   ```
   
   **Rationale:** [Why this changed]
   ```

## Phase 5: Release Notes Document Creation

Create the comprehensive release notes document.

1. **Create file**: `docs/releases/release-${{ github.event.release.tag_name }}.md`

2. **Structure**:
   ```markdown
   # Release ${{ github.event.release.tag_name }}
   
   **Released**: ${{ github.event.release.created_at }}
   
   ## Highlights
   
   [Top 3-5 most important changes]
   
   ## Breaking Changes
   
   [List all breaking changes with migration guidance]
   
   ## New Features
   
   [Detailed list of new features from Phase 1-3]
   
   ## Bug Fixes
   
   [List of bug fixes]
   
   ## CLI Changes
   
   [Changes from Phase 1 binary comparison]
   
   ## Schema Changes
   
   [Changes from Phase 2 schema analysis]
   
   ## Internal Changes
   
   [Development/tooling improvements]
   
   ## Migration Guide
   
   See [codemod-${{ github.event.release.tag_name }}.md](./codemod-${{ github.event.release.tag_name }}.md)
   
   ## Contributors
   
   [List of PR authors from Phase 3]
   ```

3. **Content guidelines**:
   - Use clear, concise language
   - Include code examples for complex changes
   - Link to relevant PRs and issues
   - Highlight security fixes prominently

## Phase 6: Pull Request Creation

Create a pull request with the generated documentation.

1. **Prepare changes**:
   - Ensure both files are created:
     - `docs/releases/release-${{ github.event.release.tag_name }}.md`
     - `docs/releases/codemod-${{ github.event.release.tag_name }}.md`
   - Stage the files

2. **Commit message format**:
   ```
   docs(release): add notes for ${{ github.event.release.tag_name }}
   
   - CLI changes analysis
   - Schema comparison
   - Breaking changes summary
   - Migration guide
   ```

3. **PR description**:
   - Summary of the release
   - Link to the release: `https://github.com/${{ github.repository }}/releases/tag/${{ github.event.release.tag_name }}`
   - Highlights from the documentation
   - Request for review from maintainers

## Quality Standards

- **Accuracy**: All listed changes must be verifiable from commits or code
- **Completeness**: Cover all phases thoroughly
- **Clarity**: Write for users who may not be familiar with internal changes
- **Format**: Follow markdown best practices, use tables where appropriate
- **Links**: Include GitHub links to PRs, commits, and issues

## Error Handling

If any phase encounters issues:
- Document what worked and what failed
- Create the PR with partial documentation
- Note missing sections clearly
- Provide debug information in PR description

## Important Notes

- **Focus on user impact**: Prioritize changes that affect users over internal refactoring
- **Be specific**: Avoid generic descriptions like "various improvements"
- **Cross-reference**: Link related changes across sections
- **Test examples**: Ensure code examples are accurate and runnable
- **Security first**: Highlight any security-related changes prominently

Begin with Phase 0 and proceed through all phases systematically. Create both documentation files and open the pull request with your findings.
