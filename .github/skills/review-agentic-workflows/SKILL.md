---
name: review-agentic-workflows
description: Review agentic workflow changes for correctness, security posture, and optimization opportunities with compile, validation, and audit evidence.
---

# Review Agentic Workflows

Use this skill when asked to review `.github/workflows/*.md` agentic workflows or their generated `.lock.yml` outputs.
Reference workflow authoring skill guidance at: https://raw.githubusercontent.com/github/gh-aw/main/.github/skills/agentic-workflows/SKILL.md

## Goals

1. Produce a security-first review of workflow changes.
2. Compile workflows with validation and security scanners.
3. Flag suspicious changes that weaken protections.
4. Use run history (`logs`/`audit`) when available to find optimization opportunities.

## Self-contained setup (do not assume environment is ready)

### Step 0) Verify CLI availability

Run from the repository root:

```bash
if gh aw --help >/dev/null 2>&1; then
  echo "gh aw is installed"
else
  echo "gh aw is missing. Run install script steps from github/gh-aw:"
  echo "  bash ./install-gh-aw.sh"
  return 1 2>/dev/null || exit 1
fi
```

## Review workflow

### 1) Scope the review

Run this scope check in the review step:

```bash
BASE_REF="${BASE_REF:-origin/main}"
if git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  git diff --name-only "$BASE_REF...HEAD" -- .github/workflows/
else
  git diff --name-only -- .github/workflows/
fi
```

If source `.md` files changed, treat generated `.lock.yml` drift as part of the review.

### 2) Compile with validation + security tools

For changed workflows, run strict compilation with validators:

```bash
gh aw compile --strict --actionlint --zizmor --poutine --runner-guard --yamllint --shellcheck
```

If `gh aw` extension is unavailable but local binary exists:

```bash
./gh-aw compile --strict --actionlint --zizmor --poutine --runner-guard --yamllint --shellcheck
```

Fail review on compilation errors or High/Critical security findings unless explicitly justified.

### 3) Enforce security best practices

Require and verify:

- least-privilege `permissions:` (no `write-all` without explicit rationale)
- pinned third-party actions by full commit SHA
- safe handling of untrusted GitHub event data (no direct template injection into shell)
- explicit `safe-outputs` limits (`max`, constrained event/action sets)
- no broadening of network/tool access without justification
- no integrity downgrades (for example lower `min-integrity`)

### 4) Detect suspicious weakening changes

Treat these as suspicious until proven safe:

- permission expansion (especially new `write` scopes or global writes)
- relaxed security controls (`strict: false`, reduced guardrails, disabled scans)
- larger write blast radius (`safe-outputs` limits removed or sharply increased)
- reduced provenance controls (unpinning actions, mutable refs)
- wider external access (new unrestricted network domains/ecosystems)
- prompt or script edits that reintroduce command/template injection risk

Use targeted diffs and call out before/after impact.

### 5) Audit history and optimize (when run data exists)

If workflow run IDs/URLs are available, audit them:

```bash
gh aw audit <run-id-or-url>
gh aw logs --start-date -14d --workflow-name <workflow-name>
```

Look for optimization opportunities:

- high token/cost usage
- repeated retries/tool failures
- long-running steps or bottleneck jobs
- unnecessary MCP/tool invocations
- firewall denials causing retries or wasted turns

Recommend minimal, safe optimizations that keep or improve security posture.

## Review output contract

Return findings in three sections:

1. **Security regressions (must-fix)** — high-confidence weakening changes.
2. **Validation/scanner results** — compile and tool outcomes.
3. **Optimization opportunities** — optional improvements backed by logs/audit evidence.

Each finding should include severity, file(s), rationale, and a concrete remediation direction.
