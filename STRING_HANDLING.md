# String Handling Functions Reference

This document provides a comprehensive inventory of all string normalization, sanitization, and cleaning functions in the gh-aw codebase. It serves as a guide for developers to choose the appropriate function for their use case.

## Table of Contents

- [Overview](#overview)
- [Decision Tree](#decision-tree)
- [Function Categories](#function-categories)
  - [Workflow Name Handling](#workflow-name-handling)
  - [Content Sanitization (Security)](#content-sanitization-security)
  - [Identifier Normalization](#identifier-normalization)
  - [Expression and Whitespace Normalization](#expression-and-whitespace-normalization)
  - [Resource Cleanup](#resource-cleanup)
  - [Error Message Cleaning](#error-message-cleaning)
- [Function Inventory](#function-inventory)
- [Known Duplications and Sync Requirements](#known-duplications-and-sync-requirements)
- [Consolidation Opportunities](#consolidation-opportunities)

## Overview

String handling in gh-aw falls into several categories based on purpose:

1. **Workflow name normalization** - Converting workflow names to filesystem/artifact-safe identifiers
2. **Content sanitization** - Removing or neutralizing potentially dangerous content (XSS, injection attacks, @mentions)
3. **Identifier creation** - Creating safe identifiers for various contexts (user agents, safe outputs, etc.)
4. **Whitespace normalization** - Standardizing whitespace for comparisons or file operations
5. **Resource cleanup** - Cleaning up temporary files and resources (not string transformation)
6. **Error message cleaning** - Improving error message readability

## Decision Tree

Use this decision tree to select the appropriate string handling function:

```
What are you trying to do?

├─ Convert a workflow name to a safe identifier?
│  ├─ For filesystem/artifact names? → SanitizeWorkflowName (Go) or sanitizeWorkflowName (JS)
│  ├─ For user agent strings? → SanitizeIdentifier (Go)
│  ├─ For safe output identifiers? → normalizeSafeOutputIdentifier (Go)
│  └─ Remove file extensions only? → normalizeWorkflowName (Go) or NormalizeWorkflowFile (Go)
│
├─ Sanitize user-provided content for security?
│  ├─ For GitHub Issues/PRs/Discussions? → sanitizeContent (JS in compute_text.cjs, sanitize_output.cjs, collect_ndjson_output.cjs)
│  └─ For labels only? → sanitizeLabelContent (JS)
│
├─ Create a git branch name?
│  └─ Use normalizeBranchName (JS in upload_assets.cjs or safe_outputs_mcp_server.cjs)
│
├─ Normalize whitespace?
│  ├─ For expression comparison? → NormalizeExpressionForComparison (Go)
│  └─ For file content? → normalizeWhitespace (Go)
│
├─ Clean MCP tool identifiers?
│  └─ Use cleanMCPToolID (Go)
│
├─ Convert repo slug to filename?
│  └─ Use sanitizeRepoSlugForFilename (Go)
│
└─ Clean error messages?
   └─ Use cleanJSONSchemaErrorMessage (Go)
```

## Function Categories

### Workflow Name Handling

These functions handle the transformation of workflow names for different contexts:

#### `SanitizeWorkflowName(name string) string` (Go)
- **Location**: `pkg/workflow/strings.go`
- **Purpose**: Sanitizes workflow names for use in artifact names and file paths
- **Transformations**:
  - Converts to lowercase
  - Replaces `:`, `\`, `/`, and spaces with hyphens
  - Replaces special characters (except `.`, `_`, `-`) with hyphens
  - Consolidates multiple hyphens into single hyphen
- **Preserves**: Dots (`.`), underscores (`_`), hyphens (`-`)
- **Example**: `"My Workflow: Test/Build"` → `"my-workflow--test-build"`
- **Use when**: Creating artifact names or filesystem paths from workflow names

#### `sanitizeWorkflowName(name)` (JavaScript)
- **Location**: `pkg/workflow/js/parse_firewall_logs.cjs`
- **Purpose**: Sanitizes workflow names for use in file paths (JavaScript equivalent)
- **Transformations**:
  - Converts to lowercase
  - Replaces `:`, `\`, `/`, and spaces with hyphens
  - Replaces special characters (except `.`, `_`, `-`) with hyphens
  - **Note**: Does NOT consolidate multiple hyphens (difference from Go version)
- **Preserves**: Dots (`.`), underscores (`_`), hyphens (`-`)
- **Example**: `"My Workflow: Test"` → `"my-workflow--test"`
- **Use when**: Processing workflow names in JavaScript firewall log parsing
- **⚠️ Potential Issue**: Missing hyphen consolidation logic compared to Go version

#### `normalizeWorkflowName(name string) string` (Go)
- **Location**: `pkg/workflow/resolve.go`
- **Purpose**: Removes file extensions to get base workflow identifier
- **Transformations**:
  - Removes `.lock.yml` extension (checked first)
  - Removes `.md` extension
  - Returns unchanged if no extension
- **Example**: `"weekly-research.lock.yml"` → `"weekly-research"`
- **Use when**: Resolving workflow files to their base identifier
- **Scope**: Private function, used internally for workflow resolution

#### `NormalizeWorkflowFile(workflowFile string) string` (Go)
- **Location**: `pkg/cli/resolver.go`
- **Purpose**: Adds `.md` extension if missing from workflow file name
- **Transformations**:
  - Appends `.md` if not already present
  - Returns unchanged if already has `.md` extension
- **Example**: `"weekly-research"` → `"weekly-research.md"`
- **Use when**: Normalizing user input for workflow file resolution
- **Scope**: Public function, exported for use in CLI

### Identifier Normalization

#### `SanitizeIdentifier(name string) string` (Go)
- **Location**: `pkg/workflow/workflow_name.go`
- **Purpose**: Creates safe identifiers for user agent strings and similar contexts
- **Transformations**:
  - Converts to lowercase
  - Replaces spaces and underscores with hyphens
  - Removes all non-alphanumeric characters except hyphens
  - Consolidates multiple hyphens
  - Trims leading/trailing hyphens
  - Returns `"github-agentic-workflow"` if result is empty
- **Preserves**: Only alphanumeric characters and hyphens
- **Example**: `"My_Workflow.Test"` → `"my-workflow-test"`
- **Use when**: Creating user agent strings or identifiers that don't need dots/underscores
- **Difference from SanitizeWorkflowName**: Removes dots and underscores; has fallback value

#### `normalizeSafeOutputIdentifier(identifier string) string` (Go)
- **Location**: `pkg/workflow/safe_outputs.go`
- **Purpose**: Converts dashes to underscores for safe output identifiers
- **Transformations**:
  - Replaces all `-` with `_`
- **Example**: `"create-issue"` → `"create_issue"`
- **Use when**: Converting safe output type names to consistent identifiers
- **Scope**: Private function, used internally for safe output processing
- **Note**: Simple transformation to ensure consistency with LLM-generated variations

#### `cleanMCPToolID(toolID string) string` (Go)
- **Location**: `pkg/cli/mcp_registry.go`
- **Purpose**: Removes common MCP prefixes/suffixes from tool IDs
- **Transformations**:
  - Removes `"mcp-"` prefix
  - Removes `"-mcp"` suffix
  - Returns original if result would be empty
- **Example**: `"mcp-notion"` → `"notion"`, `"some-mcp-server"` → `"some-server"`
- **Use when**: Cleaning MCP tool identifiers for display or comparison

#### `sanitizeRepoSlugForFilename(repoSlug string) string` (Go)
- **Location**: `pkg/cli/trial_command.go`
- **Purpose**: Converts repository slug to filename-safe string
- **Transformations**:
  - Replaces `/` with `-`
  - Returns `"clone-mode"` if input is empty
- **Example**: `"owner/repo"` → `"owner-repo"`
- **Use when**: Creating filenames from repository slugs in trial command

#### `normalizeBranchName(branchName)` (JavaScript)
- **Location**: `pkg/workflow/js/upload_assets.cjs` and `pkg/workflow/js/safe_outputs_mcp_server.cjs`
- **Purpose**: Normalizes branch names to valid git branch name format
- **Transformations**:
  1. Replaces invalid characters (anything except `a-z`, `A-Z`, `0-9`, `-`, `_`, `/`, `.`) with single dash
  2. Collapses multiple consecutive dashes to single dash
  3. Removes leading and trailing dashes
  4. Truncates to 128 characters max
  5. Removes trailing dashes after truncation
  6. Converts to lowercase
- **Preserves**: Alphanumeric, dash (`-`), underscore (`_`), forward slash (`/`), dot (`.`)
- **Example**: `"feature/My Branch!!"` → `"feature/my-branch"`
- **Use when**: Creating or validating git branch names
- **⚠️ Sync Requirement**: These two functions MUST be kept in sync (see comments in both files)

### Content Sanitization (Security)

These functions sanitize user-provided content to prevent security issues like XSS, injection attacks, and unwanted notifications.

#### `sanitizeContent(content)` (JavaScript)
- **Locations**: 
  - `pkg/workflow/js/compute_text.cjs`
  - `pkg/workflow/js/sanitize_output.cjs`
  - `pkg/workflow/js/collect_ndjson_output.cjs`
- **Purpose**: Comprehensive sanitization of user content for GitHub Actions output
- **Security Features**:
  - **@mention neutralization**: Wraps `@username` in backticks to prevent notifications
  - **Control character removal**: Strips non-printable characters (except newlines/tabs)
  - **XML tag neutralization**: Converts `<tag>` to `(tag)` to prevent injection
  - **URL protocol filtering**: Removes non-HTTPS URLs (replaces with `"(redacted)"`)
  - **Domain filtering**: Checks HTTPS URLs against allowlist, redacts unauthorized domains
  - **Length limits**: Truncates to 0.5MB max, 65k lines max
  - **ANSI escape removal**: Strips terminal control sequences
  - **Bot trigger neutralization**: Wraps `fixes #123`, `closes #456` in backticks
- **Configuration**: Reads `GH_AW_ALLOWED_DOMAINS` env var for domain allowlist
- **Default allowed domains**: `github.com`, `github.io`, `githubusercontent.com`, `githubassets.com`, `github.dev`, `codespaces.new`
- **Use when**: Processing any user-provided content from issues, PRs, comments
- **Helper functions**:
  - `sanitizeUrlDomains(s)` - Filters URLs by domain
  - `sanitizeUrlProtocols(s)` - Filters non-HTTPS protocols
  - `convertXmlTagsToParentheses(s)` - Neutralizes XML tags
  - `neutralizeMentions(s)` - Wraps @mentions in backticks
  - `neutralizeBotTriggers(s)` - Wraps bot trigger phrases in backticks

#### `sanitizeLabelContent(content)` (JavaScript)
- **Locations**:
  - `pkg/workflow/js/add_labels.cjs`
  - `pkg/workflow/js/create_issue.cjs`
- **Purpose**: Lighter sanitization specifically for label content
- **Security Features**:
  - Trims whitespace
  - Removes control characters (except newlines/tabs)
  - Removes ANSI escape sequences
  - Neutralizes @mentions with backticks
  - Removes HTML special characters (`<`, `>`, `&`, `'`, `"`)
- **Use when**: Sanitizing label names or content for GitHub API
- **Note**: Less comprehensive than `sanitizeContent` - designed for label-specific needs

### Expression and Whitespace Normalization

#### `NormalizeExpressionForComparison(expression string) string` (Go)
- **Location**: `pkg/workflow/expressions.go`
- **Purpose**: Normalizes expressions for comparison by removing formatting differences
- **Transformations**:
  - Replaces newlines and tabs with spaces
  - Consolidates multiple spaces into single spaces
  - Trims leading/trailing spaces
- **Example**: `"foo\n  bar\t  baz"` → `"foo bar baz"`
- **Use when**: Comparing multiline expressions with single-line equivalents

#### `normalizeWhitespace(content string) string` (Go)
- **Location**: `pkg/cli/update_command.go`
- **Purpose**: Normalizes trailing whitespace and newlines in file content to reduce conflicts
- **Transformations**:
  - Trims trailing whitespace from each line
  - Ensures exactly one trailing newline (if content not empty)
  - Removes excess trailing newlines
- **Use when**: Normalizing file content before writing or comparing
- **Scope**: Used in update command for workflow file handling

### Error Message Cleaning

#### `cleanJSONSchemaErrorMessage(errorMsg string) string` (Go)
- **Location**: `pkg/parser/schema.go`
- **Purpose**: Removes unhelpful prefixes from JSON schema validation errors
- **Transformations**:
  - Removes `"jsonschema validation failed"` lines
  - Removes `"- at '': "` prefixes from error descriptions
  - Cleans up error formatting for better readability
- **Use when**: Formatting JSON schema validation errors for user display
- **Scope**: Used in schema validation and error reporting

### Resource Cleanup

These functions handle cleanup of resources and temporary files. They are included for completeness but are not string transformation functions.

#### `generateCleanupStep(outputFiles []string) (string, bool)` (Go)
- **Location**: `pkg/workflow/engine_output.go`
- **Purpose**: Generates cleanup step for removing temporary output files
- **Note**: Not a string normalization function - generates workflow step

#### `GetCleanupStep(workflowData *WorkflowData) GitHubActionStep` (Go)
- **Location**: `pkg/workflow/copilot_engine.go`
- **Purpose**: Method to get cleanup step for Copilot engine
- **Note**: Not a string normalization function - workflow step generation

#### `generateAWFCleanupStep(scriptPath string) GitHubActionStep` (Go)
- **Location**: `pkg/workflow/copilot_engine.go`
- **Purpose**: Generates cleanup step for agentic workflow framework files
- **Note**: Not a string normalization function - workflow step generation

#### `generateAWFPostExecutionCleanupStep(scriptPath string) GitHubActionStep` (Go)
- **Location**: `pkg/workflow/copilot_engine.go`
- **Purpose**: Generates post-execution cleanup step
- **Note**: Not a string normalization function - workflow step generation

#### `cleanupTrialRepository(repoSlug string, verbose bool) error` (Go)
- **Location**: `pkg/cli/trial_command.go`
- **Purpose**: Cleans up trial repository after testing
- **Note**: Not a string normalization function - resource cleanup

#### `cleanupTrialSecrets(repoSlug string, tracker *TrialSecretTracker, verbose bool) error` (Go)
- **Location**: `pkg/cli/trial_command.go`
- **Purpose**: Cleans up trial secrets after testing
- **Note**: Not a string normalization function - resource cleanup

#### `checkCleanWorkingDirectory(verbose bool) error` (Go)
- **Location**: `pkg/cli/add_command.go`
- **Purpose**: Checks if git working directory is clean before operations
- **Note**: Not a string normalization function - validation check

#### `cleanupOrphanedIncludes(verbose bool) error` (Go)
- **Location**: `pkg/cli/remove_command.go`
- **Purpose**: Removes orphaned include files
- **Note**: Not a string normalization function - file cleanup

#### `cleanupAllIncludes(verbose bool) error` (Go)
- **Location**: `pkg/cli/remove_command.go`
- **Purpose**: Removes all include files
- **Note**: Not a string normalization function - file cleanup

## Function Inventory

### Go Functions (String Transformation)

| Function | File | Purpose | Scope | Category |
|----------|------|---------|-------|----------|
| `SanitizeWorkflowName` | pkg/workflow/strings.go | Workflow name → filesystem/artifact safe | Public | Workflow Name |
| `SanitizeIdentifier` | pkg/workflow/workflow_name.go | Workflow name → identifier (user agent) | Public | Identifier |
| `normalizeWorkflowName` | pkg/workflow/resolve.go | Remove file extensions | Private | Workflow Name |
| `NormalizeWorkflowFile` | pkg/cli/resolver.go | Add .md extension if missing | Public | Workflow Name |
| `normalizeSafeOutputIdentifier` | pkg/workflow/safe_outputs.go | Dashes to underscores | Private | Identifier |
| `cleanMCPToolID` | pkg/cli/mcp_registry.go | Remove MCP prefixes/suffixes | Private | Identifier |
| `sanitizeRepoSlugForFilename` | pkg/cli/trial_command.go | Repo slug → filename | Private | Identifier |
| `NormalizeExpressionForComparison` | pkg/workflow/expressions.go | Normalize whitespace for comparison | Public | Expression |
| `normalizeWhitespace` | pkg/cli/update_command.go | Normalize file content whitespace | Private | Whitespace |
| `cleanJSONSchemaErrorMessage` | pkg/parser/schema.go | Clean error message formatting | Private | Error Message |

### JavaScript Functions (String Transformation)

| Function | File(s) | Purpose | Scope | Category |
|----------|---------|---------|-------|----------|
| `sanitizeWorkflowName` | pkg/workflow/js/parse_firewall_logs.cjs | Workflow name → filesystem safe | Private | Workflow Name |
| `normalizeBranchName` | pkg/workflow/js/upload_assets.cjs, safe_outputs_mcp_server.cjs | Normalize git branch names | Private | Identifier |
| `sanitizeContent` | compute_text.cjs, sanitize_output.cjs, collect_ndjson_output.cjs | Comprehensive content sanitization | Private | Security |
| `sanitizeLabelContent` | add_labels.cjs, create_issue.cjs | Label-specific sanitization | Private | Security |
| `sanitizeUrlDomains` | compute_text.cjs, sanitize_output.cjs, collect_ndjson_output.cjs | Filter URLs by domain | Helper | Security |
| `sanitizeUrlProtocols` | compute_text.cjs, sanitize_output.cjs, collect_ndjson_output.cjs | Filter non-HTTPS protocols | Helper | Security |
| `convertXmlTagsToParentheses` | compute_text.cjs | Neutralize XML tags | Helper | Security |
| `neutralizeMentions` | compute_text.cjs | Wrap @mentions in backticks | Helper | Security |
| `neutralizeBotTriggers` | compute_text.cjs | Wrap bot triggers in backticks | Helper | Security |

## Known Duplications and Sync Requirements

### Critical: Must Be Kept In Sync

#### `normalizeBranchName` (JavaScript)
- **Files**: 
  - `pkg/workflow/js/upload_assets.cjs`
  - `pkg/workflow/js/safe_outputs_mcp_server.cjs`
- **Status**: ✅ Currently identical (verified)
- **Sync Requirement**: Both files have comments stating "IMPORTANT: Keep this function in sync"
- **Risk**: High - divergence would cause inconsistent branch name handling
- **Recommendation**: Consider extracting to shared utility module

### Potential Duplications

#### `SanitizeWorkflowName` (Go) vs `sanitizeWorkflowName` (JavaScript)
- **Files**:
  - Go: `pkg/workflow/strings.go`
  - JS: `pkg/workflow/js/parse_firewall_logs.cjs`
- **Similarity**: High (same purpose, similar logic)
- **Key Difference**: JavaScript version does NOT consolidate multiple hyphens
- **Risk**: Medium - could lead to inconsistent workflow name handling
- **Recommendation**: Consider adding hyphen consolidation to JavaScript version OR document why it's intentionally different

#### `sanitizeContent` (JavaScript)
- **Files**:
  - `pkg/workflow/js/compute_text.cjs`
  - `pkg/workflow/js/sanitize_output.cjs`
  - `pkg/workflow/js/collect_ndjson_output.cjs`
- **Similarity**: Very high - appears to be copy-pasted
- **Risk**: High - security updates must be applied to all three copies
- **Recommendation**: **HIGH PRIORITY** - Extract to shared utility module to ensure security fixes are consistent

#### `sanitizeLabelContent` (JavaScript)
- **Files**:
  - `pkg/workflow/js/add_labels.cjs`
  - `pkg/workflow/js/create_issue.cjs`
- **Similarity**: Very high - identical implementation
- **Risk**: Medium - inconsistency could affect label handling
- **Recommendation**: Extract to shared utility module

## Consolidation Opportunities

### High Priority

1. **Consolidate `sanitizeContent` duplicates** (JavaScript)
   - Create shared utility module in `pkg/workflow/js/utils/` or similar
   - Export `sanitizeContent` with all helper functions
   - Import in all three files
   - **Benefit**: Security fixes apply universally; reduced maintenance burden
   - **Estimated Effort**: 1-2 hours
   - **Risk**: Low - well-defined function boundaries

2. **Consolidate `sanitizeLabelContent` duplicates** (JavaScript)
   - Same approach as `sanitizeContent`
   - **Benefit**: Consistency across label handling
   - **Estimated Effort**: 30 minutes
   - **Risk**: Low

3. **Consolidate `normalizeBranchName` duplicates** (JavaScript)
   - Extract to shared utility
   - **Benefit**: Guaranteed consistency; single point of maintenance
   - **Estimated Effort**: 30 minutes
   - **Risk**: Low

### Medium Priority

4. **Align `SanitizeWorkflowName` implementations** (Go vs JavaScript)
   - Add hyphen consolidation to JavaScript version OR
   - Document why they differ and when to use each
   - **Benefit**: Consistency across Go/JavaScript or clear documentation
   - **Estimated Effort**: 1 hour
   - **Risk**: Medium - need to verify firewall log parsing doesn't depend on current behavior

5. **Create shared string utilities module** (Go)
   - Move similar functions (`SanitizeWorkflowName`, `SanitizeIdentifier`, etc.) to shared package
   - Improve discoverability
   - **Benefit**: Better code organization; easier to find functions
   - **Estimated Effort**: 2-3 hours
   - **Risk**: Medium - requires updating imports across codebase

### Low Priority

6. **Consider extracting whitespace normalization** (Go)
   - `NormalizeExpressionForComparison` and `normalizeWhitespace` could share logic
   - **Benefit**: Code reuse
   - **Estimated Effort**: 1 hour
   - **Risk**: Low - functions serve different purposes

## Best Practices

When adding new string handling functions:

1. **Check this document first** - Don't create new functions if existing ones can be reused
2. **Use the decision tree** - Ensure you're using the right function for your use case
3. **Consider security** - For user-provided content, always use `sanitizeContent` or similar
4. **Document sync requirements** - If creating duplicates (avoid if possible), add clear comments about sync requirements
5. **Add to this document** - Update STRING_HANDLING.md with new functions
6. **Write tests** - Ensure edge cases are covered, especially for security-critical functions

## Security Considerations

### Critical Security Functions

These functions protect against security vulnerabilities and MUST be reviewed carefully:

- `sanitizeContent` (JavaScript) - **XSS, injection, @mention bombing**
- `sanitizeLabelContent` (JavaScript) - **XSS in labels**
- `sanitizeUrlDomains` (JavaScript) - **Phishing, malicious URLs**
- `sanitizeUrlProtocols` (JavaScript) - **Protocol injection**
- `neutralizeMentions` (JavaScript) - **Notification bombing**
- `neutralizeBotTriggers` (JavaScript) - **Unintended automation**

### Security Review Requirements

When modifying these functions:
1. Consider all possible attack vectors
2. Test with malicious input patterns
3. Verify all copies are updated (if duplicated)
4. Document security implications in comments
5. Get security review from another team member

## Maintenance Notes

### Regular Maintenance Tasks

1. **Monthly**: Review for new duplications using grep/search
2. **Per Release**: Verify sync'd functions are still identical
3. **Per Security Update**: Check all content sanitization functions are updated
4. **Quarterly**: Review consolidation opportunities and prioritize

### Version History

- **Initial Version** (2024-10-30): Comprehensive audit of all string handling functions
  - Identified 10 Go functions, 9 JavaScript functions
  - Documented 3 critical duplication issues
  - Proposed 6 consolidation opportunities

## Contributing

When contributing to string handling:

1. Read this document thoroughly
2. Use existing functions when possible
3. If creating new functions:
   - Add to appropriate category above
   - Update decision tree
   - Add usage examples
   - Write comprehensive tests
4. If modifying existing functions:
   - Update documentation
   - Check for duplicates that need same changes
   - Update tests
   - Consider backwards compatibility

## Questions?

For questions about which function to use or whether to create a new one, consult this document first. If still unclear:

1. Check the function's source code and comments
2. Look at existing usage in the codebase (`grep` for the function name)
3. Ask in team chat or create a discussion issue

---

**Last Updated**: 2024-10-30  
**Maintainer**: @githubnext/gh-aw team
