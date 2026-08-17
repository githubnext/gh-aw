#!/usr/bin/env pwsh
# scripts/notes/fetch.ps1
# ─────────────────────────────────────────────────────────────────────────────
# Fetch git notes from remote. Run on every Ralph-watch startup and before
# any agent reads or writes notes.
#
# Usage:
#   ./scripts/notes/fetch.ps1           # fetch only
#   ./scripts/notes/fetch.ps1 -Setup    # first-time: add refspec + fetch
#   ./scripts/notes/fetch.ps1 -Merge    # fetch + merge (use after push conflict)
# ─────────────────────────────────────────────────────────────────────────────

[CmdletBinding()]
param(
    [string]$Remote   = "origin",
    [string]$RepoPath = ".",
    [switch]$Setup,
    [switch]$Merge,
    [switch]$Quiet
)

function Log ([string]$msg, [string]$color = "White") {
    if (-not $Quiet) { Write-Host "[notes/fetch] $msg" -ForegroundColor $color }
}

$repo = Resolve-Path $RepoPath

# ── One-time setup: add fetch refspec ──────────────────────────────────────
if ($Setup) {
    $existing = git -C $repo config --get-all "remote.$Remote.fetch" 2>&1 |
                Where-Object { $_ -match "refs/notes" }
    if ($existing) {
        Log "Notes refspec already configured." DarkGray
    } else {
        git -C $repo config --add "remote.$Remote.fetch" "refs/notes/*:refs/notes/*"
        Log "Added notes refspec to remote.$Remote.fetch" Green
    }
}

# ── Fetch notes ─────────────────────────────────────────────────────────────
Log "Fetching notes from $Remote..."
$output = git -C $repo fetch $Remote "refs/notes/*:refs/notes/*" 2>&1
if ($LASTEXITCODE -ne 0) {
    Log "Fetch warning: $output" DarkYellow
} else {
    Log "Notes fetched." Green
}

# ── Merge notes if requested (after push conflict) ──────────────────────────
if ($Merge) {
    # Abort any stale merge-in-progress state
    $mergeLock = Join-Path $repo ".git/NOTES_MERGE_PARTIAL"
    if (Test-Path $mergeLock) {
        Log "Stale notes merge in progress — aborting before retry" DarkYellow
        git -C $repo notes merge --abort 2>&1 | Out-Null
    }

    $namespaces = git -C $repo for-each-ref "refs/notes/squad/" --format="%(refname)" 2>&1
    foreach ($ref in $namespaces) {
        $ns = $ref -replace "refs/notes/", ""
        $remoteRef = "refs/notes/remotes/$Remote/$ns"
        $remoteExists = git -C $repo for-each-ref $remoteRef --format="%(refname)" 2>&1
        if ($remoteExists) {
            Log "Merging notes: $ns (cat_sort_uniq)"
            git -C $repo notes --ref=$ns merge -s cat_sort_uniq $remoteRef 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) {
                Log "  Merge failed on $ns — aborting and continuing" Red
                git -C $repo notes merge --abort 2>&1 | Out-Null
            }
        }
    }
    Log "Notes merge complete." Green
}

# ── Show available namespaces ────────────────────────────────────────────────
if (-not $Quiet) {
    $refs = git -C $repo for-each-ref "refs/notes/squad/" --format="%(refname)" 2>&1
    if ($refs) {
        Log "Available namespaces:"
        foreach ($r in $refs) {
            $count = (git -C $repo notes --ref=($r -replace "refs/notes/","") list 2>&1 |
                      Where-Object { $_ -ne "" } | Measure-Object -Line).Lines
            Log "  $r  ($count notes)" DarkGray
        }
    } else {
        Log "No squad notes yet." DarkGray
    }
}
