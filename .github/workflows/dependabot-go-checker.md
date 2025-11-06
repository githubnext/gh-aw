---
on:
  schedule:
    # Run every other business day: Monday, Wednesday, Friday at 9 AM UTC
    - cron: "0 9 * * 1,3,5"
  workflow_dispatch:

timeout_minutes: 20

permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  security-events: read

network: defaults

safe-outputs:
  create-issue:
    title-prefix: "[Dependabot] "
    labels: [dependencies, go]
    max: 5

tools:
  github:
    toolsets: [default, dependabot]
  web-fetch:
  bash: [":*"]

---
# Dependabot Go Module Dependency Checker

## Objective
Check for available Go module updates using Dependabot, evaluate their safety, and create issues for safe updates.

## Current Context
- **Repository**: ${{ github.repository }}
- **Go Module File**: `go.mod` in repository root

## Your Tasks

### Phase 1: Check Dependabot Alerts
1. Use the Dependabot toolset to check for available dependency updates for the `go.mod` file
2. Retrieve the list of alerts and update recommendations from Dependabot
3. For each potential update, gather:
   - Current version and proposed version
   - Type of update (patch, minor, major)
   - Security vulnerability information (if any)
   - Changelog or release notes (if available via web-fetch)

### Phase 2: Evaluate Update Safety
For each dependency update, evaluate:

**Safe Updates** (create issues for these):
- Patch version updates (e.g., 1.2.3 -> 1.2.4)
- Minor version updates with no breaking changes indicated in changelog
- Security updates that fix vulnerabilities
- Updates where the changelog explicitly states backward compatibility

**Unsafe/Uncertain Updates** (skip these):
- Major version updates (e.g., 1.x -> 2.x)
- Updates with breaking changes mentioned in changelog
- Updates where changelog indicates API changes
- Updates with insufficient documentation

### Phase 3: Create Task Issues
For each safe update identified, create an issue with:

**Title**: Short description of the update (e.g., "Update github.com/spf13/cobra from v1.9.1 to v1.9.2")

**Body** should include:
- **Summary**: Brief description of what needs to be updated
- **Current Version**: The version currently in go.mod
- **Proposed Version**: The version to update to
- **Update Type**: Patch/Minor/Major
- **Safety Assessment**: Why this update is considered safe
- **Changes**: Summary of changes from changelog or release notes
- **Links**: 
  - Link to the Dependabot alert (if applicable)
  - Link to the package repository
  - Link to release notes or changelog
- **Recommended Action**: Command to update (e.g., `go get -u github.com/package@v1.2.4`)
- **Testing Notes**: Any specific areas to test after applying the update

## Important Notes
- Do NOT apply updates directly - only create issues describing what should be updated
- Only create issues for updates you deem safe based on the evaluation criteria
- If no safe updates are found, exit without creating any issues
- Limit to a maximum of 5 issues per run to avoid overwhelming the repository
- For security-related updates, clearly indicate the vulnerability being fixed
- Be conservative: when in doubt about breaking changes, skip the update

## Example Issue Format

```markdown
## Summary
Update `github.com/spf13/cobra` dependency from v1.9.1 to v1.9.2

## Current State
- **Package**: github.com/spf13/cobra
- **Current Version**: v1.9.1
- **Proposed Version**: v1.9.2
- **Update Type**: Patch

## Safety Assessment
✅ **Safe to update**
- Patch version update (1.9.1 -> 1.9.2)
- No breaking changes mentioned in release notes
- Bug fixes and minor improvements only

## Changes
- Fixed issue with command completion
- Improved error messages
- Documentation updates

## Links
- [Release Notes](https://github.com/spf13/cobra/releases/tag/v1.9.2)
- [Package Repository](https://github.com/spf13/cobra)
- [Go Package](https://pkg.go.dev/github.com/spf13/cobra@v1.9.2)

## Recommended Action
```bash
go get -u github.com/spf13/cobra@v1.9.2
go mod tidy
```

## Testing Notes
- Run all tests: `make test`
- Verify CLI commands still work correctly
- Check for any deprecation warnings
```
