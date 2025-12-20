package parser

import "github.com/githubnext/gh-aw/pkg/logger"

// Package parser provides functionality for parsing and processing GitHub Agentic Workflow frontmatter.
//
// This package is organized into specialized modules:
//   - import_directive.go: Parse import/include directive syntax
//   - content_extractor.go: Extract specific fields from frontmatter
//   - tools_merger.go: Merge tool configurations from multiple sources
//   - include_processor.go: Process @include/@import directives
//   - include_expander.go: Recursively expand includes
//   - import_processor.go: Process imports field with BFS traversal
//
// The core functionality includes:
//   - Parsing YAML frontmatter from markdown files
//   - Processing import and include directives
//   - Extracting and merging configuration from multiple files
//   - Validating workflow schemas
//   - Supporting both legacy (@include/@import) and new ({{#import}}) syntax

var log = logger.New("parser:frontmatter")
