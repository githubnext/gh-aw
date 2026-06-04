# Compatibility Matrix Update Model

## Overview

The `.github/aw/compat.json` file serves as the source of truth for agent compatibility with gh-aw versions. This document describes how compatibility matrix updates are driven by canary testing and the security model that keeps gh-aw free of cross-repo secrets.

## Architecture: Push Model (Canary-Driven Bumping)

### Design

The compatibility matrix uses a **push model** where the private `agentic-workflows-canary` repository drives updates:

```
agentic-workflows-canary (private)
  ↓ (1) Run canary scenarios
  ↓ (2) Evaluate health within freshness window
  ↓ (3) Emit repository_dispatch event
  ↓
github/gh-aw (public)
  ↓ (4) Receive dispatch event
  ↓ (5) Open/update guarded compat bump PR
  ↓ (6) Automated review + manual approval
  ↓ (7) Merge to main
```

### What is "Bumping"?

**Bumping** refers to updating the `max-agent` field in a compat row when:

1. A new agent version (e.g., Copilot CLI 1.0.52) is released
2. Canary scenarios pass with the new version + current gh-aw
3. The compat row has `"open": true` (indicating it accepts automatic bumps)

Example bump:
```json
// Before:
{
  "min-gh-aw": "0.72.0",
  "max-gh-aw": "*",
  "min-agent": "1.0.21",
  "max-agent": "1.0.51",  // ← Old max
  "open": true
}

// After:
{
  "min-gh-aw": "0.72.0",
  "max-gh-aw": "*",
  "min-agent": "1.0.21",
  "max-agent": "1.0.52",  // ← Bumped to new version
  "open": true
}
```

## Guard Conditions

The canary repository emits a `repository_dispatch` event **only when ALL** of the following conditions are met:

### 1. Scenario Pass Criteria

All required canary scenarios must pass for the candidate agent version:

- **Core functionality**: Basic workflow execution (compile, activate, run)
- **Security validation**: No new security alerts from CodeQL/secret scanning
- **Compatibility tests**: Agent works with current gh-aw release range
- **Performance benchmarks**: No significant regressions in latency/memory
- **Integration tests**: MCP server communication, SDK driver compatibility

### 2. Freshness Window

Results must be within the freshness window to prevent stale data from triggering bumps:

- **Maximum age**: Scenario results must be ≤ 24 hours old
- **Minimum coverage**: All required scenarios must have fresh results
- **Health check**: Canary infrastructure itself must be healthy

### 3. Version Eligibility

The candidate agent version must meet release criteria:

- **Stable release**: No pre-release tags (e.g., `-alpha`, `-beta`)
- **Not blocked**: Not in `blockedVersions` list
- **Newer than current**: Version > current `max-agent` in compat.json
- **Row is open**: Target compat row has `"open": true`

### 4. No Active Incidents

No active incident or rollback in progress:

- **Incident status**: No P0/P1 incidents affecting gh-aw or the agent
- **Rollback state**: No ongoing rollback of previous bump
- **Manual freeze**: No operator-initiated freeze flag

## Dispatch Payload

When all guard conditions pass, canary emits:

```json
{
  "event_type": "compat-bump",
  "client_payload": {
    "agent": "copilot",
    "version": "1.0.52",
    "canary_run_id": "12345678",
    "canary_scenarios_url": "https://github.com/github/agentic-workflows-canary/actions/runs/12345678",
    "freshness_timestamp": "2026-06-04T01:00:00Z",
    "guard_conditions": {
      "scenarios_passed": true,
      "within_freshness_window": true,
      "version_eligible": true,
      "no_active_incidents": true
    }
  }
}
```

## gh-aw Bump Workflow

When gh-aw receives the `repository_dispatch` (workflow: `.github/workflows/compat-bump-from-canary-dispatch.yml`):

1. **Validate payload**: Verify all guard conditions are still true
2. **Fetch current compat.json**: Read current `max-agent` value
3. **Generate PR**: Create or update a PR with the new `max-agent`
4. **Automated checks**: Run validation workflow (schema check, test suite)
5. **Human review**: Require manual approval from compat maintainers
6. **Merge**: Automated merge after approval + passing checks

## Security Model

### Why This Model?

The push model is chosen for three key reasons:

1. **Minimize secret sprawl**: Only the canary repo needs credentials
2. **Single source of truth**: Canary owns health determination logic
3. **Principle of least privilege**: gh-aw doesn't need to read private canary data

### Credential Flow

```
┌─────────────────────────────────┐
│ agentic-workflows-canary        │
│ (private repo)                  │
│                                 │
│ ✓ Has: DISPATCH_TOKEN           │ ← Only place with cross-repo secret
│   - Scope: public_repo          │
│   - Can: send repository_dispatch to github/gh-aw
│   - Cannot: read/write gh-aw branches
└─────────────────────────────────┘
              │
              │ repository_dispatch (no authentication required to receive)
              ↓
┌─────────────────────────────────┐
│ github/gh-aw                    │
│ (public repo)                   │
│                                 │
│ ✗ No secrets to canary          │ ← Keeps gh-aw secret-free
│ ✓ Has: GITHUB_TOKEN (default)   │
│   - Can: create PRs in gh-aw    │
│   - Cannot: read canary data    │
└─────────────────────────────────┘
```

### Token Scope and Blast Radius

**DISPATCH_TOKEN** (stored in canary repo):
- **Required scope**: `public_repo` (or `repo` if gh-aw were private)
- **Blast radius if leaked**:
  - ✗ Can send repository_dispatch to github/gh-aw (trigger bump workflow)
  - ✗ Can send repository_dispatch to other public repos (if configured)
  - ✓ CANNOT read canary scenarios (private repo)
  - ✓ CANNOT push to gh-aw branches (no write access)
  - ✓ CANNOT modify gh-aw PRs directly

**Mitigation if token leaks**:
1. Revoke and rotate token immediately
2. Review recent repository_dispatch events in gh-aw
3. Revert any unauthorized compat bumps
4. No canary data exposure (private repo remains protected)

### Alternative: Polling (Not Chosen)

We explicitly **rejected** a polling model where gh-aw fetches canary status:

```
❌ Polling Model (Rejected):
   gh-aw (public) → poll → agentic-workflows-canary (private)
   
   Problems:
   - Requires gh-aw to have secrets for canary access
   - Increases secret sprawl in public repo
   - More complex secret rotation
   - Larger blast radius if gh-aw is compromised
```

## Operational Notes

### Manual Bumps

Operators can manually bump compat.json by:

1. Opening a PR directly in gh-aw
2. Bypassing the canary dispatch (for emergency updates)
3. Following the same review/approval process

### Freezing Automatic Bumps

To temporarily stop automatic bumps:

1. Set `"open": false` on the target compat row
2. Canary will skip dispatch for that agent
3. Re-enable by setting `"open": true` in a subsequent PR

### Rollback

If a bumped version causes issues:

1. Revert the compat.json PR in gh-aw
2. Set `"open": false` to prevent re-bump
3. Add problematic version to `blockedVersions` if necessary
4. Investigate root cause in canary scenarios

## References

- Compatibility matrix schema: `.github/aw/compat.schema.json`
- Installer implementation: `actions/setup/sh/install_copilot_cli.sh`
- Canary repository: `github/agentic-workflows-canary` (private)
