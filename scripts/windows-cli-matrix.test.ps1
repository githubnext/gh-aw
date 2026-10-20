$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
. "$PSScriptRoot/windows-cli-matrix.ps1"

$dimensions = Get-CliScenarioDimensions
$scenarios = @(New-CliScenarioMatrix)
$pairs = Assert-CliScenarioCoverage -Scenarios $scenarios
if ($scenarios.Count -gt 120) { throw "Matrix exceeds the bounded scenario budget: $($scenarios.Count)" }
if (@($scenarios.name | Sort-Object -Unique).Count -ne $scenarios.Count) { throw "Duplicate matrix rows" }
foreach ($row in $scenarios) {
  foreach ($key in $dimensions.Keys) {
    if ($row[$key] -notin $dimensions[$key]) { throw "Invalid matrix value: $key=$($row[$key])" }
  }
}
$repeated = @(New-CliScenarioMatrix)
if (($scenarios.name -join "`n") -cne ($repeated.name -join "`n")) { throw "Matrix is not deterministic" }
$missingRejected = $false
try {
  Assert-CliScenarioCoverage -Scenarios @($scenarios | Where-Object { $_.shell -ne "cmd" }) | Out-Null
} catch {
  if ($_.Exception.Message -notlike "Missing matrix pair:*") { throw }
  $missingRejected = $true
}
if (-not $missingRejected) { throw "Coverage assertion accepted an incomplete matrix" }
Write-Output "Matrix regression checks passed: $($scenarios.Count) scenarios, $pairs covered pairs"

# Compile the actual workflow helper and exercise stalled pipes without Win32.
$workflow = Get-Content "$PSScriptRoot/../.github/workflows/windows-cli-integration.yml" -Raw
foreach ($block in [regex]::Matches($workflow, '(?m)^        shell: pwsh\r?\n        run: \|\r?\n((?:          .*\r?\n|[ \t]*\r?\n)+)')) {
  $script = $block.Groups[1].Value -replace "(?m)^          ", ""
  $parseErrors = $null
  [System.Management.Automation.Language.Parser]::ParseInput($script, [ref]$null, [ref]$parseErrors) | Out-Null
  if ($parseErrors.Count) { throw "Workflow PowerShell parse errors: $($parseErrors.Message -join '; ')" }
}
$source = [regex]::Match($workflow, '(?s)Add-Type -TypeDefinition @"\r?\n(.*?)\r?\n          "@').Groups[1].Value
if (-not $source) { throw "ConPTY helper not found" }
$source = $source -replace "(?m)^          ", "" -replace "class ConPtySmoke", "class ConPtySmokeRegression"
Add-Type -TypeDefinition $source
$completed = [System.Threading.Tasks.Task]::CompletedTask
$captured = [System.Text.StringBuilder]::new("partial-output-marker")
$result = [ConPtySmokeRegression]::DrainOutput($completed, $completed, $captured, 50, $null)
if ($result -ne "partial-output-marker") { throw "Successful drain lost output" }

foreach ($case in @("stalled-reader", "stalled-close", "faulted-reader", "completed-reader")) {
  $pending = [System.Threading.Tasks.TaskCompletionSource[bool]]::new()
  $reader = if ($case -eq "stalled-reader") { $pending.Task } elseif ($case -eq "faulted-reader") {
    [System.Threading.Tasks.Task]::FromException([System.IO.IOException]::new("pipe failure"))
  } else { $completed }
  $closer = if ($case -eq "stalled-close") { $pending.Task } else { $completed }
  $watch = [System.Diagnostics.Stopwatch]::StartNew()
  $rejected = $false
  try {
    [ConPtySmokeRegression]::DrainOutput($closer, $reader, $captured, 50, "ConPTY child timed out after 100ms") | Out-Null
  } catch {
    $message = $_.Exception.ToString()
    if ($message -notmatch "ConPTY child timed out after 100ms" -or $message -notmatch "partial-output-marker") { throw }
    if ($case -like "stalled-*" -and $message -notmatch "drain/close timed out") { throw }
    if ($case -eq "faulted-reader" -and $message -notmatch "pipe failure") { throw }
    $rejected = $true
  } finally {
    $pending.TrySetResult($true) | Out-Null
  }
  if (-not $rejected) { throw "Child timeout was swallowed: $case" }
  if ($watch.ElapsedMilliseconds -gt 2000) { throw "Unbounded timeout cleanup: $case" }
}
Write-Output "ConPTY regression checks passed: bounded read/close waits, original timeout and partial output preserved"

if ($IsWindows) {
  $shell = (Get-Command cmd.exe).Source
  $watch = [System.Diagnostics.Stopwatch]::StartNew()
  $rejected = $false
  try {
    [ConPtySmokeRegression]::Run($shell, '/d /q /c "echo native-timeout-marker & for /L %i in (1,1,2147483647) do @rem"', 1000) | Out-Null
  } catch {
    $message = $_.Exception.ToString()
    if ($message -notmatch "ConPTY child timed out after 1000ms" -or $message -notmatch "native-timeout-marker") { throw }
    $rejected = $true
  }
  if (-not $rejected) { throw "Native ConPTY child timeout was swallowed" }
  if ($watch.ElapsedMilliseconds -gt 10000) { throw "Native ConPTY timeout cleanup exceeded its budget" }
  Write-Output "Native ConPTY timeout regression passed"
}
