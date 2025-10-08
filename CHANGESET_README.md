# Changeset CLI

A minimalistic implementation for managing version releases, inspired by `@changesets/cli`.

## Usage

```bash
# Preview next version from changesets
node changeset.js version

# Preview without modifying files (dry-run)
node changeset.js version --dry-run

# Create release and update CHANGELOG
node changeset.js release

# Preview release without modifying files (dry-run)
node changeset.js release --dry-run

# Create specific release type
node changeset.js release patch
node changeset.js release minor
node changeset.js release major

# Dry-run for specific release type
node changeset.js release minor --dry-run
```

## Dry-Run Mode

The `--dry-run` flag allows you to preview what changes would be made without actually modifying any files. This is useful for:

- Previewing the CHANGELOG entry before committing
- Checking which changeset files would be deleted
- Verifying the version bump is correct

Example output:
```bash
$ node changeset.js release --dry-run
ℹ 🔍 DRY RUN MODE - No files will be modified

ℹ Creating minor release: v0.15.0

ℹ Would add to CHANGELOG.md:
---
## v0.15.0 - 2025-10-08

### Features
- New feature description

### Bug Fixes
- Bug fix description
---

ℹ Would remove 3 changeset file(s):
  - .changeset/minor-feature.md
  - .changeset/patch-bugfix.md
```

## Changeset File Format

Changeset files are markdown files in `.changeset/` directory with YAML frontmatter:

```markdown
---
"gh-aw": patch
---

Brief description of the change
```

**Bump types:**
- `patch` - Bug fixes and minor changes (0.0.x)
- `minor` - New features, backward compatible (0.x.0)
- `major` - Breaking changes (x.0.0)

## Commands

### `version`

Analyzes changeset files and previews the next release:

```bash
node changeset.js version
```

This command:
- Reads all changeset files from `.changeset/` directory
- Determines the appropriate version bump (major > minor > patch)
- Generates or updates `CHANGELOG.md` with categorized changes
- Shows a preview of the version bump and changes

### `release [type]`

Creates a release by updating CHANGELOG and cleaning up changeset files:

```bash
node changeset.js release
```

Optional: Specify release type explicitly:

```bash
node changeset.js release patch
node changeset.js release minor
node changeset.js release major
```

This command:
- Updates `CHANGELOG.md` with the new version and changes
- Deletes processed changeset files
- Provides next steps for committing and tagging

**Safety Note:** Major releases must be explicitly specified. If changesets indicate a major bump but no type is provided, the command will fail with a safety error.

## Release Workflow

1. **Add changeset files** to `.changeset/` directory for each change:
   ```bash
   # Create a changeset file
   cat > .changeset/fix-bug.md << EOF
   ---
   "gh-aw": patch
   ---
   
   Fix critical bug in workflow compilation
   EOF
   ```

2. **Preview the release:**
   ```bash
   node changeset.js version
   ```

3. **Create the release:**
   ```bash
   node changeset.js release
   ```

4. **Review and commit:**
   ```bash
   # Review the updated CHANGELOG.md
   cat CHANGELOG.md
   
   # Commit the changes
   git add CHANGELOG.md .changeset/
   git commit -m "Release v0.15.0"
   ```

5. **Create and push tag:**
   ```bash
   git tag -a v0.15.0 -m "Release v0.15.0"
   git push origin main v0.15.0
   ```

## Features

- ✅ **Automatic Version Determination**: Analyzes all changesets and picks the highest priority bump type
- ✅ **CHANGELOG Generation**: Creates formatted entries with proper categorization (Breaking Changes, Features, Bug Fixes)
- ✅ **Git Integration**: Reads current version from git tags
- ✅ **Safety First**: Requires explicit specification for major releases
- ✅ **Clean Workflow**: Deletes processed changesets after release
- ✅ **Zero Dependencies**: Pure Node.js implementation

## Requirements

- Node.js (any recent version)
- Git repository with semantic version tags (e.g., `v1.2.3`)
