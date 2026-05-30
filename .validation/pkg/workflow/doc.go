// Package workflow provides compilation and generation of GitHub Actions workflows
// from markdown-based agentic workflow specifications.
//
// The workflow package is the core of gh-aw, handling the transformation of
// user-friendly markdown files into production-ready GitHub Actions YAML workflows.
// It supports advanced features including:
//   - Agentic AI workflow execution with Claude, Codex, and Copilot engines
//   - Safe output handling with configurable permissions
//   - MCP (Model Context Protocol) server integration
//   - Network sandboxing and firewall configuration
//   - GitHub Actions security best practices (SHA pinning, input sanitization)
//
// # Basic Usage
//
//	compiler := workflow.NewCompiler(workflow.CompilerOptions{
//		Verbose: true,
//	})
//	err := compiler.CompileWorkflow("path/to/workflow.md")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Architecture
//
// The compilation process consists of:
//  1. Frontmatter parsing - Extracts YAML metadata (triggers, permissions, tools)
//  2. Markdown processing - Extracts the AI prompt text from markdown body
//  3. Tool configuration - Configures MCP servers, safe outputs, and GitHub tools
//  4. Job generation - Creates main agent job, activation jobs, and safe output jobs
//  5. YAML generation - Produces final GitHub Actions workflow with security features
//
// # Key Components
//
// Compiler: The main orchestrator that coordinates all compilation phases.
// Handles markdown parsing, frontmatter extraction, and YAML generation.
//
// Engine Configuration: Supports multiple AI engines (copilot, claude, codex, custom).
// Each engine has specific tool integrations and execution patterns.
//
// MCP Servers: Model Context Protocol servers provide tools to AI agents.
// Configured via frontmatter and can run as stdio processes, HTTP endpoints,
// or containerized services.
//
// Safe Outputs: Write operations (create issue, add comment, etc.) are
// sanitized and executed in separate jobs with minimal permissions to
// prevent prompt injection attacks.
//
// Network Sandboxing: Firewall configuration restricts network access to
// approved domains, preventing data exfiltration and unauthorized API calls.
//
// # Security Features
//
// The workflow package implements multiple security layers:
//   - SHA-pinned GitHub Actions (no tag-based references)
//   - Input sanitization for all user-provided data
//   - Read-only execution by default with explicit safe-output gates
//   - Network firewalls with domain allowlists
//   - Tool allowlisting (MCP servers, safe outputs)
//   - Template injection prevention
//   - Secrets redaction and masking
//
// # Related Packages
//
// pkg/parser - Markdown and YAML frontmatter parsing
//
// pkg/cli - Command-line interface for compilation and workflow management
//
// pkg/types - Shared type definitions for MCP and workflow configuration
package workflow
