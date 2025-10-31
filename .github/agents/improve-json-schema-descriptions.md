---
name: improve-json-schema-descriptions
description: Systematic approach for reviewing and improving descriptions in the frontmatter JSON schema for GitHub Agentic Workflows
tools:
  - runInTerminal
  - getTerminalOutput
  - createFile
  - editFiles
  - search
  - changes
  - githubRepo
---

# Improve JSON Schema Descriptions

This prompt file documents the systematic approach for reviewing and improving descriptions in the frontmatter JSON schema for GitHub Agentic Workflows.

## Purpose

The main workflow schema (`pkg/parser/schemas/main_workflow_schema.json`) contains descriptions for all frontmatter fields that users can configure in agentic workflows. These descriptions should:

1. **Match the documentation** in `pkg/cli/templates/instructions.md` and `docs/src/content/docs/reference/frontmatter.md`
2. **Be clear and actionable** for users writing workflow frontmatter
3. **Include examples** where helpful (e.g., "supports wildcards", "e.g., 'github.com'")
4. **Highlight security considerations** (e.g., "⚠️ security consideration")
5. **Explain defaults** and when fields can be omitted

## Systematic Approach

### Phase 1: Core GitHub Actions Fields
- [x] Basic workflow metadata (`name`, `description`)
- [x] Trigger configuration (`on` field with all event types)
- [x] Permissions with detailed scope descriptions
- [x] Standard GitHub Actions fields (`runs-on`, `timeout_minutes`, `concurrency`, `run-name`)

### Phase 2: Agentic Workflow Specific Fields
- [x] Engine configuration (`claude`, `copilot`, `codex`, `custom`)
- [x] Tools configuration (`github`, `bash`, `web-fetch`, `web-search`, `edit`, `playwright`)
- [x] Network permissions with ecosystem identifiers
- [x] Safe-outputs configuration for secure GitHub API operations
- [x] Security roles configuration

### Phase 3: Complex Nested Objects
- [x] Trigger event configurations (push, pull_request, issues, etc.)
- [x] Permission scopes (actions, contents, issues, models, etc.)
- [x] Tool-specific options (Playwright domains, bash command restrictions)
- [x] Safe-output job configurations

## Key Improvements Made

### Enhanced Field Descriptions
- **Triggers**: Added specific examples and security context for command triggers (@mentions)
- **Permissions**: Added detailed explanations for each GitHub API scope
- **Engine Configuration**: Clarified AI engine options and when defaults can be used
- **Tools**: Explained security implications and use cases for each tool type
- **Safe Outputs**: Emphasized permission separation benefits

### Security-Focused Documentation
- Highlighted permission separation in safe-outputs
- Added warnings for security-sensitive fields like `roles: all`
- Explained network access controls and ecosystem identifiers
- Documented bash command restrictions and domain allowlists

### User Experience Improvements
- Added examples throughout (domain patterns, command lists, etc.)
- Explained when fields have sensible defaults and can be omitted
- Cross-referenced related features (network permissions + web tools)
- Used consistent terminology matching the main documentation

## Validation Process

After each set of changes:

1. **Syntax Validation**: Run `make recompile` to ensure JSON schema validity
2. **Workflow Compilation**: Verify all sample workflows still compile successfully  
3. **Unit Tests**: Run `make test-unit` to ensure schema validation logic works
4. **Integration Tests**: Run full test suite to verify end-to-end functionality

## Future Maintenance

When adding new fields to the schema:

1. **Document thoroughly** with examples and security considerations
2. **Cross-reference** with main documentation files
3. **Test extensively** with sample workflows
4. **Consider user experience** - what would be most helpful to understand?
5. **Maintain consistency** with existing description patterns

## Files Modified

- `pkg/parser/schemas/main_workflow_schema.json` - Primary schema file with improved descriptions
- This prompt file serves as documentation for future schema improvements

## Validation Commands

```bash
# Compile and validate all workflows
make recompile

# Run unit tests to verify schema validation
make test-unit

# Run full test suite
make test
```