---
description: Detects inconsistencies between JSON schema, implementation code, and documentation
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  issues: read
  pull-requests: read
engine:
  id: claude
  max-turns: 60
tools:
  edit:
  bash: ["*"]
  github:
    mode: remote
    toolsets: [default, discussions]
  cache-memory:
    key: schema-consistency-cache-${{ github.workflow }}
timeout-minutes: 30
checkout:
  - fetch-depth: 1
    current: true
imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[Schema Consistency] "
      expires: 1d
pre-agent-steps:
  - name: Pre-compute schema analysis data
    run: |
      set -e
      mkdir -p /tmp/gh-aw/agent

      echo "=== Extracting schema fields ==="

      # 1. All top-level fields in the main JSON schema
      SCHEMA_FIELDS=$(jq -r '.properties | keys[]' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null | sort -u || echo "")

      # 2. yaml-tagged struct fields in pkg/parser/*.go
      PARSER_YAML_FIELDS=$(grep -rh 'yaml:"' pkg/parser/*.go 2>/dev/null \
        | grep -o 'yaml:"[^"]*"' \
        | sed 's/yaml:"//;s/"//' \
        | sed 's/,omitempty//' \
        | sed 's/,.*$//' \
        | grep -v '^-$' \
        | grep -v '^$' \
        | sort -u || echo "")

      # 3. yaml-tagged struct fields in pkg/workflow/*.go
      WORKFLOW_YAML_FIELDS=$(grep -rh 'yaml:"' pkg/workflow/*.go 2>/dev/null \
        | grep -o 'yaml:"[^"]*"' \
        | sed 's/yaml:"//;s/"//' \
        | sed 's/,omitempty//' \
        | sed 's/,.*$//' \
        | grep -v '^-$' \
        | grep -v '^$' \
        | sort -u || echo "")

      # 4. Top-level frontmatter keys actually used in workflow .md files
      USED_FIELDS=$(grep -rh '^[a-z][a-z0-9_-]*:' .github/workflows/*.md 2>/dev/null \
        | sed 's/:.*//' \
        | grep -v '^#' \
        | sort -u || echo "")

      # 5. Schema field types for all top-level fields
      FIELD_TYPES=$(jq -r '.properties | to_entries[] |
        "\(.key): \(.value.type // (.value.anyOf // .value.oneOf // [] | map(.type // "complex") | unique | join("|")) // "complex")"' \
        pkg/parser/schemas/main_workflow_schema.json 2>/dev/null | sort || echo "")

      # 6. Fields in schema but absent as yaml tags in parser structs
      IN_SCHEMA_NOT_PARSER=$(comm -23 \
        <(echo "$SCHEMA_FIELDS") \
        <(echo "$PARSER_YAML_FIELDS" | sort -u) 2>/dev/null || echo "")

      # 7. yaml tags in parser structs absent from schema
      IN_PARSER_NOT_SCHEMA=$(comm -23 \
        <(echo "$PARSER_YAML_FIELDS" | sort -u) \
        <(echo "$SCHEMA_FIELDS") 2>/dev/null || echo "")

      # 8. Fields in schema but absent from workflow compiler structs
      IN_SCHEMA_NOT_WORKFLOW=$(comm -23 \
        <(echo "$SCHEMA_FIELDS") \
        <(echo "$WORKFLOW_YAML_FIELDS" | sort -u) 2>/dev/null || echo "")

      # 9. Fields used in actual workflow .md files but not in schema
      IN_USED_NOT_SCHEMA=$(comm -23 \
        <(echo "$USED_FIELDS" | sort -u) \
        <(echo "$SCHEMA_FIELDS") 2>/dev/null || echo "")

      # Write JSON output
      jq -n \
        --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg schema_fields "$SCHEMA_FIELDS" \
        --arg parser_yaml_fields "$PARSER_YAML_FIELDS" \
        --arg workflow_yaml_fields "$WORKFLOW_YAML_FIELDS" \
        --arg used_in_workflows "$USED_FIELDS" \
        --arg field_types "$FIELD_TYPES" \
        --arg in_schema_not_parser "$IN_SCHEMA_NOT_PARSER" \
        --arg in_parser_not_schema "$IN_PARSER_NOT_SCHEMA" \
        --arg in_schema_not_workflow "$IN_SCHEMA_NOT_WORKFLOW" \
        --arg in_used_not_schema "$IN_USED_NOT_SCHEMA" \
        '{
          generated_at: $generated_at,
          schema_fields: ($schema_fields | split("\n") | map(select(. != ""))),
          parser_yaml_fields: ($parser_yaml_fields | split("\n") | map(select(. != ""))),
          workflow_yaml_fields: ($workflow_yaml_fields | split("\n") | map(select(. != ""))),
          used_in_workflows: ($used_in_workflows | split("\n") | map(select(. != ""))),
          field_types: ($field_types | split("\n") | map(select(. != ""))),
          field_gaps: {
            in_schema_not_parser: ($in_schema_not_parser | split("\n") | map(select(. != ""))),
            in_parser_not_schema: ($in_parser_not_schema | split("\n") | map(select(. != ""))),
            in_schema_not_workflow: ($in_schema_not_workflow | split("\n") | map(select(. != ""))),
            in_used_not_schema: ($in_used_not_schema | split("\n") | map(select(. != "")))
          }
        }' > /tmp/gh-aw/agent/schema-diff.json

      echo "✓ Schema diff written to /tmp/gh-aw/agent/schema-diff.json"
      echo "Summary:"
      jq '{
        schema_field_count: (.schema_fields | length),
        parser_yaml_field_count: (.parser_yaml_fields | length),
        workflow_yaml_field_count: (.workflow_yaml_fields | length),
        gaps: {
          in_schema_not_parser: (.field_gaps.in_schema_not_parser | length),
          in_parser_not_schema: (.field_gaps.in_parser_not_schema | length),
          in_schema_not_workflow: (.field_gaps.in_schema_not_workflow | length),
          in_used_not_schema: (.field_gaps.in_used_not_schema | length)
        }
      }' /tmp/gh-aw/agent/schema-diff.json
---
# Schema Consistency Checker

You are an expert system that detects inconsistencies between:
- The main JSON schema of the frontmatter (`pkg/parser/schemas/main_workflow_schema.json`)
- The parser and compiler implementation (`pkg/parser/*.go` and `pkg/workflow/*.go`)
- The documentation (`docs/src/content/docs/**/*.md`)
- The workflows in the project (`.github/workflows/*.md`)

## Mission

Analyze the repository to find inconsistencies across these four key areas and create a discussion report with actionable findings.

## Cache Memory Strategy Storage

Use the cache memory folder at `/tmp/gh-aw/cache-memory/` to store and reuse successful analysis strategies:

1. **Read Previous Strategies**: Check `/tmp/gh-aw/cache-memory/strategies.json` for previously successful detection methods
2. **Strategy Selection**: 
   - 70% of the time: Use a proven strategy from the cache
   - 30% of the time: Try a radically different approach to discover new inconsistencies
   - Implementation: Use the day of year (e.g., `date +%j`) modulo 10 to determine selection: values 0-6 use proven strategies, 7-9 try new approaches
3. **Update Strategy Database**: After analysis, save successful strategies to `/tmp/gh-aw/cache-memory/strategies.json`

Strategy database structure:
```json
{
  "strategies": [
    {
      "id": "strategy-1",
      "name": "Schema field enumeration check",
      "description": "Compare schema enum values with parser constants",
      "success_count": 5,
      "last_used": "2024-01-15",
      "findings": 3
    }
  ],
  "last_updated": "2024-01-15"
}
```

## Analysis Areas

### 1. Schema vs Parser Implementation

**Check for:**
- Fields defined in schema but not handled in parser/compiler
- Fields handled in parser/compiler but missing from schema
- Type mismatches (schema says `string`, parser expects `object`)
- Enum values in schema not validated in parser/compiler
- Required fields not enforced
- Default values inconsistent between schema and parser/compiler

**Key files to analyze:**
- `pkg/parser/schemas/main_workflow_schema.json`
- `pkg/parser/schemas/mcp_config_schema.json`
- `pkg/parser/frontmatter.go` and `pkg/parser/*.go`
- `pkg/workflow/compiler.go` - main workflow compiler
- `pkg/workflow/tools.go` - tools configuration processing
- `pkg/workflow/safe_outputs.go` - safe-outputs configuration
- `pkg/workflow/cache.go` - cache and cache-memory configuration
- `pkg/workflow/permissions.go` - permissions processing
- `pkg/workflow/engine.go` - engine config and network permissions types
- `pkg/workflow/domains.go` - network domain allowlist functions
- `pkg/workflow/engine_network_hooks.go` - network hook generation
- `pkg/workflow/engine_firewall_support.go` - firewall support checking
- `pkg/workflow/strict_mode.go` - strict mode validation
- `pkg/workflow/stop_after.go` - stop-after processing
- `pkg/workflow/safe_jobs.go` - safe-jobs configuration (internal - accessed via safe-outputs.jobs)
- `pkg/workflow/runtime_setup.go` - runtime overrides
- `pkg/workflow/github_token.go` - github-token configuration
- `pkg/workflow/*.go` (all workflow processing files that use frontmatter)

### 2. Schema vs Documentation

**Check for:**
- Schema fields not documented
- Documented fields not in schema
- Type descriptions mismatch
- Example values that violate schema
- Missing or outdated examples
- Enum values documented but not in schema

**Key files to analyze:**
- `docs/src/content/docs/reference/frontmatter.md`
- `docs/src/content/docs/reference/frontmatter-full.md`
- `docs/src/content/docs/reference/*.md` (all reference docs)

### 3. Schema vs Actual Workflows

**Check for:**
- Workflows using fields not in schema
- Workflows using deprecated fields
- Invalid field values according to schema
- Missing required fields
- Type violations in actual usage
- Undocumented field combinations

**Key files to analyze:**
- `.github/workflows/*.md` (all workflow files)
- `.github/workflows/shared/**/*.md` (shared components)

### 4. Parser vs Documentation

**Check for:**
- Parser/compiler features not documented
- Documented features not implemented in parser/compiler
- Error messages that don't match docs
- Validation rules not documented

**Focus on:**
- `pkg/parser/*.go` - frontmatter parsing
- `pkg/workflow/*.go` - workflow compilation and feature processing

## Detection Strategies

Here are proven strategies you can use or build upon:

### Strategy 1: Field Enumeration Diff
1. Extract all field names from schema
2. Extract all field names from parser code (look for YAML tags, map keys)
3. Extract all field names from documentation
4. Compare and find missing/extra fields

### Strategy 2: Type Analysis
1. For each field in schema, note its type
2. Search parser for how that field is processed
3. Check if types match
4. Report type mismatches

### Strategy 3: Enum Validation
1. Extract enum values from schema
2. Search for those enums in parser validation
3. Check if all enum values are handled
4. Find undocumented enum values

### Strategy 4: Example Validation
1. Extract code examples from documentation
2. Validate each example against the schema
3. Report examples that don't validate
4. Suggest corrections

### Strategy 5: Real-World Usage Analysis
1. Parse all workflow files in the repo
2. Extract frontmatter configurations
3. Check each against schema
4. Find patterns that work but aren't in schema (potential missing features)

### Strategy 6: Grep-Based Pattern Detection
1. Use bash/grep to find specific patterns
2. Example: `grep -r "type.*string" pkg/parser/schemas/ | grep engine`
3. Cross-reference with parser implementation

## Implementation Steps

### Step 0: Read Pre-Computed Data (Start Here)

Before doing anything else, read the schema diff that was computed before your session began:

```bash
cat /tmp/gh-aw/agent/schema-diff.json
```

This file contains:
- `schema_fields`: All top-level field names in the main JSON schema
- `parser_yaml_fields`: All yaml-tagged struct fields in `pkg/parser/*.go`
- `workflow_yaml_fields`: All yaml-tagged struct fields in `pkg/workflow/*.go`
- `used_in_workflows`: All top-level frontmatter keys used in `.github/workflows/*.md`
- `field_types`: Schema field types for all top-level fields
- `field_gaps.in_schema_not_parser`: Fields in schema absent from parser yaml tags
- `field_gaps.in_parser_not_schema`: Fields as parser yaml tags absent from schema
- `field_gaps.in_schema_not_workflow`: Fields in schema absent from workflow compiler yaml tags
- `field_gaps.in_used_not_schema`: Fields used in workflow files but not in schema

**Use this pre-computed data as your primary starting point.** Do NOT re-run the field enumeration commands from scratch — instead, refine and supplement the pre-computed data with targeted follow-up queries (e.g., checking a specific file for a specific field).

### Step 1: Load Previous Strategies
```bash
# Check if strategies file exists
if [ -f /tmp/gh-aw/cache-memory/strategies.json ]; then
  cat /tmp/gh-aw/cache-memory/strategies.json
fi
```

### Step 2: Choose Analysis Focus

Using the pre-computed `field_gaps` from Step 0 plus the strategy cache from Step 1:
- If `field_gaps` show promising leads, start there (they are likely high-signal)
- If cache has strategies, use a proven strategy 70% of the time; try a new approach 30% of the time

```bash
# Determine selection mode (0-6 = proven strategy, 7-9 = new approach)
day_mod=$(( $(date +%j) % 10 ))
if [ "$day_mod" -le 6 ]; then
  echo "Use proven strategy from cache"
else
  echo "Try new approach"
fi
```

### Step 3: Execute Targeted Analysis

Use the pre-computed data as context and run **targeted** follow-up commands only when
deeper inspection is needed (e.g., checking how a specific field is actually processed in code).

**Example: Verify a gap from pre-computed data**
```bash
# Verify a specific field gap by searching implementation files
grep -r "fieldName" pkg/parser/ pkg/workflow/ 2>/dev/null | grep -v "_test.go"
```

**Example: Type checking for a specific field**
```bash
# Find schema field types (handles different JSON Schema patterns)
jq -r '
  (.properties // {}) | to_entries[] | 
  "\(.key): \(.value.type // .value.oneOf // .value.anyOf // .value.allOf // "complex")"
' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null || echo "Failed to parse schema"
```

### Step 4: Record Findings
Create a structured list of inconsistencies found:

```markdown
## Inconsistencies Found

### Schema ↔ Parser Mismatches
1. **Field `engine.version`**: 
   - Schema: defines as string
   - Parser: not validated in frontmatter.go
   - Impact: Invalid values could pass through

### Schema ↔ Documentation Mismatches  
1. **Field `cache-memory`**:
   - Schema: defines array of objects with `id` and `key`
   - Docs: only shows simple boolean example
   - Impact: Advanced usage not documented

### Parser ↔ Documentation Mismatches
1. **Error message for invalid `on` field**:
   - Parser: "trigger configuration is required"
   - Docs: doesn't mention this error
   - Impact: Users may not understand error
```

### Step 5: Update Cache
Save successful strategy and findings to cache:
```bash
# Update strategies.json with results
cat > /tmp/gh-aw/cache-memory/strategies.json << 'EOF'
{
  "strategies": [...],
  "last_updated": "2024-XX-XX"
}
EOF
```

### Step 6: Create Discussion
Generate a comprehensive report for discussion output.

## Discussion Report Format

Create a well-structured discussion report:

```markdown
# 🔍 Schema Consistency Check - [DATE]

## Summary

- **Inconsistencies Found**: [NUMBER]
- **Categories Analyzed**: Schema, Parser, Documentation, Workflows
- **Strategy Used**: [STRATEGY NAME]
- **New Strategy**: [YES/NO]

## Critical Issues

[List high-priority inconsistencies that could cause bugs]

## Documentation Gaps

[List areas where docs don't match reality]

## Schema Improvements Needed

[List schema enhancements needed]

## Parser Updates Required

[List parser code that needs updates]

## Workflow Violations

[List workflows using invalid/undocumented features]

## Recommendations

1. [Specific actionable recommendation]
2. [Specific actionable recommendation]
3. [...]

## Strategy Performance

- **Strategy Used**: [NAME]
- **Findings**: [COUNT]
- **Effectiveness**: [HIGH/MEDIUM/LOW]
- **Should Reuse**: [YES/NO]

## Next Steps

- [ ] Fix schema definitions
- [ ] Update parser validation
- [ ] Update documentation
- [ ] Fix workflow files
```

## Important Guidelines

### Security
- Never execute untrusted code from workflows
- Validate all file paths before reading
- Sanitize all grep/bash commands
- Read-only access to schema, parser, and documentation files for analysis
- Only modify files in `/tmp/gh-aw/cache-memory/` (never modify source files)

### Quality
- Be thorough but focused on actionable findings
- Prioritize issues by severity (critical bugs vs documentation gaps)
- Provide specific file:line references when possible
- Include code snippets to illustrate issues
- Suggest concrete fixes

### Efficiency  
- **Always start from `/tmp/gh-aw/agent/schema-diff.json`** — this pre-computed diff eliminates the need to re-read all source files
- Use targeted bash commands to verify specific leads from the pre-computed data
- Cache results when re-analyzing same data
- Don't re-check things found in previous runs (check cache first)
- Focus on high-impact areas (field gaps with parser mismatches are usually most critical)

### Strategy Evolution
- Try genuinely different approaches when not using cached strategies
- Document why a strategy worked or failed
- Update success metrics in cache
- Consider combining successful strategies

## Tools Available

You have access to:
- **bash**: Any command (use grep, jq, find, cat, etc.)
- **edit**: Create/modify files in cache memory
- **github**: Read repository data, discussions

## Success Criteria

A successful run:
- ✅ Analyzes all 4 areas (schema, parser, docs, workflows)
- ✅ Uses or creates an effective detection strategy
- ✅ Updates cache with strategy results
- ✅ Finds at least one category of inconsistencies OR confirms consistency
- ✅ Creates a detailed discussion report
- ✅ Provides actionable recommendations

Begin your analysis now. Check the cache, choose a strategy, execute it, and report your findings in a discussion.

{{#import shared/noop-reminder.md}}
