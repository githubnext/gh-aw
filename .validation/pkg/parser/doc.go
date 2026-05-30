// Package parser provides markdown frontmatter parsing and content extraction
// for agentic workflow files.
//
// The parser package handles the extraction and validation of YAML frontmatter
// from markdown files, which defines workflow configuration (triggers, permissions,
// tools, safe outputs). It also extracts the markdown body content which serves
// as the AI agent's prompt text.
//
// # Key Functionality
//
// Frontmatter Parsing: Extracts YAML configuration blocks from markdown files
// using delimiters (---). Supports nested structures, includes, and imports.
//
// Content Extraction: Separates frontmatter from markdown body content while
// preserving formatting and structure.
//
// Import Processing: Resolves @import directives to include external workflow
// fragments, templates, and shared configurations.
//
// GitHub URL Resolution: Fetches workflow content from GitHub repositories
// using various URL formats (raw.githubusercontent.com, github.com URLs).
//
// ANSI Strip: Removes terminal color codes from content to prevent YAML
// parsing issues.
//
// # Basic Usage
//
//	// Parse frontmatter from markdown content
//	frontmatter, content, err := parser.ParseFrontmatter([]byte(markdown))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Process imports in workflow file
//	processor := parser.NewImportProcessor(parser.ImportProcessorConfig{
//		BasePath: "/path/to/workflows",
//	})
//	resolvedFrontmatter, err := processor.Process(frontmatter)
//
// # Architecture
//
// The parsing flow:
//  1. Read markdown file content
//  2. Extract YAML frontmatter block between --- delimiters
//  3. Parse YAML into structured configuration
//  4. Resolve @import directives recursively
//  5. Merge imported configurations
//  6. Extract remaining markdown as prompt content
//  7. Validate configuration structure
//
// # Import System
//
// The parser supports flexible import directives:
//   - Local files: @import path/to/template.md
//   - GitHub URLs: @import github.com/owner/repo/workflow.md
//   - Fragments: @import path/to/fragment#section
//   - Includes: Include specific frontmatter sections from other files
//
// Imports are cached to avoid redundant fetches and processed recursively
// to support multi-level includes.
//
// # Configuration Structure
//
// Parsed frontmatter maps to structured types in pkg/workflow:
//   - Trigger configuration (on: schedule, pull_request, etc.)
//   - Permissions (contents: read, issues: write, etc.)
//   - Tools (MCP servers, safe outputs, GitHub toolsets)
//   - Engine settings (copilot, claude, codex, custom)
//   - Network restrictions (allowed/blocked domains)
//   - Runtime overrides (node, python, go versions)
//
// # Related Packages
//
// pkg/workflow - Consumes parsed frontmatter for workflow compilation
//
// pkg/types - Shared type definitions (BaseMCPServerConfig)
package parser
