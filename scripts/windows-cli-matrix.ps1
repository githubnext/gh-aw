Set-StrictMode -Version Latest

function Get-CliScenarioDimensions {
  return [ordered]@{
    shell = @("pwsh-default", "pwsh-noprofile", "powershell-default", "powershell-noprofile", "cmd")
    launch = @("direct", "path")
    env = @("default", "minimal", "no-color", "term-dumb", "term-unset", "ci-true", "ci-unset")
    path = @("default", "spaces", "parentheses", "mixed-slashes", "unicode")
    order = @("workspace-first", "toolcache-first", "duplicate-workspace", "shadow")
    pathext = @("default", "exe-first", "duplicate-exe", "missing-exe")
  }
}

function New-CliScenarioMatrix {
  param([System.Collections.IDictionary] $Dimensions = (Get-CliScenarioDimensions))

  $keys = @($Dimensions.Keys)
  $covered = [System.Collections.Generic.HashSet[string]]::new()
  $scenarios = [System.Collections.Generic.List[hashtable]]::new()

  # Seed every uncovered pair, then greedily fill the other dimensions. Unlike
  # separate pair loops with fixed defaults, each row combines all dimensions.
  for ($i = 0; $i -lt $keys.Count; $i++) {
    for ($j = $i + 1; $j -lt $keys.Count; $j++) {
      foreach ($left in $Dimensions[$keys[$i]]) {
        foreach ($right in $Dimensions[$keys[$j]]) {
          if ($covered.Contains("$i/$left/$j/$right")) { continue }
          $row = @{ $keys[$i] = $left; $keys[$j] = $right }
          for ($k = 0; $k -lt $keys.Count; $k++) {
            if ($row.ContainsKey($keys[$k])) { continue }
            $bestScore = -1
            $bestValue = $null
            foreach ($value in $Dimensions[$keys[$k]]) {
              $score = 0
              for ($n = 0; $n -lt $keys.Count; $n++) {
                if (-not $row.ContainsKey($keys[$n])) { continue }
                $pair = if ($n -lt $k) { "$n/$($row[$keys[$n]])/$k/$value" } else { "$k/$value/$n/$($row[$keys[$n]])" }
                if (-not $covered.Contains($pair)) { $score++ }
              }
              if ($score -gt $bestScore) { $bestScore = $score; $bestValue = $value }
            }
            $row[$keys[$k]] = $bestValue
          }
          for ($a = 0; $a -lt $keys.Count; $a++) {
            for ($b = $a + 1; $b -lt $keys.Count; $b++) {
              [void]$covered.Add("$a/$($row[$keys[$a]])/$b/$($row[$keys[$b]])")
            }
          }
          $row.name = ($keys | ForEach-Object { "$_=$($row[$_])" }) -join ", "
          $scenarios.Add($row)
        }
      }
    }
  }
  return $scenarios.ToArray()
}

function Assert-CliScenarioCoverage {
  param(
    [Parameter(Mandatory)] [hashtable[]] $Scenarios,
    [System.Collections.IDictionary] $Dimensions = (Get-CliScenarioDimensions)
  )

  $keys = @($Dimensions.Keys)
  $pairs = 0
  for ($i = 0; $i -lt $keys.Count; $i++) {
    for ($j = $i + 1; $j -lt $keys.Count; $j++) {
      foreach ($left in $Dimensions[$keys[$i]]) {
        foreach ($right in $Dimensions[$keys[$j]]) {
          $found = $false
          foreach ($row in $Scenarios) {
            if ($row[$keys[$i]] -eq $left -and $row[$keys[$j]] -eq $right) { $found = $true; break }
          }
          if (-not $found) { throw "Missing matrix pair: $($keys[$i])=$left, $($keys[$j])=$right" }
          $pairs++
        }
      }
    }
  }
  return $pairs
}
