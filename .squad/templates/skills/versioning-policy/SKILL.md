---
name: "versioning-policy"
description: "Semver versioning rules for Squad SDK and CLI — prevents prerelease version incidents"
domain: "release, versioning, npm, CI"
confidence: "medium"
source: "earned (PR #640 workspace resolution incident, PR #116 prerelease leak, CI gate implementation)"
---

## Context

Squad is a monorepo with two publishable npm packages (`@bradygaster/squad-sdk` and `@bradygaster/squad-cli`) managed via npm workspaces. Version mismatches and prerelease leaks have caused production incidents — most notably PR #640, where a `-build.N` prerelease version silently broke workspace dependency resolution.

This skill codifies the versioning rules every agent must follow.

## 1. Version Format

All packages use **strict semver**: `MAJOR.MINOR.PATCH`

- ✅ `0.9.1`, `1.0.0`, `0.10.0`
- ❌ `0.9.1-build.4`, `0.9.1-preview.1`, `0.8.6.1-preview`

No prerelease suffixes on `dev` or `main` branches — ever.

## 2. Prerelease Versions Are Ephemeral

The `scripts/bump-build.mjs` script creates `-build.N` versions (e.g., `0.9.1-build.4`) for **local development testing only**.

Rules:
- `-build.N` versions are created automatically during local `npm run build`
- They are **never committed** to `dev` or `main`
- The script skips itself in CI (`CI=true` or `SKIP_BUILD_BUMP=1`)
- If you see a `-build.N` version in a PR diff, it is a bug — reject the PR

## 3. SDK and CLI Version Sync

Both `@bradygaster/squad-sdk` and `@bradygaster/squad-cli` **MUST have the same version** at all times. The root `package.json` version must also match.

`bump-build.mjs` enforces this by updating all three `package.json` files in lockstep (root + `packages/squad-sdk` + `packages/squad-cli`).

If versions diverge, workspace resolution silently breaks (see §4).

## 4. npm Workspace Semver Footgun

The CLI depends on the SDK via a workspace dependency with a semver range:

```json
"@bradygaster/squad-sdk": ">=0.9.0"
```

**Critical:** Per the semver specification, `>=0.9.0` does **NOT** match `0.9.1-build.4`.

Semver prerelease versions (anything with a `-` suffix) are only matched by ranges that explicitly reference the same `MAJOR.MINOR.PATCH` base with a prerelease comparator. A bare `>=0.9.0` range skips all prerelease versions.

**What happens:** When the local SDK has version `0.9.1-build.4`, npm's workspace resolution fails to match the `>=0.9.0` range. npm then **silently installs a stale published version** from the npm registry instead of using the local workspace link. The build succeeds but runs against old SDK code.

This is the root cause of the **PR #640 incident**, where workspace packages appeared linked but were actually running against stale registry versions.

## 5. Who Bumps Versions

**Surgeon (Release Manager) owns all version bumps.**

| Agent | May modify `version` in package.json? |
|-------|---------------------------------------|
| Surgeon | ✅ Yes — sole owner of version bumps |
| Any other agent | ❌ No — unless explicitly fixing a prerelease leak |

If you discover a prerelease version committed to `dev` or `main`, you may fix it (revert to the clean release version) without Surgeon's approval. This is a safety escape hatch, not a license to manage versions.

## 6. Version Bump Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│  Development phase                                      │
│  Versions stay at current release: 0.9.1                │
│  bump-build.mjs creates -build.N locally (not committed)│
├─────────────────────────────────────────────────────────┤
│  Pre-release testing                                    │
│  bump-build.mjs → 0.9.1-build.1, -build.2, ...         │
│  Local only. Never committed. Never pushed.             │
├─────────────────────────────────────────────────────────┤
│  Release                                                │
│  Surgeon bumps to next version (e.g., 0.9.2 or 0.10.0) │
│  Tags, publishes to npm registry                        │
├─────────────────────────────────────────────────────────┤
│  Post-release                                           │
│  Versions stay at the new release version (e.g., 0.9.2) │
│  Development continues on clean version                 │
└─────────────────────────────────────────────────────────┘
```

## 7. CI Enforcement

The **`prerelease-version-guard`** CI gate blocks any PR to `dev` or `main` that contains prerelease version strings in `package.json` files.

- The gate scans all three `package.json` files for `-` in the version field
- PRs with prerelease versions **cannot merge** until the version is cleaned
- The `skip-version-check` label bypasses the gate — use **only** for the bump-build script's own PR (if applicable), and only with Surgeon's approval

## 8. Incident Reference — PR #640

**PR #640** is the cautionary tale for this entire policy.

**What happened:** Prerelease versions (`0.9.1-build.4`) were committed to a branch. The workspace dependency `>=0.9.0` failed to match the prerelease version per semver spec. npm silently installed a stale published SDK from the registry instead of linking the local workspace copy. Four PRs (#637–#640) attempted iterative patches before the root cause was identified.

**Root cause:** No versioning policy existed. Agents didn't know that prerelease versions break workspace resolution, or that only Surgeon should modify versions.

**Resolution:** This skill, the `prerelease-version-guard` CI gate, and the team decision to centralize version ownership under Surgeon.

## Quick Reference

| Rule | Summary |
|------|---------|
| Format | `MAJOR.MINOR.PATCH` — no prerelease on dev/main |
| Prerelease | `-build.N` is local-only, never committed |
| Sync | SDK + CLI + root must have identical versions |
| Ownership | Surgeon bumps versions; others don't touch them |
| CI gate | `prerelease-version-guard` blocks prerelease PRs |
| Escape hatch | Any agent may revert a prerelease leak to clean version |
| Footgun | `>=0.9.0` does NOT match `0.9.1-build.4` per semver |
