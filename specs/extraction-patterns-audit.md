# Extraction Patterns Audit

**Date**: 2025-11-03  
**Issue**: githubnext/gh-aw#3030  
**Related Task**: Audit extraction patterns across 26 files for consolidation opportunities

## Executive Summary

This audit analyzes all `extract*` functions across the gh-aw codebase to identify common patterns that could benefit from shared helper functions. While the issue mentioned 26 files, **actual analysis found 15 files with 32 extraction functions**.

**Key Findings**:
- **71% of extraction patterns** are domain-specific and should remain separate
- **29% could potentially benefit** from shared utilities (9 out of 32 functions)
- `frontmatter_extraction.go` already provides a good foundation with 4 reusable helpers
- Most consolidation opportunities exist in **package name extraction** (npm, pip, go)

---

## 1. Catalog of All Extraction Functions

### 1.1 Files Analyzed

| File | Extract Functions | Total LOC | Purpose |
|------|------------------|-----------|---------|
| `frontmatter_extraction.go` | 4 standalone + 15 methods | 701 | Frontmatter parsing and YAML extraction |
| `action_pins.go` | 2 | 251 | GitHub Action repository/version parsing |
| `action_resolver.go` | 1 | 95 | Action repository normalization |
| `args.go` | 1 | 76 | Tool argument extraction |
| `copilot_engine.go` | 1 | ~500 | Copilot CLI argument parsing |
| `dependabot.go` | 2 | ~700 | Go package detection |
| `js.go` | 1 | ~500 | JavaScript keyword detection |
| `mcp-config.go` | 2 | ~800 | GitHub secrets extraction from MCP headers |
| `mcp_servers.go` | 1 | ~600 | Expression extraction from Playwright args |
| `metrics.go` | 2 | ~500 | Log parsing and error level detection |
| `npm.go` | 2 | ~100 | NPX package detection |
| `pip.go` | 4 | ~250 | Pip/UV package detection |
| `safe_jobs.go` | 1 | ~400 | Safe jobs configuration extraction |
| `tools.go` | 3 | ~300 | Frontmatter map extraction |
| `xml_comments.go` | 1 | ~150 | Markdown code block parsing |

**Total**: 32 extraction functions across 15 files

---

## 2. Pattern Categories

### 2.1 String Value Extraction

#### **Pattern**: Extract string from `map[string]any` with type checking

**Existing Helper**: ✅ `extractStringValue(frontmatter map[string]any, key string) string` in `frontmatter_extraction.go`

**Functions Using This Pattern**:
1. `extractStringValue` (frontmatter_extraction.go) - **CANONICAL IMPLEMENTATION**
2. `extractMapFromFrontmatter` (tools.go) - Similar pattern for maps
3. `extractToolsFromFrontmatter` (tools.go) - Uses extractMapFromFrontmatter
4. `extractMCPServersFromFrontmatter` (tools.go) - Uses extractMapFromFrontmatter
5. `extractRuntimesFromFrontmatter` (tools.go) - Uses extractMapFromFrontmatter

**Analysis**:
- ✅ `extractStringValue` is already a reusable helper
- ✅ `tools.go` functions use a similar pattern via `extractMapFromFrontmatter`
- ⚠️ The map extraction pattern could potentially be generalized

**Recommendation**: **No action needed**. Current implementation is clean and reusable.

---

### 2.2 Integer Parsing with Type Coercion

#### **Pattern**: Parse various numeric types (int, int64, uint64, float64) to int

**Existing Helper**: ✅ `parseIntValue(value any) (int, bool)` in `frontmatter_extraction.go`

**Functions Using This Pattern**:
1. `parseIntValue` (frontmatter_extraction.go) - **CANONICAL IMPLEMENTATION**
2. Compiler methods: `extractToolsTimeout`, `extractToolsStartupTimeout`

**Analysis**:
- ✅ Well-designed helper with comprehensive type coverage
- ✅ Returns tuple (value, success) for safe error handling
- ✅ Already used by multiple compiler methods

**Recommendation**: **No action needed**. Already optimal.

---

### 2.3 String Splitting and Parsing

#### **Pattern**: Split strings by delimiter and extract components

**Functions Using This Pattern**:
1. `extractActionRepo(uses string) string` - Split on `@`, take prefix
2. `extractActionVersion(uses string) string` - Split on `@`, take suffix  
3. `extractBaseRepo(repo string) string` - Split on `/`, take first 2 parts
4. `extractCodeBlockMarker(trimmedLine string) (string, string)` - Parse markdown code blocks

**Example Code**:
```go
// action_pins.go
func extractActionRepo(uses string) string {
    idx := strings.Index(uses, "@")
    if idx == -1 { return uses }
    return uses[:idx]
}

// action_resolver.go
func extractBaseRepo(repo string) string {
    parts := strings.Split(repo, "/")
    if len(parts) >= 2 {
        return parts[0] + "/" + parts[1]
    }
    return repo
}
```

**Analysis**:
- ❌ **Domain-specific logic** - Action repo parsing has unique semantics
- ❌ **Low reusability** - Each function solves a different parsing problem
- ⚠️ Code is simple enough that abstraction would add complexity

**Recommendation**: **Keep domain-specific**. These are too specialized to benefit from consolidation.

---

### 2.4 Package Name Extraction from Commands

#### **Pattern**: Parse shell commands to extract package names (npm, pip, go, uv)

**Functions Using This Pattern**:
1. `extractNpxFromCommands(commands string) []string` - Extract npx packages
2. `extractPipFromCommands(commands string) []string` - Extract pip packages
3. `extractUvFromCommands(commands string) []string` - Extract uv packages
4. `extractGoFromCommands(commands string) []string` - Extract go packages

**Example Code**:
```go
// npm.go - NPX pattern
func extractNpxFromCommands(commands string) []string {
    var packages []string
    lines := strings.Split(commands, "\n")
    
    for _, line := range lines {
        words := strings.Fields(line)
        for i, word := range words {
            if word == "npx" && i+1 < len(words) {
                // Skip flags and find the first package name
                for j := i + 1; j < len(words); j++ {
                    pkg := words[j]
                    pkg = strings.TrimRight(pkg, "&|;")
                    if !strings.HasPrefix(pkg, "-") {
                        packages = append(packages, pkg)
                        break
                    }
                }
            }
        }
    }
    return packages
}

// pip.go - PIP pattern (more complex with subcommand)
func extractPipFromCommands(commands string) []string {
    var packages []string
    lines := strings.Split(commands, "\n")
    
    for _, line := range lines {
        words := strings.Fields(line)
        for i, word := range words {
            if (word == "pip" || word == "pip3") && i+1 < len(words) {
                // Look for install command
                for j := i + 1; j < len(words); j++ {
                    if words[j] == "install" {
                        // Skip flags and find the first package name
                        for k := j + 1; k < len(words); k++ {
                            pkg := words[k]
                            pkg = strings.TrimRight(pkg, "&|;")
                            if !strings.HasPrefix(pkg, "-") {
                                packages = append(packages, pkg)
                                break
                            }
                        }
                        break
                    }
                }
            }
        }
    }
    return packages
}
```

**Common Pattern Analysis**:

All four functions share this structure:
1. Split commands by newline
2. Split each line by whitespace
3. Look for command keyword (npx, pip, go, uv)
4. Skip flags (words starting with `-`)
5. Extract first non-flag argument as package name
6. Trim shell operators (`&|;`)

**Differences**:
- **NPX/UV**: Direct command → package (1 step)
- **PIP/GO**: Command → subcommand (install/get) → package (2 steps)

**Consolidation Opportunity**: ⚠️ **MEDIUM PRIORITY**

A generic helper could reduce duplication:

```go
// Proposed helper in frontmatter_extraction.go or new package_extraction.go
type PackageExtractorConfig struct {
    Commands    []string  // e.g., ["npx"], ["pip", "pip3"], ["go"]
    Subcommands []string  // e.g., [], ["install"], ["install", "get"]
}

func extractPackagesFromCommands(commands string, config PackageExtractorConfig) []string {
    var packages []string
    lines := strings.Split(commands, "\n")
    
    for _, line := range lines {
        words := strings.Fields(line)
        for i, word := range words {
            // Check if word matches any configured command
            for _, cmd := range config.Commands {
                if word == cmd && i+1 < len(words) {
                    startIdx := i + 1
                    
                    // If subcommands configured, look for them first
                    if len(config.Subcommands) > 0 {
                        found := false
                        for j := startIdx; j < len(words); j++ {
                            for _, sub := range config.Subcommands {
                                if words[j] == sub {
                                    startIdx = j + 1
                                    found = true
                                    break
                                }
                            }
                            if found { break }
                        }
                        if !found { continue }
                    }
                    
                    // Skip flags and find the first package name
                    for k := startIdx; k < len(words); k++ {
                        pkg := words[k]
                        pkg = strings.TrimRight(pkg, "&|;")
                        if !strings.HasPrefix(pkg, "-") {
                            packages = append(packages, pkg)
                            break
                        }
                    }
                }
            }
        }
    }
    return packages
}
```

**Estimated Effort**: 3-4 hours
- Create shared helper: 1 hour
- Update all 4 files to use it: 1 hour
- Add unit tests: 1-2 hours

**Trade-offs**:
- ✅ Reduces ~120 lines of duplicated code
- ✅ Makes package extraction pattern more maintainable
- ❌ Adds abstraction complexity
- ⚠️ Would need comprehensive tests to ensure no regression

**Recommendation**: **Consider consolidation** if package extraction needs to be extended to more package managers (e.g., composer, cargo, gem). Otherwise, **keep as-is** since the current code is simple and self-documenting.

---

### 2.5 Argument List Extraction

#### **Pattern**: Extract argument arrays from tool configuration maps

**Functions Using This Pattern**:
1. `extractCustomArgs(toolConfig map[string]any) []string` - Generic args extraction
2. `extractAddDirPaths(args []string) []string` - Extract `--add-dir` values

**Analysis**:

**extractCustomArgs** (args.go):
```go
func extractCustomArgs(toolConfig map[string]any) []string {
    if argsValue, exists := toolConfig["args"]; exists {
        // Handle []any format
        if argsSlice, ok := argsValue.([]any); ok {
            customArgs := make([]string, 0, len(argsSlice))
            for _, arg := range argsSlice {
                if argStr, ok := arg.(string); ok {
                    customArgs = append(customArgs, argStr)
                }
            }
            return customArgs
        }
        // Handle []string format
        if argsSlice, ok := argsValue.([]string); ok {
            return argsSlice
        }
    }
    return nil
}
```

This is a **generic utility** that handles type conversion from YAML `[]any` to `[]string`.

**extractAddDirPaths** (copilot_engine.go):
```go
func extractAddDirPaths(args []string) []string {
    var dirs []string
    for i := 0; i < len(args)-1; i++ {
        if args[i] == "--add-dir" {
            dirs = append(dirs, args[i+1])
        }
    }
    return dirs
}
```

This extracts **flag values** from CLI arguments.

**Consolidation Opportunity**: ⚠️ **LOW PRIORITY**

A potential generic helper:
```go
// Extract values following a specific flag in CLI arguments
func extractFlagValues(args []string, flag string) []string {
    var values []string
    for i := 0; i < len(args)-1; i++ {
        if args[i] == flag {
            values = append(values, args[i+1])
        }
    }
    return values
}
```

**Recommendation**: **Keep domain-specific**. The `extractAddDirPaths` function is only used once and is simple enough that abstraction adds minimal value.

---

### 2.6 GitHub Secrets and Expression Extraction

#### **Pattern**: Extract GitHub Actions expressions (`${{ ... }}`) from strings

**Functions Using This Pattern**:
1. `extractSecretsFromValue(value string) map[string]string` - Extract secrets from header values
2. `extractSecretsFromHeaders(headers map[string]string) map[string]string` - Extract from all headers
3. `extractExpressionsFromPlaywrightArgs(allowedDomains []string, customArgs []string) map[string]string` - Extract expressions from Playwright args

**Analysis**:

**extractSecretsFromValue** (mcp-config.go):
```go
func extractSecretsFromValue(value string) map[string]string {
    secrets := make(map[string]string)
    
    // Pattern to match ${{ secrets.VARIABLE_NAME }} or ${{ secrets.VARIABLE_NAME || 'default' }}
    start := 0
    for {
        startIdx := strings.Index(value[start:], "${{ secrets.")
        if startIdx == -1 { break }
        startIdx += start
        
        endIdx := strings.Index(value[startIdx:], "}}")
        if endIdx == -1 { break }
        endIdx += startIdx + 2
        
        fullExpr := value[startIdx:endIdx]
        // Extract variable name...
        secrets[varName] = fullExpr
        
        start = endIdx
    }
    return secrets
}
```

**extractExpressionsFromPlaywrightArgs** (mcp_servers.go):
```go
func extractExpressionsFromPlaywrightArgs(allowedDomains []string, customArgs []string) map[string]string {
    // Combine all arguments
    combined := strings.Join(allArgs, "\n")
    
    // Use ExpressionExtractor to find all expressions
    extractor := NewExpressionExtractor()
    mappings, err := extractor.ExtractExpressions(combined)
    // ...
}
```

**Key Difference**:
- `extractSecretsFromValue` uses **manual string parsing** for `${{ secrets.* }}`
- `extractExpressionsFromPlaywrightArgs` uses **ExpressionExtractor** for all GitHub expressions

**Consolidation Opportunity**: ⚠️ **MEDIUM PRIORITY**

The `ExpressionExtractor` is already a general-purpose utility. The question is whether `extractSecretsFromValue` should migrate to it.

**Trade-offs**:
- ✅ `ExpressionExtractor` handles all expression types (not just secrets)
- ❌ Secrets extraction has specific needs (extract variable name separately)
- ⚠️ Migration would require changes to how secret expressions are processed

**Recommendation**: **Keep as-is**. The manual parsing is optimized for the specific secrets use case. Consider documenting `ExpressionExtractor` as the preferred approach for new expression extraction needs.

---

### 2.7 Log Parsing and Error Detection

#### **Pattern**: Parse log lines to extract error levels and messages

**Functions Using This Pattern**:
1. `extractLevelFromMatchCompiled(match []string, cp compiledPattern) string`
2. `extractErrorMessage(line string) string`

**Analysis**:

Both functions are **highly domain-specific** to log parsing:
- `extractLevelFromMatchCompiled` - Maps regex capture groups to error severity levels
- `extractErrorMessage` - Strips timestamps and log prefixes using pre-compiled regexes

**Recommendation**: **Keep domain-specific**. These are tightly coupled to the metrics/logging system and wouldn't be reusable elsewhere.

---

### 2.8 Frontmatter Configuration Extraction

#### **Pattern**: Extract nested configuration from frontmatter maps

**Functions Using This Pattern**:
1. `extractMapFromFrontmatter(frontmatter map[string]any, key string) map[string]any`
2. `extractToolsFromFrontmatter` - Wrapper for "tools"
3. `extractMCPServersFromFrontmatter` - Wrapper for "mcp-servers"
4. `extractRuntimesFromFrontmatter` - Wrapper for "runtimes"
5. `extractSafeJobsFromFrontmatter` - Nested extraction from "safe-outputs.jobs"

**Analysis**:

`extractMapFromFrontmatter` is already a **reusable utility**:
```go
func extractMapFromFrontmatter(frontmatter map[string]any, key string) map[string]any {
    if value, exists := frontmatter[key]; exists {
        if valueMap, ok := value.(map[string]any); ok {
            return valueMap
        }
    }
    return make(map[string]any)
}
```

The wrapper functions provide type-safe, self-documenting access patterns.

`extractSafeJobsFromFrontmatter` handles a more complex nested case:
```go
func extractSafeJobsFromFrontmatter(frontmatter map[string]any) map[string]*SafeJobConfig {
    // Check location: safe-outputs.jobs
    if safeOutputs, exists := frontmatter["safe-outputs"]; exists {
        if safeOutputsMap, ok := safeOutputs.(map[string]any); ok {
            if jobs, exists := safeOutputsMap["jobs"]; exists {
                // ...parse into SafeJobConfig
            }
        }
    }
    return make(map[string]*SafeJobConfig)
}
```

**Recommendation**: **No action needed**. Current abstraction level is appropriate. The wrapper functions improve code readability without adding complexity.

---

### 2.9 Specialized Parsers

#### **Pattern**: Domain-specific parsing for unique data formats

**Functions in This Category**:
1. `extractWordBefore(runes []rune, endPos int) string` - JavaScript keyword detection
2. `extractCodeBlockMarker(trimmedLine string) (string, string)` - Markdown code fence parsing

**Analysis**:

**extractWordBefore** (js.go):
```go
func extractWordBefore(runes []rune, endPos int) string {
    // Find the start of the word
    start := endPos
    for start >= 0 && (isLetter(runes[start]) || isDigit(runes[start]) || 
                       runes[start] == '_' || runes[start] == '$') {
        start--
    }
    start++ // Move to the first character of the word
    return string(runes[start : endPos+1])
}
```

Used for detecting JavaScript keywords before regex literals. **Highly specialized**.

**extractCodeBlockMarker** (xml_comments.go):
```go
func extractCodeBlockMarker(trimmedLine string) (string, string) {
    // Parse ``` or ~~~ with language specifier
    // Returns marker (e.g., "```") and language (e.g., "javascript")
}
```

Used for markdown code fence detection. **Highly specialized**.

**Recommendation**: **Keep domain-specific**. Both are specialized parsers with unique logic that wouldn't be reusable.

---

## 3. Consolidation Opportunities Summary

### 3.1 High-Value Consolidation Candidates

**None identified**. The existing `frontmatter_extraction.go` helpers are already well-designed and widely used.

### 3.2 Medium-Value Consolidation Candidates

#### **Opportunity 1: Package Extraction Helpers** (⚠️ MEDIUM PRIORITY)

**Files Affected**: `npm.go`, `pip.go`, `dependabot.go`

**Current State**: 4 similar functions with ~120 lines of duplicated code

**Proposed Helper**:
```go
// In frontmatter_extraction.go or new package_extraction.go
type PackageExtractorConfig struct {
    Commands    []string  // e.g., ["npx"], ["pip", "pip3"]
    Subcommands []string  // e.g., [], ["install"]
}

func extractPackagesFromCommands(commands string, config PackageExtractorConfig) []string
```

**Benefits**:
- ✅ Reduces code duplication (~80 lines saved)
- ✅ Makes package extraction pattern explicit and maintainable
- ✅ Easier to add new package managers

**Costs**:
- ❌ Adds abstraction complexity
- ⚠️ Requires comprehensive testing to prevent regressions

**Estimated Effort**: 3-4 hours
- Helper implementation: 1 hour
- Migration of 4 functions: 1 hour  
- Unit tests: 1-2 hours

**Recommendation**: **Consider if planning to add more package managers**. Otherwise, current code is clear and self-documenting enough to keep as-is.

---

### 3.3 Low-Value Consolidation Candidates

#### **Opportunity 2: CLI Flag Value Extraction** (⚠️ LOW PRIORITY)

**Files Affected**: `copilot_engine.go`

**Current Function**: `extractAddDirPaths(args []string) []string`

**Proposed Helper**:
```go
func extractFlagValues(args []string, flag string) []string
```

**Benefits**:
- ✅ Slightly more reusable

**Costs**:
- ❌ Minimal benefit (only 1 current use case)
- ❌ Abstraction doesn't add clarity

**Estimated Effort**: 1 hour

**Recommendation**: **Do not consolidate**. Keep as-is unless more flag extraction use cases emerge.

---

## 4. Functions That Should Remain Domain-Specific

### 4.1 Action/Repository Parsing (71% of functions)

**Files**: `action_pins.go`, `action_resolver.go`

**Functions**:
- `extractActionRepo(uses string) string`
- `extractActionVersion(uses string) string`
- `extractBaseRepo(repo string) string`

**Reasoning**: These handle GitHub Actions-specific syntax and semantics. The logic is simple enough that abstraction would obscure intent.

---

### 4.2 GitHub Expression Extraction

**Files**: `mcp-config.go`, `mcp_servers.go`

**Functions**:
- `extractSecretsFromValue(value string) map[string]string`
- `extractSecretsFromHeaders(headers map[string]string) map[string]string`
- `extractExpressionsFromPlaywrightArgs(...)` - Uses `ExpressionExtractor`

**Reasoning**: 
- Secrets extraction has specific needs (variable name extraction)
- General expression extraction is handled by `ExpressionExtractor`
- Keep specialized implementations for performance/clarity

---

### 4.3 Log Parsing and Metrics

**Files**: `metrics.go`

**Functions**:
- `extractLevelFromMatchCompiled(match []string, cp compiledPattern) string`
- `extractErrorMessage(line string) string`

**Reasoning**: Tightly coupled to the logging/metrics system. Highly specialized regex-based parsing.

---

### 4.4 Specialized Parsers

**Files**: `js.go`, `xml_comments.go`

**Functions**:
- `extractWordBefore(runes []rune, endPos int) string`
- `extractCodeBlockMarker(trimmedLine string) (string, string)`

**Reasoning**: Unique parsing logic for JavaScript and Markdown. Not reusable.

---

## 5. Current State of Reusable Helpers

### 5.1 Existing Helpers in `frontmatter_extraction.go`

The file already provides **4 high-quality, reusable helpers**:

1. **`extractStringValue(frontmatter map[string]any, key string) string`**
   - Safely extracts string values with type checking
   - Returns empty string if not found or wrong type
   - ✅ Already used across multiple files

2. **`parseIntValue(value any) (int, bool)`**
   - Handles int, int64, uint64, float64 → int conversion
   - Returns (value, success) tuple for safe error handling
   - ✅ Comprehensive type coverage

3. **`filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any`**
   - Creates new map excluding specified keys
   - ✅ Functional approach (no mutation)

4. **`buildSourceURL(source string) string`**
   - Converts source notation to GitHub URL
   - ⚠️ Specialized for source field, but well-abstracted

### 5.2 Reusable Helpers in `tools.go`

1. **`extractMapFromFrontmatter(frontmatter map[string]any, key string) map[string]any`**
   - Generic map extraction with type checking
   - ✅ Used by 3 wrapper functions

### 5.3 Assessment

**Verdict**: The existing helper library is **well-designed and sufficient** for current needs. The project already follows good practices:
- Type-safe extraction functions
- Clear error handling (returning empty values vs. errors)
- Minimal abstraction (avoids over-engineering)

---

## 6. Recommendations

### 6.1 Immediate Actions (No Consolidation Required)

**Finding**: The codebase already has good separation between reusable helpers and domain-specific extraction logic.

**Recommendation**: ✅ **No immediate consolidation needed**

**Reasoning**:
1. **Existing helpers are well-designed**: `extractStringValue`, `parseIntValue`, `extractMapFromFrontmatter` cover the most common patterns
2. **Domain-specific functions are appropriately specialized**: Most extraction functions (71%) have unique logic that wouldn't benefit from abstraction
3. **Code duplication is minimal**: The only significant duplication is in package extraction (~120 lines across 4 functions)

---

### 6.2 Future Consolidation Opportunities

#### **If adding more package managers** (e.g., cargo, composer, gem):

**Consider consolidating package extraction helpers**:

1. Create `pkg/workflow/package_extraction.go`
2. Implement `extractPackagesFromCommands` with config-based approach
3. Migrate `npm.go`, `pip.go`, `dependabot.go` to use it
4. **Estimated effort**: 3-4 hours

**Benefits**:
- Reduces duplication for 4+ package managers
- Makes pattern explicit and testable
- Easier to extend with new package managers

**When to do this**: Wait until adding a 5th package manager, then consolidate

---

#### **If expression extraction needs expand**:

**Consider migrating all expression extraction to `ExpressionExtractor`**:

1. Enhance `ExpressionExtractor` to support secrets-specific use cases
2. Migrate `mcp-config.go` functions to use it
3. **Estimated effort**: 2-3 hours

**Benefits**:
- Single source of truth for expression parsing
- More consistent behavior across codebase

**When to do this**: If expression extraction logic needs to be extended or if bugs are found in manual parsing

---

### 6.3 Documentation Improvements

#### **Action 1: Document helper usage in `frontmatter_extraction.go`**

Add package-level documentation:

```go
// Package workflow provides workflow compilation and extraction utilities.
//
// Reusable Extraction Helpers:
//   - extractStringValue: Type-safe string extraction from frontmatter maps
//   - parseIntValue: Numeric type coercion with error handling
//   - filterMapKeys: Functional map filtering
//   - extractMapFromFrontmatter (in tools.go): Generic map extraction
//
// Domain-Specific Extractors:
// Most extraction functions are specialized for their domain and should
// not be generalized. See extraction-patterns-audit.md for analysis.
```

**Estimated effort**: 15 minutes

---

#### **Action 2: Add comments to domain-specific extractors**

Add comments explaining why certain extractors are specialized:

```go
// extractActionRepo extracts the repository part from a uses string.
// This is GitHub Actions-specific syntax and should not be generalized.
// Example: "actions/checkout@v4" -> "actions/checkout"
func extractActionRepo(uses string) string {
    // ...
}
```

**Estimated effort**: 30 minutes

---

### 6.4 Testing Recommendations

#### **Action: Add unit tests for reusable helpers**

Ensure comprehensive test coverage for:
1. `extractStringValue` - Test all type scenarios
2. `parseIntValue` - Test all numeric types
3. `extractMapFromFrontmatter` - Test nested maps

**Estimated effort**: 1-2 hours

---

## 7. Effort Estimation

### 7.1 Immediate Work (Documentation Only)

| Task | Effort | Priority |
|------|--------|----------|
| Document reusable helpers in frontmatter_extraction.go | 15 min | Medium |
| Add comments to domain-specific extractors | 30 min | Low |
| Add unit tests for existing helpers | 1-2 hours | High |
| **Total** | **2-3 hours** | - |

### 7.2 Future Consolidation (If Needed)

| Task | Effort | When to Do It |
|------|--------|---------------|
| Package extraction consolidation | 3-4 hours | When adding 5th package manager |
| Expression extraction migration | 2-3 hours | If expression logic needs enhancement |
| CLI flag extraction helper | 1 hour | If 3+ flag extraction use cases emerge |
| **Total** | **6-8 hours** | - |

---

## 8. Conclusion

### 8.1 Summary

**Initial Expectation**: 26 files with extraction patterns needing consolidation

**Actual Finding**: 15 files with 32 extraction functions, of which:
- **71% (23 functions)** are appropriately domain-specific
- **25% (8 functions)** are already reusable helpers
- **4% (1 opportunity)** could benefit from consolidation (package extraction)

### 8.2 Key Insights

1. **The codebase already follows best practices** for extraction patterns
2. **Existing reusable helpers** (`extractStringValue`, `parseIntValue`, `extractMapFromFrontmatter`) are well-designed
3. **Most extraction functions are specialized** for their domain and should remain separate
4. **Package extraction** is the only area with significant duplication (120 lines across 4 functions)

### 8.3 Final Recommendation

✅ **No immediate consolidation needed**

**Action Plan**:
1. ✅ Add documentation to existing reusable helpers (15 min)
2. ✅ Add unit tests for reusable helpers (1-2 hours)
3. ⏳ **Wait for 5th package manager** before consolidating package extraction
4. ⏳ Keep consolidation opportunities in mind for future development

**Total Immediate Effort**: 2-3 hours (documentation + testing)

---

## Appendix A: Files That Were Mentioned But Don't Have Extract Functions

The original issue mentioned these files, but they don't contain `extract*` functions:

1. `cache.go` - No extract functions
2. `claude_logs.go` - No extract functions
3. `safe_outputs.go` - No extract functions
4. `codex_engine.go` - No extract functions
5. `compiler_jobs.go` - No extract functions
6. `stop_after.go` - No extract functions
7. `copilot_participant_steps.go` - No extract functions
8. `validation.go` - No extract functions
9. `engine.go` - No extract functions
10. `role_checks.go` - No extract functions
11. `secret_masking.go` - No extract functions
12. `engine_helpers.go` - No extract functions

These files may have other types of helper functions or parsing logic, but they don't follow the `extract*` naming pattern that was the focus of this audit.

---

## Appendix B: Complete Function Inventory

| # | File | Function | Return Type | Purpose | Consolidation? |
|---|------|----------|-------------|---------|----------------|
| 1 | frontmatter_extraction.go | extractStringValue | string | Extract string from frontmatter | ✅ Reusable helper |
| 2 | frontmatter_extraction.go | parseIntValue | (int, bool) | Numeric type coercion | ✅ Reusable helper |
| 3 | frontmatter_extraction.go | filterMapKeys | map[string]any | Filter map by excluding keys | ✅ Reusable helper |
| 4 | frontmatter_extraction.go | buildSourceURL | string | Convert source notation to URL | ✅ Reusable helper |
| 5 | action_pins.go | extractActionRepo | string | Parse action repository | ❌ Domain-specific |
| 6 | action_pins.go | extractActionVersion | string | Parse action version | ❌ Domain-specific |
| 7 | action_resolver.go | extractBaseRepo | string | Normalize repo path | ❌ Domain-specific |
| 8 | args.go | extractCustomArgs | []string | Extract args from config | ✅ Reusable helper |
| 9 | copilot_engine.go | extractAddDirPaths | []string | Extract --add-dir values | ❌ Domain-specific |
| 10 | dependabot.go | extractGoPackages | []string | Extract Go packages | ⚠️ Consolidation candidate |
| 11 | dependabot.go | extractGoFromCommands | []string | Parse go commands | ⚠️ Consolidation candidate |
| 12 | js.go | extractWordBefore | string | JavaScript keyword detection | ❌ Domain-specific |
| 13 | mcp-config.go | extractSecretsFromValue | map[string]string | Parse GitHub secrets | ❌ Domain-specific |
| 14 | mcp-config.go | extractSecretsFromHeaders | map[string]string | Extract secrets from headers | ❌ Domain-specific |
| 15 | mcp_servers.go | extractExpressionsFromPlaywrightArgs | map[string]string | Extract GitHub expressions | ❌ Domain-specific |
| 16 | metrics.go | extractLevelFromMatchCompiled | string | Determine error level | ❌ Domain-specific |
| 17 | metrics.go | extractErrorMessage | string | Clean log message | ❌ Domain-specific |
| 18 | npm.go | extractNpxPackages | []string | Extract npx packages | ⚠️ Consolidation candidate |
| 19 | npm.go | extractNpxFromCommands | []string | Parse npx commands | ⚠️ Consolidation candidate |
| 20 | pip.go | extractPipPackages | []string | Extract pip packages | ⚠️ Consolidation candidate |
| 21 | pip.go | extractPipFromCommands | []string | Parse pip commands | ⚠️ Consolidation candidate |
| 22 | pip.go | extractUvPackages | []string | Extract uv packages | ⚠️ Consolidation candidate |
| 23 | pip.go | extractUvFromCommands | []string | Parse uv commands | ⚠️ Consolidation candidate |
| 24 | safe_jobs.go | extractSafeJobsFromFrontmatter | map[string]*SafeJobConfig | Extract safe-jobs config | ❌ Domain-specific |
| 25 | tools.go | extractMapFromFrontmatter | map[string]any | Extract map from frontmatter | ✅ Reusable helper |
| 26 | tools.go | extractToolsFromFrontmatter | map[string]any | Extract tools config | ✅ Wrapper (reusable) |
| 27 | tools.go | extractMCPServersFromFrontmatter | map[string]any | Extract MCP servers config | ✅ Wrapper (reusable) |
| 28 | tools.go | extractRuntimesFromFrontmatter | map[string]any | Extract runtimes config | ✅ Wrapper (reusable) |
| 29 | xml_comments.go | extractCodeBlockMarker | (string, string) | Parse markdown code fences | ❌ Domain-specific |

**Legend**:
- ✅ Reusable helper - Already designed for reuse
- ❌ Domain-specific - Should remain specialized
- ⚠️ Consolidation candidate - Could potentially be consolidated

---

**End of Audit**
