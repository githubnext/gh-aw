# Code Organization Patterns

This document describes the organizational patterns used in the gh-aw codebase, based on semantic function clustering analysis.

## Overview

The gh-aw codebase (401 non-test Go files) follows clear semantic grouping patterns that improve maintainability and discoverability. This document captures these patterns for future reference.

## Package Organization

### pkg/workflow (227 files)

The largest package, organized by functional domain:

#### Compiler Files
- `compiler.go` - Core compiler interface
- `compiler_*.go` - Compiler feature modules (jobs, events, permissions, etc.)
- Pattern: `compiler_{feature}.go` for specific compilation concerns

#### Engine Files
- `{engine}_engine.go` - Engine implementation (claude, codex, copilot, custom)
- `{engine}_*.go` - Engine-specific features (logs, mcp, tools, etc.)
- Pattern: Each engine has a focused set of files for its concerns

#### Validation Files (30 files)
- `*_validation.go` - Domain-specific validation logic
- Examples: `mcp_config_validation.go`, `dangerous_permissions_validation.go`
- Pattern: One validation file per validation domain (follows specs/validation-refactoring.md)

#### Helper Files (14 files)
- `*_helpers.go` - Shared utilities for specific domains
- Each helper file includes organization rationale in header comments
- Pattern: Functions used by 3+ callers that share a clear domain focus

### pkg/stringutil

Focused utility package for string operations:
- `stringutil.go` - General-purpose utilities (Truncate, NormalizeWhitespace, ParseVersionValue)
- `identifiers.go` - Workflow and identifier normalization (NormalizeWorkflowName, NormalizeSafeOutputIdentifier)
- `sanitize.go` - Security-focused sanitization (SanitizeErrorMessage, SanitizeParameterName)
- `paths.go` - Path manipulation utilities

## Helper File Patterns

Helper files in `pkg/workflow` follow consistent conventions:

### 1. **config_helpers.go** (353 lines)
**Purpose**: Safe output configuration parsing
**Key Functions**:
- `ParseStringArrayFromConfig()` - Generic string array extraction
- `parseLabelsFromConfig()` - Extract labels array
- `parseTargetRepoFromConfig()` - Extract target repository
**Used By**: Safe output processors (3+ callers)

### 2. **engine_helpers.go** (348 lines)
**Purpose**: Shared AI engine utilities
**Key Functions**:
- `GenerateAgentInstallSteps()` - Agent installation workflow
- `GenerateNpmInstallStep()` - NPM package installation
- `GenerateMultiSecretValidationStep()` - Secret validation
**Used By**: Claude, Codex, Copilot, Custom engines

### 3. **error_helpers.go** (268 lines)
**Purpose**: Validation error construction
**Key Types**:
- `ValidationError` - Structured validation errors
- `NewValidationError()` - Error constructor with context
**Used By**: All validation modules

### 4. **compiler_yaml_helpers.go** (195 lines)
**Purpose**: YAML generation utilities
**Used By**: Compiler modules

### 5. **close_entity_helpers.go** (211 lines)
**Purpose**: Entity closing operations (issues, PRs, discussions)
**Used By**: Close action handlers

### 6. **update_entity_helpers.go** (389 lines)
**Purpose**: Entity update operations
**Used By**: Update action handlers

### 7. **safe_outputs_config_helpers.go** (43 lines)
**Purpose**: Safe outputs configuration utilities
**Used By**: Safe outputs generation

### 8. **safe_outputs_config_helpers_reflection.go** (92 lines)
**Purpose**: Reflection-based config generation
**Used By**: Safe outputs configuration

### 9. **safe_outputs_config_generation_helpers.go** (134 lines)
**Purpose**: Safe outputs config generation
**Used By**: Safe outputs generation

### 10. **git_helpers.go** (63 lines)
**Purpose**: Git command generation
**Used By**: Git-related actions

### 11. **map_helpers.go** (75 lines)
**Purpose**: Map manipulation utilities
**Used By**: Configuration parsing

### 12. **prompt_step_helper.go** (119 lines)
**Purpose**: Prompt step construction
**Used By**: Prompt generation

### 13. **validation_helpers.go** (38 lines)
**Purpose**: Shared validation utilities
**Used By**: Validation modules

### 14. **compiler_test_helpers.go** (not counted - test file)
**Purpose**: Test utilities
**Used By**: Compiler tests

## String Processing Organization

Two distinct patterns documented in `pkg/workflow/strings.go`:

### Sanitize Pattern (Character Validity)
**Location**: `pkg/workflow/strings.go`
**Purpose**: Remove or replace invalid characters

Functions:
- `SanitizeName()` - Configurable sanitization
- `SanitizeWorkflowName()` - Artifact names and file paths
- `SanitizeIdentifier()` (in workflow_name.go) - User agent identifiers

Use when:
- Processing user input with invalid characters
- Creating identifiers, artifact names, or file paths
- Ensuring character validity for a specific context

### Normalize Pattern (Format Standardization)
**Location**: `pkg/stringutil/identifiers.go`
**Purpose**: Standardize format between representations

Functions:
- `NormalizeWorkflowName()` - Remove file extensions (.md, .lock.yml)
- `NormalizeSafeOutputIdentifier()` - Convert dashes to underscores
- `MarkdownToLockFile()` - Convert .md to .lock.yml paths
- `LockFileToMarkdown()` - Convert .lock.yml to .md paths

Use when:
- Converting between file names and identifiers
- Standardizing naming conventions
- Input is valid but needs format conversion

**Key Distinction**: Sanitize ensures character validity; Normalize converts between valid formats.

See `pkg/workflow/strings.go` (lines 1-75) for comprehensive documentation.

## Engine Implementation Pattern

All engines extend `BaseEngine` and follow a consistent structure:

### Core Files
1. `{engine}_engine.go` - Constructor and interface implementation
2. `{engine}_logs.go` - Log parsing and metrics extraction (if applicable)
3. `{engine}_mcp.go` - MCP server configuration (if applicable)
4. `{engine}_tools.go` - Tool configuration (if applicable)

### Example: Copilot Engine
- `copilot_engine.go` - Core engine interface
- `copilot_engine_installation.go` - Installation workflow
- `copilot_engine_execution.go` - Execution workflow
- `copilot_engine_tools.go` - Tool permissions
- `copilot_logs.go` - Log parsing
- `copilot_mcp.go` - MCP configuration
- `copilot_srt.go` - Sandbox runtime
- `copilot_participant_steps.go` - Participant steps

This modular organization improves maintainability and makes it easy to locate specific functionality.

## Validation File Organization

Validation files follow domain-specific grouping:

**Pattern**: `{domain}_validation.go` or `{domain}_{subdomain}_validation.go`

Examples:
- `mcp_config_validation.go` - MCP configuration validation
- `dangerous_permissions_validation.go` - Permission security checks
- `bundler_runtime_validation.go` - Bundler runtime checks
- `bundler_safety_validation.go` - Bundler safety checks
- `bundler_script_validation.go` - Bundler script validation

**Guidelines**:
- Target size: 100-200 lines per validator
- Hard limit: 300 lines (refactor if exceeded)
- Split when file contains 2+ unrelated domains
- See `specs/validation-refactoring.md` for details

## Test Organization

Tests co-locate with implementation:
- `{file}.go` → `{file}_test.go` pattern
- Unit tests excluded with `! -name "*_test.go"` filters
- Integration tests marked with `//go:build integration` tag

## Key Principles

1. **Semantic Grouping**: Files grouped by domain/feature, not by type
2. **Clear Naming**: File names indicate purpose (compiler, engine, validation, helper)
3. **Focused Modules**: Each file has a clear, single responsibility
4. **Documentation**: Helper files include organization rationale in headers
5. **Reusability**: Helper files contain functions used by 3+ callers
6. **Stability**: Helper functions change infrequently

## Cross-References

- Helper file conventions: `skills/developer/SKILL.md#helper-file-conventions`
- Validation refactoring guide: `specs/validation-refactoring.md`
- String sanitization patterns: `specs/string-sanitization-normalization.md`
- CLI command patterns: `specs/cli-command-patterns.md`
- Testing guidelines: `specs/testing.md`

## Analysis Summary

This organization emerged from analyzing 401 non-test Go files across the repository:
- **pkg/workflow**: 227 files (compiler, engines, validation, safe outputs)
- **pkg/cli**: 136 files (command implementations)
- **Other packages**: 38 files (utilities, constants, logger, etc.)

**Key Finding**: The codebase exhibits strong semantic organization with clear patterns. Helper files serve distinct, documented purposes. Validation is appropriately distributed. Engine implementations follow consistent structure.

**Recommendation**: Maintain current organization patterns. New code should follow these established conventions.
