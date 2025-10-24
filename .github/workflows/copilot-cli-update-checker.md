---
on:
  schedule:
    - cron: "0 16 * * 1"  # Weekly on Monday at 4 PM UTC
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine:
  id: claude
  max-turns: 50
network: 
  allowed: [defaults, node, "api.github.com", "registry.npmjs.org"]
imports:
  - shared/jqschema.md
tools:
  web-fetch:
  cache-memory: true
  bash:
    - "cat *"
    - "ls *"
    - "grep *"
    - "git *"
    - "npm view *"
    - "npm list *"
    - "npm install *"
    - "copilot --help"
    - "copilot --version"
  edit:
safe-outputs:
  create-issue:
    title-prefix: "[copilot-cli] "
    labels: [automation, dependencies, copilot]
timeout_minutes: 15
strict: true
---

# Copilot CLI Update Checker

Check for GitHub Copilot CLI updates and provide detailed update guidance.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

## Objective

Monitor the GitHub Copilot CLI (`@github/copilot`) for new releases and analyze what needs to be reviewed when updating the version in this repository.

## Process

### 1. Cache Check (Efficiency First)

Before starting:
1. Check cache-memory at `/tmp/gh-aw/cache-memory/` for previous version checks
2. Look for cached file: `copilot-cli-last-check.json` containing:
   - Last check timestamp
   - Last known version
   - Last known help output
3. If cached data exists and is recent (< 7 days), load it to compare
4. If no version changes detected, exit early with success

### 2. Version Detection

**Current Version**: Check `./pkg/constants/constants.go` for `DefaultCopilotVersion`

**Latest Version**: Use npm registry to get latest version:
```bash
npm view @github/copilot version
```

**Version Comparison**:
- If current version matches latest, save timestamp to cache and exit successfully
- If update available, proceed with detailed analysis

### 3. Release Analysis

For the version update (or multiple intermediate versions if skipped), analyze:

#### A. Release Metadata
- **Version Number**: Semantic versioning (major.minor.patch)
- **Release Date**: When was it published
- **Time Since Previous**: How frequently are updates released
- **Changelog URL**: `https://www.npmjs.com/package/@github/copilot/v/<version>`

#### B. Changes Classification

Categorize changes into:
- **🔴 Breaking Changes**: API changes, removed features, incompatible changes
- **🟢 New Features**: New commands, flags, capabilities, MCP improvements
- **🔵 Bug Fixes**: Fixed issues, stability improvements
- **🟡 Security**: CVEs, security patches, vulnerability fixes
- **⚡ Performance**: Speed improvements, optimization
- **📚 Documentation**: Updated docs, improved help text
- **🔧 Dependencies**: Updated npm dependencies

#### C. CLI Command Discovery

**IMPORTANT**: Install the new version and compare command-line interface:

1. **Install New Version** (in isolated environment):
   ```bash
   npm install -g @github/copilot@<new-version>
   ```

2. **Extract Help Output**:
   ```bash
   copilot --help > /tmp/gh-aw/cache-memory/copilot-<new-version>-help.txt
   copilot --version
   ```

3. **Compare with Previous Version**:
   - Load cached help from previous version (if available)
   - Identify NEW commands or subcommands
   - Identify NEW flags or options
   - Identify DEPRECATED or REMOVED features
   - Note any changes in default behaviors
   - Check for new MCP-related features

4. **Save to Cache**:
   - Store new help output for future comparisons
   - Update `copilot-cli-last-check.json` with new version info

### 4. Impact Assessment

Evaluate impact on gh-aw integration:

#### Files to Review
1. **`pkg/constants/constants.go`**: Update `DefaultCopilotVersion`
2. **`pkg/workflow/copilot_engine.go`**: Core engine implementation
3. **`.github/instructions/copilot-cli.instructions.md`**: Documentation
4. **`pkg/workflow/js/parse_copilot_log.cjs`**: Log parsing
5. **Related test files**: Any test files that reference Copilot CLI

#### Integration Points to Check
- **MCP Configuration**: Does the update change MCP server support?
- **Authentication**: Any changes to token handling or auth flow?
- **Command-line Arguments**: Are our current args still valid?
- **Log Format**: Does log parsing need updates?
- **Environment Variables**: New or changed env var requirements?
- **Node.js Version**: Any Node.js version requirement changes?

#### Risk Assessment
Assign a risk level based on:
- **Low Risk**: Patch version, bug fixes only, no breaking changes
- **Medium Risk**: Minor version, new features but backward compatible
- **High Risk**: Major version, breaking changes, requires code updates

### 5. Testing Recommendations

Provide testing guidance:
- Which workflows should be tested with the new version
- Specific MCP server integrations to verify
- Authentication and permission scenarios to test
- Command-line arguments to validate

### 6. Update Process Documentation

If an update is available, create an issue via `safe-outputs.create-issue` with:

**Issue Template**:
```markdown
# GitHub Copilot CLI Update: v{old} → v{new}

## Summary
- **Current Version**: {old_version}
- **Latest Version**: {new_version}
- **Release Date**: {release_date}
- **Risk Level**: {Low|Medium|High}

## Release Information
- **NPM Package**: https://www.npmjs.com/package/@github/copilot/v/{new_version}
- **Time Since Last Update**: {days/weeks}
- **Intermediate Versions**: {list if multiple}

## Changes

### 🔴 Breaking Changes
{list or "None detected"}

### 🟢 New Features
{list}

### 🔵 Bug Fixes
{list}

### 🟡 Security Updates
{list or "None"}

### ⚡ Performance Improvements
{list or "None"}

### 🔧 CLI Changes
**New Commands/Flags**:
{list or "None detected"}

**Deprecated/Removed**:
{list or "None detected"}

**Changed Defaults**:
{list or "None detected"}

## Impact Assessment

### Affected Files
- [ ] `pkg/constants/constants.go` - Update version constant
- [ ] `pkg/workflow/copilot_engine.go` - Review for compatibility
- [ ] `.github/instructions/copilot-cli.instructions.md` - Update docs
- [ ] `pkg/workflow/js/parse_copilot_log.cjs` - Review log parsing
- [ ] Test files - Update version references

### Integration Points
- **MCP Support**: {impact description}
- **Authentication**: {impact description}
- **CLI Arguments**: {impact description}
- **Log Parsing**: {impact description}
- **Environment Variables**: {impact description}

### Testing Requirements
- [ ] Test with existing agentic workflows
- [ ] Verify MCP server integrations
- [ ] Validate authentication flow
- [ ] Check log parsing compatibility
- [ ] Run full test suite

## Update Steps

1. Update version in `pkg/constants/constants.go`:
   ```go
   const DefaultCopilotVersion = "{new_version}"
   ```

2. Review and update related code if needed

3. Update documentation if CLI features changed

4. Run validation:
   ```bash
   make build
   make test-unit
   make recompile
   make lint
   ```

5. Test with sample workflows

6. Commit changes

## Migration Notes
{specific migration steps if breaking changes exist}

## References
- NPM Package: https://www.npmjs.com/package/@github/copilot
- Version {new_version}: https://www.npmjs.com/package/@github/copilot/v/{new_version}
- Copilot CLI Docs: https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line
```

## Cache Management

Save the following to cache-memory for future runs:
- `copilot-cli-last-check.json`: Check timestamp and version info
- `copilot-{version}-help.txt`: Help output for version comparison
- `copilot-{version}-changelog.md`: Extracted changelog (if fetched)

## Exit Conditions

- ✅ **Success with No Update**: Current version is latest, cache updated
- ✅ **Success with Issue**: Update found, issue created with analysis
- ⚠️ **Partial Success**: Unable to fetch some data, issue created with available info
- ❌ **Failure**: Critical error (network down, npm registry unavailable)

## Error Handling

- Retry npm registry calls once after 30s delay
- Continue with partial data if changelog fetch fails
- Document what couldn't be analyzed in the issue
- Save progress to cache before exiting on errors

## Notes

- This workflow runs **weekly** to catch updates promptly
- Uses cache to avoid redundant work
- Focuses specifically on Copilot CLI (unlike general CLI version checker)
- Provides actionable update guidance tailored to gh-aw integration
- Creates issues only when updates are detected
