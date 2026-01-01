# Documentation Campaign Usage Guide

This guide explains how to use the Documentation Quality & Completeness Campaign to track and manage all documentation-related work in the GitHub Agentic Workflows repository.

## Overview

The documentation campaign provides a structured way to:
- Track all documentation-related issues and pull requests
- Monitor documentation quality and completeness
- Measure progress toward documentation goals
- Coordinate documentation work across the team

## Quick Start

### 1. Setup (One-Time)

1. **Create a GitHub Project board** for documentation tracking:
   - Go to your organization's Projects page
   - Create a new Project (Table or Board view)
   - Copy the project URL

2. **Update the campaign configuration**:
   - Edit `.github/workflows/documentation-tasks.campaign.md`
   - Replace `https://github.com/orgs/githubnext/projects/TBD` with your actual project URL
   - Commit and push the changes

3. **Recompile the campaign**:
   ```bash
   gh aw compile .github/workflows/documentation-tasks.campaign.md
   ```

4. **Commit the updated orchestrator**:
   ```bash
   git add .github/workflows/documentation-tasks.campaign.g.*
   git commit -m "Update documentation campaign with project URL"
   git push
   ```

### 2. Using the Campaign

#### Tracking Documentation Work

Add the campaign tracker label to any documentation-related issue or PR:

```
Label: campaign:documentation-tasks
```

The campaign orchestrator will automatically:
- Discover labeled issues and PRs
- Add them to the project board
- Update their status
- Track metrics and KPIs

#### Creating Documentation Issues

When you identify documentation work that needs to be done:

1. Create a GitHub issue
2. Add a clear description of what documentation needs to be created or updated
3. Add the `campaign:documentation-tasks` label
4. Optionally add specific labels like:
   - `docs:setup` - Setup/getting started documentation
   - `docs:reference` - Reference documentation
   - `docs:guide` - How-to guides and tutorials
   - `docs:example` - Example workflows

#### Working on Documentation

1. Find work on the project board
2. Assign yourself to an issue
3. Create a pull request with your documentation changes
4. Add the `campaign:documentation-tasks` label to your PR
5. Link your PR to the issue

### 3. Monitoring Progress

#### Using the CLI

View campaign status:
```bash
gh aw campaign status documentation-tasks
```

View campaign details:
```bash
gh aw campaign documentation-tasks
```

View all campaigns:
```bash
gh aw campaign
```

#### Using the Project Board

The GitHub Project board provides a visual dashboard showing:
- Open documentation tasks
- In-progress work
- Completed items
- Overall campaign progress

#### Campaign Metrics

The campaign tracks three key performance indicators:

1. **Documentation Coverage** (Primary KPI)
   - Target: 95% of features have documentation
   - Timeframe: 90 days

2. **Broken Links Fixed** (Supporting KPI)
   - Target: 0 broken links
   - Timeframe: 30 days

3. **Documentation Freshness** (Supporting KPI)
   - Target: 90% of docs updated recently
   - Timeframe: 60 days

Metrics are stored in: `memory/campaigns/documentation-tasks/metrics/`

## Campaign Structure

### Files

- `.github/workflows/documentation-tasks.campaign.md` - Campaign specification (source of truth)
- `.github/workflows/documentation-tasks.campaign.g.md` - Generated orchestrator workflow
- `.github/workflows/documentation-tasks.campaign.g.lock.yml` - Compiled orchestrator
- `.github/workflows/documentation-tracker.md` - Placeholder tracking workflow
- `.github/workflows/documentation-tracker.lock.yml` - Compiled tracker workflow

### Labels

- `campaign:documentation-tasks` - Main campaign tracker label
- `documentation` - General documentation label (optional)
- Additional `docs:*` labels for categorization (optional)

### Memory Storage

Campaign state is persisted in:
- `memory/campaigns/documentation-tasks/cursor.json` - Progress cursor
- `memory/campaigns/documentation-tasks/metrics/YYYY-MM-DD.json` - Daily metrics snapshots

## Documentation Categories

The campaign tracks work across four main documentation areas:

### 1. User Documentation (`docs/src/content/docs/`)
- Setup guides
- Reference documentation
- How-to guides
- Examples
- Troubleshooting

### 2. Developer Documentation
- DEVGUIDE.md
- CONTRIBUTING.md
- AGENTS.md
- specs/ directory

### 3. Skills Documentation (`skills/`)
- Domain-specific guides
- Integration documentation
- Best practices

### 4. In-Code Documentation
- README files
- Code comments
- Workflow documentation

## Best Practices

### For Documentation Contributors

1. **Check the project board first** - See what work is available
2. **Follow the Diátaxis framework** - Understand whether you're writing a tutorial, how-to guide, reference, or explanation
3. **Use Astro Starlight syntax** - Follow the conventions in `skills/documentation/SKILL.md`
4. **Test your changes** - Build the docs locally to verify everything works
5. **Link issues and PRs** - Connect your work to the tracking system

### For Maintainers

1. **Create issues early** - When you see documentation gaps, create issues immediately
2. **Label consistently** - Always use `campaign:documentation-tasks` for tracking
3. **Review promptly** - Documentation PRs should be reviewed quickly
4. **Monitor the board** - Check for blockers and stalled work
5. **Celebrate progress** - Acknowledge documentation contributions

### For AI Agents

Future workflows can be added to automatically:
- Scan for documentation gaps
- Check for broken links
- Validate documentation structure
- Monitor freshness
- Create issues for missing docs

## Governance

The campaign follows these governance policies:

- **Max new items per run**: 10
- **Max discovery items per run**: 100
- **Max discovery pages per run**: 10
- **Max project updates per run**: 15
- **Max comments per run**: 15

These limits ensure sustainable progress without overwhelming the team or hitting API rate limits.

## Opt-Out

To exclude specific issues or PRs from campaign tracking, add one of these labels:
- `no-campaign` - Explicitly excluded from all campaigns
- `no-bot` - No automated actions

## Troubleshooting

### Campaign not tracking my issue

1. Verify the issue has the `campaign:documentation-tasks` label
2. Wait for the orchestrator to run (runs daily at 18:00 UTC)
3. Or manually trigger the orchestrator from GitHub Actions

### Project board not updating

1. Check that the project URL in the campaign spec is correct
2. Verify the `GH_AW_PROJECT_GITHUB_TOKEN` secret is configured
3. Check the orchestrator workflow logs for errors

### Campaign validation errors

Run validation to check for issues:
```bash
gh aw campaign validate
```

Fix any reported problems and recompile.

## Next Steps

1. **Create the project board** and update the campaign with the URL
2. **Start labeling existing documentation issues** with `campaign:documentation-tasks`
3. **Create new issues** for documentation work you've identified
4. **Run the orchestrator** manually to populate the project board
5. **Monitor progress** and adjust as needed

## Resources

- **Campaign Specification**: `.github/workflows/documentation-tasks.campaign.md`
- **Documentation Skill**: `skills/documentation/SKILL.md`
- **Campaign Guide**: `docs/src/content/docs/guides/campaigns.md`
- **Diátaxis Framework**: https://diataxis.fr/
- **Astro Starlight**: https://starlight.astro.build/

---

For questions or issues with the campaign, create an issue with the `campaign:documentation-tasks` label.
