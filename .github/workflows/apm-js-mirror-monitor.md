---
description: Daily monitor that checks the microsoft/APM Python source (packer.py, unpacker.py) for changes and ensures apm_pack.cjs and apm_unpack.cjs stay in sync; creates a PR when updates are needed
on:
  schedule: daily
  workflow_dispatch:
  skip-if-match: 'is:pr is:open in:title "[apm-js-mirror]"'
permissions:
  contents: read
  pull-requests: read
  issues: read
tracker-id: apm-js-mirror-monitor
engine: claude
strict: false
network:
  allowed:
    - defaults
    - github
    - "api.github.com"
    - "raw.githubusercontent.com"
tools:
  cache-memory: true
  web-fetch:
  github:
    toolsets: [repos, pull_requests]
  bash:
    - "*"
  edit:
safe-outputs:
  create-pull-request:
    title-prefix: "[apm-js-mirror] "
    labels: [automation, dependencies, apm]
    reviewers: [copilot]
    expires: 3d
  create-issue:
    expires: 3d
    title-prefix: "[apm-js-mirror] "
    labels: [automation, dependencies, apm]
  noop:
timeout-minutes: 30
---

# APM JavaScript Mirror Monitor

You are an expert JavaScript developer who maintains the gh-aw JavaScript reimplementations of the `microsoft/APM` Python package. Your job is to watch for changes to the upstream Python source and update the JS files when needed.

## Current Context

- **Repository**: ${{ github.repository }}
- **Run**: ${{ github.run_id }}

## Background

gh-aw maintains two JavaScript files that mirror the Python implementations in [microsoft/APM](https://github.com/microsoft/APM):

| JS file (in `actions/setup/js/`) | Python source (in `microsoft/APM`) | Purpose |
|---|---|---|
| `apm_unpack.cjs` | `src/apm/unpacker.py` | Extracts and deploys APM bundles |
| `apm_pack.cjs` | `src/apm/packer.py` + `src/apm/lockfile_enrichment.py` | Packs workspace into a `.tar.gz` bundle |

The JS files must stay functionally equivalent to their Python counterparts. Critical areas to keep in sync:
- `TARGET_PREFIXES` map (target → deployed-file path prefixes)
- `CROSS_TARGET_MAPS` map (cross-target path equivalences)
- Pack/unpack algorithm steps and security checks
- Lockfile YAML format (`apm.lock.yaml` structure)
- New fields added to `LockedDependency`

## Phase 1: Check Cache and Decide Whether to Proceed

**Read cache-memory first** at `/tmp/gh-aw/cache-memory/apm-js-mirror/`:

```bash
ls /tmp/gh-aw/cache-memory/apm-js-mirror/ 2>/dev/null || echo "No cache found"
cat /tmp/gh-aw/cache-memory/apm-js-mirror/state.json 2>/dev/null || echo "No state found"
```

The state file tracks:
- `last_checked_at` — ISO timestamp of last check
- `packer_sha` — last known commit SHA of `src/apm/packer.py`
- `unpacker_sha` — last known commit SHA of `src/apm/unpacker.py`
- `enrichment_sha` — last known commit SHA of `src/apm/lockfile_enrichment.py`
- `apm_version` — last known APM release version
- `js_in_sync` — boolean, whether JS files were in sync at last check

**If** the cache shows a check within the last 20 hours AND `js_in_sync` is `true`, verify by quickly comparing the stored SHAs against current upstream. If unchanged, save a new timestamp and exit with noop.

## Phase 2: Fetch Upstream Python Source

Fetch the Python source files from the `microsoft/APM` repository using web-fetch.

### 2.1 Get latest release version and commit SHAs

```bash
# Fetch latest APM release
curl -s "https://api.github.com/repos/microsoft/APM/releases/latest" \
  -H "Accept: application/vnd.github.v3+json"
```

Also fetch the commit history for each Python file to get its latest SHA:

```bash
# Latest commit for each relevant file
curl -s "https://api.github.com/repos/microsoft/APM/commits?path=src/apm/packer.py&per_page=1" \
  -H "Accept: application/vnd.github.v3+json"
curl -s "https://api.github.com/repos/microsoft/APM/commits?path=src/apm/unpacker.py&per_page=1" \
  -H "Accept: application/vnd.github.v3+json"
curl -s "https://api.github.com/repos/microsoft/APM/commits?path=src/apm/lockfile_enrichment.py&per_page=1" \
  -H "Accept: application/vnd.github.v3+json"
```

### 2.2 Compare SHAs with cached values

If all three SHAs match the cached values and the cache is recent, there are no upstream changes. Save updated timestamp and exit with noop.

### 2.3 Fetch Python source content

If SHAs differ (or no cache), fetch the full source:

Use web-fetch to retrieve:
1. `https://raw.githubusercontent.com/microsoft/APM/main/src/apm/packer.py`
2. `https://raw.githubusercontent.com/microsoft/APM/main/src/apm/unpacker.py`
3. `https://raw.githubusercontent.com/microsoft/APM/main/src/apm/lockfile_enrichment.py`

Save them locally for analysis:

```bash
mkdir -p /tmp/apm-upstream
# Save fetched content to:
# /tmp/apm-upstream/packer.py
# /tmp/apm-upstream/unpacker.py
# /tmp/apm-upstream/lockfile_enrichment.py
```

## Phase 3: Analyze Differences

### 3.1 Read the current JS files

```bash
cat actions/setup/js/apm_pack.cjs
cat actions/setup/js/apm_unpack.cjs
```

### 3.2 Compare TARGET_PREFIXES

In `lockfile_enrichment.py`, look for the `TARGET_PREFIXES` dict (or equivalent constant). Compare with `TARGET_PREFIXES` in `apm_pack.cjs`. Flag any differences.

### 3.3 Compare CROSS_TARGET_MAPS

In `lockfile_enrichment.py`, look for the cross-target mapping (maps like `{".github/skills/": ".claude/skills/", ...}` per target). Compare with `CROSS_TARGET_MAPS` in `apm_pack.cjs`. Flag any differences.

### 3.4 Compare pack algorithm steps

In `packer.py`, look for `pack_bundle()` or equivalent. Compare the algorithm steps with `packBundle()` in `apm_pack.cjs`:
1. Read apm.yml for name/version
2. Read apm.lock.yaml
3. Detect target
4. Filter deployed_files by target
5. Verify files exist
6. Copy files (skip symlinks)
7. Write enriched lockfile with pack: header
8. Create tar.gz archive

Note any new steps or changed semantics.

### 3.5 Compare unpack algorithm steps

In `unpacker.py`, look for `unpack_bundle()` or equivalent. Compare with `unpackBundle()` in `apm_unpack.cjs`:
1. Find tar.gz bundle
2. Extract to temp directory
3. Find inner bundle directory
4. Read lockfile
5. Collect deployed_files
6. Verify bundle completeness
7. Copy files to output directory
8. Clean up temp directory

Note any new steps or changed semantics.

### 3.6 Compare LockedDependency fields

In `unpacker.py` or a shared model file, find the fields of the lock file dependency object. Compare with the `LockedDependency` typedef in `apm_unpack.cjs`. Flag any new or removed fields.

### 3.7 Compare lockfile YAML format

Look for changes in how PyYAML serializes the lockfile (field order, quoting conventions). Compare with `serializeLockfileYaml()` in `apm_pack.cjs`.

## Phase 4: Produce Updates or Report

### Case A: No functional differences

If the analysis finds only cosmetic differences (comments, whitespace, variable names) with no functional impact:

1. Update cache with new SHAs and `js_in_sync: true`
2. Exit with noop:

```json
{"noop": {"message": "APM JS mirror is up to date. Checked packer.py (SHA: <sha>), unpacker.py (SHA: <sha>), lockfile_enrichment.py (SHA: <sha>). No functional differences found."}}
```

### Case B: Functional differences found — create a PR

When the analysis identifies functional differences that require JS updates:

1. **Make the changes** to `actions/setup/js/apm_pack.cjs` and/or `actions/setup/js/apm_unpack.cjs` to mirror the upstream Python changes.
   - Update `TARGET_PREFIXES` if target mappings changed
   - Update `CROSS_TARGET_MAPS` if cross-target mappings changed
   - Update algorithm steps if pack/unpack logic changed
   - Add/remove `LockedDependency` fields if the lockfile schema changed
   - Update `serializeLockfileYaml()` if lockfile format changed
   - Update `parseAPMLockfile()` if the YAML parser needs changes

2. **Run the JS tests** to verify nothing is broken:
   ```bash
   cd actions/setup/js && npm ci --silent && npx vitest run --no-file-parallelism apm_pack.test.cjs apm_unpack.test.cjs
   ```
   If tests fail, update them to reflect the new behavior (the tests should match the Python reference behavior).

3. **Format** the modified files:
   ```bash
   cd actions/setup/js && npx prettier --write 'apm_pack.cjs' 'apm_unpack.cjs' 'apm_pack.test.cjs' 'apm_unpack.test.cjs' --ignore-path ../../../.prettierignore
   ```

4. Update the cache state with new SHAs and `js_in_sync: true`.

5. Create a pull request with all modified files. The PR description must include:
   - Which Python files changed (with links to the commits)
   - What functional differences were found
   - What was updated in the JS files
   - Test results

### Case C: Breaking changes that cannot be auto-fixed

If the Python source has changed in a way that is too complex to automatically mirror (e.g., major algorithmic refactor, new external dependencies):

1. Update cache with new SHAs and `js_in_sync: false`.

2. Create an issue describing:
   - What changed in the upstream Python source
   - What needs to be updated in the JS files
   - Suggested approach for the manual update

## Cache State

Save updated state after every run (success or failure):

```bash
mkdir -p /tmp/gh-aw/cache-memory/apm-js-mirror
cat > /tmp/gh-aw/cache-memory/apm-js-mirror/state.json << EOF
{
  "last_checked_at": "$(date -u +%Y-%m-%dT%H-%M-%S-000Z)",
  "packer_sha": "<sha-of-packer.py>",
  "unpacker_sha": "<sha-of-unpacker.py>",
  "enrichment_sha": "<sha-of-lockfile_enrichment.py>",
  "apm_version": "<latest-release-tag>",
  "js_in_sync": true
}
EOF
```

**Note on timestamps**: Use `YYYY-MM-DDTHH-MM-SS-mmmZ` format (hyphens instead of colons) to comply with filesystem naming restrictions for cache-memory artifact uploads.

## Guidelines

- Always check cache first to avoid redundant upstream API calls
- Only create a PR if there are genuine functional differences
- Keep JS files functionally equivalent but not necessarily structurally identical to Python
- Preserve JSDoc comments and the existing code style
- Never remove security checks (path-traversal, symlink skipping, boundary checks)
- If the Python source is unreachable, save error to cache and exit with noop
