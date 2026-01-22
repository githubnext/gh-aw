# Workflow Inventory Package

The `pkg/cli/inventory` package provides unified helpers for extracting, normalizing, and managing workflow file names and paths.

## Overview

This package centralizes workflow name handling logic that was previously scattered across multiple files. It provides a single source of truth for:
- Extracting workflow names from file paths
- Normalizing user-provided workflow names
- Converting workflow names to file paths
- Discovering workflow files in directories

## Functions

### ExtractWorkflowName

Extracts the normalized workflow name from any file path or filename.

```go
import "github.com/githubnext/gh-aw/pkg/cli/inventory"

// Regular workflows
name := inventory.ExtractWorkflowName("my-workflow.md")              // "my-workflow"
name := inventory.ExtractWorkflowName(".github/workflows/deploy.md") // "deploy"

// Lock files
name := inventory.ExtractWorkflowName("workflow.lock.yml")           // "workflow"

// Campaign workflows
name := inventory.ExtractWorkflowName("security.campaign.md")        // "security"
name := inventory.ExtractWorkflowName("security.campaign.lock.yml")  // "security"
name := inventory.ExtractWorkflowName("security.campaign.g.md")      // "security"
```

**Supported file types:**
- Regular workflows: `.md`
- Lock files: `.lock.yml`
- Campaign workflows: `.campaign.md`
- Campaign lock files: `.campaign.lock.yml`
- Generated campaign orchestrators: `.campaign.g.md`

### NormalizeWorkflowName

Normalizes user input to a workflow name. This is semantically the same as `ExtractWorkflowName`, but indicates the input is from a user.

```go
// User provides various formats
name := inventory.NormalizeWorkflowName("my-workflow")                 // "my-workflow"
name := inventory.NormalizeWorkflowName("my-workflow.md")              // "my-workflow"
name := inventory.NormalizeWorkflowName(".github/workflows/deploy.md") // "deploy"
```

### GetWorkflowPath

Converts a workflow name to its markdown file path.

```go
// Default directory (.github/workflows)
path := inventory.GetWorkflowPath("my-workflow", "")
// ".github/workflows/my-workflow.md"

// Custom directory
path := inventory.GetWorkflowPath("deploy", "/custom/path")
// "/custom/path/deploy.md"
```

### GetLockFilePath

Returns the lock file path for a workflow.

```go
// Regular workflow
lockPath := inventory.GetLockFilePath("workflow.md", "")
// ".github/workflows/workflow.lock.yml"

// Campaign workflow
lockPath := inventory.GetLockFilePath("security.campaign.md", "")
// ".github/workflows/security.campaign.lock.yml"

// Generated campaign orchestrator
lockPath := inventory.GetLockFilePath("security.campaign.g.md", "")
// ".github/workflows/security.campaign.lock.yml"
```

### ListWorkflowFiles

Discovers all workflow files in a directory with filtering options.

```go
// List only regular workflows (default)
workflows, err := inventory.ListWorkflowFiles("", false, false)

// Include campaign workflows
workflows, err := inventory.ListWorkflowFiles("", true, false)

// Include generated files
workflows, err := inventory.ListWorkflowFiles("", false, true)

// Include everything
workflows, err := inventory.ListWorkflowFiles("", true, true)

// Each workflow contains:
for _, wf := range workflows {
    fmt.Printf("Name: %s\n", wf.Name)       // Normalized name
    fmt.Printf("Path: %s\n", wf.Path)       // Full path to .md file
    fmt.Printf("Type: %d\n", wf.Type)       // WorkflowType enum
    fmt.Printf("Lock: %s\n", wf.LockPath)   // Path to lock file
}
```

**WorkflowType enum values:**
- `WorkflowTypeRegular` (0) - Standard workflow (.md)
- `WorkflowTypeCampaign` (1) - Campaign spec (.campaign.md)
- `WorkflowTypeCampaignGenerated` (2) - Generated orchestrator (.campaign.g.md)

**Filtering:**
- `README.md` files are always excluded (case-insensitive)
- Files with README in the middle (e.g., `README-test.md`) are included
- By default, only regular `.md` workflows are returned
- Campaign files (`.campaign.md`) require `includeCampaigns=true`
- Generated files (`.campaign.g.md`) require `includeGenerated=true`

## Usage Examples

### Command-line workflow argument handling

```go
import "github.com/githubnext/gh-aw/pkg/cli/inventory"

func handleWorkflowCommand(userInput string) error {
    // Normalize user input (strips .md, handles paths)
    workflowName := inventory.NormalizeWorkflowName(userInput)
    
    // Get the actual workflow file path
    workflowPath := inventory.GetWorkflowPath(workflowName, "")
    
    // Read and process the workflow
    content, err := os.ReadFile(workflowPath)
    // ...
}
```

### Listing workflows for status display

```go
import "github.com/githubnext/gh-aw/pkg/cli/inventory"

func listAllWorkflows() error {
    // Get all workflows including campaigns
    workflows, err := inventory.ListWorkflowFiles("", true, false)
    if err != nil {
        return err
    }
    
    for _, wf := range workflows {
        fmt.Printf("Workflow: %s (%s)\n", wf.Name, wf.Path)
        
        // Check if lock file exists
        if fileExists(wf.LockPath) {
            fmt.Printf("  ✓ Compiled: %s\n", wf.LockPath)
        } else {
            fmt.Printf("  ✗ Not compiled\n")
        }
    }
    
    return nil
}
```

### Extracting names from GitHub API responses

```go
import "github.com/githubnext/gh-aw/pkg/cli/inventory"

func processGitHubWorkflows(apiWorkflows []GitHubWorkflow) {
    for _, apiWorkflow := range apiWorkflows {
        // API returns paths like ".github/workflows/ci.lock.yml"
        workflowName := inventory.ExtractWorkflowName(apiWorkflow.Path)
        fmt.Printf("Workflow: %s\n", workflowName)
    }
}
```

## Migration Guide

### Before (scattered logic)

```go
// In workflows.go
func extractWorkflowNameFromPath(path string) string {
    base := filepath.Base(path)
    name := strings.TrimSuffix(base, filepath.Ext(base))
    return strings.TrimSuffix(name, ".lock")
}

// In status_command.go (duplicate)
func getMarkdownWorkflowFiles(dir string) ([]string, error) {
    mdFiles, err := filepath.Glob(filepath.Join(dir, "*.md"))
    // ...
}

// In many files
workflowName := strings.TrimSuffix(filepath.Base(file), ".md")
```

### After (unified inventory package)

```go
import "github.com/githubnext/gh-aw/pkg/cli/inventory"

// Unified extraction
workflowName := inventory.ExtractWorkflowName(path)

// Unified listing
workflows, err := inventory.ListWorkflowFiles("", false, false)

// Normalized names
name := inventory.NormalizeWorkflowName(userInput)
```

## Benefits

1. **Single Source of Truth**: All workflow name logic in one place
2. **Comprehensive**: Handles all workflow types (regular, campaign, generated)
3. **Well-Tested**: 56+ test cases covering edge cases
4. **Type-Safe**: Strongly-typed WorkflowFile struct
5. **Discoverable**: Clear function names and documentation
6. **Extensible**: Easy to add new workflow types or filters

## Design Decisions

### Why separate ExtractWorkflowName and NormalizeWorkflowName?

While they do the same thing functionally, the semantic difference is important:
- `ExtractWorkflowName`: Used when processing known file paths (API responses, file listings)
- `NormalizeWorkflowName`: Used when handling user input (CLI arguments, interactive prompts)

This makes code intent clearer and helps future developers understand the context.

### Why filter README.md?

README.md files in the workflows directory are documentation, not workflows. The filter:
- Is case-insensitive (matches README.md, readme.md, ReadMe.md)
- Only filters exact matches (allows README-test.md, test-README.md)
- Is applied automatically to protect against common mistakes

### Why separate include flags for campaigns and generated files?

Different use cases need different visibility:
- **User-facing commands** (status, list): Show regular and campaign workflows, hide generated
- **Internal operations** (compile): Need to see all files including generated
- **Cleanup operations**: Might need to target only generated files

## Testing

The package has comprehensive test coverage:

```bash
# Run inventory package tests
go test ./pkg/cli/inventory/...

# Run with verbose output
go test -v ./pkg/cli/inventory/...

# Check test coverage
go test -cover ./pkg/cli/inventory/...
```

Test coverage includes:
- All file type variations (regular, campaign, lock, generated)
- Path handling (relative, absolute, with/without directories)
- Edge cases (empty input, no extension, multiple dots)
- README.md filtering (all case variations)
- Directory listing with filtering options
- Error conditions (non-existent directories)

## Future Enhancements

Possible future additions to this package:

1. **Workflow validation**: Check if workflow files are valid
2. **Dependency tracking**: Find workflows that include/reference others
3. **Metadata extraction**: Parse frontmatter without full compilation
4. **Workflow templates**: Support for workflow templates/scaffolding
5. **Batch operations**: Rename, move, or organize multiple workflows
6. **Search/filtering**: Find workflows by name pattern, tags, or content

## Related Packages

- `pkg/parser`: Parses workflow markdown and frontmatter
- `pkg/workflow`: Compiles workflows to GitHub Actions YAML
- `pkg/campaign`: Campaign workflow orchestration
- `pkg/cli/fileutil`: General file utilities
