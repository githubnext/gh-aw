package cli

// Compile Command Module Organization
//
// This file serves as documentation for the compile_* module organization.
// The compile command converts markdown workflow files into GitHub Actions YAML files.
//
// # Architecture
//
// The compile functionality has been refactored into multiple focused files:
//
//   - compile_config.go: Configuration types (CompileConfig, CompileOptions)
//   - compile_helpers.go: Utility functions (path resolution, file detection)
//   - compile_validation.go: Input validation logic (file existence, permissions)
//   - compile_watch.go: Watch mode functionality (file watching, auto-recompilation)
//   - compile_campaign.go: Campaign validation (multi-workflow compilation)
//   - compile_orchestrator.go: Main orchestration (CompileWorkflows function - entry point)
//
// # Usage
//
// The main entry point is CompileWorkflows() in compile_orchestrator.go:
//
//   err := cli.CompileWorkflows(cli.CompileConfig{
//       WorkflowFiles: []string{"workflow.md"},
//       Verbose:       false,
//       Force:         false,
//   })
//
// # Compilation Flow
//
// The compilation process:
//  1. Validate input files and configuration (compile_validation.go)
//  2. Detect workflow files or campaigns (compile_helpers.go)
//  3. For each workflow:
//     a. Parse markdown and frontmatter (via pkg/parser)
//     b. Compile to GitHub Actions YAML (via pkg/workflow)
//     c. Write .lock.yml output file
//  4. Handle watch mode if enabled (compile_watch.go)
//  5. Report compilation statistics and errors
//
// # Related Commands
//
// See NewCompileCommand() in commands.go for the Cobra command definition
// and flag configuration.
