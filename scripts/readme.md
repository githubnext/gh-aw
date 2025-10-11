# Changeset CLI

A minimalistic implementation for managing version releases, inspired by `@changesets/cli`.

## Usage

```bash
# Preview next version from changesets (always read-only)
node scripts/changeset.js version
# Or use make target
make changeset-version

# Create release and update CHANGELOG
node scripts/changeset.js release
# Or use make target
make changeset-release

# Create specific release type
node scripts/changeset.js release patch
node scripts/changeset.js release minor
node scripts/changeset.js release major
```

## Commands

### `version`

The `version` command always operates in preview mode and never modifies files. It shows what the next version would be based on the changesets.

```bash
node scripts/changeset.js version
```

This command:
- Reads all changeset files from `.changeset/` directory
- Determines the appropriate version bump (major > minor > patch)
- Shows a preview of the CHANGELOG entry that would be added
- Never modifies any files

### `release [type]`

The `release` command creates an actual release by updating files.

```bash
node scripts/changeset.js release
# Or specify type explicitly
node scripts/changeset.js release minor
```

This command:
- Checks prerequisites (clean tree, main branch)
- **Prompts for confirmation** before proceeding with the release
- Updates `CHANGELOG.md` with the new version and changes (if changesets exist)
- Deletes processed changeset files
- Automatically commits the changes
- Creates and pushes a git tag for the release

**Patch releases without changesets:**
- Patch releases can be created without any changeset files
- Use `node scripts/changeset.js release patch` to create a patch release without changesets
- Non-patch releases (minor, major) require changeset files

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

## Prerequisites for Release

When running `node changeset.js release`, the script checks:

1. **Clean working tree**: All changes must be committed or stashed
2. **On main branch**: Must be on the `main` branch to create a release

Example error when not on main branch:
```bash
$ node scripts/changeset.js release
✗ Must be on 'main' branch to create a release (currently on 'feature-branch')
```

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
   node scripts/changeset.js version
   ```

3. **Create the release:**
   ```bash
   node scripts/changeset.js release
   ```
   
   This will:
   - Show the version that will be created
   - **Ask for confirmation** (y/N)
   - Update CHANGELOG.md (if changesets exist)
   - Delete changeset files (if any)
   - Commit the changes
   - Create a git tag
   - Push the tag to remote

4. **Patch releases without changesets:**
   ```bash
   # Create a patch release without any changesets
   node scripts/changeset.js release patch
   ```
   
   This is useful for:
   - Emergency hotfixes
   - Version bumps without feature changes
   - Infrastructure updates that don't affect functionality

## Features

- ✅ **Automatic Version Determination**: Analyzes all changesets and picks the highest priority bump type
- ✅ **CHANGELOG Generation**: Creates formatted entries with proper categorization (Breaking Changes, Features, Bug Fixes)
- ✅ **Git Integration**: Reads current version from git tags
- ✅ **Automated Git Operations**: Automatically commits, tags, and pushes releases
- ✅ **User Confirmation**: Prompts for confirmation before executing release operations
- ✅ **Patch Releases Without Changesets**: Allows patch releases without requiring changeset files
- ✅ **Safety First**: Requires explicit specification for major releases
- ✅ **Clean Workflow**: Deletes processed changesets after release
- ✅ **Zero Dependencies**: Pure Node.js implementation

## Requirements

- Node.js (any recent version)
- Git repository with semantic version tags (e.g., `v1.2.3`)
