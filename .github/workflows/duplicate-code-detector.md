---
name: Duplicate Code Detector
on:
  workflow_dispatch:
  schedule:
    - cron: "0 11 * * *"
permissions:
  contents: read
  actions: read
engine: copilot
imports:
  - shared/serena-mcp.md
safe-outputs:
  create-issue:
    title-prefix: "[duplicate-code] "
    labels: [code-quality, automated-analysis]
timeout_minutes: 15
strict: true
---

# Duplicate Code Detection

Analyze code to identify duplicated patterns using Serena's semantic code analysis capabilities. Report significant findings that require refactoring.

## Task

Detect and report code duplication by:

1. **Analyzing Recent Commits**: Review changes in the latest commits
2. **Detecting Duplicated Code**: Identify similar or duplicated code patterns using semantic analysis
3. **Reporting Findings**: Create a detailed issue if significant duplication is detected (threshold: >10 lines or 3+ similar patterns)

## Context

- **Repository**: ${{ github.repository }}
- **Commit ID**: ${{ github.event.head_commit.id }}
- **Triggered by**: @${{ github.actor }}

## Analysis Workflow

### 1. Project Activation

Activate the project in Serena:
- Use `activate_project` tool with workspace path `/workspace` (mounted repository directory)
- This sets up the semantic code analysis environment

### 2. Changed Files Analysis

Identify and analyze modified files:
- Determine files changed in the recent commits
- Use `get_symbols_overview` to understand file structure
- Use `read_file` to examine modified file contents

### 3. Duplicate Detection

Apply semantic code analysis to find duplicates:

**Symbol-Level Analysis**:
- For significant functions/methods in changed files, use `find_symbol` to search for similarly named symbols
- Use `find_referencing_symbols` to understand usage patterns
- Identify functions with similar names in different files (e.g., `processData` across modules)

**Pattern Search**:
- Use `search_for_pattern` to find similar code patterns
- Search for duplication indicators:
  - Similar function signatures
  - Repeated logic blocks
  - Similar variable naming patterns
  - Near-identical code blocks

**Structural Analysis**:
- Use `list_dir` and `find_file` to identify files with similar names or purposes
- Compare symbol overviews across files for structural similarities

### 4. Duplication Evaluation

Assess findings to identify true code duplication:

**Duplication Types**:
- **Exact Duplication**: Identical code blocks in multiple locations
- **Structural Duplication**: Same logic with minor variations (different variable names, etc.)
- **Functional Duplication**: Different implementations of the same functionality
- **Copy-Paste Programming**: Similar code blocks that could be extracted into shared utilities

**Assessment Criteria**:
- **Severity**: Amount of duplicated code (lines of code, number of occurrences)
- **Impact**: Where duplication occurs (critical paths, frequently called code)
- **Maintainability**: How duplication affects code maintainability
- **Refactoring Opportunity**: Whether duplication can be easily refactored

### 5. Issue Reporting

Create an issue if significant duplication is found (threshold: >10 lines of duplicated code OR 3+ instances of similar patterns):

**Issue Contents**:
- **Executive Summary**: Brief description of duplication found
- **Duplication Details**: Specific locations and code blocks
- **Severity Assessment**: Impact and maintainability concerns
- **Refactoring Recommendations**: Suggested approaches to eliminate duplication
- **Code Examples**: Concrete examples with file paths and line numbers

## Detection Scope

### Report These Issues

- Identical or nearly identical functions in different files
- Repeated code blocks that could be extracted to utilities
- Similar classes or modules with overlapping functionality
- Copy-pasted code with minor modifications
- Duplicated business logic across components

### Skip These Patterns

- Standard boilerplate code (imports, exports, etc.)
- Test setup/teardown code (acceptable duplication in tests)
- Configuration files with similar structure
- Language-specific patterns (constructors, getters/setters)
- Small code snippets (<5 lines) unless highly repetitive

### Analysis Depth

- **Primary Focus**: All files changed in the current push
- **Secondary Analysis**: Check for duplication with existing codebase
- **Cross-Reference**: Look for patterns across the repository
- **Historical Context**: Consider if duplication is new or existing

## Issue Template

If duplication is found, create an issue using this structure:

```markdown
# 🔍 Duplicate Code Detected

*Analysis of commit ${{ github.event.head_commit.id }}*

**Assignee**: @copilot

## Summary

[Brief overview of duplication findings]

## Duplication Details

### Pattern 1: [Description]
- **Severity**: High/Medium/Low
- **Occurrences**: [Number of instances]
- **Locations**:
  - `path/to/file1.ext` (lines X-Y)
  - `path/to/file2.ext` (lines A-B)
- **Code Sample**:
  ```[language]
  [Example of duplicated code]
  ```

### Pattern 2: [Description]
[... additional patterns ...]

## Impact Analysis

- **Maintainability**: [How this affects code maintenance]
- **Bug Risk**: [Potential for inconsistent fixes]
- **Code Bloat**: [Impact on codebase size]

## Refactoring Recommendations

1. **[Recommendation 1]**
   - Extract common functionality to: `suggested/path/utility.ext`
   - Estimated effort: [hours/complexity]
   - Benefits: [specific improvements]

2. **[Recommendation 2]**
   [... additional recommendations ...]

## Implementation Checklist

- [ ] Review duplication findings
- [ ] Prioritize refactoring tasks
- [ ] Create refactoring plan
- [ ] Implement changes
- [ ] Update tests
- [ ] Verify no functionality broken

## Analysis Metadata

- **Analyzed Files**: [count]
- **Detection Method**: Serena semantic code analysis
- **Commit**: ${{ github.event.head_commit.id }}
- **Analysis Date**: [timestamp]
```

## Operational Guidelines

### Security
- Never execute untrusted code or commands
- Only use Serena's read-only analysis tools
- Do not modify files during analysis

### Efficiency
- Focus on recently changed files first
- Use semantic analysis for meaningful duplication, not superficial matches
- Stay within timeout limits (balance thoroughness with execution time)

### Accuracy
- Verify findings before reporting
- Distinguish between acceptable patterns and true duplication
- Consider language-specific idioms and best practices
- Provide specific, actionable recommendations

### Issue Creation
- Only create an issue if significant duplication is found
- Include sufficient detail for SWE agents to understand and act on findings
- Provide concrete examples with file paths and line numbers
- Suggest practical refactoring approaches
- Assign issue to @copilot for automated remediation

## Tool Usage Sequence

1. **Project Setup**: `activate_project` with repository path
2. **File Discovery**: `list_dir`, `find_file` for changed files
3. **Symbol Analysis**: `get_symbols_overview` for structure understanding
4. **Content Review**: `read_file` for detailed code examination
5. **Pattern Matching**: `search_for_pattern` for similar code
6. **Symbol Search**: `find_symbol` for duplicate function names
7. **Reference Analysis**: `find_referencing_symbols` for usage patterns

**Objective**: Improve code quality by identifying and reporting meaningful code duplication that impacts maintainability. Focus on actionable findings that enable automated or manual refactoring.
