---
name: Daily Safe Output Integrator
description: Daily workflow that inspects test workflows in pkg/cli/workflows for safe-output coverage, detects missing safe-output types, creates test workflows and Go compilation tests for any missing types, then creates a PR or reports NOOP
on:
  schedule: daily
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tracker-id: daily-safe-output-integrator
engine: copilot
strict: true

tools:
  mount-as-clis: true
  github:
    toolsets: [default]
  bash:
    - "find pkg/cli/workflows -name 'test-*.md' -type f"
    - "ls pkg/cli/workflows/"
    - "grep -rn 'safe-outputs:' pkg/cli/workflows/*.md"
    - "grep -n 'yaml:.*' pkg/workflow/compiler_types.go"
    - "cat pkg/workflow/compiler_types.go"
    - "cat pkg/workflow/safe_outputs_validation_config.go"
    - "cat pkg/workflow/js/safe_outputs_tools.json"
    - "cat pkg/parser/schemas/main_workflow_schema.json"
    - "cat pkg/cli/workflows/*.md"
    - "git status"
    - "git diff --name-only"
    - "python3 *"
  edit:

safe-outputs:
  create-pull-request:
    expires: 3d
    title-prefix: "[safe-output-integrator] "
    labels: [safe-outputs, testing, automation]
    draft: false
  noop:

timeout-minutes: 20

imports:
  - shared/reporting-otlp.md

features:
  mcp-cli: true
  copilot-requests: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Safe Output Integrator

You are the **Daily Safe Output Integrator** — an automated agent that ensures all safe-output types are covered by test workflows and compiler tests. Your mission is to maintain full test coverage for every safe-output type supported by the gh-aw compiler.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run Date**: $(date +%Y-%m-%d)
- **Workspace**: ${{ github.workspace }}

## Background

The gh-aw compiler supports many safe-output types (e.g., `create-issue`, `add-comment`, `update-project`). Each type needs:
1. A **test workflow** in `pkg/cli/workflows/` — a markdown file that demonstrates usage and serves as a compilation fixture
2. A **Go compiler test** — a test function in `pkg/workflow/compiler_safe_outputs_config_test.go` that verifies the type compiles correctly

Your job is to detect any safe-output types that lack test coverage and create the missing artifacts.

## Phase 1: Discover All Safe-Output Types

Extract the complete list of safe-output YAML keys from the `SafeOutputsConfig` struct in `pkg/workflow/compiler_types.go`.

```bash
grep -n 'yaml:"[a-z-]*,omitempty"' pkg/workflow/compiler_types.go
```

The keys you're looking for are the YAML field names in the `SafeOutputsConfig` struct (lines ~443-490). These are the authoritative list of supported safe-output types.

**Extract and record**: Every `yaml:"<key>,omitempty"` found in `SafeOutputsConfig`. Skip internal/special fields like `jobs`, `github-app`, `env`, `github-token`, `allowed-domains`, `allowed-github-references`, `staged`, and `threat-detection` — those are configuration, not output types.

The **testable safe-output types** are:
- `create-issue`
- `create-discussion`
- `update-discussion`
- `close-discussion`
- `close-issue`
- `close-pull-request`
- `mark-pull-request-as-ready-for-review`
- `add-comment`
- `create-pull-request`
- `create-pull-request-review-comment`
- `submit-pull-request-review`
- `reply-to-pull-request-review-comment`
- `resolve-pull-request-review-thread`
- `create-code-scanning-alerts`
- `autofix-code-scanning-alert`
- `add-labels`
- `remove-labels`
- `add-reviewer`
- `assign-milestone`
- `assign-to-agent`
- `assign-to-user`
- `unassign-from-user`
- `update-issue`
- `update-pull-request`
- `push-to-pull-request-branch`
- `upload-asset`
- `update-release`
- `create-agent-session`
- `update-project`
- `create-project`
- `create-project-status-update`
- `link-sub-issue`
- `hide-comment`
- `set-issue-type`
- `dispatch-workflow`
- `call-workflow`
- `missing-tool`
- `missing-data`
- `noop`

## Phase 2: Scan Existing Test Workflows

List all test workflow files in `pkg/cli/workflows/`:

```bash
find pkg/cli/workflows -name 'test-*.md' -type f | sort
```

For each safe-output type, check if any test workflow uses it:

```bash
grep -rn 'safe-outputs:' pkg/cli/workflows/*.md
```

For each safe-output key (e.g., `create-issue`), a workflow "covers" it if the key appears after a `safe-outputs:` block in any `.md` file in `pkg/cli/workflows/`. Use this Python script for precise detection:

```python
import re, os, glob

workflows_dir = 'pkg/cli/workflows'
files = glob.glob(os.path.join(workflows_dir, '*.md'))

safe_output_types = [
    'create-issue', 'create-discussion', 'update-discussion', 'close-discussion',
    'close-issue', 'close-pull-request', 'mark-pull-request-as-ready-for-review',
    'add-comment', 'create-pull-request', 'create-pull-request-review-comment',
    'submit-pull-request-review', 'reply-to-pull-request-review-comment',
    'resolve-pull-request-review-thread', 'create-code-scanning-alerts',
    'autofix-code-scanning-alert', 'add-labels', 'remove-labels', 'add-reviewer',
    'assign-milestone', 'assign-to-agent', 'assign-to-user', 'unassign-from-user',
    'update-issue', 'update-pull-request', 'push-to-pull-request-branch',
    'upload-asset', 'update-release', 'create-agent-session', 'update-project',
    'create-project', 'create-project-status-update', 'link-sub-issue',
    'hide-comment', 'set-issue-type', 'dispatch-workflow', 'call-workflow',
    'missing-tool', 'missing-data', 'noop'
]

covered = {}
for sotype in safe_output_types:
    covered[sotype] = []

for f in sorted(files):
    with open(f) as fp:
        content = fp.read()
    # Only check for safe-output keys appearing in frontmatter (between --- markers)
    # Extract frontmatter
    parts = content.split('---', 2)
    if len(parts) < 3:
        continue
    frontmatter = parts[1]
    basename = os.path.basename(f)
    for sotype in safe_output_types:
        if (sotype + ':') in frontmatter or (sotype + '\n') in frontmatter:
            covered[sotype].append(basename)

print('COVERED:')
for k, v in covered.items():
    if v:
        print(f'  {k}: {v[0]} (+{len(v)-1} more)' if len(v) > 1 else f'  {k}: {v[0]}')

print()
print('NOT COVERED (need test workflows):')
missing = []
for k, v in covered.items():
    if not v:
        print(f'  {k}')
        missing.append(k)
print(f'Total missing: {len(missing)}')
```

Run the script:
```bash
python3 /tmp/check_coverage.py
```

(Save the script content to `/tmp/check_coverage.py` first using the edit tool, then run it.)

## Phase 3: Scan Existing Go Tests

Check whether the Go test file `pkg/workflow/compiler_safe_outputs_config_test.go` already has test cases for each safe-output type. Use the underscore form (e.g., `create_issue` for `create-issue`).

```bash
grep -n '"[a-z_]+ config"' pkg/workflow/compiler_safe_outputs_config_test.go
```

Record which types have Go test cases and which don't.

## Phase 4: Create Missing Test Workflow Files

For each safe-output type that lacks a test workflow, create a file in `pkg/cli/workflows/` named `test-copilot-<type>.md` (unless a file with that name already exists for a different reason).

**File naming convention**: `test-copilot-<safe-output-type>.md`
- For `update-pull-request` → `test-copilot-update-pull-request.md`
- For `create-discussion` → `test-copilot-create-discussion.md`
- etc.

**Template for test workflow files**:

```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  <type-key>:
    max: <sensible-default-from-validation-config>
    <relevant-options-for-this-type>
---

# Test Copilot <Type Name>

Test the `<type_underscore>` safe output type with the Copilot engine.

## Task

<Clear, minimal instructions for the agent to exercise this safe-output type.>

Output results in JSONL format using the `<type_underscore>` tool.
```

**Permissions guidance by type**:
- Types related to issues/PRs/discussions: `issues: read`, `pull-requests: read`
- Types related to code scanning: `security-events: read`
- Types related to projects: `projects: read` (if supported)
- Types related to releases/assets: `contents: read`
- Types related to workflow dispatch: `actions: read`
- When in doubt, use `contents: read`

**Configuration guidance** — consult `pkg/workflow/safe_outputs_validation_config.go` for each type's `DefaultMax` and fields. Here are tailored templates for each missing type:

### `update-pull-request`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  update-pull-request:
    max: 1
---

# Test Copilot Update Pull Request

Test the `update_pull_request` safe output type with the Copilot engine.

## Task

Update pull request #1 with a new title "Updated PR Title" and body "This PR body was updated by the test workflow."

Output results in JSONL format using the `update_pull_request` tool.
```

### `submit-pull-request-review`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  submit-pull-request-review:
    max: 1
---

# Test Copilot Submit Pull Request Review

Test the `submit_pull_request_review` safe output type with the Copilot engine.

## Task

Submit a COMMENT review on pull request #1 with the body "This is a test review comment submitted by the automated test workflow."

Output results in JSONL format using the `submit_pull_request_review` tool.
```

### `reply-to-pull-request-review-comment`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  reply-to-pull-request-review-comment:
    max: 1
---

# Test Copilot Reply to Pull Request Review Comment

Test the `reply_to_pull_request_review_comment` safe output type with the Copilot engine.

## Task

Reply to pull request review comment #1 with the body "Thank you for the review comment. This is an automated test reply."

Output results in JSONL format using the `reply_to_pull_request_review_comment` tool.
```

### `resolve-pull-request-review-thread`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
engine: copilot
safe-outputs:
  resolve-pull-request-review-thread:
    max: 5
---

# Test Copilot Resolve Pull Request Review Thread

Test the `resolve_pull_request_review_thread` safe output type with the Copilot engine.

## Task

Resolve the pull request review thread with thread ID "PRRT_test123". This indicates the discussion in the thread has been addressed.

Output results in JSONL format using the `resolve_pull_request_review_thread` tool.
```

### `create-discussion`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
engine: copilot
safe-outputs:
  create-discussion:
    max: 1
---

# Test Copilot Create Discussion

Test the `create_discussion` safe output type with the Copilot engine.

## Task

Create a new GitHub discussion with:
- Title: "Test Discussion from Copilot"
- Body: "This discussion was created automatically by the Copilot test workflow to verify the create_discussion safe output type works correctly."

Output results in JSONL format using the `create_discussion` tool.
```

### `close-issue`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  close-issue:
    max: 1
---

# Test Copilot Close Issue

Test the `close_issue` safe output type with the Copilot engine.

## Task

Close issue #1 with a reason of "completed" and a comment "Closing this issue as it has been resolved."

Output results in JSONL format using the `close_issue` tool.
```

### `create-code-scanning-alerts`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  security-events: read
engine: copilot
safe-outputs:
  create-code-scanning-alerts:
    driver: "Test Scanner"
    max: 3
timeout-minutes: 5
---

# Test Copilot Create Code Scanning Alerts

Test the `create_code_scanning_alert` safe output type with the Copilot engine.

## Task

Create a code scanning alert with the following details:
- **rule_id**: "TEST001"
- **rule_description**: "Test security rule for automated testing"
- **message**: "Found a potential test vulnerability"
- **path**: "src/test.js"
- **start_line**: 42
- **severity**: "warning"

Output results in JSONL format using the `create_code_scanning_alert` tool.
```

### `link-sub-issue`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  link-sub-issue:
    max: 5
---

# Test Copilot Link Sub-Issue

Test the `link_sub_issue` safe output type with the Copilot engine.

## Task

Link issue #2 as a sub-issue of issue #1. This establishes a parent-child relationship between the two issues.

Output results in JSONL format using the `link_sub_issue` tool.
```

### `update-project`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  update-project:
    max: 5
---

# Test Copilot Update Project

Test the `update_project` safe output type with the Copilot engine.

## Task

Add issue #1 to a GitHub Project V2. Set the status field to "In Progress" for the added item.

Output results in JSONL format using the `update_project` tool.
```

### `create-project`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  create-project:
    max: 1
---

# Test Copilot Create Project

Test the `create_project` safe output type with the Copilot engine.

## Task

Create a new GitHub Project V2 with:
- Title: "Test Project from Copilot"
- Description: "This project was created automatically by the Copilot test workflow."

Output results in JSONL format using the `create_project` tool.
```

### `create-project-status-update`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  create-project-status-update:
    max: 1
---

# Test Copilot Create Project Status Update

Test the `create_project_status_update` safe output type with the Copilot engine.

## Task

Create a status update for a GitHub Project V2. Set the status to "ON_TRACK" with a body message "All tasks are progressing as planned. No blockers identified."

Output results in JSONL format using the `create_project_status_update` tool.
```

### `remove-labels`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  remove-labels:
    max: 5
---

# Test Copilot Remove Labels

Test the `remove_labels` safe output type with the Copilot engine.

## Task

Remove the label "bug" from issue #1.

Output results in JSONL format using the `remove_labels` tool.
```

### `missing-data`
```markdown
---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  missing-data:
    max: 5
timeout-minutes: 5
---

# Test Missing Data Safe Output

Test the `missing_data` safe output functionality.

Report missing data with transparency messages:
- "Required issue number not found in the workflow trigger context"
- "Expected pull request branch name but no PR was associated with this run"
- "Configuration file 'config.json' not found in the repository root"

Output as JSONL format using the `missing_data` tool.
```

## Phase 5: Add Go Compiler Tests

For each safe-output type that lacks a Go test case in `pkg/workflow/compiler_safe_outputs_config_test.go`, add a new test case to the `TestAddHandlerManagerConfigEnvVar` function.

**Pattern to follow** (based on existing test cases):

```go
{
    name: "<type> config",
    safeOutputs: &SafeOutputsConfig{
        <FieldName>: &<ConfigType>{
            BaseSafeOutputConfig: BaseSafeOutputConfig{
                Max: strPtr("<default-max>"),
            },
            // Add type-specific fields as needed
        },
    },
    checkContains: []string{
        "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
    },
    checkJSON:    true,
    expectedKeys: []string{"<type_underscore>"},
},
```

Use the Go struct field names from `pkg/workflow/compiler_types.go` to find the correct field name (e.g., `UpdatePullRequests` for `update-pull-request`).

Use the type name in underscore form for `expectedKeys` (e.g., `update_pull_request` for `update-pull-request`).

**Reference**: Look at the existing test cases for `create_issue`, `add_comment`, and `create_discussion` in `pkg/workflow/compiler_safe_outputs_config_test.go` to understand the exact pattern.

**For each missing Go test, add a case like:**

- `update-pull-request` → field: `UpdatePullRequests`, type: `UpdatePullRequestsConfig`, key: `update_pull_request`
- `submit-pull-request-review` → field: `SubmitPullRequestReview`, type: `SubmitPullRequestReviewConfig`, key: `submit_pull_request_review`
- `reply-to-pull-request-review-comment` → field: `ReplyToPullRequestReviewComment`, type: `ReplyToPullRequestReviewCommentConfig`, key: `reply_to_pull_request_review_comment`
- `resolve-pull-request-review-thread` → field: `ResolvePullRequestReviewThread`, type: `ResolvePullRequestReviewThreadConfig`, key: `resolve_pull_request_review_thread`
- `create-discussion` → already has test case (verify)
- `close-issue` → field: `CloseIssues`, type: `CloseIssuesConfig`, key: `close_issue`
- `create-code-scanning-alerts` → field: `CreateCodeScanningAlerts`, type: `CreateCodeScanningAlertsConfig`, key: `create_code_scanning_alert`
- `link-sub-issue` → field: `LinkSubIssue`, type: `LinkSubIssueConfig`, key: `link_sub_issue`
- `update-project` → field: `UpdateProjects`, type: `UpdateProjectConfig`, key: `update_project`
- `create-project` → field: `CreateProjects`, type: `CreateProjectsConfig`, key: `create_project`
- `create-project-status-update` → field: `CreateProjectStatusUpdates`, type: `CreateProjectStatusUpdateConfig`, key: `create_project_status_update`
- `remove-labels` → field: `RemoveLabels`, type: `RemoveLabelsConfig`, key: `remove_labels`
- `missing-data` → field: `MissingData`, type: `MissingDataConfig`, key: `missing_data`

## Phase 6: Verify Changes

After creating files, verify:

1. **List created files**: Check all new files in `pkg/cli/workflows/`
2. **Verify git status**: Confirm only intended files were modified

```bash
git status
git diff --name-only
```

3. **Validate test workflow format**: Each test workflow should have:
   - Valid YAML frontmatter between `---` markers
   - An `on:` trigger (at least `workflow_dispatch`)
   - A `permissions:` section
   - An `engine:` field
   - A `safe-outputs:` section with the target type
   - A clear task description in the body

## Phase 7: Create PR or Report NOOP

### If files were created:

Create a pull request with all the new test workflows and Go tests.

The PR should:
- Target the `main` branch
- Include all new files in `pkg/cli/workflows/` and changes to `pkg/workflow/compiler_safe_outputs_config_test.go`
- Have a descriptive body listing which safe-output types were added

Use the `create_pull_request` tool:
```json
{
  "title": "Add missing safe-output test workflows and compiler tests",
  "body": "## Safe Output Test Coverage\n\nThis PR adds test workflows and Go compiler tests for safe-output types that lacked test coverage.\n\n### New Test Workflows Added\n\n[List each file created]\n\n### Go Tests Added\n\n[Describe test cases added to compiler_safe_outputs_config_test.go]\n\n### Summary\n\nAll [N] previously uncovered safe-output types now have test coverage.",
  "branch": "safe-output-integrator/$(date +%Y-%m-%d)",
  "labels": ["safe-outputs", "testing", "automation"]
}
```

### If no files were needed:

All safe-output types are already covered. Call the `noop` tool:
```json
{"noop": {"message": "No action needed: All safe-output types already have test workflows in pkg/cli/workflows/ and Go compiler tests in compiler_safe_outputs_config_test.go. Coverage is complete."}}
```

## Important Guidelines

### File Creation Rules

- **ONLY** create files for truly missing coverage
- **Do NOT** modify existing test workflows
- **Do NOT** create duplicate files (check if `test-copilot-<type>.md` already exists)
- **Do NOT** create files for configuration-only fields (`jobs`, `github-app`, `env`, `github-token`, `allowed-domains`, `staged`, `threat-detection`)
- If a type is covered by a file with a different name (e.g., `test-close-discussion.md` covers `close-discussion`), do NOT create another file for it

### Test Workflow Quality

Each test workflow must:
- Be a **minimal**, focused example of the safe-output type
- Use `engine: copilot` for consistency with other test-copilot-*.md files  
- Include only the permissions actually needed
- Have a clear task in the body that exercises the safe-output type
- Follow the `test-copilot-<type>.md` naming convention

### Go Test Quality

Each new Go test case must:
- Follow the existing pattern in `TestAddHandlerManagerConfigEnvVar`
- Use the correct field names from `SafeOutputsConfig`
- Include `expectedKeys` with the underscore-form type name
- Test that the config appears in the `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` env var

### Error Handling

- If you cannot find a file, use bash to verify paths
- If a safe-output type appears in `compiler_types.go` but not in `safe_outputs_validation_config.go`, include it in the test workflow but note the discrepancy
- If the Go test file structure differs from expected, read it carefully before making changes

## Success Criteria

A successful run:
- ✅ Identifies all safe-output types from `pkg/workflow/compiler_types.go`
- ✅ Cross-references with existing test workflows in `pkg/cli/workflows/`
- ✅ Cross-references with existing Go tests in `compiler_safe_outputs_config_test.go`
- ✅ Creates missing test workflow files with correct format
- ✅ Adds missing Go test cases to `compiler_safe_outputs_config_test.go`
- ✅ Creates a PR with all changes (or calls NOOP if everything is covered)
- ✅ PR body clearly describes what was added and why

**Important**: You MUST call either `create_pull_request` (if changes were made) or `noop` (if everything is already covered). Failing to call a safe-output tool will cause this workflow run to be marked as failed.
