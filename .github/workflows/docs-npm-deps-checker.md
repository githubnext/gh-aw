---
description: Checks for npm dependency updates in docs/package.json and creates issues using a three-tier categorization strategy
on:
  schedule: daily
  workflow_dispatch:

timeout-minutes: 20

permissions:
  contents: read
  issues: read
  pull-requests: read

network:
  allowed:
    - defaults
    - node

safe-outputs:
  close-issue:
    required-title-prefix: "[docs-deps]"
    target: "*"
    max: 20
  create-issue:
    expires: 2d
    title-prefix: "[docs-deps]"
    labels: [dependencies, docs, npm]
    max: 10
    group: true

tools:
  github:
  web-fetch:
  bash: [":*"]

imports:
  - shared/reporting.md
---
# Docs npm Dependency Checker

## Objective

Close any existing open dependency update issues with the `[docs-deps]` prefix, then check for available npm dependency updates for the `docs/package.json` file, categorize them by safety level, and create issues using a three-tier strategy: group safe patch updates into a single consolidated issue, create individual issues for potentially problematic updates, and skip major version updates.

## Current Context

- **Repository**: ${{ github.repository }}
- **npm Manifest**: `docs/package.json`

## Your Tasks

### Phase 0: Close Existing Dependency Issues (CRITICAL FIRST STEP)

**Before performing any analysis**, you must close existing open issues with the `[docs-deps]` title prefix to prevent duplicate dependency update issues.

Use the GitHub API tools to:
1. Search for open issues with title starting with `[docs-deps]` in repository ${{ github.repository }}
2. Close each found issue with a comment explaining that a new dependency check is being performed
3. Use the `close_issue` safe output to close these issues with reason "not_planned"

**Important**: The `close-issue` safe output is configured with:
- `required-title-prefix: "[docs-deps]"` - Only issues starting with this prefix will be closed
- `target: "*"` - Can close any issue by number (not just triggering issue)
- `max: 20` - Can close up to 20 issues in one run

To close an existing dependency issue, emit:
```
close_issue(issue_number=123, body="Closing this issue as a new dependency check is being performed.")
```

**Do not proceed to Phase 1 until all existing `[docs-deps]` issues are closed.**

### Phase 1: Discover Available npm Updates

1. Run `npm outdated --json` in the `docs/` directory to list all packages with available updates:
   ```bash
   cd docs && npm outdated --json 2>/dev/null || true
   ```
   The command exits with code 1 when outdated packages exist, so `|| true` is required to capture the output.

2. Parse the JSON output. Each entry has the shape:
   ```json
   {
     "package-name": {
       "current": "1.2.3",
       "wanted": "1.2.4",
       "latest": "2.0.0",
       "location": "...",
       "type": "dependencies"
     }
   }
   ```
   - `current` — installed version
   - `wanted` — highest version satisfying the semver range in `package.json`
   - `latest` — latest published version on npm

3. For each package with `wanted != current` or `latest != current`, record:
   - Package name
   - Current version
   - Wanted version (range-compatible update)
   - Latest version (absolute latest on npm)
   - Dependency type (`dependencies` or `devDependencies`)

4. For each outdated package, fetch its npm registry metadata to retrieve the changelog or release notes:
   ```
   https://registry.npmjs.org/<package-name>
   ```
   Look at the `versions` object to understand what changed between `current` and `wanted`/`latest`.

### Phase 2: Categorize Updates (Three-Tier Strategy)

For each dependency update, use the **wanted** version (the semver-range-compatible version) as the proposed update target, unless the wanted version equals the current version — in that case use the **latest** version.

Categorize each update into one of three tiers:

**Category A: Safe Patches** (group into ONE consolidated issue):
- Patch version updates ONLY (e.g., `1.2.3` → `1.2.4`)
- Single-version increments (not multi-version jumps like `1.2.3` → `1.2.7`)
- Bug fixes and stability improvements only (no new features)
- No breaking changes or behavior modifications
- Security patches that only fix vulnerabilities without API changes
- Explicitly backward compatible per changelog or release notes

**Category B: Potentially Problematic** (create INDIVIDUAL issues):
- Minor version updates (e.g., `1.2.x` → `1.3.x`)
- Multi-version jumps in patch versions (e.g., `1.2.3` → `1.2.7`)
- Updates with new features or API additions
- Updates with behavior changes mentioned in release notes
- Updates that require configuration or code changes
- Security updates that include API changes
- Any update where safety is uncertain

**Category C: Skip** (do NOT create any issues):
- Major version updates (e.g., `1.x.x` → `2.x.x`)
- Updates with breaking changes explicitly mentioned
- Updates requiring significant refactoring
- Updates with insufficient documentation to assess safety

### Phase 3: Create Issues Based on Categorization

**For Category A (Safe Patches)**: Create ONE consolidated issue grouping all safe patch updates together.

**For Category B (Potentially Problematic)**: Create INDIVIDUAL issues for each update.

**For Category C**: Do not create any issues.

#### Consolidated Issue Format (Category A)

**Title**: "Update safe patch npm dependencies in docs/ (N updates)"

**Body** should include:
- **Summary**: Brief overview of grouped safe patch updates
- **Updates Table**: Table listing all safe patch updates with columns:
  - Package name
  - Current version
  - Proposed version
  - Dependency type (`dependencies` or `devDependencies`)
  - Key changes
- **Safety Assessment**: Why all these updates are considered safe patches
- **Recommended Action**: Single command block to apply all updates at once:
  ```bash
  cd docs
  npm install <package1>@<version1> <package2>@<version2>
  ```
- **Testing Notes**: General testing guidance for the docs site

#### Individual Issue Format (Category B)

**Title**: Short description of the specific update (e.g., "Update astro from 6.0.8 to 6.1.0 in docs/")

**Body** should include:
- **Summary**: Brief description of what needs to be updated
- **Current Version**: The version currently installed
- **Proposed Version**: The version to update to
- **Dependency Type**: `dependencies` or `devDependencies`
- **Update Type**: Minor / Multi-version patch jump
- **Why Separate Issue**: Clear explanation of why this update needs individual review
- **Safety Assessment**: Detailed assessment of risks and considerations
- **Changes**: Summary of changes from changelog or release notes
- **Links**:
  - Link to the npm package page: `https://www.npmjs.com/package/<name>`
  - Link to the GitHub repository or release notes (if available via registry metadata)
- **Recommended Action**:
  ```bash
  cd docs
  npm install <package>@<version>
  ```
- **Testing Notes**: Specific areas to test after applying the update (e.g., run `npm run build`, check docs site renders correctly)

## Important Notes

- Do NOT apply updates directly — only create issues describing what should be updated
- Use three-tier categorization: Group Category A (safe patches), individual issues for Category B (potentially problematic), skip Category C (major versions)
- Category A updates should be grouped into ONE consolidated issue with a table format
- Category B updates should each get their own issue with a "Why Separate Issue" explanation
- If no outdated packages are found, or all updates fall into Category C, exit without creating any issues
- Limit to a maximum of 10 issues per run (up to 1 grouped issue for Category A + remaining individual issues for Category B)
- For security-related updates, clearly indicate the vulnerability being fixed
- Be conservative: when in doubt about breaking changes or behavior modifications, categorize as Category B (individual issue) or Category C (skip)
- Only true single-version patch updates with bug fixes belong in Category A

## Example Issue Formats

### Example 1: Consolidated Issue for Safe Patches (Category A)

```markdown
## Summary
This issue groups together multiple safe patch updates for `docs/package.json` that can be applied together. All updates are single-version patch increments with bug fixes only and no breaking changes.

## Updates

| Package | Current | Proposed | Type | Key Changes |
|---------|---------|----------|------|-------------|
| mermaid | 11.13.0 | 11.13.1 | dependencies | Bug fix in diagram rendering |
| sharp | 0.34.5 | 0.34.6 | dependencies | Build compatibility fix |

## Safety Assessment
✅ **All updates are safe patches**
- All are single-version patch increments
- Only bug fixes and stability improvements, no new features
- No breaking changes or behavior modifications

## Recommended Action
Apply all updates together:

```bash
cd docs
npm install mermaid@11.13.1 sharp@0.34.6
npm run build
```

## Testing Notes
- Run `cd docs && npm run build` to verify the docs site still builds
- Check that diagrams, images, and static assets render correctly
- Run `cd docs && npm test` if Playwright tests are configured
```

### Example 2: Individual Issue for Minor Update (Category B)

```markdown
## Summary
Update `astro` dependency in `docs/package.json` from `6.0.8` to `6.1.0`.

## Current State
- **Package**: astro
- **Current Version**: 6.0.8
- **Proposed Version**: 6.1.0
- **Dependency Type**: dependencies
- **Update Type**: Minor

## Why Separate Issue
⚠️ **Minor version update with new features**
- This is a minor version update (6.0.8 → 6.1.0)
- May introduce new APIs or behavior changes
- The docs build configuration may need review
- Needs individual testing before merging

## Safety Assessment
⚠️ **Requires careful review**
- Minor version update indicates new features
- Review release notes for any deprecations or behavior changes
- Test the full docs build after applying

## Links
- [npm: astro](https://www.npmjs.com/package/astro)
- [Astro GitHub Releases](https://github.com/withastro/astro/releases)

## Recommended Action
```bash
cd docs
npm install astro@6.1.0
npm run build
```

## Testing Notes
- Run `cd docs && npm run build` and verify no errors
- Spot-check pages in the generated `dist/` output
- Run `cd docs && npm test` for Playwright smoke tests
```

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
