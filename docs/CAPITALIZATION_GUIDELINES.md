# Capitalization Guidelines for gh-aw CLI

This document defines the capitalization rules for the gh-aw CLI to ensure consistency across all command descriptions and documentation.

## Decision: Option 2 - Context-Based Capitalization

The gh-aw CLI follows **Option 2** for capitalization:

### Rules

1. **Use lowercase "agentic workflows"** when referring generically to:
   - Workflow files and functionality
   - The concept of agentic workflows
   - Actions users perform on workflows (enable, disable, compile, etc.)

2. **Use capitalized "Agentic Workflows"** only when:
   - Explicitly referring to the product as a whole
   - Part of the full product name "GitHub Agentic Workflows"

3. **Keep technical terms capitalized:**
   - Markdown
   - YAML
   - MCP (Model Context Protocol)
   - Other proper nouns and acronyms

## Examples

### ✅ Correct Usage

**Product Name (Capitalized):**
```
"GitHub Agentic Workflows CLI from GitHub Next"
"GitHub Agentic Workflows from GitHub Next"
```

**Generic Usage (Lowercase):**
```
"Enable agentic workflows"
"Disable agentic workflows and cancel any in-progress runs"
"Show status of agentic workflows"
"Initialize repository for agentic workflows"
"Download and analyze agentic workflow logs with aggregated metrics"
"Add an MCP tool to an agentic workflow"
"List MCP servers defined in agentic workflows"
"Run one or more agentic workflows on GitHub Actions"
"Trial one or more agentic workflows as if they were running in a repository"
```

**Technical Terms (Capitalized):**
```
"Compile Markdown to YAML workflows"
"MCP helpers"
```

### ❌ Incorrect Usage

**Do NOT capitalize when referring to generic usage:**
```
❌ "Enable Agentic Workflows"
❌ "Show status of Agentic Workflows"
❌ "Add an MCP tool to an Agentic Workflow"
```

**Do NOT use all lowercase for product name:**
```
❌ "github agentic workflows CLI from GitHub Next"
```

**Do NOT lowercase technical terms:**
```
❌ "Compile markdown to yaml workflows"
```

## Testing

Capitalization consistency is enforced by automated tests in:
- `cmd/gh-aw/capitalization_test.go`

These tests verify that:
1. The root command and version command use the capitalized product name
2. All other commands use lowercase for generic workflow references
3. Technical terms remain properly capitalized

## Rationale

This approach provides:
- **Clarity**: Distinguishes between the product (GitHub Agentic Workflows) and the concept (agentic workflows)
- **Consistency**: Follows standard practices for product names vs. generic terms
- **Professional Appearance**: Maintains consistent messaging across the CLI
- **User Understanding**: Makes it easier to understand when referring to the product vs. the concept

## References

- Issue: [cli-consistency] Inconsistent capitalization of "agentic workflows" in command descriptions
- Test file: `cmd/gh-aw/capitalization_test.go`
