---
name: "pr-screenshots"
description: "Capture Playwright screenshots and embed them in GitHub PR descriptions"
domain: "pull-requests, visual-review, docs, testing"
confidence: "high"
source: "earned (multiple sessions establishing the pattern for PR #11 TypeDoc API reference)"
---

## Context

When a PR includes visual changes (docs sites, UI components, generated pages), reviewers
need to see what the PR delivers without checking out the branch. Screenshots belong in
the **PR description body**, not as committed files and not as text descriptions.

Use this skill whenever:
- A PR touches docs site pages (Astro, Starlight, etc.)
- A PR adds or changes UI components
- A PR generates visual artifacts (TypeDoc, Storybook, diagrams)
- Playwright tests already capture screenshots as part of testing

## Patterns

### 1. Capture screenshots with Playwright

If Playwright tests already exist and produce screenshots, reuse those. Otherwise,
write a minimal capture script:

```javascript
// scripts/capture-pr-screenshots.mjs
import { chromium } from 'playwright';

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });

const screenshots = [
  { url: 'http://localhost:4321/path/to/page', name: 'feature-landing' },
  { url: 'http://localhost:4321/path/to/detail', name: 'feature-detail' },
];

for (const { url, name } of screenshots) {
  await page.goto(url, { waitUntil: 'networkidle' });
  await page.screenshot({ path: `screenshots/${name}.png`, fullPage: false });
}

await browser.close();
```

### 2. Host screenshots on a temporary branch

GitHub PR descriptions render images via URLs. The `gh` CLI cannot upload binary
images directly. Use a temporary orphan branch to host the images:

```powershell
# Save current branch
$currentBranch = git branch --show-current

# Create orphan branch with only screenshot files
git checkout --orphan screenshots-temp
git reset
git add screenshots/*.png
git commit -m "screenshots for PR review"
git push origin screenshots-temp --force

# Build raw URLs
$base = "https://raw.githubusercontent.com/{owner}/{repo}/screenshots-temp/screenshots"
# Each image: $base/{name}.png

# Return to working branch
git checkout -f $currentBranch
```

### 3. Embed in PR description

Use `gh pr edit` with the raw URLs embedded as markdown images:

```powershell
$base = "https://raw.githubusercontent.com/{owner}/{repo}/screenshots-temp/screenshots"

gh pr edit {PR_NUMBER} --repo {owner}/{repo} --body @"
## {PR Title}

### What this PR delivers
- {bullet points of changes}

---

### Screenshots

#### {Page/Feature Name}
![{alt text}]($base/{name}.png)

#### {Another Page}
![{alt text}]($base/{another-name}.png)

---

### To verify locally
```bash
{commands to run locally}
```
"@
```

### 4. Cleanup after merge

After the PR is merged, delete the temporary branch:

```bash
git push origin --delete screenshots-temp
```

### 5. Gitignore screenshots locally

Screenshots are build artifacts — never commit them to feature branches:

```gitignore
# PR screenshots (hosted on temp branch, not committed to features)
screenshots/
docs/tests/screenshots/
```

## Examples

### Example: Docs site PR with 3 pages

1. Start dev server: `cd docs && npm run dev`
2. Run Playwright tests (they capture screenshots as a side effect)
3. Push screenshots to `screenshots-temp` branch
4. Update PR body with embedded `![...]()` image references
5. Reviewer sees the pages inline without checking out the branch

### Example: Reusing existing Playwright test screenshots

If tests at `docs/tests/*.spec.mjs` already save to `docs/tests/screenshots/`:

```powershell
cd docs && npx playwright test tests/api-reference.spec.mjs
# Screenshots now at docs/tests/screenshots/*.png
# Push those to screenshots-temp and embed in PR
```

## Anti-Patterns

- ❌ **Committing screenshots to feature branches** — they bloat the repo and go stale
- ❌ **Posting text descriptions instead of actual images** — reviewers can't see what they're getting
- ❌ **Using `gh` CLI to "upload" images** — `gh issue comment` and `gh pr edit` don't support binary uploads
- ❌ **Asking the user to manually drag-drop images** — automate it with the temp branch pattern
- ❌ **Skipping screenshots for visual PRs** — if the PR changes what users see, show what users see
- ❌ **Leaving the screenshots-temp branch around forever** — clean up after merge
