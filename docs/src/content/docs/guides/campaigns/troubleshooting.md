---
title: Troubleshooting
description: Common campaign issues and how to resolve them
---

This guide helps you diagnose and fix common issues with campaigns.

## Campaign creation issues

### Label applied but nothing happens

**Symptoms:** You applied the `create-agentic-campaign` label to an issue, but no workflow runs.

**Possible causes:**
1. Campaign generator workflow not installed in repository
2. Workflow disabled in GitHub Actions settings
3. Workflow permissions insufficient

**Solution:**
```bash
# Check if campaign generator exists
ls .github/workflows/campaign-generator.md

# Check GitHub Actions tab for workflow run
# Look for "Campaign Generator" workflow

# If workflow missing, install it:
gh aw add githubnext/agentics/campaign-generator
gh aw compile
```

### Campaign generation fails

**Symptoms:** Campaign generator workflow runs but fails with errors.

**Common error messages:**

**"Failed to create project board"**
- **Cause:** Insufficient permissions to create projects
- **Solution:** Ensure workflow has `projects: write` permission or create project board manually and provide URL in issue body

**"No workflows discovered"**
- **Cause:** Repository has no agentic workflows
- **Solution:** This is expected for new repositories. The generated campaign will have an empty workflows list. Add workflows later by editing the spec.

**"Compilation failed"**
- **Cause:** Generated spec has validation errors
- **Solution:** Check the PR for compilation error details. Edit the spec to fix issues before merging.

## Discovery issues

### Items not being discovered

**Symptoms:** Workflow creates issues/PRs, but orchestrator doesn't find them.

**Diagnostic steps:**

1. **Check tracker label matches:**
   ```yaml
   # In campaign spec
   tracker-label: "campaign:framework-upgrade"
   
   # In worker workflow safe-outputs
   create-issue:
     labels:
       - "campaign:framework-upgrade"  # Must match exactly
   ```

2. **Verify discovery limits:**
   ```yaml
   governance:
     max-discovery-items-per-run: 100  # May be too low
     max-discovery-pages-per-run: 10    # May be too low
   ```

3. **Check for opt-out labels:**
   ```yaml
   governance:
     opt-out-labels: ["no-campaign", "no-bot"]
   ```
   Items with these labels are intentionally excluded.

4. **Review orchestrator logs:**
   ```bash
   # Download latest run logs
   gh run view --log
   
   # Look for discovery step output
   # Check "discovered items" count in logs
   ```

**Solution:** Adjust governance limits or fix label mismatches.

### Discovery finds too many items

**Symptoms:** Orchestrator hits discovery limits, doesn't process all items.

**Solution:** This is expected behavior. Campaigns process items incrementally:

```yaml
# Increase discovery limits if needed
governance:
  max-discovery-items-per-run: 200  # Up from 100
  max-discovery-pages-per-run: 20   # Up from 10
```

> [!NOTE]
> Higher limits use more API quota. Start conservative and increase gradually.

## Project board issues

### Items not appearing on board

**Symptoms:** Discovery finds items, but they don't appear on project board.

**Diagnostic steps:**

1. **Check max-project-updates limit:**
   ```yaml
   governance:
     max-project-updates-per-run: 10  # Items processed per run
   ```
   If discovery finds 50 items but limit is 10, only first 10 are processed. Remaining items process on next run.

2. **Verify project-url is correct:**
   ```yaml
   project-url: "https://github.com/orgs/ORG/projects/123"
   # Must be exact URL, check org vs user projects
   ```

3. **Check token permissions:**
   ```yaml
   # If using custom token
   project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
   # Token must have projects:write permission
   ```

4. **Review orchestrator errors:**
   Look for `update-project` failures in orchestrator logs.

**Solution:** Increase limits, fix URL, or adjust permissions.

### Project fields not updating

**Symptoms:** Items appear on board but custom fields are empty or incorrect.

**Common causes:**

1. **Field name mismatch:**
   ```yaml
   # In orchestrator
   fields:
     worker_workflow: "my-worker"  # Underscores
   
   # But field on board is named:
   # "Worker/Workflow"  # Slash, not underscore
   ```
   Field names are case-sensitive and exact.

2. **Field type mismatch:**
   ```yaml
   fields:
     priority: 5  # Number
   # But field expects: "High", "Medium", "Low" (strings)
   ```

3. **Field doesn't exist:**
   Orchestrator tries to set field that wasn't created on project board.

**Solution:** 
- Match field names exactly (use underscores: `worker_workflow`, `start_date`)
- Use correct value types for each field
- Verify fields exist on project board in GitHub UI

## Orchestrator execution issues

### Orchestrator not running on schedule

**Symptoms:** Orchestrator doesn't run at scheduled time (6 PM UTC daily).

**Diagnostic steps:**

1. **Check workflow is enabled:**
   - Go to Actions → Workflows
   - Find campaign orchestrator
   - Verify not disabled

2. **Check state field:**
   ```yaml
   state: active  # Must be active for scheduled runs
   # state: paused  # Would prevent scheduled runs
   ```

3. **Review recent runs:**
   - Check for any error patterns
   - Look for rate limiting or permissions issues

**Solution:** Enable workflow, set state to `active`, or manually trigger run.

### Orchestrator fails with rate limiting

**Symptoms:** Error messages about API rate limits.

**Solution:** Reduce API usage:

```yaml
governance:
  max-discovery-items-per-run: 50    # Down from 100
  max-discovery-pages-per-run: 5     # Down from 10
  max-project-updates-per-run: 5     # Down from 10
  max-comments-per-run: 5            # Down from 10
```

**Long-term:** Campaigns will take longer but avoid rate limits.

### Orchestrator fails with permission errors

**Symptoms:** "403 Forbidden" or "Resource not accessible" errors.

**Diagnostic steps:**

1. **Check repository permissions:**
   ```yaml
   # In .campaign.lock.yml
   permissions:
     contents: read
     issues: write
     pull-requests: write
     projects: write  # Required for project updates
   ```

2. **Check allowed-repos:**
   ```yaml
   allowed-repos:
     - "myorg/repo-a"
     - "myorg/repo-b"
   # Must include all repos where campaign operates
   ```

3. **Verify project-github-token:**
   If using custom token, check it has sufficient permissions in GitHub settings.

**Solution:** Adjust permissions, expand allowed-repos, or fix token.

## Workflow coordination issues

### Worker workflows not executing

**Symptoms:** Campaign spec lists workflows, but orchestrator doesn't run them.

**Possible causes:**

1. **execute-workflows not enabled:**
   ```yaml
   # Add this to enable orchestration mode
   execute-workflows: true
   ```

2. **Workflow files don't exist:**
   ```yaml
   workflows:
     - framework-scanner  # Must exist at .github/workflows/framework-scanner.md
   ```

3. **Worker workflow missing workflow_dispatch:**
   ```yaml
   # Worker must have:
   on:
     workflow_dispatch:  # Required for orchestrator to trigger it
   ```

**Solution:** Enable orchestration, create missing workflows, or add `workflow_dispatch` trigger.

### Worker workflows running duplicate times

**Symptoms:** Workflows run both on their schedule and when orchestrator triggers them.

**Cause:** Worker has multiple triggers active:

```yaml
# WRONG - Both triggers active
on:
  schedule: daily           # Worker runs independently
  workflow_dispatch:        # Campaign also triggers it
```

**Solution:** Choose one coordination mode:

**Option 1: Campaign orchestrates (recommended)**
```yaml
# Worker workflow
on:
  # schedule: daily      # DISABLE original schedule
  workflow_dispatch:      # Only campaign triggers

# Campaign spec
execute-workflows: true   # Campaign orchestrates
workflows:
  - my-worker
```

**Option 2: Independent execution**
```yaml
# Worker workflow  
on:
  schedule: daily         # Worker runs independently
  workflow_dispatch:      # Optional manual trigger

# Campaign spec
# execute-workflows: false  # Campaign only tracks, doesn't orchestrate
tracker-label: "campaign:my-campaign"  # Discover outputs via label
```

> [!CAUTION]
> Never have both campaign orchestration AND independent scheduling active simultaneously.

## Performance issues

### Campaign runs too slowly

**Symptoms:** Campaign takes many runs to process all items.

**Causes:**
- Governance limits too low
- Too many items to process
- Workers creating items faster than orchestrator can process

**Solutions:**

1. **Increase processing limits:**
   ```yaml
   governance:
     max-project-updates-per-run: 20  # Up from 10
   ```

2. **Increase orchestrator frequency:**
   ```yaml
   # Edit .campaign.lock.yml manually
   on:
     schedule:
       - cron: "0 */6 * * *"  # Every 6 hours instead of daily
   ```

3. **Reduce item creation rate:**
   Adjust worker workflow schedules to create items more slowly.

### Campaign using too much API quota

**Symptoms:** Rate limiting errors, slow API responses.

**Solution:** Reduce API usage:

```yaml
governance:
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 5
```

Balance between speed and API usage based on your rate limit headroom.

## Debugging techniques

### Enable debug logging

Add to orchestrator's agent instructions (edit `.campaign.lock.yml`):

```yaml
# In the agent job
env:
  DEBUG: campaign:*
```

Then check workflow run logs for detailed debug output.

### Inspect discovery manifest

Discovery precomputation outputs a manifest:

```bash
# Download workflow artifacts after run
gh run download <run-id>

# Check discovery manifest
cat .gh-aw/campaign.discovery.json
```

Manifest shows:
- All discovered items
- Discovery budget usage
- Summary counts
- Cursor position

### Review safe-output logs

Safe-output operations are logged:

```bash
# In workflow run logs, look for:
# - update-project requests
# - create-project-status-update requests
# - Errors or warnings from safe-output system
```

### Test manually

Run individual steps manually to isolate issues:

```bash
# 1. Manually trigger orchestrator
gh workflow run <campaign-id>.campaign.lock.yml

# 2. Check discovery step separately
# View discovery step in workflow run logs

# 3. Test project updates
# Create test issue with campaign label
# Verify orchestrator picks it up on next run
```

## Getting help

If you've tried troubleshooting steps and still have issues:

1. **Check documentation:**
   - [Campaign Specs](/gh-aw/guides/campaigns/specs/) for configuration reference
   - [Campaign Flow](/gh-aw/guides/campaigns/flow/) for execution details
   - [Technical Overview](/gh-aw/guides/campaigns/technical-overview/) for architecture

2. **Review examples:**
   - Check [example campaigns](/gh-aw/examples/campaigns/) for working patterns
   - Compare your spec to working examples

3. **Ask for help:**
   - Create GitHub issue with:
     - Campaign spec (redact sensitive info)
     - Orchestrator logs
     - Description of expected vs actual behavior
   - Include workflow run URLs for investigation
