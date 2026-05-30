// Package logger provides namespace-based debug logging with zero overhead
// when disabled.
//
// The logger package implements a lightweight debug logging system inspired by
// the debug npm package. It uses the DEBUG environment variable for selective
// log enabling with pattern matching and namespace coloring. When logging is
// disabled for a namespace, log calls have zero overhead (not even string
// formatting).
//
// # Basic Usage
//
//	var log = logger.New("cli:compile")
//
//	func CompileWorkflow(path string) error {
//		log.Printf("Compiling workflow: %s", path)
//		// Only executed if DEBUG matches "cli:compile" or "cli:*"
//
//		if log.Enabled() {
//			// Expensive operation only when logging is enabled
//			log.Printf("Details: %+v", expensiveDebugInfo())
//		}
//		return nil
//	}
//
// # Environment Variables
//
// DEBUG - Controls which namespaces are enabled:
//
//	DEBUG=*              # Enable all namespaces
//	DEBUG=cli:*          # Enable all cli namespaces
//	DEBUG=workflow:*     # Enable all workflow namespaces
//	DEBUG=cli:*,parser:* # Enable multiple patterns
//	DEBUG=*,-test:*      # Enable all except test namespaces
//
// DEBUG_COLORS - Controls color output (default: enabled in terminals):
//
//	DEBUG_COLORS=0       # Disable colors (auto-disabled when piping)
//
// # Namespace Convention
//
// Follow the pattern: pkg:filename or pkg:component
//
//	logger.New("cli:compile_command")   # Command-specific
//	logger.New("workflow:compiler")     # Core component
//	logger.New("parser:frontmatter")    # Subcomponent
//	logger.New("mcp:gateway")           # Feature-specific
//
// Use consistent naming across the codebase for easy filtering.
//
// # Features
//
// Zero Overhead: Log calls are no-ops when the namespace is disabled.
// No string formatting or function calls occur.
//
// Time Deltas: Each log shows time elapsed since the previous log in that
// namespace (e.g., +50ms, +2.5s, +1m30s).
//
// Auto-Colors: Each namespace gets a consistent color in terminals. Colors
// are generated deterministically from the namespace string.
//
// Pattern Matching: Supports wildcards (*) and exclusions (-pattern) for
// flexible namespace filtering.
//
// # Performance
//
//	// No overhead - neither Printf nor Enabled() called when disabled
//	log.Printf("Debug info: %s", expensiveFunction())
//
//	// Check first for expensive operations
//	if log.Enabled() {
//		result := expensiveFunction()
//		log.Printf("Result: %+v", result)
//	}
//
// # Use Cases
//
// Development Debugging: Enable specific namespaces during development
// to trace execution flow without adding/removing print statements.
//
// Performance Analysis: Time deltas help identify slow operations and
// bottlenecks in the compilation pipeline.
//
// Production Diagnostics: Users can enable logging to diagnose issues
// by setting DEBUG environment variable before running commands.
//
// Integration Testing: Tests can enable logging selectively to verify
// behavior without affecting test output.
//
// # Output Format
//
//	workflow:compiler Parsing workflow file +0ms
//	workflow:compiler Generating YAML +125ms
//	cli:compile Compilation complete +2.5s
//
// Each line shows: namespace (colored), message, time delta
//
// # Best Practices
//
// Use logger.New at package level with consistent namespace naming.
//
// Log significant operations, not every line - focus on key decision points.
//
// Check log.Enabled() before expensive debug operations like JSON marshaling.
//
// Use descriptive messages that make sense without source code context.
//
// Prefer structured data in messages for easy parsing if needed later.
//
// # Related Packages
//
// pkg/console - User-facing output formatting (errors, success, warnings)
//
// pkg/timeutil - Time formatting utilities used for delta calculations
//
// pkg/tty - Terminal detection for color support
package logger
