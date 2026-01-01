# Architecture

This document explains the architectural patterns and design decisions that govern how GitHub Agentic Workflows is organized.

## Overview

GitHub Agentic Workflows implements a clear, maintainable codebase through **five distinct architectural patterns**. These patterns emerged organically as the system grew and have proven effective at maintaining code quality while enabling rapid feature development.

**Key Metrics**:
- **367** implementation files (non-test Go files)
- **703** test files
- **1.9:1** test-to-implementation ratio
- **194** files in `pkg/workflow` (the core compilation package)

## Table of Contents

- [The Five Architectural Patterns](#the-five-architectural-patterns)
  - [Pattern 1: Safe-Output Operations](#pattern-1-safe-output-operations)
  - [Pattern 2: Complex Feature Architecture](#pattern-2-complex-feature-architecture)
  - [Pattern 3: Feature-Specific Files](#pattern-3-feature-specific-files)
  - [Pattern 4: Core Infrastructure](#pattern-4-core-infrastructure)
  - [Pattern 5: Passthrough Extraction](#pattern-5-passthrough-extraction)
- [File Naming Conventions](#file-naming-conventions)
- [Decision Trees](#decision-trees)
- [Package Organization](#package-organization)
- [Testing Strategy](#testing-strategy)
- [Related Documentation](#related-documentation)

---

## The Five Architectural Patterns

### Pattern 1: Safe-Output Operations

**Purpose**: Handle GitHub API write operations through a unified, secure interface

**Characteristics**:
- One `create_*.go` file per GitHub entity type
- Comprehensive configuration parsing
- Input sanitization and validation
- Staged mode support (preview without writing)

**File Structure** (30+ files):
```
pkg/workflow/
├── create_issue.go                    # Issue creation
├── create_pull_request.go             # PR creation
├── create_discussion.go               # Discussion creation
├── create_code_scanning_alert.go      # Code scanning alert
├── create_pr_review_comment.go        # PR review comments
├── create_agent_task.go               # Agent task creation
├── compiler_safe_outputs.go           # Compiler integration
├── compiler_safe_outputs_core.go      # Core safe output logic
├── compiler_safe_outputs_prs.go       # PR-specific compilation
├── compiler_safe_outputs_discussions.go # Discussion-specific
├── compiler_safe_outputs_shared.go    # Shared safe output utilities
├── compiler_safe_outputs_specialized.go # Specialized operations
├── safe_outputs.go                    # Module documentation
├── safe_outputs_config.go             # Configuration parsing
├── safe_outputs_steps.go              # Step builders
├── safe_outputs_env.go                # Environment variables
├── safe_outputs_jobs.go               # Job assembly
└── [20+ additional safe output files]
```

**Examples**:

**Issue Creation** (`create_issue.go`):
```go
type CreateIssuesConfig struct {
    BaseSafeOutputConfig `yaml:",inline"`
    TitlePrefix          string   `yaml:"title-prefix,omitempty"`
    Labels               []string `yaml:"labels,omitempty"`
    AllowedLabels        []string `yaml:"allowed-labels,omitempty"`
    Assignees            []string `yaml:"assignees,omitempty"`
    TargetRepoSlug       string   `yaml:"target-repo,omitempty"`
    AllowedRepos         []string `yaml:"allowed-repos,omitempty"`
    Expires              int      `yaml:"expires,omitempty"`
}
```

**Pull Request Creation** (`create_pull_request.go`):
```go
type CreatePullRequestConfig struct {
    BaseSafeOutputConfig `yaml:",inline"`
    TitlePrefix          string   `yaml:"title-prefix,omitempty"`
    Labels               []string `yaml:"labels,omitempty"`
    AllowedLabels        []string `yaml:"allowed-labels,omitempty"`
    Base                 string   `yaml:"base,omitempty"`
    Reviewers            []string `yaml:"reviewers,omitempty"`
    Draft                bool     `yaml:"draft,omitempty"`
}
```

**When to Use**:
- ✅ Adding new GitHub entity write operations
- ✅ Creating safe output handlers
- ✅ Implementing API integrations with validation

**Pattern Benefits**:
- **Isolation**: Each entity type has dedicated implementation
- **Testability**: Clear boundaries for unit and integration tests
- **Security**: Centralized validation and sanitization
- **Discoverability**: Predictable naming makes features easy to find

---

### Pattern 2: Complex Feature Architecture

**Purpose**: Implement sophisticated features that span multiple aspects with 5+ dedicated files

**Characteristics**:
- Main implementation file
- Multiple domain-specific files
- Comprehensive test coverage
- Shared helper utilities

**File Structure** (5+ files per feature):

**Engine Architecture** (6 files):
```
pkg/workflow/
├── agentic_engine.go          # Base interface (450 lines)
├── copilot_engine.go          # Copilot implementation (971 lines)
├── claude_engine.go           # Claude implementation (340 lines)
├── codex_engine.go            # Codex implementation (639 lines)
├── custom_engine.go           # Custom engine (300 lines)
└── engine_helpers.go          # Shared utilities (424 lines)
```

**MCP (Model Context Protocol)** (8+ files):
```
pkg/workflow/
├── mcp-config.go              # Main configuration (1121 lines)
├── mcp_servers.go             # Server management
├── mcp_renderer.go            # Template rendering
├── gateway.go                 # Gateway integration
├── gateway_validation.go      # Gateway validation
└── [additional MCP files]
```

**Expression Handling** (6+ files):
```
pkg/workflow/
├── expressions.go             # Expression building (948 lines)
├── expression_extraction.go   # Expression extraction
├── expression_parser.go       # Expression parsing
├── expression_nodes.go        # AST nodes
├── expression_builder.go      # Builder pattern
└── expression_validation.go   # Validation logic
```

**Examples**:

**Copilot Engine** (`copilot_engine.go`):
```go
// CompileCopilotEngine generates GitHub Actions configuration for GitHub Copilot
// This includes:
// - Tool configuration and setup
// - GitHub MCP server integration  
// - Participant configuration
// - Prompt step generation
func (c *Compiler) CompileCopilotEngine(workflowData *WorkflowData, ...) ([]map[string]any, error) {
    // 971 lines of Copilot-specific logic
}
```

**Engine Helpers** (`engine_helpers.go`):
```go
// Shared utilities used across all engines:
// - Tool configuration parsing
// - Permission validation
// - Environment variable setup
// - Network configuration
```

**When to Use**:
- ✅ Implementing AI engine support
- ✅ Complex protocol integrations (MCP, HTTP)
- ✅ Features with multiple specialized aspects
- ✅ Systems requiring extensive configuration

**Pattern Benefits**:
- **Modularity**: Each aspect lives in a focused file
- **Maintainability**: Clear separation of concerns
- **Extensibility**: Easy to add new engines or features
- **Reusability**: Shared helpers eliminate duplication

---

### Pattern 3: Feature-Specific Files

**Purpose**: Implement focused features in 1-2 self-contained files

**Characteristics**:
- Single implementation file
- Co-located tests
- Clear, specific naming
- Minimal dependencies

**File Structure** (1-2 files):
```
pkg/workflow/
├── strings.go                 # String utilities (153 lines)
├── strings_test.go            # String utility tests
├── artifacts.go               # Artifact handling (60 lines)
├── artifacts_test.go          # Artifact tests
├── args.go                    # Argument parsing (65 lines)
├── args_test.go               # Argument tests
├── concurrency.go             # Concurrency settings
├── concurrency_test.go        # Concurrency tests
├── features.go                # Feature flags
├── features_test.go           # Feature flag tests
└── [many other focused files]
```

**Examples**:

**String Utilities** (`strings.go` - 153 lines):
```go
// normalizeWhitespace standardizes whitespace in text
func normalizeWhitespace(s string) string

// sanitizeGitHubLabel ensures labels meet GitHub requirements
func sanitizeGitHubLabel(label string) string

// sanitizeGitHubBranch validates branch names
func sanitizeGitHubBranch(branch string) error
```

**Artifact Handling** (`artifacts.go` - 60 lines):
```go
// parseArtifacts extracts artifact configuration
func (c *Compiler) parseArtifacts(frontmatter map[string]any) []ArtifactConfig

// validateArtifacts validates artifact paths and names
func validateArtifacts(artifacts []ArtifactConfig) error
```

**Arguments** (`args.go` - 65 lines):
```go
// parseArgs extracts command-line arguments for runtime tools
func (c *Compiler) parseArgs(frontmatter map[string]any) map[string]string

// validateArgs ensures argument values are safe
func validateArgs(args map[string]string) error
```

**When to Use**:
- ✅ Self-contained utility functions
- ✅ Domain-specific helpers
- ✅ Configuration parsers
- ✅ Validation logic
- ✅ Simple data structures

**Pattern Benefits**:
- **Simplicity**: Easy to understand and modify
- **Focused**: Single responsibility per file
- **Quick Navigation**: Find features instantly
- **Low Overhead**: Minimal abstraction

---

### Pattern 4: Core Infrastructure

**Purpose**: Foundational files that power the entire compilation system

**Characteristics**:
- Large, cohesive files (500-1600 lines)
- Central to system functionality
- Well-documented
- Comprehensive test coverage

**File Structure** (37 foundation files):

**Compiler Core**:
```
pkg/workflow/
├── compiler.go                # Main compilation (1596 lines)
├── compiler_yaml.go           # YAML generation (1020 lines)
├── compiler_jobs.go           # Job generation (806 lines)
├── compiler_orchestrator.go   # Compilation orchestration
└── compiler_types.go          # Core type definitions
```

**Validation System**:
```
pkg/workflow/
├── validation.go              # Centralized validation (782 lines)
├── strict_mode_validation.go  # Strict mode enforcement (190 lines)
├── docker_validation.go       # Docker image validation
├── npm_validation.go          # NPM package validation
├── pip_validation.go          # Python package validation
└── [validation files]
```

**JavaScript Integration**:
```
pkg/workflow/
├── js.go                      # JavaScript bundling (914 lines)
├── bundler.go                 # Code bundler
├── bundler_validation.go      # Bundler validation (360 lines)
└── script_registry.go         # Script management
```

**Permission System**:
```
pkg/workflow/
├── permissions.go             # Permission handling (945 lines)
├── permissions_validator.go   # Permission validation
└── permissions_enum_test.go   # Enum validation tests
```

**Examples**:

**Compiler** (`compiler.go` - 1596 lines):
```go
// CompileWorkflow converts markdown to GitHub Actions YAML
func (c *Compiler) CompileWorkflow(markdownPath string) error

// CompileWorkflowData compiles from already-parsed data
func (c *Compiler) CompileWorkflowData(workflowData *WorkflowData, ...) error

// ParseWorkflowFile parses markdown and extracts frontmatter
func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error)
```

**Validation** (`validation.go` - 782 lines):
```go
// validateExpressionSizes ensures GitHub Actions limits
func validateExpressionSizes(data map[string]any) error

// validateContainerImages checks Docker images exist
func validateContainerImages(containers []string) error

// validateRepositoryFeatures checks repo capabilities
func (c *Compiler) validateRepositoryFeatures(workflowData *WorkflowData) error
```

**When to Use**:
- ✅ Core compilation logic
- ✅ Cross-cutting validation
- ✅ System-wide configuration
- ✅ Central type definitions

**Pattern Benefits**:
- **Cohesion**: Related functionality stays together
- **Performance**: Reduced file loading overhead
- **Comprehensiveness**: Complete feature implementation
- **Authority**: Single source of truth

**Guideline**: Core infrastructure files can be large (500-1600 lines) if they maintain cohesion. Split if distinct responsibilities emerge.

---

### Pattern 5: Passthrough Extraction

**Purpose**: Simple field extraction from YAML frontmatter with minimal logic

**Characteristics**:
- Direct mapping from YAML to structs
- No validation (handled elsewhere)
- No transformation (used as-is)
- Minimal code (5-20 lines per field)

**Common Passthrough Fields** (14 fields):
```go
// Simple string fields
workflowName     := extractString(frontmatter, "name")
description      := extractString(frontmatter, "description")
runName          := extractString(frontmatter, "run-name")

// Simple boolean fields
draft            := extractBool(frontmatter, "draft")
allowForks       := extractBool(frontmatter, "allow-forks")
verbose          := extractBool(frontmatter, "verbose")

// Simple integer fields
maxTurns         := extractInt(frontmatter, "max-turns")
timeoutMinutes   := extractInt(frontmatter, "timeout-minutes")

// Simple list fields
dependsOn        := extractStringList(frontmatter, "depends-on")
environments     := extractStringList(frontmatter, "environments")

// Simple map fields
env              := extractMap(frontmatter, "env")
secrets          := extractMap(frontmatter, "secrets")
```

**Implementation Pattern**:
```go
// Centralized extraction helpers in validation.go
func extractString(data map[string]any, key string) string {
    if val, ok := data[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}

func extractBool(data map[string]any, key string) bool {
    if val, ok := data[key]; ok {
        if b, ok := val.(bool); ok {
            return b
        }
    }
    return false
}

func extractStringList(data map[string]any, key string) []string {
    if val, ok := data[key]; ok {
        if list, ok := val.([]any); ok {
            result := make([]string, 0, len(list))
            for _, item := range list {
                if str, ok := item.(string); ok {
                    result = append(result, str)
                }
            }
            return result
        }
    }
    return nil
}
```

**Examples from Codebase**:

**Simple Fields** (compiler.go):
```go
// Extract workflow metadata
workflowName := extractString(frontmatter, "name")
description := extractString(frontmatter, "description")
runName := extractString(frontmatter, "run-name")

// Extract flags
draft := extractBool(frontmatter, "draft")
allowForks := extractBool(frontmatter, "allow-forks")
```

**Environment Variables** (env.go):
```go
// Direct passthrough of environment variables
envVars := extractMap(frontmatter, "env")
for key, value := range envVars {
    // Use directly in GitHub Actions YAML
    workflow["env"][key] = value
}
```

**When to Use**:
- ✅ Simple YAML field extraction
- ✅ Direct value passthrough
- ✅ No business logic required
- ✅ Validation handled separately

**When NOT to Use**:
- ❌ Complex parsing logic needed
- ❌ Value transformation required
- ❌ Validation mixed with extraction
- ❌ Domain-specific processing

**Pattern Benefits**:
- **Simplicity**: Minimal code, easy to understand
- **Consistency**: Same extraction helpers everywhere
- **Separation**: Extraction separate from validation
- **Maintainability**: Changes to extraction don't affect validation

**Anti-Pattern**: Don't mix extraction with validation or transformation:
```go
// ❌ BAD: Mixing extraction with validation
func extractAndValidatePort(data map[string]any) (int, error) {
    port := extractInt(data, "port")
    if port < 1 || port > 65535 {
        return 0, fmt.Errorf("invalid port")
    }
    return port, nil
}

// ✅ GOOD: Separate extraction and validation
port := extractInt(frontmatter, "port")
if err := validatePort(port); err != nil {
    return err
}
```

---

## File Naming Conventions

The codebase uses consistent naming patterns to indicate file purpose and content type.

### YAML and JSON Schema Files: Kebab-Case

**Pattern**: `lowercase-with-hyphens`

**Examples**:
```
pkg/parser/schemas/
├── main-workflow-schema.json
├── included-file-schema.json
└── mcp_config_schema.json
```

**Rationale**: 
- Matches YAML frontmatter field naming
- Consistent with configuration file conventions
- External file format standards

### Go Implementation Files: Snake_Case

**Pattern**: `lowercase_with_underscores`

**Examples**:
```
pkg/workflow/
├── create_issue.go
├── create_pull_request.go
├── compiler_safe_outputs.go
├── strict_mode_validation.go
└── engine_helpers.go
```

**Rationale**:
- Go standard practice for file names
- Consistent with Go ecosystem
- Clear word separation
- Matches function naming

### Test Files: Feature_test.go

**Pattern**: `feature_test.go` or `feature_scenario_test.go`

**Examples**:
```
pkg/workflow/
├── create_issue_test.go
├── create_issue_assignees_test.go
├── create_issue_backward_compat_test.go
├── copilot_engine_test.go
└── engine_helpers_integration_test.go
```

**Test Type Suffixes**:
- `_test.go` - Unit tests
- `_integration_test.go` - Integration tests
- `_benchmark_test.go` - Benchmark tests
- `_fuzz_test.go` - Fuzz tests

### Common Naming Patterns

**Create Operations**:
- Pattern: `create_<entity>.go`
- Examples: `create_issue.go`, `create_pull_request.go`

**Engine Implementations**:
- Pattern: `<engine>_engine.go`
- Examples: `copilot_engine.go`, `claude_engine.go`

**Validation Logic**:
- Pattern: `<domain>_validation.go`
- Examples: `strict_mode_validation.go`, `docker_validation.go`

**Helper Functions**:
- Pattern: `<subsystem>_helpers.go`
- Examples: `engine_helpers.go`, `validation_helpers.go`

**Compiler Components**:
- Pattern: `compiler_<aspect>.go`
- Examples: `compiler_yaml.go`, `compiler_jobs.go`

---

## Decision Trees

### Choosing the Right Pattern

```
┌─────────────────────────────────────┐
│  Need to implement new feature?     │
└──────────────┬──────────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Is it a       │
       │ GitHub API    │     YES     ┌─────────────────────┐
       │ write         ├────────────►│ Pattern 1:          │
       │ operation?    │             │ Safe-Output         │
       └───────┬───────┘             │ Operations          │
               │ NO                  │ (create_*.go)       │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Does it need  │
       │ 5+ files?     │     YES     ┌─────────────────────┐
       │ Multiple      ├────────────►│ Pattern 2:          │
       │ aspects?      │             │ Complex Feature     │
       └───────┬───────┘             │ Architecture        │
               │ NO                  │ (engine_*.go, etc)  │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Is it just    │
       │ extracting    │     YES     ┌─────────────────────┐
       │ YAML fields?  ├────────────►│ Pattern 5:          │
       │ No logic?     │             │ Passthrough         │
       └───────┬───────┘             │ Extraction          │
               │ NO                  │ (extractString)     │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Is it core    │
       │ to entire     │     YES     ┌─────────────────────┐
       │ compilation   ├────────────►│ Pattern 4:          │
       │ system?       │             │ Core Infrastructure │
       └───────┬───────┘             │ (compiler.go, etc)  │
               │ NO                  └─────────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Self-contained│     YES     ┌─────────────────────┐
       │ feature?      ├────────────►│ Pattern 3:          │
       │ 1-2 files?    │             │ Feature-Specific    │
       └───────────────┘             │ (strings.go, etc)   │
                                     └─────────────────────┘
```

### Where to Add New Validation

```
┌─────────────────────────────────────┐
│  Need to add validation?            │
└──────────────┬──────────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Is it about   │
       │ security or   │     YES     ┌─────────────────────┐
       │ strict mode?  ├────────────►│ Add to:             │
       └───────┬───────┘             │ strict_mode_        │
               │ NO                  │ validation.go       │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Is it for     │
       │ external      │     YES     ┌─────────────────────┐
       │ resources?    ├────────────►│ Add to domain file: │
       │ (Docker, NPM) │             │ docker_validation.go│
       └───────┬───────┘             │ npm_validation.go   │
               │ NO                  │ pip_validation.go   │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Is it cross-  │
       │ cutting?      │     YES     ┌─────────────────────┐
       │ Affects       ├────────────►│ Add to:             │
       │ multiple      │             │ validation.go       │
       │ domains?      │             │ (centralized)       │
       └───────┬───────┘             └─────────────────────┘
               │ NO
               │
               ▼
       ┌───────────────┐
       │ Feature-      │     YES     ┌─────────────────────┐
       │ specific      ├────────────►│ Add to feature file:│
       │ validation?   │             │ Near the code it    │
       └───────────────┘             │ validates           │
                                     └─────────────────────┘
```

### Should I Split a File?

```
┌─────────────────────────────────────┐
│  File size or complexity question?  │
└──────────────┬──────────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Is file       │
       │ > 1000 lines? │     YES     ┌─────────────────────┐
       └───────┬───────┘             │ SHOULD split by     │
               │ NO                  │ logical boundaries  │
               │                     └─────────────────────┘
               ▼
       ┌───────────────┐
       │ Is file       │
       │ > 800 lines?  │     YES     ┌─────────────────────┐
       └───────┬───────┘             │ CONSIDER splitting  │
               │ NO                  │ if distinct         │
               │                     │ responsibilities    │
               ▼                     └─────────────────────┘
       ┌───────────────┐
       │ Multiple      │
       │ unrelated     │     YES     ┌─────────────────────┐
       │ responsibilities│───────────►│ SHOULD split by     │
       │ ?             │             │ responsibility      │
       └───────┬───────┘             └─────────────────────┘
               │ NO
               │
               ▼
       ┌───────────────┐
       │ Frequent      │
       │ merge         │     YES     ┌─────────────────────┐
       │ conflicts?    ├────────────►│ CONSIDER splitting  │
       └───────┬───────┘             │ into focused files  │
               │ NO                  └─────────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Core          │
       │ infrastructure│     YES     ┌─────────────────────┐
       │ cohesive?     ├────────────►│ Keep as-is          │
       └───────┬───────┘             │ (Pattern 4)         │
               │ NO                  └─────────────────────┘
               │
               ▼
       ┌───────────────┐
               │ Keep as-is          │
               └─────────────────────┘
```

---

## Package Organization

### pkg/workflow (194 files)

**Purpose**: Core workflow compilation, validation, and GitHub Actions generation

**Key Subsystems**:
- **Compiler**: Main compilation logic (`compiler.go`, `compiler_yaml.go`)
- **Engines**: AI engine implementations (`*_engine.go`)
- **Safe Outputs**: GitHub API operations (`create_*.go`, `safe_outputs_*.go`)
- **Validation**: Input validation (`validation.go`, `*_validation.go`)
- **JavaScript**: JS bundling and execution (`js.go`, `bundler.go`)
- **MCP**: Model Context Protocol support (`mcp*.go`)
- **Expressions**: GitHub Actions expressions (`expression*.go`)

### pkg/parser (20 files)

**Purpose**: Markdown frontmatter parsing and schema validation

**Key Components**:
- Schema definitions (JSON schemas)
- YAML parsing and validation
- Frontmatter extraction
- Include file processing

### pkg/cli (124 files)

**Purpose**: Command-line interface implementation

**Key Commands**:
- `compile` - Compile workflows
- `mcp` - MCP server management
- `logs` - Log analysis
- `audit` - Workflow auditing

### pkg/console (small package)

**Purpose**: Formatted console output for CLI

**Key Features**:
- Colored output
- Error formatting
- Progress indicators
- Message styling

### pkg/logger (small package)

**Purpose**: Debug logging with namespace-based filtering

**Key Features**:
- Selective logging via DEBUG environment variable
- Automatic color assignment per namespace
- Performance timing

---

## Testing Strategy

### Test Organization

**Unit Tests**: Co-located with implementation
```
feature.go
feature_test.go
```

**Integration Tests**: Marked with `_integration_test.go`
```
feature_integration_test.go
```

**Specialized Tests**:
- `*_benchmark_test.go` - Performance tests
- `*_fuzz_test.go` - Fuzz testing
- `*_regression_test.go` - Security regression tests

### Test Patterns

**Table-Driven Tests**:
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
    }{
        {"valid input", "test", false},
        {"invalid input", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            if (err != nil) != tt.expectError {
                t.Errorf("expected error: %v, got: %v", 
                    tt.expectError, err)
            }
        })
    }
}
```

**Test Ratio**: 1.9 test files per implementation file
- Comprehensive coverage
- Multiple test files per feature
- Integration and unit tests
- Security regression tests

### Running Tests

```bash
make test-unit       # Fast unit tests (~25s)
make test            # Full test suite (~30s)
make agent-finish    # Complete validation
```

---

## Related Documentation

### Core Documentation

- **[specs/code-organization.md](specs/code-organization.md)** - Detailed file organization guidance
- **[specs/validation-architecture.md](specs/validation-architecture.md)** - Validation system architecture
- **[specs/testing.md](specs/testing.md)** - Comprehensive testing guidelines
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[DEVGUIDE.md](DEVGUIDE.md)** - Development guide

### Specification Files

- **[specs/safe-output-messages.md](specs/safe-output-messages.md)** - Safe output system design
- **[specs/breaking-cli-rules.md](specs/breaking-cli-rules.md)** - CLI compatibility rules
- **[specs/capitalization.md](specs/capitalization.md)** - Capitalization guidelines
- **[specs/yaml-version-gotchas.md](specs/yaml-version-gotchas.md)** - YAML compatibility

### Skills Documentation

- **[skills/developer.md](skills/developer.md)** - Developer instructions
- **[skills/console-rendering.md](skills/console-rendering.md)** - Console output formatting
- **[skills/error-messages.md](skills/error-messages.md)** - Error message style guide

---

## Summary

GitHub Agentic Workflows maintains code quality through five architectural patterns:

1. **Safe-Output Operations** - Unified GitHub API operations (30+ files)
2. **Complex Feature Architecture** - Multi-file features (5+ files each)
3. **Feature-Specific Files** - Focused implementations (1-2 files)
4. **Core Infrastructure** - Foundation files (37 files)
5. **Passthrough Extraction** - Simple YAML extraction (14 fields)

These patterns emerged organically and have proven effective at maintaining a large codebase (367 implementation files, 703 test files) with high code quality and test coverage (1.9:1 ratio).

**Key Principles**:
- **Clarity**: Predictable naming and organization
- **Isolation**: Clear boundaries between features
- **Testability**: Comprehensive test coverage
- **Maintainability**: Patterns that scale with growth
- **Discoverability**: Easy to find and understand code

---

**Last Updated**: 2025-01-01  
**Related Issues**: [#8372](https://github.com/githubnext/gh-aw/discussions/8372), [#8374](https://github.com/githubnext/gh-aw/issues/8374)
