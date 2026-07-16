---
private: true
on:
  schedule: daily
  workflow_dispatch: null
permissions:
  contents: read
  issues: read
  pull-requests: read

sandbox:
  agent:
    sudo: false

imports:
- shared/otlp.md
safe-outputs:
  create-pull-request:
    draft: false
    expires: 2d
    labels:
    - enhancement
    - automation
    title-prefix: "[log] "
steps:
- name: Build deterministic logger manifest
  id: logger_manifest
  run: |
    set -euo pipefail
    cache_dir="/tmp/gh-aw/cache-memory/go-logger"
    out_dir="/tmp/gh-aw/agent/go-logger"
    mkdir -p "$cache_dir" "$out_dir"

    current_sha="$(git rev-parse HEAD)"
    current_files="$out_dir/current-files.txt"
    processed_files="$cache_dir/processed-files.txt"
    find pkg -name '*.go' -type f ! -name '*_test.go' | sort > "$current_files"
    [ -f "$processed_files" ] || : > "$processed_files"

    new_files="$out_dir/new-files.txt"
    comm -23 "$current_files" "$processed_files" > "$new_files" || true

    last_sha=""
    if [ -f "$cache_dir/last-run.json" ]; then
      last_sha="$(jq -r '.commit_sha // empty' "$cache_dir/last-run.json" 2>/dev/null || true)"
    fi

    files_needing_logger="$out_dir/files-needing-logger.txt"
    files_missing_logger_import="$out_dir/files-missing-logger-import.txt"
    call_sites="$out_dir/call-sites.tsv"
    : > "$files_needing_logger"
    : > "$files_missing_logger_import"
    : > "$call_sites"

    while IFS= read -r rel; do
      [ -f "$rel" ] || continue
      if ! grep -q "var log = logger.New" "$rel"; then
        echo "$rel" >> "$files_needing_logger"
      fi
      if grep -q "log\\." "$rel" && ! grep -q '"github.com/github/gh-aw/pkg/logger"' "$rel"; then
        echo "$rel" >> "$files_missing_logger_import"
      fi
      while IFS=: read -r line_number match_line; do
        function_name="$(printf '%s' "$match_line" | sed -E 's/^[[:space:]]*func[[:space:]]+([A-Za-z0-9_]+).*/\\1/')"
        printf "%s\\t%s\\t%s\\n" "$rel" "$line_number" "$function_name" >> "$call_sites"
      done < <(grep -nE "^[[:space:]]*func[[:space:]]+[A-Za-z0-9_]+" "$rel" || true)
    done < "$current_files"

    # Write each JSON payload to a file to avoid exceeding ARG_MAX with large datasets.
    jq -R -s 'split("\n") | map(select(length > 0))' "$files_needing_logger" \
      > "$out_dir/files-needing-logger.json"
    jq -R -s 'split("\n") | map(select(length > 0))' "$files_missing_logger_import" \
      > "$out_dir/files-missing-logger-import.json"
    jq -R -s 'split("\n") | map(select(length > 0) | split("\t") | {file: .[0], line: (.[1] | tonumber), function: .[2]})' "$call_sites" \
      > "$out_dir/candidate-call-sites.json"

    # Build manifest by reading files via --slurpfile (no large args on argv).
    jq -n \
      --slurpfile files_needing_logger "$out_dir/files-needing-logger.json" \
      --slurpfile missing_logger_import "$out_dir/files-missing-logger-import.json" \
      --slurpfile candidate_call_sites "$out_dir/candidate-call-sites.json" \
      '{files_needing_logger: $files_needing_logger[0], missing_logger_import: $missing_logger_import[0], candidate_call_sites: $candidate_call_sites[0]}' \
      > "$out_dir/manifest.json"

    should_run=true
    if [ "$current_sha" = "$last_sha" ] && [ ! -s "$new_files" ]; then
      should_run=false
    fi
    echo "{\"should_run\": \"$should_run\", \"current_sha\": \"$current_sha\", \"last_sha\": \"$last_sha\", \"manifest\": \"$out_dir/manifest.json\", \"new_files\": \"$new_files\"}" > "$out_dir/preflight.json"
    echo "should_run=$should_run" >> "$GITHUB_OUTPUT"
- name: Setup Node.js
  uses: actions/setup-node@v7.0.0
  with:
    cache: npm
    cache-dependency-path: actions/setup/js/package-lock.json
    node-version: "24"
- name: Setup Go
  uses: actions/setup-go@v7.0.0
  with:
    cache: true
    go-version-file: go.mod
- name: Install npm dependencies
  run: npm ci
  working-directory: ./actions/setup/js
description: Analyzes and enhances Go logging practices across the codebase for improved debugging and observability
emoji: 📝
engine: claude
name: Go Logger Enhancement
timeout-minutes: 15
tools:
  bash:
  - cat /tmp/gh-aw/agent/go-logger/preflight.json
  - cat /tmp/gh-aw/agent/go-logger/manifest.json
  - cat /tmp/gh-aw/agent/go-logger/new-files.txt
  - make build
  - make fmt
  - make recompile
  - ./gh-aw compile
  - git
  cache-memory: null
  cli-proxy: true
  edit: null
  github:
    mode: gh-proxy
    toolsets:
    - default
---
# Go Logger Enhancement

You are an AI agent that improves Go code by adding debug logging statements to help with troubleshooting and development.

## Validation Commands

Use **bash** for all build and validation commands in this workflow to avoid MCP connection timeouts during long file-exploration phases.

```bash
make build && make fmt       # Build the project and check formatting
make recompile               # Recompile workflows only if you changed .md files
```

## Efficiency First: Use Pre-flight Outputs

Before analyzing files, read `/tmp/gh-aw/agent/go-logger/preflight.json` and `/tmp/gh-aw/agent/go-logger/manifest.json`.

- The pre-flight step already computed whether this run should proceed.
- If cache files are missing (cold cache / first run), treat that as expected and continue.
- Only report `missing_data` when cache files exist but are unreadable/corrupted.
- Update cache after processing:
  - Save list of processed files to `processed-files.txt`
  - Save current commit SHA to `last-run.json`
  - Save summary of changes made

## Mission

Add meaningful debug logging calls to Go files in the `pkg/` directory following the project's logging guidelines from AGENTS.md.

## Important Constraints

1. **Maximum 5 files per pull request** - Keep changes focused and reviewable
2. **Skip test files** - Never modify files ending in `_test.go`
3. **No side effects** - Logger arguments must NOT compute anything or cause side effects
4. **Follow logger naming convention** - Use `pkg:filename` pattern (e.g., `workflow:compiler`)

## Logger Guidelines from AGENTS.md

Read the **Debug Logging** section of `AGENTS.md` with the read or bash tools, then follow its logger naming convention (`pkg:filename`), usage patterns, and "When to Add Logging" guidance.

## Task Steps

### 1. Read Deterministic Candidate Manifest

Use `/tmp/gh-aw/agent/go-logger/manifest.json` as the source of truth for:
- `files_needing_logger`
- `missing_logger_import`
- `candidate_call_sites`

### 2. Select Files for Enhancement

From the list of Go files:
1. Prioritize files without loggers or with minimal logging
2. Focus on files with complex logic (workflows, parsers, compilers)
3. Avoid trivial files with just simple functions
4. **Select exactly 5 files maximum** for this PR

### 3. Analyze Each Selected File

For each selected file:
1. Read the file content to understand its structure
2. Identify functions that would benefit from logging
3. Check if the file already has a logger declaration
4. Plan where to add logging calls

### 4. Add Logger and Logging Calls

For each file:

1. **Add logger declaration if missing:**
   - Add import: `"github.com/github/gh-aw/pkg/logger"`
   - Add logger variable using correct naming: `var log = logger.New("pkg:filename")`

2. **Add meaningful logging calls:**
   - Add logging at function entry for important functions
   - Add logging before/after state changes
   - Add logging for control flow decisions
   - Ensure log arguments don't have side effects
   - Use `log.Enabled()` check for expensive debug info

3. **Keep it focused:**
   - 2-5 logging calls per file is usually sufficient
   - Don't over-log - focus on the most useful information
   - Ensure messages are meaningful and helpful for debugging

### 5. Validation (After All Files)

After adding logging to **all selected files**, validate your changes before creating a PR:

1. **Build the project and check formatting:**
   ```bash
   make build && make fmt
   ```
   This catches compilation errors and import formatting issues without the full unit test suite.

2. **If needed, recompile workflows:**
   ```bash
   make recompile
   ```
   Only run this if you changed any `.md` workflow files during this session.

### 6. Create Pull Request

After validating your changes:

1. The safe-outputs create-pull-request will automatically create a PR
2. Ensure your changes follow the guidelines above
3. The PR title will automatically have the "[log] " prefix

## Quality Checklist

Before creating the PR, verify:

- [ ] Maximum 5 files modified
- [ ] No test files modified (`*_test.go`)
- [ ] Each file has logger declaration with correct naming convention
- [ ] Logger arguments don't compute anything or cause side effects
- [ ] Logging messages are meaningful and helpful
- [ ] No duplicate logging with existing logs
- [ ] Import statements are properly formatted
- [ ] Changes validated with `make build && make fmt`

## Important Notes

- You have access to the edit tool to modify files
- You have access to bash commands to explore the codebase
- The safe-outputs create-pull-request will automatically create the PR
- Focus on quality over quantity - 5 well-logged files is better than 10 poorly-logged files
- Remember: debug logs are for developers, not end users

## Structured Patch Output

When proposing per-file logger changes, use this compact schema in your reasoning/output to reduce verbose prose:

```json
{
  "patches": [
    {
      "file": "pkg/path/file.go",
      "logger_name": "pkg:filename",
      "add_import": true,
      "add_logger_var": true,
      "call_sites": [
        {"line": 123, "function": "Run", "message": "enter Run"}
      ]
    }
  ]
}
```

Good luck enhancing the codebase with better logging!