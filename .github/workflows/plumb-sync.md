---
description: Automatically runs plumb to keep spec, tests, and code in sync when a PR is ready for review or receives new commits — no human intervention required
on:
  pull_request:
    types: [ready_for_review, synchronize]
permissions:
  contents: read
  pull-requests: read
engine: copilot
checkout:
  fetch-depth: 0
network:
  allowed:
    - defaults
    - github
    - python
    - api.anthropic.com
secrets:
  ANTHROPIC_API_KEY:
    value: ${{ secrets.ANTHROPIC_API_KEY }}
    description: "Anthropic API key required by plumb for LLM-powered decision extraction"
tools:
  bash: ["*"]
  edit:
safe-outputs:
  add-comment:
    max: 1
    hide-older-comments: true
    discussions: false
    issues: false
  create-pull-request:
    expires: 7d
    title-prefix: "[plumb-sync] "
    labels: [spec-sync, automated]
    draft: false
    allowed-base-branches:
      - "*"
  messages:
    footer: "> 🔧 *Plumb sync by [{workflow_name}]({run_url})*{effective_tokens_suffix}{history_link}"
    run-started: "🪛 [{workflow_name}]({run_url}) is running plumb to sync spec and tests on this {event_type}..."
    run-success: "✅ [{workflow_name}]({run_url}) completed plumb analysis."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status} during plumb analysis."
timeout-minutes: 20
---

# Plumb Sync

You are an automated plumb agent. Your mission is to analyze the changes in this pull request using [plumb](https://github.com/dbreunig/plumb), automatically approve all detected design decisions, sync them back to the spec and tests, and report results — all without human intervention.

## Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **PR Title**: "${{ github.event.pull_request.title }}"
- **Base SHA**: ${{ github.event.pull_request.base.sha }}
- **Head SHA**: ${{ github.event.pull_request.head.sha }}

## Step 1: Check Plumb Initialization

First, verify that plumb has been initialized in this repository:

```bash
ls -la .plumb/ 2>/dev/null && cat .plumb/config.json || echo "NOT_INITIALIZED"
```

**If `.plumb/` does not exist** (output contains `NOT_INITIALIZED`), this repository has not yet been set up with plumb. Post a setup guide comment and stop:

```json
{
  "add-comment": {
    "body": "## 🔧 Plumb Not Initialized\n\nThis workflow detected that [plumb](https://github.com/dbreunig/plumb) has not been initialized in this repository.\n\nTo enable automated spec/test sync on every PR, run the following once in your repository:\n\n```bash\npip install plumb-dev\nplumb init\n```\n\nPlumb will ask for:\n1. Paths to your spec markdown file(s)\n2. Your test directory\n\nThen commit the generated `.plumb/` directory to version control. Once initialized, this workflow will automatically extract design decisions from every PR and keep your spec and tests in sync — no human review required."
  }
}
```

Then call noop:

```json
{"noop": {"message": "Plumb not initialized in this repository. Posted setup instructions on PR #${{ github.event.pull_request.number }}."}}
```

## Step 2: Install Plumb

Install plumb and verify the installation:

```bash
pip install plumb-dev --quiet 2>&1 | tail -5
plumb --version 2>/dev/null || plumb --help 2>&1 | head -3
```

## Step 3: Stage the PR Diff for Analysis

Plumb analyzes **staged changes** (git index). The repository is currently checked out at the PR's HEAD commit with all changes already committed. You must re-stage the PR's diff so plumb can analyze it.

```bash
# Record current HEAD for reference
echo "Currently at HEAD: $(git rev-parse HEAD)"

# Fetch enough history to have the base commit available
git fetch origin ${{ github.event.pull_request.base.sha }} 2>&1 || git fetch --unshallow 2>&1 || true

# Move working tree to base commit (detached HEAD)
git checkout --detach ${{ github.event.pull_request.base.sha }} 2>&1

# Apply the PR diff as staged (indexed) changes — simulates the PR being staged
git diff ${{ github.event.pull_request.base.sha }} ${{ github.event.pull_request.head.sha }} | git apply --index --allow-empty 2>&1
echo "Staging result: $?"

# Show what is staged
git diff --stat --cached
```

If the `git apply` fails, try the fallback:

```bash
# Fallback: manually stage changed files from PR HEAD
CHANGED_FILES=$(git diff --name-only ${{ github.event.pull_request.base.sha }} ${{ github.event.pull_request.head.sha }})
if [ -n "$CHANGED_FILES" ]; then
  git checkout ${{ github.event.pull_request.head.sha }} -- $CHANGED_FILES
  git add -A
  echo "Fallback staging complete"
  git diff --stat --cached
else
  echo "No changed files found between base and head"
fi
```

## Step 4: Extract Pending Decisions

Run plumb diff to preview what decisions it would extract from the staged changes:

```bash
cd $GITHUB_WORKSPACE
plumb diff 2>&1 | tee /tmp/plumb-diff-output.txt
echo "---plumb diff exit code: $?"
```

Read the output:

```bash
cat /tmp/plumb-diff-output.txt
```

**If plumb reports no decisions** (e.g., output is empty, contains "No staged changes", "No decisions found", "No pending decisions", or similar), then no spec/test sync is needed. Call noop:

```json
{"noop": {"message": "Plumb found no design decisions in PR #${{ github.event.pull_request.number }}. Spec and tests are already consistent."}}
```

## Step 5: Auto-Approve All Decisions

Automatically approve every pending decision without human review:

```bash
cd $GITHUB_WORKSPACE
plumb approve --all 2>&1 | tee /tmp/plumb-approve-output.txt
echo "---plumb approve exit code: $?"
cat /tmp/plumb-approve-output.txt
```

## Step 6: Sync Spec and Tests

Sync all approved decisions back to the spec and test files:

```bash
cd $GITHUB_WORKSPACE
plumb sync 2>&1 | tee /tmp/plumb-sync-output.txt
echo "---plumb sync exit code: $?"
cat /tmp/plumb-sync-output.txt
```

Check what files were modified by the sync:

```bash
git diff --name-only
```

Save the list of changed files:

```bash
git diff --name-only > /tmp/changed-files.txt
cat /tmp/changed-files.txt
```

## Step 7: Optional — Run Coverage Report

Run plumb coverage for a full spec/test/code alignment summary:

```bash
cd $GITHUB_WORKSPACE
plumb coverage 2>&1 | tee /tmp/plumb-coverage.txt || echo "(Coverage report unavailable)"
cat /tmp/plumb-coverage.txt
```

## Step 8: Post Summary Comment

Post a comment on PR #${{ github.event.pull_request.number }} summarizing the plumb analysis using `add-comment`. The comment body should be:

```markdown
## 🔧 Plumb Sync — Automated Spec/Test Alignment

Plumb analyzed the PR changes, extracted design decisions, auto-approved them all, and synced the spec and tests.

### Decisions Extracted

<details>
<summary>Decision analysis output</summary>

[paste contents of /tmp/plumb-diff-output.txt here]

</details>

### Auto-Approved

[paste contents of /tmp/plumb-approve-output.txt here]

### Synced Files

[if changed-files.txt is non-empty: list each file; otherwise write "No spec or test files required updates"]

### Coverage

<details>
<summary>Spec · Test · Code coverage</summary>

[paste contents of /tmp/plumb-coverage.txt here, or "(not available)" if plumb coverage was not run]

</details>
```

## Step 9: Create Companion PR with Synced Changes

If `changed-files.txt` is non-empty (i.e., plumb sync modified spec or test files):

1. Use the `edit` tool to write the updated content for each file listed in `changed-files.txt` — this registers the changes with the safe-output processor. Read each file's current content from the filesystem (the plumb sync has already modified them in-place) and pass it to `edit`.
2. Discover the PR's head branch name at runtime:
   ```bash
   gh pr view ${{ github.event.pull_request.number }} --repo ${{ github.repository }} --json headRefName --jq .headRefName
   ```
3. Call `create-pull-request` with:
   - **Title**: `Sync spec and tests for PR #${{ github.event.pull_request.number }}: ${{ github.event.pull_request.title }}`
   - **Body**: A concise description listing the files updated, the decisions synced, and a reference to PR #${{ github.event.pull_request.number }}
   - **Base**: the head branch name discovered in step 2 (so synced changes land directly in the same PR)

If `changed-files.txt` is empty, no companion PR is needed. Call noop:

```json
{"noop": {"message": "Plumb sync complete for PR #${{ github.event.pull_request.number }}. No spec or test files were modified — everything is already consistent."}}
```

## Important Notes

- All plumb commands must run from `$GITHUB_WORKSPACE` (the repository root)
- The `ANTHROPIC_API_KEY` environment variable is automatically available for plumb's LLM-powered analysis
- Plumb requires Python 3.10+ (the default GitHub Actions runner provides Python 3.12)
- The `.plumb/config.json` file defines which spec and test paths plumb manages
- If plumb exits non-zero during `approve --all` or `sync`, read the error and post it in the summary comment before stopping

{{#runtime-import shared/noop-reminder.md}}
