# Skill: Retro Enforcement

## Purpose

Ensure retrospectives happen on schedule and that their action items are tracked in GitHub Issues — not markdown checklists.

This skill addresses a specific, measured failure mode: **0% completion rate on markdown retro action items across 6 consecutive retrospectives**. GitHub Issues have an 85%+ completion rate in the same squad. The format was the problem, not the people.

## Core Function: Test-RetroOverdue

```powershell
function Test-RetroOverdue {
    param(
        [string]$LogDir    = ".squad/log",
        [int]$WindowDays   = 7,
        [string]$Pattern   = "*retrospective*"
    )

    $cutoff = (Get-Date).AddDays(-$WindowDays)

    $retroLogs = Get-ChildItem -Path $LogDir -Filter $Pattern -ErrorAction SilentlyContinue |
                 Where-Object { $_.LastWriteTime -ge $cutoff }

    return ($retroLogs.Count -eq 0)
}
```

### Returns
- `$true` — No retro log found within the window. **Retro is overdue. Block other work.**
- `$false` — At least one retro log found within the window. Proceed normally.

### Detection Logic

The function checks `.squad/log/` for any file matching `*retrospective*` dated within the last `$WindowDays` days (default: 7). If none is found, the retro is overdue.

**File naming convention:** `.squad/log/{ISO8601-timestamp}-retrospective.md`

Example: `.squad/log/2026-03-24T14-45-00Z-retrospective.md`

## Coordinator Integration

Call `Test-RetroOverdue` **at the start of every round**, before building the work queue.

```powershell
# At round start — before any work queue construction
if (Test-RetroOverdue -LogDir ".squad/log" -WindowDays 7) {
    Write-Host "[RETRO] Retrospective overdue. Running before other work."

    # Spawn retro facilitator
    Invoke-RetroSession -Mode "catch-up"

    # Wait for retro log to be written
    # Then resume normal round
}

# Proceed with normal work queue
$workQueue = Get-PendingIssues | Sort-Object -Property Priority
```

### Blocking Semantics

When `Test-RetroOverdue` returns `$true`:

1. **Do not start any other work** until the retro completes
2. **Spawn the facilitator agent** (Scribe or designated) with retro mode
3. **Wait for the log file** to be written to `.squad/log/`
4. **Verify action items** were created as GitHub Issues (not markdown)
5. **Resume normal round** after retro log confirmed

## Action Item Enforcement

Every retro action item MUST become a GitHub Issue. The facilitator agent is responsible for this. The coordinator verifies.

### Verification Check

```powershell
function Test-RetroActionItemsCreated {
    param([string]$RetroLogPath)

    $content = Get-Content $RetroLogPath -Raw

    # Check for Issue references (e.g., #1478, https://github.com/.../issues/1478)
    $issueRefs = [regex]::Matches($content, '(?:#\d{3,}|issues/\d{3,})')

    # Check for unclosed markdown checkboxes (bad pattern)
    $openCheckboxes = [regex]::Matches($content, '- \[ \]')

    if ($openCheckboxes.Count -gt 0) {
        Write-Warning "[RETRO] Found $($openCheckboxes.Count) markdown checkboxes — convert to Issues"
        return $false
    }

    return ($issueRefs.Count -gt 0)
}
```

### Why Not Markdown Checklists

From production data in tamirdresher/tamresearch1:

| Retro | Action Items Format | Completion |
|-------|---------------------|------------|
| 2025-12-05 | Markdown `- [ ]` | 0/4 = **0%** |
| 2025-12-19 | Markdown `- [ ]` | 0/3 = **0%** |
| 2026-01-09 | Markdown `- [ ]` | 0/5 = **0%** |
| 2026-01-23 | Markdown `- [ ]` | 0/4 = **0%** |
| 2026-02-07 | Markdown `- [ ]` | 0/3 = **0%** |
| 2026-02-21 | Markdown `- [ ]` | 0/4 = **0%** |
| 2026-03-24 | GitHub Issues | 4/4 = **100%** (after enforcement) |

**Root cause:** Markdown checklists have no assignee, no notifications, no close event, and no query surface. They are invisible to every workflow that drives completion.

## Cadence Enforcement

### Recommended schedule
- Weekly squads: window = 7 days
- Bi-weekly squads: window = 14 days

### Ralph integration example

```powershell
# ralph-watch.ps1 — round start hook
function Invoke-RoundStart {
    # 1. Always check retro first
    if (Test-RetroOverdue -LogDir "$RepoRoot/.squad/log" -WindowDays 7) {
        Write-Host "[RALPH] Retro overdue — enforcing before work queue"
        Invoke-RetroSession
        return  # Re-enter round after retro completes
    }

    # 2. Normal work queue
    $issues = Get-ReadyIssues
    foreach ($issue in $issues) {
        Invoke-WorkItem -Issue $issue
    }
}
```

## Skill Metadata

| Field | Value |
|-------|-------|
| **Skill ID** | `retro-enforcement` |
| **Category** | Ceremonies / Process |
| **Trigger** | Coordinator round start |
| **Dependencies** | `.squad/log/` directory, GitHub Issues API |
| **Tested in** | tamirdresher/tamresearch1 (production, March 2026) |
| **Outcome** | Retro cadence restored; action item completion 0% → 100% |
