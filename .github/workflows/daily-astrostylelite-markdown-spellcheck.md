---
name: Daily AstroStyleLite Markdown Spellcheck
description: Runs daily American English spellcheck for AstroStyleLite docs content and opens a safe PR only when findings exist
on:
  schedule:
    - cron: daily
  workflow_dispatch:

permissions:
  contents: read
  pull-requests: read

tracker-id: daily-astrostylelite-markdown-spellcheck
engine: claude
strict: true

if: needs.spellcheck.outputs.has_findings == 'true'

jobs:
  spellcheck:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    outputs:
      has_findings: ${{ steps.run_spellcheck.outputs.has_findings }}
      findings_count: ${{ steps.run_spellcheck.outputs.findings_count }}
      files_checked: ${{ steps.run_spellcheck.outputs.files_checked }}
      dictionary_path: ${{ steps.run_spellcheck.outputs.dictionary_path }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v6.0.2
        with:
          persist-credentials: false

      - name: Setup Node.js
        uses: actions/setup-node@v6.3.0
        with:
          node-version: "24"

      - name: Run Markdown spellcheck (pinned)
        id: run_spellcheck
        shell: bash
        run: |
          set -euo pipefail

          ARTIFACT_DIR="/tmp/gh-aw/spellcheck"
          mkdir -p "$ARTIFACT_DIR"

          find docs/src/content -type f \( -name '*.md' -o -name '*.mdx' \) | sort > "$ARTIFACT_DIR/files.txt"

          FILES_CHECKED=$(wc -l < "$ARTIFACT_DIR/files.txt" | tr -d ' ')

          DICTIONARY_PATH=""
          for candidate in \
            docs/.cspell-words.txt \
            docs/.spellcheck-ignore.txt \
            .cspell-words.txt \
            .spellcheck-ignore.txt \
            .github/spellcheck-ignore.txt
          do
            if [ -f "$candidate" ]; then
              DICTIONARY_PATH="$candidate"
              break
            fi
          done

          if [ "$FILES_CHECKED" -eq 0 ]; then
            echo '[]' > "$ARTIFACT_DIR/cspell-results.json"
          else
            npx --yes cspell@8.19.4 lint \
              --no-progress \
              --no-summary \
              --show-suggestions \
              --format json \
              --config docs/.cspell.docs.json \
              --file-list "$ARTIFACT_DIR/files.txt" \
              > "$ARTIFACT_DIR/cspell-results.json" || true
          fi

          FINDINGS_COUNT=$(jq '[.. | objects | select(has("issues")) | .issues[]? | select((.isFlagged // true) == true)] | length' "$ARTIFACT_DIR/cspell-results.json")

          jq -c '.. | objects | select(has("issues")) | .issues[]? | select((.isFlagged // true) == true) | {
            file: (.uri // .filename // .file // ""),
            line: (.row // .line // null),
            column: (.col // .column // null),
            word: (.text // .word // ""),
            suggestions: (.suggestions // []),
            flagged: (.isFlagged // true)
          }' "$ARTIFACT_DIR/cspell-results.json" > "$ARTIFACT_DIR/findings.ndjson"

          jq -n \
            --arg tool "cspell@8.19.4" \
            --arg locale "en-US" \
            --arg scope "docs/src/content/" \
            --arg dict "$DICTIONARY_PATH" \
            --argjson files_checked "$FILES_CHECKED" \
            --argjson findings "$FINDINGS_COUNT" \
            '{
              tool: $tool,
              locale: $locale,
              scope: $scope,
              file_patterns: ["**/*.md", "**/*.mdx"],
              files_checked: $files_checked,
              findings: $findings,
              has_findings: ($findings > 0),
              dictionary: {
                supported: true,
                path: (if $dict == "" then null else $dict end)
              }
            }' > "$ARTIFACT_DIR/summary.json"

          if [ "$FINDINGS_COUNT" -gt 0 ]; then
            echo "has_findings=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_findings=false" >> "$GITHUB_OUTPUT"
          fi

          echo "findings_count=$FINDINGS_COUNT" >> "$GITHUB_OUTPUT"
          echo "files_checked=$FILES_CHECKED" >> "$GITHUB_OUTPUT"
          echo "dictionary_path=$DICTIONARY_PATH" >> "$GITHUB_OUTPUT"

      - name: Upload spellcheck artifact
        if: success()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: spellcheck-results
          path: |
            /tmp/gh-aw/spellcheck/summary.json
            /tmp/gh-aw/spellcheck/cspell-results.json
            /tmp/gh-aw/spellcheck/findings.ndjson
            /tmp/gh-aw/spellcheck/files.txt
            docs/.cspell.docs.json
          if-no-files-found: error
          retention-days: 3

safe-outputs:
  create-pull-request:
    expires: 3d
    title-prefix: "[docs] "
    labels: [documentation, spellcheck, automation]
    draft: false
    fallback-as-issue: false
    preserve-branch-name: true
    allowed-files:
      - docs/src/content/**/*.md
      - docs/src/content/**/*.mdx

steps:
  - name: Download spellcheck artifact
    uses: actions/download-artifact@v8.0.1
    with:
      name: spellcheck-results
      path: /tmp/gh-aw/spellcheck

tools:
  mount-as-clis: true
  bash: true
  edit:

features:
  mcp-cli: true
---

# Daily AstroStyleLite Markdown Spellcheck

You maintain spelling quality for AstroStyleLite documentation under `docs/src/content/`.

## Scope

- Only process files under `docs/src/content/`
- Only modify markdown content files:
  - `*.md`
  - `*.mdx`
- Use American English conventions

## Inputs from the Spellcheck Job

The spellcheck job runs after activation and before the agent job, and stores machine-readable results at:

- `/tmp/gh-aw/spellcheck/summary.json`
- `/tmp/gh-aw/spellcheck/cspell-results.json`
- `/tmp/gh-aw/spellcheck/findings.ndjson`
- `/tmp/gh-aw/spellcheck/files.txt`
- `docs/.cspell.docs.json`

Dictionary source files referenced by `docs/.cspell.docs.json` are optional:

- `docs/.cspell-words.txt`
- `docs/.spellcheck-ignore.txt`
- `.cspell-words.txt`
- `.spellcheck-ignore.txt`
- `.github/spellcheck-ignore.txt`

If a dictionary file is absent, spellcheck continues without it.

Spellcheck summary:

- Files checked: `${{ needs.spellcheck.outputs.files_checked }}`
- Findings: `${{ needs.spellcheck.outputs.findings_count }}`
- Dictionary file (if any): `${{ needs.spellcheck.outputs.dictionary_path }}`

## Conditional Execution

This workflow is intentionally gated so the agent path only runs when `needs.spellcheck.outputs.has_findings == 'true'`.
When no findings exist, the workflow stops after spellcheck and skips agent execution.

## Task

1. Read `/tmp/gh-aw/spellcheck/summary.json` and `/tmp/gh-aw/spellcheck/findings.ndjson`.
2. Apply only justified spelling fixes in `docs/src/content/**/*.md` and `docs/src/content/**/*.mdx`.
3. Preserve technical terms, product names, code symbols, and intentional capitalization.
4. Do not re-run spellcheck in the agent job; use the provided artifact as the source of truth.
5. If there are no safe fixes to apply, call `noop`.

## Pull Request Requirements

When creating the pull request:

- Use branch name format: `spellcheck/YYYY-MM-DD` (prefix must be `spellcheck/`)
- Provide the full branch name in the `create_pull_request` call (the prefix is not auto-added)
- State that the run is automated
- State the scope is `docs/src/content/`
- State changes are markdown spellcheck fixes only
- Summarize changed files
- Mention dictionary path only if one was used

## Safety Constraints

- Do not modify files outside `docs/src/content/`
- Do not modify non-markdown files
- Use only the safe output `create_pull_request` for repository writes

If no action is needed, call:

```json
{"noop": {"message": "No valid markdown spellcheck fixes needed for docs/src/content/."}}
```
