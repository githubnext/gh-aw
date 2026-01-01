# Campaign Infrastructure Validation

This directory contains the infrastructure validation script for GitHub Agentic Workflows campaigns.

## Validation Script

### `validate-campaign-infrastructure.sh`

Validates that all required infrastructure is in place for campaign execution.

**Usage:**
```bash
./scripts/validate-campaign-infrastructure.sh
```

**What it validates:**

1. **Campaign Orchestrators** (6 checks)
   - Orchestrator markdown files exist
   - Orchestrator lock files are compiled
   - Orchestrators are scheduled for 18:00 UTC daily

2. **Worker Workflows** (7 checks)
   - All worker workflow files exist
   - All worker workflows are compiled to lock files
   - Project 67: 6 workflows (daily-doc-updater, docs-noob-tester, daily-multi-device-docs-tester, unbloat-docs, developer-docs-consolidator, technical-doc-writer)
   - Project 64: 1 workflow (daily-file-diet)

3. **Memory Configuration** (6 checks)
   - Campaign orchestrators configured for memory/campaigns branch
   - Campaign IDs properly set
   - Memory branches exist in repository
   - File glob patterns correctly configured

4. **Metrics Infrastructure** (5 checks)
   - Metrics collector workflow exists and compiled
   - Metrics collector scheduled for daily execution
   - Repo-memory configured for metrics storage
   - Proper file paths configured

5. **Project Board Access** (2 checks + 2 warnings)
   - GitHub CLI available
   - Tests project access when GH_TOKEN is set
   - Validates safe-output token references

6. **Safe Output Configuration** (4 checks)
   - Both campaigns configured for update-project
   - Both campaigns reference GH_AW_PROJECT_GITHUB_TOKEN secret
   - Rate limits properly configured

**Output:**

The script provides color-coded output:
- ✓ Green: Check passed
- ⚠ Yellow: Warning (non-critical)
- ✗ Red: Check failed

**Exit codes:**
- `0`: All critical validations passed
- `1`: One or more validations failed

**Example output:**
```
========================================
Validation Summary
========================================

Passed:  29
Warnings: 2
Failed:  0

✓ All critical validations passed!
⚠ Some warnings present - review above
```

## Related Documentation

- **Validation Report:** `docs/campaign-infrastructure-validation-report.md`
- **Campaign System:** See `.github/workflows/*.campaign.g.md` files
- **Memory Documentation:** `docs/src/content/docs/reference/memory.md`

## Campaigns Validated

Currently validates infrastructure for:
- **Project 67:** Documentation Quality & Maintenance Campaign
- **Project 64:** Go File Size Reduction Campaign

## Adding New Campaigns

To add validation for new campaigns:

1. Add orchestrator file check to `validate_campaign_orchestrators()`
2. Add worker workflows to `validate_worker_workflows()`
3. Add memory configuration checks to `validate_memory_configuration()`
4. Update documentation in this README

## Troubleshooting

**Warning: "GH_TOKEN not set"**
- This is expected in local development
- In GitHub Actions, set `GH_TOKEN: ${{ github.token }}`
- For project access, workflows use `GH_AW_PROJECT_GITHUB_TOKEN` secret

**Failed: "orchestrator schedule NOT FOUND"**
- Run `make recompile` to regenerate lock files
- Ensure orchestrator markdown has proper schedule in frontmatter

**Warning: "memory branch does NOT exist yet"**
- This is normal for new campaigns
- Memory branches are auto-created on first workflow run
- Use `create-orphan: true` in repo-memory config (default)

## CI Integration

The validation script can be integrated into CI workflows:

```yaml
- name: Validate campaign infrastructure
  run: ./scripts/validate-campaign-infrastructure.sh
```

The script exits with code 0 on success, making it suitable for CI gates.
