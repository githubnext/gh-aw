---
description: Guidance for creating agentic workflows that analyze test coverage — prefer reading pre-computed CI artifacts over re-running tests.
---

# Test Coverage Workflow Guidance

Consult this file when creating or updating an agentic workflow that analyzes test coverage (e.g., coverage PR analyzers, coverage trend reporters, coverage gate enforcers).

## Core Principle: Read Artifacts First

**Always prefer reading pre-computed coverage artifacts from a previous CI run over re-running the full test suite inside the workflow.**

Re-running tests is slow, resource-intensive, and duplicates work that CI has already done. Coverage data is almost always available as a CI artifact from the same commit or PR.

## Two Patterns: Preferred vs Fallback

### ✅ Preferred: Fetch Pre-Computed Coverage Artifacts

Use this when a CI workflow already runs tests and uploads coverage reports as artifacts.

**Using `gh run download`** (bash tool):

```bash
# Find the latest successful CI run for this commit/branch
gh run list --branch "$BRANCH" --workflow ci.yml --status success --limit 1 --json databaseId -q '.[0].databaseId'

# Download the coverage artifact
gh run download "$RUN_ID" --name coverage-report --dir /tmp/coverage

# Parse coverage data (format varies: lcov, JSON, XML, plain text)
cat /tmp/coverage/coverage-summary.json
```

**Using the `actions` toolset** (MCP tool):

```yaml
tools:
  github:
    toolsets: [actions]
```

Then in the prompt:
```
Use the `list_workflow_run_artifacts` tool to find the coverage artifact from the latest
CI run on this PR's head commit. Download and parse it using `download_workflow_run_artifact`.
```

### ⚠️ Fallback: Run Tests to Compute Fresh Coverage

Use this **only when**:
- No prior CI run exists for this commit (e.g., first push on a brand-new branch)
- The existing CI does not upload coverage artifacts
- The user explicitly requests fresh coverage data

```bash
# Example: run tests with coverage (language-specific)
# Node.js
npx jest --coverage --coverageReporters=json-summary

# Python
python -m pytest --cov=src --cov-report=json

# Go
go test ./... -coverprofile=/tmp/coverage.out
go tool cover -func=/tmp/coverage.out
```

When using the fallback, always inform the user that no prior coverage artifact was found and tests are being re-run.

## Detecting Coverage Artifact Availability

Before falling back to running tests, the agent should:

1. **Check for prior CI runs** on the same commit or branch head:
   ```bash
   gh run list --commit "$HEAD_SHA" --status success --json databaseId,workflowName
   ```

2. **Check for coverage artifacts** on found runs:
   ```bash
   gh run download "$RUN_ID" --name coverage-report --dir /tmp/coverage 2>/dev/null \
     && echo "artifact found" || echo "no artifact"
   ```

3. **Fall back gracefully** if no artifact is found.

## Recommended Decision Logic for the Agent Prompt

Include this decision logic in the workflow prompt body:

```
## Coverage Data Strategy

1. First, look for a pre-computed coverage artifact from the latest successful CI run
   on this PR's head commit using `gh run download --name <artifact-name>`.
2. If an artifact is found, parse and analyze it directly — do NOT re-run tests.
3. If no artifact is found, run the test suite with coverage enabled and note in
   your report that coverage was computed fresh (not from CI artifacts).
```

## Frontmatter Configuration

Coverage analysis workflows typically need:

```yaml
engine: copilot
triggers:
  pull_request:
    types: [opened, synchronize]
permissions:
  pull-requests: write    # to post coverage comment
  actions: read           # to download artifacts
network:
  defaults: true
  # Add language ecosystem if running tests as fallback:
  # egosystems: [node]  # or python, go, etc.
tools:
  github:
    toolsets: [default, actions]
safe-outputs:
  add-comment:
    hide-older-comments: true  # replace previous coverage comment
```

## Common Coverage Report Formats

| Tool | Artifact format | Key file |
|---|---|---|
| Jest (JS) | `coverage-report` | `coverage-summary.json` |
| Istanbul/nyc | `coverage-report` | `coverage-summary.json` |
| pytest-cov | `coverage-report` | `coverage.json` |
| Go cover | `coverage-report` | `coverage.out` |
| Cobertura (XML) | `coverage-report` | `coverage.xml` |
| lcov | `coverage-report` | `lcov.info` |
| simplecov (Ruby) | `coverage-report` | `.last_run.json` |

## Example: Coverage PR Analyzer

Below is a minimal example workflow that reads coverage artifacts and posts a summary comment:

```markdown
---
engine: copilot
triggers:
  pull_request:
    types: [opened, synchronize]
permissions:
  pull-requests: write
  actions: read
network:
  defaults: true
tools:
  github:
    toolsets: [default, actions]
safe-outputs:
  add-comment:
    hide-older-comments: true
---

Analyze test coverage for this pull request.

## Coverage Data Strategy

1. Find the latest successful CI run for the PR's head commit:
   `gh run list --commit "${{ github.event.pull_request.head.sha }}" --status success --limit 5 --json databaseId,workflowName`

2. Download the coverage artifact (try common names: `coverage-report`, `coverage`, `test-results`):
   `gh run download <run-id> --name coverage-report --dir /tmp/coverage`

3. If a coverage artifact is found, parse it and report coverage metrics.
   **Do NOT re-run tests** — use the existing CI data.

4. If no artifact is found, run the test suite with coverage enabled and note
   that coverage was computed fresh.

## Report Format

Post a comment on the PR with:
- Overall coverage percentage (and delta vs base branch if determinable)
- Files with decreased coverage (⚠️)
- Files with increased coverage (✅)
- Uncovered lines for changed files
```

## Anti-Patterns to Avoid

❌ **Never default to re-running tests** without first checking for artifacts:
```
# BAD: Always re-runs tests even when CI already computed coverage
npm test -- --coverage
```

❌ **Never ignore the `actions` toolset** when coverage data is available in GitHub:
```yaml
# BAD: Missing actions toolset
tools:
  github:
    toolsets: [default]
```

✅ **Always check for artifacts first**, then fall back:
```
# GOOD: Check artifact → fallback to test run only if needed
gh run download "$RUN_ID" --name coverage-report --dir /tmp/coverage \
  && analyze_artifact || run_tests_with_coverage
```
