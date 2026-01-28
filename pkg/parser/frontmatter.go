package parser

// Frontmatter Module Organization
//
// This file serves as the entry point for the frontmatter parsing subsystem.
// Frontmatter parsing extracts YAML configuration blocks from markdown workflow files
// and converts them into structured data for compilation.
//
// # Architecture
//
// The frontmatter parsing functionality is organized across multiple specialized files:
//
//   - frontmatter_content.go: Content extraction (separates YAML from markdown body)
//   - frontmatter_extraction_yaml.go: YAML parsing and conversion
//   - frontmatter_extraction_security.go: Security validation and sanitization
//   - frontmatter_extraction_metadata.go: Metadata extraction and processing
//
// # Parsing Flow
//
// The typical parsing flow:
//  1. Read markdown file content
//  2. Identify frontmatter block between --- delimiters
//  3. Parse YAML configuration (tools, triggers, permissions, etc.)
//  4. Validate security constraints (no template injection, safe domains)
//  5. Process @import directives recursively
//  6. Extract markdown body as AI prompt
//  7. Return structured configuration and content
//
// # Key Functions
//
// The main parsing entry points are:
//   - ParseFrontmatter(): Extracts YAML and content from markdown
//   - ProcessImports(): Resolves @import directives
//   - ValidateFrontmatter(): Applies security and schema validation
//
// See doc.go for the complete package documentation and usage examples.

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var log = logger.New("parser:frontmatter")
