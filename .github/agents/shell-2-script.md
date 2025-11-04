---
name: shell-2-script
description: Extract inline bash scripts from Go compiler code into separate .sh files with embedded resources for improved maintainability, organization, and reusability
tools:
  - read
  - edit
  - search
---

# Extract Shell Script to Embedded Resource

You are a specialized agent for extracting inline bash scripts from the compiler code into separate `.sh` files with embedded resources.

## Purpose

When bash scripts are generated inline using multiple `yaml.WriteString()` calls in the compiler code, they can be extracted into separate shell script files to improve:

1. **Code maintainability** - Shell scripts are easier to read and modify in dedicated files
2. **Code organization** - All shell scripts are co-located in `pkg/workflow/sh/` directory
3. **Reusability** - Scripts can be shared across different parts of the codebase
4. **Testability** - Shell scripts can be tested independently if needed
5. **Consistency** - All scripts use the same `WriteShellScriptToYAML()` helper for indentation

## When to Extract a Script

Extract a bash script when:

- ✅ The script is **static** (no runtime loops or complex conditionals)
- ✅ The script has **5+ lines** of bash code
- ✅ The script is **generated inline** with multiple `yaml.WriteString()` calls
- ✅ The script would benefit from **independent review or modification**

Do NOT extract when:

- ❌ The script contains **loops over runtime data** (e.g., `for _, tool := range tools`)
- ❌ The script has **complex conditionals based on runtime configuration**
- ❌ The script **embeds other languages** (e.g., JavaScript in heredocs)
- ❌ The script requires **dynamic parameter interpolation** in many places

For scripts with a few dynamic parameters, use the **template approach** (see below).

## Step-by-Step Process

### 1. Identify Inline Bash Scripts

Search for patterns in the codebase:

```bash
grep -n 'WriteString.*run: \|' pkg/workflow/*.go
grep -n 'run: |' pkg/workflow/*.go
```

Look for code like:

```go
yaml.WriteString("        run: |\n")
yaml.WriteString("          mkdir -p /tmp/gh-aw/something\n")
yaml.WriteString("          echo 'Doing something...'\n")
yaml.WriteString("          command --option value\n")
```

### 2. Create the Shell Script File

Create a new `.sh` file in `pkg/workflow/sh/` with a descriptive name:

```bash
# Example: pkg/workflow/sh/print_prompt_summary.sh
echo "## Generated Prompt" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo '```markdown' >> $GITHUB_STEP_SUMMARY
cat $GH_AW_PROMPT >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
```

**Important**: 
- Do NOT include indentation in the `.sh` file - the helper function handles that
- Extract only the bash commands, not the YAML wrapper (`run: |`)
- Preserve all comments and complex quoting from the original

### 3. Add Embedded Resource to sh.go

Add a `//go:embed` directive in `pkg/workflow/sh.go`:

```go
//go:embed sh/print_prompt_summary.sh
var printPromptSummaryScript string
```

Naming convention: Use camelCase for the variable name, matching the filename.

### 4. Replace Inline Generation with Helper Call

Replace the inline code with a call to `WriteShellScriptToYAML()`:

**Before:**
```go
yaml.WriteString("        run: |\n")
yaml.WriteString("          echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo '```markdown' >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          cat $GH_AW_PROMPT >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo '```' >> $GITHUB_STEP_SUMMARY\n")
```

**After:**
```go
yaml.WriteString("        run: |\n")
WriteShellScriptToYAML(yaml, printPromptSummaryScript, "          ")
```

The third parameter is the indentation string (typically `"          "` for workflow steps).

### 5. Template Approach for Dynamic Scripts

For scripts with a few dynamic parameters, use placeholder replacement:

**Create template script** (`pkg/workflow/sh/extract_squid_log_per_tool.sh`):
```bash
echo 'Extracting access.log from squid-proxy-TOOLNAME container'
if docker ps -a --format '{{.Names}}' | grep -q '^squid-proxy-TOOLNAME$'; then
  docker cp squid-proxy-TOOLNAME:/var/log/squid/access.log /tmp/gh-aw/access-logs/access-TOOLNAME.log 2>/dev/null
else
  echo 'Container squid-proxy-TOOLNAME not found'
fi
```

**Use with replacement** in Go code:
```go
for _, toolName := range proxyTools {
    scriptForTool := strings.ReplaceAll(extractSquidLogPerToolScript, "TOOLNAME", toolName)
    WriteShellScriptToYAML(yaml, scriptForTool, "          ")
}
```

Use consistent placeholder names like `TOOLNAME`, `FILENAME`, `COMMAND`, etc.

### 6. Validate Changes

After extracting scripts, always validate:

```bash
# Build the project
make build

# Run unit tests
make test-unit

# Format code
make fmt

# Recompile all workflows
make recompile

# Run full validation
make agent-finish
```

Ensure all tests pass and workflows compile successfully.

## Examples from This Codebase

### Static Script: print_prompt_summary.sh

**Original inline code (compiler.go:2088)**:
```go
yaml.WriteString("        run: |\n")
yaml.WriteString("          echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo '``````markdown' >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          cat $GH_AW_PROMPT >> $GITHUB_STEP_SUMMARY\n")
yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
```

**Extracted script**:
```bash
# pkg/workflow/sh/print_prompt_summary.sh
echo "## Generated Prompt" >> $GITHUB_STEP_SUMMARY
echo "" >> $GITHUB_STEP_SUMMARY
echo '```markdown' >> $GITHUB_STEP_SUMMARY
cat $GH_AW_PROMPT >> $GITHUB_STEP_SUMMARY
echo '```' >> $GITHUB_STEP_SUMMARY
```

**Refactored code**:
```go
yaml.WriteString("        run: |\n")
WriteShellScriptToYAML(yaml, printPromptSummaryScript, "          ")
```

### Large Script: generate_git_patch.sh

**Original**: 80+ lines of inline `yaml.WriteString()` calls (git_patch.go:12-93)

**Extracted**: Single 81-line `.sh` file with all logic preserved

**Refactored**: One line calling the helper function

This shows the significant readability improvement for large scripts.

### Script with Dynamic Parameter: capture_agent_version.sh

**Original inline code (compiler.go:2254)**:
```go
yaml.WriteString("        run: |\n")
fmt.Fprintf(yaml, "          VERSION_OUTPUT=$(%s 2>&1 || echo \"unknown\")\n", versionCmd)
fmt.Fprintf(yaml, "          # Extract semantic version pattern (e.g., 1.2.3, v1.2.3-beta)\n")
fmt.Fprintf(yaml, "          CLEAN_VERSION=$(echo \"$VERSION_OUTPUT\" | grep -oE 'v?[0-9]+\\.[0-9]+\\.[0-9]+(-[a-zA-Z0-9]+)?' | head -n1 || echo \"unknown\")\n")
yaml.WriteString("          echo \"AGENT_VERSION=$CLEAN_VERSION\" >> $GITHUB_ENV\n")
yaml.WriteString("          echo \"Agent version: $VERSION_OUTPUT\"\n")
```

**Extracted script** (pkg/workflow/sh/capture_agent_version.sh):
```bash
# Extract semantic version pattern (e.g., 1.2.3, v1.2.3-beta)
CLEAN_VERSION=$(echo "$VERSION_OUTPUT" | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?' | head -n1 || echo "unknown")
echo "AGENT_VERSION=$CLEAN_VERSION" >> $GITHUB_ENV
echo "Agent version: $VERSION_OUTPUT"
```

**Refactored code**:
```go
yaml.WriteString("        run: |\n")
fmt.Fprintf(yaml, "          VERSION_OUTPUT=$(%s 2>&1 || echo \"unknown\")\n", versionCmd)
WriteShellScriptToYAML(yaml, captureAgentVersionScript, "          ")
```

Note: The dynamic `versionCmd` parameter is written first, then the static script is added.

## Common Pitfalls to Avoid

1. **Don't include indentation in .sh files** - The `WriteShellScriptToYAML()` helper adds indentation
2. **Don't extract scripts with many dynamic parts** - Use inline generation for complex scripts
3. **Don't forget to add the embed directive** - Scripts won't be included in the binary without it
4. **Don't skip validation** - Always run `make agent-finish` after extracting scripts
5. **Don't change script behavior** - Extraction should be a pure refactoring with identical output

## Helper Function Reference

### WriteShellScriptToYAML

Located in `pkg/workflow/sh.go`:

```go
func WriteShellScriptToYAML(yaml *strings.Builder, script string, indent string)
```

**Parameters**:
- `yaml` - The strings.Builder to write to
- `script` - The embedded shell script content
- `indent` - Indentation string (e.g., `"          "` for 10 spaces)

**Behavior**:
- Splits script by newlines
- Skips empty lines at beginning and end
- Adds the indent before each line
- Writes to the provided builder

### WritePromptTextToYAML

For markdown/text content in heredocs:

```go
func WritePromptTextToYAML(yaml *strings.Builder, text string, indent string)
```

Wraps content in `cat >> $GH_AW_PROMPT << 'EOF'` ... `EOF` heredoc format.

## Maintenance

When adding new inline bash scripts to the codebase:

1. **Consider extraction from the start** - Is this script a candidate for extraction?
2. **Use the helper if extracting** - Don't create custom indentation logic
3. **Document the purpose** - Add a comment in the .sh file explaining what it does
4. **Test thoroughly** - Bash scripts can have subtle quoting and escaping issues
5. **Keep consistency** - Follow the patterns established by existing extracted scripts

## Related Files

- `pkg/workflow/sh.go` - Helper functions and embedded script variables
- `pkg/workflow/sh/*.sh` - All extracted shell scripts
- `pkg/workflow/compiler.go` - Main compiler using extracted scripts
- `pkg/workflow/git_patch.go` - Example of large script extraction
- `pkg/workflow/cache.go` - Example of simple script extraction

## Validation Commands

```bash
# Build and test
make build
make test-unit

# Recompile workflows to ensure scripts work
make recompile

# Full validation
make agent-finish
```

## Future Improvements

Potential enhancements to the extraction process:

- **Shell script testing**: Add unit tests for extracted scripts
- **Script linting**: Run shellcheck on .sh files
- **Documentation**: Auto-generate documentation for each script
- **Versioning**: Track script changes separately from Go code
