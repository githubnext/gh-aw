---
description: Checks for Go module and NPM dependency updates and analyzes Dependabot PRs for compatibility and breaking changes
on:
  schedule:
    # Run every other business day: Monday, Wednesday, Friday at 9 AM UTC
    - cron: "0 9 * * 1,3,5"
  workflow_dispatch:

timeout-minutes: 20

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
# Dependabot Dependency Checker

## Objective
Check for available Go module and NPM dependency updates using Dependabot, evaluate their safety, and create issues for safe updates.

## Current Context
- **Repository**: ${{ github.repository }}
- **Go Module File**: `go.mod` in repository root
- **NPM Packages**: Check for `@playwright/mcp` updates in constants.go

## Your Tasks

### Phase 1: Check Dependabot Alerts
1. Use the Dependabot toolset to check for available dependency updates for the `go.mod` file
2. Retrieve the list of alerts and update recommendations from Dependabot
3. For each potential update, gather:
   - Current version and proposed version
   - Type of update (patch, minor, major)
   - Security vulnerability information (if any)
   - Changelog or release notes (if available via web-fetch)

### Phase 1.5: Check Playwright NPM Package Updates
1. Check the current `@playwright/mcp` version in `pkg/constants/constants.go`:
   - Look for `DefaultPlaywrightVersion` constant
   - Extract the current version number
2. Check for newer versions on NPM:
   - Use web-fetch to query `https://registry.npmjs.org/@playwright/mcp`
   - Compare the latest version with the current version in constants.go
   - Get release information and changelog if available
3. Evaluate the update:
   - Check if it's a patch, minor, or major version update
   - Look for breaking changes in release notes
   - Consider security fixes and improvements

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

### Phase 2.5: Repository Detection
Before creating issues, determine the actual source repository for each Go module:

**GitHub Packages** (`github.com/*`):
- Remove version suffixes like `/v2`, `/v3`, `/v4` from the module path
- Example: `github.com/spf13/cobra/v2` → repository is `github.com/spf13/cobra`
- Repository URL: `https://github.com/{owner}/{repo}`
- Release URL: `https://github.com/{owner}/{repo}/releases/tag/{version}`

**golang.org/x Packages**:
- These are NOT hosted on GitHub
- Repository: `https://go.googlesource.com/{package-name}`
- Example: `golang.org/x/sys` → `https://go.googlesource.com/sys`
- Commit history: `https://go.googlesource.com/{package-name}/+log`
- Do NOT link to GitHub release pages (they don't exist)

**Other Packages**:
- Use `pkg.go.dev/{module-path}` to find the repository URL
- Look for the "Repository" or "Source" link on the package page
- Use the discovered repository for links

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
  - Link to the actual source repository (detected per Repository Detection rules)
  - Link to release notes or changelog (if available)
  - For GitHub packages: link to release page
  - For golang.org/x packages: link to commit history instead
- **Recommended Action**: Command to update (e.g., `go get -u github.com/package@v1.2.4`)
- **Testing Notes**: Any specific areas to test after applying the update

## Important Notes
- Do NOT apply updates directly - only create issues describing what should be updated
- Only create issues for updates you deem safe based on the evaluation criteria
- If no safe updates are found, exit without creating any issues
- Limit to a maximum of 5 issues per run to avoid overwhelming the repository
- For security-related updates, clearly indicate the vulnerability being fixed
- Be conservative: when in doubt about breaking changes, skip the update

**CRITICAL - Repository Detection**:
- **Never assume all Go packages are on GitHub**
- **golang.org/x packages** are hosted at `go.googlesource.com`, NOT GitHub
- **Always remove version suffixes** (e.g., `/v2`, `/v3`) when constructing repository URLs for GitHub packages
- **Use pkg.go.dev** to find the actual repository for packages not on GitHub or golang.org/x
- **Do NOT create GitHub release links** for packages that don't use GitHub releases

## Example Issue Formats

### Example 1: GitHub Package Update

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

### Example 2: golang.org/x Package Update

```markdown
## Summary
Update `golang.org/x/sys` dependency from v0.15.0 to v0.16.0

## Current State
- **Package**: golang.org/x/sys
- **Current Version**: v0.15.0
- **Proposed Version**: v0.16.0
- **Update Type**: Minor

## Safety Assessment
✅ **Safe to update**
- Minor version update (0.15.0 -> 0.16.0)
- No breaking changes in commit history
- Standard system call updates and bug fixes

## Changes
- Added support for new Linux syscalls
- Fixed Windows file system handling
- Performance improvements for Unix systems
- Bug fixes in signal handling

## Links
- [Source Repository](https://go.googlesource.com/sys)
- [Commit History](https://go.googlesource.com/sys/+log)
- [Go Package](https://pkg.go.dev/golang.org/x/sys@v0.16.0)

**Note**: This package is hosted on Google's Git (go.googlesource.com), not GitHub. There are no GitHub release pages.

## Recommended Action
```bash
go get -u golang.org/x/sys@v0.16.0
go mod tidy
```

## Testing Notes
- Run all tests: `make test`
- Test system-specific functionality
- Verify cross-platform compatibility
```

### Example 3: Playwright NPM Package Update

```markdown
## Summary
Update `@playwright/mcp` package from 1.56.1 to 1.57.0

## Current State
- **Package**: @playwright/mcp
- **Current Version**: 1.56.1 (in pkg/constants/constants.go - DefaultPlaywrightVersion)
- **Proposed Version**: 1.57.0
- **Update Type**: Minor

## Safety Assessment
✅ **Safe to update**
- Minor version update (1.56.1 -> 1.57.0)
- No breaking changes mentioned in release notes
- Includes bug fixes and new features
- Backward compatible

## Changes
- Added support for new Playwright features
- Improved MCP server stability
- Bug fixes in browser automation
- Performance improvements

## Links
- [NPM Package](https://www.npmjs.com/package/@playwright/mcp)
- [Release Notes](https://github.com/microsoft/playwright/releases/tag/v1.57.0)
- [Source Repository](https://github.com/microsoft/playwright)

## Recommended Action
```bash
# Update the constant in pkg/constants/constants.go
# Change: const DefaultPlaywrightVersion = "1.56.1"
# To:     const DefaultPlaywrightVersion = "1.57.0"

# Then run tests to verify
make test-unit
```

## Testing Notes
- Run unit tests: `make test-unit`
- Verify Playwright MCP configuration generation
- Test browser automation workflows with playwright tool
- Check that version is correctly used in compiled workflows
```
