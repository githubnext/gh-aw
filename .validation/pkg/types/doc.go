// Package types provides shared type definitions used across gh-aw packages.
//
// This package contains common data structures and interfaces that are used by
// multiple packages to avoid circular dependencies and maintain clean separation
// of concerns. The types here are focused on configuration structures that need
// to be shared between parsing and workflow compilation.
//
// # Key Types
//
// BaseMCPServerConfig: The foundational configuration structure for MCP
// (Model Context Protocol) servers. This type is embedded by both parser
// and workflow packages to maintain consistency while allowing each package
// to add domain-specific fields.
//
// MCP servers are AI tool providers that can run as:
//   - stdio processes (command + args)
//   - HTTP endpoints (url + headers)
//   - Container services (container image + mounts)
//
// # Basic Usage
//
//	config := types.BaseMCPServerConfig{
//		Type:    "stdio",
//		Command: "npx",
//		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
//		Env: map[string]string{
//			"ALLOWED_PATHS": "/workspace",
//		},
//	}
//
// # Architecture
//
// This package serves as a bridge between the parser package (which reads
// workflow markdown files) and the workflow package (which generates GitHub
// Actions YAML). By defining shared types here, we avoid circular imports
// and ensure consistent configuration structures.
//
// The types are designed to be:
//   - Serializable to JSON and YAML
//   - Embeddable by other packages
//   - Extensible with package-specific fields
//   - Well-documented with struct tags
//
// # Related Packages
//
// pkg/parser - Embeds BaseMCPServerConfig in parser.MCPServerConfig
//
// pkg/workflow - Embeds BaseMCPServerConfig in workflow.MCPServerConfig
package types
