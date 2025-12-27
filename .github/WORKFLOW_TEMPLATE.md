---
name: Workflow Template
description: Template demonstrating best practices for agentic workflows with correct shell scripting patterns
on:
  workflow_dispatch:
  # schedule:
  #   - cron: "0 0 * * *"  # Daily at midnight UTC

permissions:
  contents: read
  issues: read
  # pull-requests: read

# Unique identifier for this workflow (used in logs and tracking)
tracker-id: workflow-template

# AI engine configuration
engine: copilot

# Optional: Import shared instructions or boilerplate
# imports:
#   - shared/reporting.md
#   - shared/security-notice.md

# Configure safe outputs (what the AI can do)
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
    labels: [automated]
    max: 1
  # create-pull-request:
  # create-discussion:

# Tool configuration
tools:
  github:
    # Specify explicit toolsets to avoid permission warnings
    # default includes: repos, issues, pull_requests, discussions, etc.
    # For minimal permissions, specify only what you need:
    toolsets: [repos, issues]
    # Or use [default] and ensure pull-requests: read permission is set
  
  # Bash tool configuration with correct shell scripting patterns
  bash:
    # ✅ Correct: Use find instead of ls
    - "find . -name '*.md' -type f -printf '%T@ %p\\n' | sort -rn | head -10"
    
    # ✅ Correct: Quote variables in grep
    - "grep -r 'pattern' . --include='*.go'"
    
    # ✅ Correct: Use wc with find
    - "find . -name '*.go' ! -name '*_test.go' -type f -exec wc -l {} \\;"
    
    # ❌ Avoid: ls -t *.yml | head -5
    # ❌ Avoid: grep $pattern $file (unquoted variables)

# Set timeout to prevent runaway costs
timeout-minutes: 15

# Use strict mode for additional security validation
strict: true
---

# Workflow Instructions

## Overview

Brief description of what this workflow does.

## Task

Detailed instructions for the AI agent:

1. **Analyze repository**: Describe what to analyze
2. **Generate insights**: What kind of insights to generate
3. **Create output**: Use safe-outputs to create issues/PRs/discussions

## Shell Scripting Best Practices

When the workflow runs bash commands, follow these patterns:

### ✅ Correct Patterns

**Separate declaration from assignment:**
```bash
local result
result=$(command_that_might_fail)
if [[ $? -ne 0 ]]; then
  echo "Command failed"
fi
```

**Use find instead of ls:**
```bash
# Find recent files
find . -name "*.yml" -type f -printf '%T@ %p\n' | sort -rn | head -5

# Iterate over files safely
find . -name "*.txt" -type f -print0 | while IFS= read -r -d '' file; do
  echo "Processing: $file"
done
```

**Quote all variables:**
```bash
# Always quote to prevent word splitting
echo "$variable"
rm "$file"
grep "$pattern" "$filename"

# Quote array expansions
files=("file1.txt" "file2.txt")
command "${files[@]}"
```

**Group redirects:**
```bash
# Group multiple writes
{
  echo "Header"
  command1
  echo "Footer"
} > output.txt
```

### ❌ Patterns to Avoid

**Don't combine local and assignment:**
```bash
# This masks the exit status
local result=$(command)
```

**Don't parse ls output:**
```bash
# Breaks with spaces and special characters
ls -t *.yml | head -5
for file in $(ls *.txt); do
  echo $file
done
```

**Don't leave variables unquoted:**
```bash
# Causes word splitting and globbing
echo $variable
rm $file
```

**Don't use multiple redirects:**
```bash
# Inefficient - opens file multiple times
echo "line1" >> file
echo "line2" >> file
```

## Output Format

Describe the expected output format for issues/PRs/discussions.

### Example Output

```markdown
## Analysis Results

- Finding 1
- Finding 2
- Finding 3

### Recommendations

1. Recommendation 1
2. Recommendation 2
```

## Validation

Run actionlint to validate this workflow:

```bash
gh aw compile workflow-template --actionlint
```

This includes shellcheck validation to catch common shell scripting errors.

## Notes

- Keep the workflow focused on a single task
- Set appropriate timeouts
- Use strict mode for additional security
- Test bash commands with files containing spaces in names
- Review generated issues/PRs before merging workflow changes
