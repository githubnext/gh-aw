# Custom GitHub Actions Build System

> Specification documenting the custom actions directory structure, build tooling, and architectural decisions.
> Last updated: 2025-12-09

## Overview

This document describes the custom GitHub Actions build system implemented to support migrating from inline JavaScript (using `actions/github-script`) to standalone custom actions. The system provides a foundation for creating, building, and managing custom GitHub Actions that can be referenced in compiled workflows.

## Table of Contents

- [Motivation](#motivation)
- [Architecture](#architecture)
- [Directory Structure](#directory-structure)
- [Build System](#build-system)
- [Architectural Decisions](#architectural-decisions)
- [Usage Guide](#usage-guide)
- [CI Integration](#ci-integration)
- [Development Guide](#development-guide)
- [Future Work](#future-work)

## Motivation

### Problem Statement

The workflow compiler generates inline JavaScript code embedded in YAML files using `actions/github-script`. This approach has several limitations:

1. **No Version Control**: JavaScript code is embedded in compiled `.lock.yml` files without semantic versioning
2. **Limited Reusability**: Same JavaScript logic is duplicated across multiple workflows
3. **Testing Challenges**: Inline scripts are harder to test independently
4. **Maintenance Burden**: Changes require recompiling all affected workflows
5. **Distribution Issues**: Cannot easily share actions across repositories

### Solution

Create a custom actions system that:
- Stores actions in a dedicated `actions/` directory
- Provides versioning through Git tags/releases
- Enables reuse via `uses: ./actions/{action-name}`
- Supports independent testing and validation
- Leverages existing bundler infrastructure from workflow compilation

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│                    gh-aw CLI Tool                        │
│  ┌────────────────────────────────────────────────────┐ │
│  │  actions-build  │  actions-validate  │  actions-clean│ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│            pkg/cli/actions_build_command.go              │
│  ┌────────────────────────────────────────────────────┐ │
│  │  • ActionsBuildCommand()                           │ │
│  │  • ActionsValidateCommand()                        │ │
│  │  • ActionsCleanCommand()                           │ │
│  │  • getActionDependencies() - Manual mapping        │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│         Reused Workflow Infrastructure                   │
│  ┌────────────────────────────────────────────────────┐ │
│  │  workflow.GetJavaScriptSources()                   │ │
│  │    - All embedded .cjs files from pkg/workflow/js/ │ │
│  │    - Accessed via map[string]string               │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    actions/ Directory                    │
│  ┌────────────────────────────────────────────────────┐ │
│  │  setup-safe-inputs/    setup-safe-outputs/         │ │
│  │  ├── action.yml        ├── action.yml              │ │
│  │  ├── index.js          ├── index.js                │ │
│  │  ├── src/              ├── src/                    │ │
│  │  │   └── index.js      │   └── index.js            │ │
│  │  └── README.md         └── README.md               │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### 1. CLI Commands (`cmd/gh-aw/main.go`)
- Cobra command definitions for `actions-build`, `actions-validate`, `actions-clean`
- Integrated into `gh aw` command hierarchy under "Development Commands" group
- Commands delegate to `pkg/cli/actions_build_command.go`

#### 2. Build Command Implementation (`pkg/cli/actions_build_command.go`)
- **ActionsBuildCommand()**: Builds all actions by bundling dependencies
- **ActionsValidateCommand()**: Validates action.yml files
- **ActionsCleanCommand()**: Removes generated index.js files
- **getActionDependencies()**: Maps action names to required JavaScript files

#### 3. JavaScript Sources (`pkg/workflow/js.go`)
- `GetJavaScriptSources()`: Returns map of all embedded JavaScript files
- Files are embedded using `//go:embed` directives
- Single source of truth for JavaScript dependencies

#### 4. Actions Directory (`actions/`)
- Contains custom action subdirectories
- Each action follows GitHub Actions standard structure
- Source files in `src/`, compiled output in root

## Directory Structure

### Repository Layout

```
gh-aw/
├── actions/                          # Custom GitHub Actions
│   ├── README.md                     # Actions documentation
│   ├── setup-safe-inputs/           # Safe inputs MCP server setup
│   │   ├── action.yml               # Action metadata
│   │   ├── index.js                 # Bundled output (committed)
│   │   ├── src/                     # Source files
│   │   │   └── index.js             # Source that references FILES constant
│   │   └── README.md                # Action-specific docs
│   └── setup-safe-outputs/          # Safe outputs MCP server setup
│       ├── action.yml               # Action metadata
│       ├── index.js                 # Bundled output (committed)
│       ├── src/                     # Source files
│       │   └── index.js             # Source that references FILES constant
│       └── README.md                # Action-specific docs
├── pkg/
│   ├── cli/
│   │   └── actions_build_command.go # Build system implementation
│   └── workflow/
│       ├── js.go                    # JavaScript sources map
│       └── js/                      # Embedded JavaScript files
│           ├── *.cjs                # CommonJS modules
│           └── *.json               # JSON configuration files
├── cmd/gh-aw/
│   └── main.go                      # CLI entry point with commands
├── Makefile                         # Build targets
├── .gitattributes                   # Mark generated files
└── .github/workflows/
    └── ci.yml                       # CI pipeline with actions-build job
```

### Action Structure

Each action follows this template:

```
actions/{action-name}/
├── action.yml          # Metadata: name, description, inputs, outputs, runs
├── index.js            # Bundled JavaScript (generated, committed)
├── src/                # Source files
│   └── index.js        # Main entry point with FILES placeholder
└── README.md           # Action documentation
```

### action.yml Format

```yaml
name: 'Action Name'
description: 'Action description'
inputs:
  destination:
    description: 'Destination directory path'
    required: true
    default: '/tmp/action-files'
runs:
  using: 'node20'
  main: 'index.js'
```

### Source File Pattern

Source files use a `FILES` constant that gets replaced during build:

```javascript
const fs = require('fs');
const path = require('path');

// This object is populated during build with embedded file contents
const FILES = {};

// Main action code that uses FILES
const destinationDir = process.env.INPUT_DESTINATION || '/tmp/action-files';

// Create directory and write files
for (const [filename, content] of Object.entries(FILES)) {
  const filepath = path.join(destinationDir, filename);
  fs.mkdirSync(path.dirname(filepath), { recursive: true });
  fs.writeFileSync(filepath, content, 'utf8');
}
```

## Build System

### Build Process

The build system follows these steps:

1. **Discovery**: Scans `actions/` directory for action subdirectories
2. **Validation**: Validates each `action.yml` file structure
3. **Dependency Resolution**: Maps action name to required JavaScript files
4. **File Reading**: Retrieves file contents from `workflow.GetJavaScriptSources()`
5. **Bundling**: Creates JSON object with all dependencies
6. **Code Generation**: Replaces `FILES` placeholder in source with bundled content
7. **Output**: Writes bundled `index.js` to action directory

### Build Commands

#### Command Line

```bash
# Build all actions
gh aw actions-build

# Validate action.yml files
gh aw actions-validate

# Clean generated files
gh aw actions-clean
```

#### Makefile

```bash
# Build all actions
make actions-build

# Validate action.yml files
make actions-validate

# Clean generated files
make actions-clean
```

### Implementation Details

#### Dependency Mapping

Currently uses manual mapping in `getActionDependencies()`:

```go
func getActionDependencies(actionName string) []string {
    dependencyMap := map[string][]string{
        "setup-safe-outputs": {
            "safe_outputs_mcp_server.cjs",
            "safe_outputs_bootstrap.cjs",
            "safe_outputs_tools_loader.cjs",
            "safe_outputs_config.cjs",
            "safe_outputs_handlers.cjs",
            "safe_outputs_tools.json",
            "mcp_server_core.cjs",
            "mcp_logger.cjs",
            "messages.cjs",
        },
        "setup-safe-inputs": {
            "safe_inputs_mcp_server.cjs",
            "safe_inputs_bootstrap.cjs",
            "safe_inputs_config_loader.cjs",
            "safe_inputs_tool_factory.cjs",
            "safe_inputs_validation.cjs",
            "mcp_server_core.cjs",
            "mcp_logger.cjs",
        },
    }
    
    if deps, ok := dependencyMap[actionName]; ok {
        return deps
    }
    return []string{}
}
```

#### File Embedding

Files are embedded at build time using regex replacement:

```go
// Replace the FILES placeholder in source
filesRegex := regexp.MustCompile(`(?s)const FILES = \{[^}]*\};`)
outputContent := filesRegex.ReplaceAllString(
    string(sourceContent), 
    fmt.Sprintf("const FILES = %s;", strings.TrimSpace(indentedJSON))
)
```

## Architectural Decisions

### Decision 1: Reuse Workflow Bundler Infrastructure

**Decision**: Leverage existing `workflow.GetJavaScriptSources()` instead of creating separate bundling system.

**Rationale**:
- Eliminates code duplication
- Single source of truth for JavaScript files
- Maintains consistency with workflow compilation
- Reduces maintenance burden

**Implications**:
- Actions and workflows share same JavaScript dependencies
- Changes to embedded files affect both systems
- Build system must stay in sync with workflow compiler

### Decision 2: Manual Dependency Mapping

**Decision**: Use explicit map of action names to required files rather than automatic dependency resolution.

**Rationale**:
- Simpler implementation for initial version
- Explicit dependencies are easier to understand
- Fewer moving parts reduces complexity
- Can migrate to automatic resolution later

**Trade-offs**:
- Must manually update when dependencies change
- Risk of forgetting to update mapping
- More maintenance overhead

**Future**: Implement automatic dependency resolution using `FindJavaScriptDependencies()` from bundler.

### Decision 3: Commit Bundled Files

**Decision**: Commit generated `index.js` files to Git, marked as `linguist-generated`.

**Rationale**:
- GitHub Actions requires files to be in repository
- No runtime build step needed in workflows
- Easier for consumers to use actions
- Git diff shows what changed

**Implications**:
- Repository size increases with bundled files
- Must rebuild and commit after changes
- Generated files appear in diffs (marked as generated)

### Decision 4: Use `go run` for Development Commands

**Decision**: Run action commands via `go run ./cmd/gh-aw` instead of building binary first.

**Rationale**:
- Faster iteration during development
- No stale binary issues
- Simpler developer workflow
- Commands are project-specific

**Trade-offs**:
- Slightly slower execution (compilation overhead)
- Requires Go toolchain

### Decision 5: Node.js 20 Runtime

**Decision**: Use `node20` as the runtime for all actions.

**Rationale**:
- Latest stable Node.js version supported by GitHub Actions
- Modern JavaScript features available
- Consistent with workflow compilation environment

## Usage Guide

### Using Actions in Workflows

Actions can be referenced using relative paths:

```yaml
jobs:
  my-job:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Safe Inputs
        uses: ./actions/setup-safe-inputs
        with:
          destination: /tmp/safe-inputs
      
      - name: Setup Safe Outputs
        uses: ./actions/setup-safe-outputs
        with:
          destination: /tmp/safe-outputs
```

### Creating a New Action

1. **Create directory structure**:
   ```bash
   mkdir -p actions/my-action/src
   ```

2. **Create action.yml**:
   ```yaml
   name: 'My Action'
   description: 'Description of my action'
   inputs:
     destination:
       description: 'Destination directory'
       required: true
       default: '/tmp/my-action'
   runs:
     using: 'node20'
     main: 'index.js'
   ```

3. **Create src/index.js**:
   ```javascript
   const fs = require('fs');
   const path = require('path');
   
   const FILES = {};
   
   const destinationDir = process.env.INPUT_DESTINATION || '/tmp/my-action';
   
   for (const [filename, content] of Object.entries(FILES)) {
     const filepath = path.join(destinationDir, filename);
     fs.mkdirSync(path.dirname(filepath), { recursive: true });
     fs.writeFileSync(filepath, content, 'utf8');
   }
   
   console.log(`Files copied to ${destinationDir}`);
   ```

4. **Update dependency mapping** in `pkg/cli/actions_build_command.go`:
   ```go
   func getActionDependencies(actionName string) []string {
       dependencyMap := map[string][]string{
           // ... existing mappings ...
           "my-action": {
               "required_file1.cjs",
               "required_file2.cjs",
           },
       }
       // ...
   }
   ```

5. **Build and test**:
   ```bash
   make actions-build
   make actions-validate
   ```

6. **Create README.md** documenting the action

### Modifying Existing Actions

1. **Edit source files** in `actions/{action-name}/src/`
2. **Update dependencies** if needed in `pkg/cli/actions_build_command.go`
3. **Rebuild**: `make actions-build`
4. **Validate**: `make actions-validate`
5. **Test**: Use action in a workflow and verify behavior
6. **Commit**: Include both source and generated `index.js` changes

## CI Integration

### Actions Build Job

The CI pipeline includes an `actions-build` job that validates actions on every pull request:

```yaml
actions-build:
  needs: [lint]
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v5
    - uses: actions/setup-go@v6
      with:
        go-version-file: go.mod
        cache: true
    - run: go mod verify
    - run: make actions-build
    - run: make actions-validate
```

### Trigger Conditions

The CI runs when:
- Any `.go` file changes
- Any file in `actions/**` changes
- `.github/workflows/ci.yml` changes
- Workflow markdown files change

### What Gets Validated

1. **Go code compilation**: Ensures build system compiles
2. **Action building**: All actions must build successfully
3. **Action validation**: All action.yml files must be valid
4. **Dependency resolution**: All referenced files must exist

## Development Guide

### For Future Agents

#### Quick Start

1. **Understand the structure**:
   ```bash
   tree actions/
   ```

2. **Explore build system**:
   - Read `pkg/cli/actions_build_command.go`
   - Check `getActionDependencies()` for mapping

3. **Test locally**:
   ```bash
   make actions-build
   make actions-validate
   ```

4. **Check CI**:
   - Look at `.github/workflows/ci.yml`
   - Find `actions-build` job

#### Common Tasks

**Add a new action**:
1. Create directory structure in `actions/`
2. Write `action.yml`, `src/index.js`, `README.md`
3. Update `getActionDependencies()` in `actions_build_command.go`
4. Run `make actions-build`
5. Test in a workflow

**Update dependencies**:
1. Modify `getActionDependencies()` mapping
2. Rebuild: `make actions-build`
3. Verify: Check generated `index.js` has new files

**Add new JavaScript source**:
1. Add file to `pkg/workflow/js/`
2. Add `//go:embed` directive in `js.go`
3. Add to `GetJavaScriptSources()` map
4. Update action dependencies as needed
5. Run `make build` to embed new file

#### Key Files to Know

- `pkg/cli/actions_build_command.go` - Build system logic
- `pkg/workflow/js.go` - JavaScript source map
- `cmd/gh-aw/main.go` - CLI command definitions
- `Makefile` - Build targets
- `.github/workflows/ci.yml` - CI validation
- `actions/README.md` - Actions documentation

#### Debugging Tips

**Action won't build**:
- Check if all dependencies exist in `GetJavaScriptSources()`
- Verify `getActionDependencies()` mapping is correct
- Look for typos in filenames

**Generated file looks wrong**:
- Check regex pattern in `buildAction()`
- Verify source file has `const FILES = {};` placeholder
- Ensure JSON indentation is correct

**CI failing**:
- Run `make actions-build` locally first
- Check Go syntax errors
- Verify action.yml is valid YAML

## Future Work

### Planned Improvements

1. **Automatic Dependency Resolution**
   - Use `FindJavaScriptDependencies()` from bundler
   - Eliminate manual mapping
   - Parse `require()` statements automatically

2. **Action Versioning**
   - Git tags for action versions
   - Semantic versioning support
   - Version pinning in workflows

3. **Testing Infrastructure**
   - Unit tests for actions
   - Integration tests with workflows
   - Mock GitHub Actions environment

4. **Enhanced Validation**
   - Lint JavaScript code
   - Validate against Actions schema
   - Check for common mistakes

5. **Distribution**
   - Publish actions to GitHub Marketplace
   - Support external repository references
   - Create action templates

6. **Developer Experience**
   - Interactive action creation wizard
   - Auto-generate action.yml from source
   - Hot reload during development

### Migration Path

**Phase 1**: Current State (Complete)
- Directory structure established
- Build system working
- Two initial actions created
- CI integration complete

**Phase 2**: Enhanced Tooling (Next)
- Automatic dependency resolution
- Better error messages
- Validation improvements

**Phase 3**: Workflow Migration (Future)
- Identify inline scripts to migrate
- Create actions from inline code
- Update workflows to use actions

**Phase 4**: Distribution (Future)
- Version and publish actions
- External repository support
- Marketplace presence

## Summary

The custom GitHub Actions build system provides a foundation for migrating from inline JavaScript to versioned, reusable actions. Key achievements:

✅ **Structured directory layout** following GitHub Actions conventions
✅ **Go-based build system** reusing workflow bundler infrastructure  
✅ **CLI integration** with `gh aw actions-*` commands
✅ **CI validation** ensuring actions stay buildable
✅ **Two initial actions** (setup-safe-inputs, setup-safe-outputs)
✅ **Comprehensive documentation** for future development

The system is production-ready and extensible, with clear paths for enhancement and migration of existing inline scripts.
