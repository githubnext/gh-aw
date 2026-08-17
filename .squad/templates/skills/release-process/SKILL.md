---
name: "release-process"
description: "Pre-release validation, npm publish procedures, and post-publish verification"
domain: "release"
confidence: "high"
source: "earned"
---

# Release Process

> Earned knowledge from the v0.9.0→v0.9.1 and v0.9.4 incidents. Every agent involved in releases MUST read this before starting release work.
> See also: `.github/skills/release-process/SKILL.md` for the Copilot-facing runbook.

## SCOPE

✅ THIS SKILL PRODUCES:
- Pre-release validation checks that prevent broken publishes
- Correct npm publish commands (never workspace-scoped)
- Fallback procedures when CI workflows fail
- Post-publish verification steps

❌ THIS SKILL DOES NOT PRODUCE:
- Feature implementation or test code
- Architecture decisions
- Documentation content

## Confidence: high

Established through the v0.9.1 incident (8-hour recovery) and reinforced by the v0.9.4 release delay (PRs #1042, #1043, #1044). Every rule below is battle-tested.

## Context

Squad publishes two npm packages: `@bradygaster/squad-sdk` and `@bradygaster/squad-cli`. The release pipeline flows: dev → preview → main → GitHub Release → npm publish. Brady (project owner) triggers releases — the coordinator does NOT.

## Rules (Non-Negotiable)

### 1. Coordinator Does NOT Publish

The coordinator routes work and manages agents. It does NOT run `npm publish`, trigger release workflows, or make release decisions. Brady owns the release trigger. If an agent or the coordinator is asked to publish, escalate to Brady.

### 2. Pre-Publish Dependency Validation

Before ANY release is tagged, scan every `packages/*/package.json` for:
- `file:` references (workspace leak — the v0.9.0 root cause)
- `link:` references
- Absolute paths in dependency values
- Non-semver version strings

**Command:**
```bash
grep -r '"file:\|"link:\|"/' packages/*/package.json
```
If anything matches, STOP. Do not proceed. Fix the reference first.

### 3. Never Use `npm -w` for Publishing

`npm -w packages/squad-sdk publish` hangs silently when 2FA is enabled. Always `cd` into the package directory:

```bash
cd packages/squad-sdk && npm publish --access public
cd packages/squad-cli && npm publish --access public
```

### 4. Fallback Protocol

If `workflow_dispatch` or the publish workflow fails:
1. Try once more (ONE retry, not four)
2. If it fails again → local publish immediately
3. Do NOT attempt GitHub UI file operations to fix workflow indexing
4. GitHub has a ~15min workflow cache TTL after file renames/deletes — waiting helps, retrying doesn't

### 5. Post-Publish Smoke Test

After every publish, verify in a clean shell:
```bash
npm install -g @bradygaster/squad-cli@latest
squad --version    # should match published version
squad doctor       # should pass in a test repo
```

If the smoke test fails, rollback immediately.

### 6. npm Token Must Be Automation Type

NPM_TOKEN in CI must be an Automation token (not a user token with 2FA prompts). User tokens with `auth-and-writes` 2FA cause silent hangs in non-interactive environments.

### 7. No Draft GitHub Releases

Never create draft GitHub Releases. The `release: published` event only fires when a release is published — drafts don't trigger the npm publish workflow.

### 8. Version Format

Semantic versioning only: `MAJOR.MINOR.PATCH` (e.g., `0.9.1`). Four-part versions like `0.8.21.4` are NOT valid semver and will break npm publish.

### 9. SKIP_BUILD_BUMP=1 in CI

Set this environment variable in all CI build steps to prevent the build script from mutating versions during CI runs.

## Release Checklist (Quick Reference)

```
□ All tests passing on dev
□ No file:/link: references in packages/*/package.json
□ Root package.json version matches sub-packages (v0.9.4 lesson — PR #1043)
□ CHANGELOG.md has ## [$VERSION] section (not just [Unreleased]) (v0.9.4 lesson — PR #1042)
□ Version bumps committed: npm version $VERSION --workspaces --include-workspace-root --no-git-tag-version
□ npm auth verified (Automation token)
□ No draft GitHub Releases pending
□ Local build + test: npm run build && npx vitest run
□ Push dev → CI green
□ Promote dev → preview (squad-promote workflow)
□ Preview CI green (squad-preview validates)
□ Promote preview → main
□ squad-release auto-creates GitHub Release
□ squad-npm-publish auto-triggers (⚠️ may be BLOCKED — see GITHUB_TOKEN limitation below)
□ If publish didn't trigger: gh workflow run squad-npm-publish.yml --ref main -f version=X.Y.Z
□ Monitor publish workflow
□ Post-publish smoke test
```

## Known Gotchas

| Gotcha | Impact | Mitigation |
|--------|--------|------------|
| npm workspaces rewrite `"*"` → `"file:../path"` | Broken global installs | Preflight scan in CI (squad-npm-publish.yml) |
| GitHub Actions workflow cache (~15min TTL) | 422 on workflow_dispatch after file renames | Wait 15min or use local publish fallback |
| `npm -w publish` hangs with 2FA | Silent hang, no error | Never use `-w` for publish |
| Draft GitHub Releases | npm publish workflow doesn't trigger | Never create drafts |
| User npm tokens with 2FA | EOTP errors in CI | Use Automation token type |
| Root package.json version drift (v0.9.4) | squad-release.yml fails CHANGELOG check | Always bump all 3 package.json files together (PR #1043) |
| CHANGELOG.md missing `## [$VERSION]` (v0.9.4) | squad-release.yml exits with error | Convert `[Unreleased]` → `[$VERSION] - YYYY-MM-DD` before promoting to main (PR #1042) |
| GITHUB_TOKEN can't trigger downstream workflows (v0.9.4) | squad-npm-publish.yml never fires | Manual `gh workflow run` or use PAT/GitHub App token (see below) |
| Lockfile integrity check rejects workspace packages (v0.9.4) | False failures in squad-npm-publish.yml | Only validate packages resolved from npm registry (`startsWith('https://')`) (PR #1044) |
| `prebuild` version bump breaks workspace linking (v0.9.4) | Local builds fail after bump-build.mjs runs | `git checkout -- package.json packages/*/package.json` then fresh install |

## v0.9.4 Incident Learnings

> Source: v0.9.4 release session. PRs #1042, #1043, #1044.

### Root Package.json Version Must Match Sub-Packages

`squad-release.yml` reads version from ROOT `package.json` (lines 31-35):
```bash
VERSION=$(node -e "console.log(require('./package.json').version)")
if ! grep -q "## \[$VERSION\]" CHANGELOG.md; then
  echo "::error::Version $VERSION not found in CHANGELOG.md"
  exit 1
fi
```
If root package.json is behind (e.g., 0.9.1 while sub-packages are 0.9.4), the release workflow FAILS. This was the root cause of the v0.9.4 release delay — PR #1043 fixed it.

**Rule:** When bumping versions, ALWAYS bump all 3 package.json files together:
```bash
npm version $VERSION --workspaces --include-workspace-root --no-git-tag-version
```

### CHANGELOG.md Must Have Version Entry

`squad-release.yml` validates that `CHANGELOG.md` contains `## [$VERSION]`. If the version section is still `[Unreleased]` and no `[$VERSION]` section exists, the release workflow exits with error. PR #1042 fixed this for v0.9.4.

**Rule:** Before promoting to main, convert `[Unreleased]` to `[$VERSION] - YYYY-MM-DD` in CHANGELOG.md and add a fresh `[Unreleased]` section above it.

### GITHUB_TOKEN Event Propagation Limitation (CRITICAL)

When `squad-release.yml` creates a GitHub Release using the default `GITHUB_TOKEN`, the `release: published` event does NOT trigger `squad-npm-publish.yml`. This is a GitHub security feature to prevent infinite workflow loops.

**Workaround:** After the release workflow succeeds and creates the tag + GitHub Release, manually trigger the publish workflow:
```bash
gh workflow run squad-npm-publish.yml --ref main -f version=X.Y.Z
```
IMPORTANT: Use `--ref main` to ensure the workflow runs against the main branch (where the release artifacts exist).

**Permanent fix (TODO):** Use a PAT or GitHub App token in `squad-release.yml` instead of `GITHUB_TOKEN`.

### Lockfile Integrity — Workspace Package Handling

The lockfile stability check in `squad-npm-publish.yml` (line 82) filters packages for integrity hashes. Workspace packages resolve to bare relative paths (e.g., `packages/squad-sdk`), NOT `file:` URLs. The check must filter for registry-resolved packages only (`startsWith('https://')`). PR #1044 fixed this.

### Prebuild Version Bump Breaks Local Workspace Resolution

`scripts/bump-build.mjs` runs during `npm run prebuild` and bumps versions like `0.9.4` → `0.9.4-build.1`. This breaks workspace linking because CLI depends on exact `"@bradygaster/squad-sdk": "0.9.4"` but SDK becomes `0.9.4-build.1`.

**Fix for local dev:**
```bash
git checkout -- package.json packages/*/package.json
rm -rf node_modules packages/*/node_modules
npm install
npm run build
```

### The Full Promotion Chain (v0.9.4 Documented)

```
dev → preview → main (via squad-promote.yml)
main push → squad-release.yml validates CHANGELOG, creates tag + GitHub Release
release published → squad-npm-publish.yml (⚠️ BLOCKED by GITHUB_TOKEN limitation)
manual workaround → gh workflow run squad-npm-publish.yml --ref main -f version=X.Y.Z
```

### npm Publish Workflow Dispatch Target

When using `workflow_dispatch` to trigger `squad-npm-publish.yml`, the default ref is the repo's default branch (`dev`). Always specify `--ref main` explicitly to ensure the workflow runs against the branch with the release tag and latest workflow fixes.

## CI Gate: Workspace Publish Policy

The `publish-policy` job in `squad-ci.yml` scans all workflow files for bare `npm publish` commands that are missing `-w`/`--workspace` flags. Any workflow that attempts a non-workspace-scoped publish will fail CI. This prevents accidental root-level publishes that would push the wrong `package.json` to npm.

See `.github/workflows/squad-ci.yml` → `publish-policy` job for implementation details.

## Related

- Issues: #556–#564 (release:next)
- v0.9.4 fixes: PR #1042 (CHANGELOG), PR #1043 (root package.json), PR #1044 (lockfile integrity)
- Retro: `.squad/decisions/inbox/surgeon-v091-retrospective.md`
- CI audit: `.squad/decisions/inbox/booster-ci-audit.md`
- Copilot-level skill: `.github/skills/release-process/SKILL.md`
- Playbook: `PUBLISH-README.md` (repo root)
