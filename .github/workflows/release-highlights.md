---
name: Release
description: Build, test, and release gh-aw extension, then generate and prepend release highlights
on:
  push:
    tags:
      - 'v*.*.*'
  workflow_dispatch:
    inputs:
      release_tag:
        description: "Release tag to update (leave empty for latest release)"
        required: false
        type: string
permissions:
  contents: read
  pull-requests: read
  actions: read
  issues: read
roles:
  - admin
  - maintainer
engine: copilot
timeout-minutes: 20
network:
  allowed:
    - defaults
    - node
  firewall: true
tools:
  bash:
    - "*"
  edit:
safe-outputs:
  update-release:
jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write
      id-token: write
      attestations: write
    outputs:
      release_id: ${{ steps.get_release.outputs.release_id }}
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0
          
      - name: Release with gh-extension-precompile
        uses: cli/gh-extension-precompile@v2
        with:
          go_version_file: go.mod
          build_script_override: scripts/build-release.sh

      - name: Get release ID
        id: get_release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          RELEASE_TAG="${GITHUB_REF#refs/tags/}"
          echo "Getting release ID for tag: $RELEASE_TAG"
          RELEASE_ID=$(gh release view "$RELEASE_TAG" --json databaseId --jq '.databaseId')
          echo "release_id=$RELEASE_ID" >> $GITHUB_OUTPUT
          echo "✓ Release ID: $RELEASE_ID"
steps:
  - name: Setup environment and fetch release data
    run: |
      set -e
      mkdir -p /tmp/gh-aw/release-data
      
      # Use the release ID from the release job
      echo "Release ID from release job: ${{ needs.release.outputs.release_id }}"
      
      # Determine which release to analyze
      if [ "${{ github.event_name }}" = "push" ]; then
        if [[ ! "$GITHUB_REF" == refs/tags/* ]]; then
          echo "Error: Push event triggered but GITHUB_REF is not a tag: $GITHUB_REF"
          exit 1
        fi
        RELEASE_TAG="${GITHUB_REF#refs/tags/}"
        echo "Processing release from push event: $RELEASE_TAG"
      elif [ -n "${{ github.event.inputs.release_tag }}" ]; then
        RELEASE_TAG="${{ github.event.inputs.release_tag }}"
        echo "Processing release from workflow input: $RELEASE_TAG"
      else
        # Get latest release tag
        RELEASE_TAG=$(gh release list --limit 1 --json tagName --jq '.[0].tagName')
        echo "Processing latest release: $RELEASE_TAG"
      fi
      
      echo "RELEASE_TAG=$RELEASE_TAG" >> $GITHUB_ENV
      
      # Get the current release information
      gh release view "$RELEASE_TAG" --json name,tagName,createdAt,publishedAt,url,body > /tmp/gh-aw/release-data/current_release.json
      echo "✓ Fetched current release information"
      
      # Get the previous release to determine the range
      PREV_RELEASE_TAG=$(gh release list --limit 2 --json tagName --jq '.[1].tagName // empty')
      
      if [ -z "$PREV_RELEASE_TAG" ]; then
        echo "No previous release found. This appears to be the first release."
        echo "PREV_RELEASE_TAG=" >> $GITHUB_ENV
        touch /tmp/gh-aw/release-data/pull_requests.json
        echo "[]" > /tmp/gh-aw/release-data/pull_requests.json
      else
        echo "Previous release: $PREV_RELEASE_TAG"
        echo "PREV_RELEASE_TAG=$PREV_RELEASE_TAG" >> $GITHUB_ENV
        
        # Get commits between releases
        echo "Fetching commits between $PREV_RELEASE_TAG and $RELEASE_TAG..."
        git fetch --unshallow 2>/dev/null || git fetch --depth=1000
        
        # Get all merged PRs between the two releases
        echo "Fetching pull requests merged between releases..."
        PREV_PUBLISHED_AT=$(gh release view "$PREV_RELEASE_TAG" --json publishedAt --jq .publishedAt)
        CURR_PUBLISHED_AT=$(gh release view "$RELEASE_TAG" --json publishedAt --jq .publishedAt)
        gh pr list \
          --state merged \
          --limit 1000 \
          --json number,title,author,labels,mergedAt,url,body \
          --jq "[.[] | select(.mergedAt >= \"$PREV_PUBLISHED_AT\" and .mergedAt <= \"$CURR_PUBLISHED_AT\")]" \
          > /tmp/gh-aw/release-data/pull_requests.json
        
        PR_COUNT=$(jq length /tmp/gh-aw/release-data/pull_requests.json)
        echo "✓ Fetched $PR_COUNT pull requests"
      fi
      
      # Get the CHANGELOG.md content around this version
      if [ -f "CHANGELOG.md" ]; then
        cp CHANGELOG.md /tmp/gh-aw/release-data/CHANGELOG.md
        echo "✓ Copied CHANGELOG.md for reference"
      fi
      
      # List documentation files for linking
      find docs -type f -name "*.md" 2>/dev/null > /tmp/gh-aw/release-data/docs_files.txt || echo "No docs directory found"
      
      echo "✓ Setup complete. Data available in /tmp/gh-aw/release-data/"
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Release Highlights Generator

Generate an engaging release highlights summary for **${{ github.repository }}** release `${RELEASE_TAG}`.

**Release ID**: ${{ needs.release.outputs.release_id }}

## Data Available

All data is pre-fetched in `/tmp/gh-aw/release-data/`:
- `current_release.json` - Release metadata (tag, name, dates, existing body)
- `pull_requests.json` - PRs merged between `${PREV_RELEASE_TAG}` and `${RELEASE_TAG}` (empty array if first release)
- `CHANGELOG.md` - Full changelog for context (if exists)
- `docs_files.txt` - Available documentation files for linking

## Output Requirements

Create a **"🌟 Release Highlights"** section that:
- Is concise and scannable (users grasp key changes in 30 seconds)
- Uses professional, enthusiastic tone (not overly casual)
- Categorizes changes logically (features, fixes, docs, breaking changes)
- Links to relevant documentation where helpful
- Focuses on user impact (why changes matter, not just what changed)

## Workflow

### 1. Load Data

```bash
# View release metadata
cat /tmp/gh-aw/release-data/current_release.json | jq

# List PRs (empty if first release)
cat /tmp/gh-aw/release-data/pull_requests.json | jq -r '.[] | "- #\(.number): \(.title) by @\(.author.login)"'

# Check CHANGELOG context
head -100 /tmp/gh-aw/release-data/CHANGELOG.md 2>/dev/null || echo "No CHANGELOG"

# View available docs
cat /tmp/gh-aw/release-data/docs_files.txt
```

### 2. Categorize & Prioritize

Group PRs by category (omit categories with no items):
- **✨ New Features** - User-facing capabilities
- **🐛 Bug Fixes** - Issue resolutions
- **⚡ Performance** - Speed/efficiency improvements
- **📚 Documentation** - Guide/reference updates
- **⚠️ Breaking Changes** - Requires user action (ALWAYS list first if present)
- **🔧 Internal** - Refactoring, dependencies (usually omit from highlights)

### 3. Write Highlights

Structure:
```markdown
## 🌟 Release Highlights

[1-2 sentence summary of the release theme/focus]

### ⚠️ Breaking Changes
[If any - list FIRST with migration guidance]

### ✨ What's New
[Top 3-5 features with user benefit, link docs when relevant]

### 🐛 Bug Fixes & Improvements
[Notable fixes - focus on user impact]

### 📚 Documentation
[Only if significant doc additions/improvements]

---
For complete details, see [CHANGELOG](https://github.com/githubnext/gh-aw/blob/main/CHANGELOG.md).
```

**Writing Guidelines:**
- Lead with benefits: "GitHub MCP now supports remote mode" not "Added remote mode"
- Be specific: "Reduced compilation time by 40%" not "Faster compilation"
- Skip internal changes unless they have user impact
- Use docs links: `[Learn more](https://githubnext.github.io/gh-aw/path/)`
- Keep breaking changes prominent with action items

### 4. Handle Special Cases

**First Release** (no `${PREV_RELEASE_TAG}`):
```markdown
## 🎉 First Release

Welcome to the inaugural release! This version includes [core capabilities].

### Key Features
[List primary features with brief descriptions]
```

**Maintenance Release** (no user-facing changes):
```markdown
## 🔧 Maintenance Release

Dependency updates and internal improvements to keep things running smoothly.
```

## Output Format

**CRITICAL**: You MUST call the `update_release` tool to update the release with the generated highlights:

```javascript
update_release({
  tag: "${RELEASE_TAG}",
  operation: "prepend",
  body: "## 🌟 Release Highlights\n\n[Your complete markdown highlights here]"
})
```

**Required Parameters:**
- `tag` - Release tag from `${RELEASE_TAG}` environment variable (e.g., "v0.30.2")
- `operation` - Must be `"prepend"` to add before existing notes
- `body` - Complete markdown content (include all formatting, emojis, links)

**WARNING**: If you don't call the `update_release` tool, the release notes will NOT be updated!

**Documentation Base URLs:**
- User docs: `https://githubnext.github.io/gh-aw/`
- Reference: `https://githubnext.github.io/gh-aw/reference/`
- Setup: `https://githubnext.github.io/gh-aw/setup/`

Verify paths exist in `docs_files.txt` before linking.
