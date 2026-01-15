---
name: Release
description: Build, test, and release gh-aw extension, then generate and prepend release highlights
on:
  push:
    tags:
      - 'v*.*.*'
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
    - "githubnext.github.io"
sandbox:
  agent: awf  # Firewall enabled (migrated from network.firewall)
tools:
  bash:
    - "*"
  edit:
safe-outputs:
  update-release:
jobs:
  release:
    needs: ["activation"]
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write
      id-token: write
      attestations: write
    outputs:
      release_id: ${{ steps.get_release.outputs.release_id }}
      release_tag: ${{ steps.get_release.outputs.release_tag }}
    steps:
      - name: Checkout
        uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8 # v5.0.0
        with:
          fetch-depth: 0
          persist-credentials: false
          
      - name: Release with gh-extension-precompile
        uses: cli/gh-extension-precompile@6f13f31f798a93a6b08d3be0727120e9af35851f # v2.1.0
        with:
          go_version_file: go.mod
          build_script_override: scripts/build-release.sh

      - name: Upload checksums file
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          RELEASE_TAG="${GITHUB_REF#refs/tags/}"
          if [ -f "dist/checksums.txt" ]; then
            echo "Uploading checksums file to release: $RELEASE_TAG"
            gh release upload "$RELEASE_TAG" dist/checksums.txt --clobber
            echo "✓ Checksums file uploaded to release"
          else
            echo "Warning: checksums.txt not found in dist/"
          fi

      - name: Get release ID
        id: get_release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          RELEASE_TAG="${GITHUB_REF#refs/tags/}"
          echo "Getting release ID for tag: $RELEASE_TAG"
          RELEASE_ID=$(gh release view "$RELEASE_TAG" --json databaseId --jq '.databaseId')
          echo "release_id=$RELEASE_ID" >> "$GITHUB_OUTPUT"
          echo "release_tag=$RELEASE_TAG" >> "$GITHUB_OUTPUT"
          echo "✓ Release ID: $RELEASE_ID"
          echo "✓ Release Tag: $RELEASE_TAG"
  generate-sbom:
    needs: ["release"]
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8 # v5.0.0

      - name: Setup Go
        uses: actions/setup-go@4469467cea6daeb81c49688e3f738b3ea61cc4e1 # v6.0.0
        with:
          go-version-file: go.mod
          cache: false  # Disabled for release security - prevent cache poisoning attacks

      - name: Download Go modules
        run: go mod download

      - name: Generate SBOM (SPDX format)
        uses: anchore/sbom-action@fbfd9c6c0a5723f5b15376258af3142b3d6a83bb # v0.20.10
        with:
          artifact-name: sbom.spdx.json
          output-file: sbom.spdx.json
          format: spdx-json

      - name: Generate SBOM (CycloneDX format)
        uses: anchore/sbom-action@fbfd9c6c0a5723f5b15376258af3142b3d6a83bb # v0.20.10
        with:
          artifact-name: sbom.cdx.json
          output-file: sbom.cdx.json
          format: cyclonedx-json

      - name: Audit SBOM files for secrets
        run: |
          echo "Auditing SBOM files for potential secrets..."
          if grep -rE "GITHUB_TOKEN|SECRET|PASSWORD|API_KEY|PRIVATE_KEY" sbom.*.json; then
            echo "Error: Potential secrets found in SBOM files"
            exit 1
          fi
          echo "✓ No secrets detected in SBOM files"

      - name: Upload SBOM artifacts
        uses: actions/upload-artifact@b7c566a0745ede1831f8ca951aaab692e8d836c2 # v6.0.0
        with:
          name: sbom-artifacts
          path: |
            sbom.spdx.json
            sbom.cdx.json
          retention-days: 7  # Minimize exposure window

      - name: Attach SBOM to release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          RELEASE_TAG: ${{ needs.release.outputs.release_tag }}
        run: |
          echo "Attaching SBOM files to release: $RELEASE_TAG"
          gh release upload "$RELEASE_TAG" sbom.spdx.json sbom.cdx.json --clobber
          echo "✓ SBOM files attached to release"
  docker-image:
    needs: ["release"]
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
      id-token: write
      attestations: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8 # v5.0.0

      - name: Setup Docker Buildx
        uses: docker/setup-buildx-action@8d2750ceccfa2109d028e60fbdcf2e87b3ce84a2 # v3.12.0

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@5e57cd11039ae84fdace9dfebfd0ed0a3282deb0 # v3.6.0
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Download release artifacts
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          RELEASE_TAG: ${{ needs.release.outputs.release_tag }}
        run: |
          echo "Downloading release binaries..."
          mkdir -p dist
          gh release download "$RELEASE_TAG" --pattern "linux-*" --dir dist
          ls -lh dist/
          echo "✓ Release binaries downloaded"

      - name: Extract metadata for Docker
        id: meta
        uses: docker/metadata-action@c299e40ca79d9ee606ef6f4365af95e9a7ca7f9f # v5.10.0
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=sha,format=long
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Docker image (amd64)
        id: build
        uses: docker/build-push-action@8c6338f942d2d9576ac98c87becb29da981ca7e8 # v6
        with:
          context: .
          platforms: linux/amd64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            BINARY=dist/linux-amd64
          cache-from: type=gha
          cache-to: type=gha,mode=max

      - name: Generate SBOM for Docker image
        uses: anchore/sbom-action@fbfd9c6c0a5723f5b15376258af3142b3d6a83bb # v0.20.10
        with:
          image: ghcr.io/${{ github.repository }}:${{ needs.release.outputs.release_tag }}
          artifact-name: docker-sbom.spdx.json
          output-file: docker-sbom.spdx.json
          format: spdx-json

      - name: Attest Docker image
        uses: actions/attest-build-provenance@e8998f985e7ebc42bf28d5f01b12f7a9a44b30bb # v2.4.0
        with:
          subject-name: ghcr.io/${{ github.repository }}
          subject-digest: ${{ steps.build.outputs.digest }}
          push-to-registry: true
steps:
  - name: Setup environment and fetch release data
    env:
      RELEASE_ID: ${{ needs.release.outputs.release_id }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e
      mkdir -p /tmp/gh-aw/release-data
      
      # Use the release ID from the release job
      echo "Release ID from release job: $RELEASE_ID"
      
      # Get the release tag from the push event
      if [[ ! "$GITHUB_REF" == refs/tags/* ]]; then
        echo "Error: Push event triggered but GITHUB_REF is not a tag: $GITHUB_REF"
        exit 1
      fi
      RELEASE_TAG="${GITHUB_REF#refs/tags/}"
      echo "Processing release: $RELEASE_TAG"
      
      echo "RELEASE_TAG=$RELEASE_TAG" >> "$GITHUB_ENV"
      
      # Get the current release information
      gh release view "$RELEASE_TAG" --json name,tagName,createdAt,publishedAt,url,body > /tmp/gh-aw/release-data/current_release.json
      echo "✓ Fetched current release information"
      
      # Get the previous release to determine the range
      PREV_RELEASE_TAG=$(gh release list --limit 2 --json tagName --jq '.[1].tagName // empty')
      
      if [ -z "$PREV_RELEASE_TAG" ]; then
        echo "No previous release found. This appears to be the first release."
        echo "PREV_RELEASE_TAG=" >> "$GITHUB_ENV"
        touch /tmp/gh-aw/release-data/pull_requests.json
        echo "[]" > /tmp/gh-aw/release-data/pull_requests.json
      else
        echo "Previous release: $PREV_RELEASE_TAG"
        echo "PREV_RELEASE_TAG=$PREV_RELEASE_TAG" >> "$GITHUB_ENV"
        
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
        
        PR_COUNT=$(jq length "/tmp/gh-aw/release-data/pull_requests.json")
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