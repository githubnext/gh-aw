// Package workflow implements the agentic workflow compiler that transforms
// markdown-based workflow definitions into GitHub Actions YAML files.
//
// # Type Safety and Dynamic Configuration
//
// This package makes extensive use of map[string]any for parsing YAML/JSON
// frontmatter and GitHub Actions configurations. This is intentional and
// follows Go best practices for handling dynamic data structures:
//
//  1. YAML unmarshaling produces map[string]any for structures unknown at compile time
//  2. Code validates types at runtime using type assertions with error handling
//  3. Validated data is converted to strongly-typed domain objects where possible
//  4. Type safety is maintained through typed constants and domain models
//
// # Common Patterns
//
// Dynamic Frontmatter Parsing:
//
//	var frontmatter map[string]any
//	yaml.Unmarshal(data, &frontmatter)
//	// Then extract and validate specific fields
//
// Runtime Type Checking:
//
//	if stepsList, ok := steps.([]any); ok {
//	    // Process as list
//	} else if stepMap, ok := steps.(map[string]any); ok {
//	    // Process as map
//	}
//
// Extensible Configuration:
//
//	type Config struct {
//	    KnownField string
//	    CustomFields map[string]any `yaml:",inline"`
//	}
//
// Do not attempt to eliminate all uses of 'any' - it is required for
// dynamic configuration parsing and provides necessary flexibility for
// extensible configurations like MCP server settings.
//
// # Type Hierarchies
//
// The package uses base types with embedded fields to share common
// configuration while allowing domain-specific extensions:
//
//	types.BaseMCPServerConfig (shared fields)
//	  ├─ parser.MCPServerConfig (parser-specific)
//	  └─ workflow.MCPServerConfig (workflow-specific)
//
// This pattern prevents duplication while maintaining domain boundaries.
//
// # When to Use map[string]any
//
// ✅ Use map[string]any when:
//   - Parsing YAML/JSON with unknown structure
//   - Handling GitHub Actions configurations (dynamic fields)
//   - Processing extensible feature flags
//   - Working with custom MCP server configurations
//   - Intermediate representation during compilation
//
// ❌ Avoid map[string]any when:
//   - The structure is known and can be typed
//   - Internal APIs with stable schemas
//   - Return values where type is deterministic
//
// # Runtime Type Assertions
//
// When working with dynamic types, always use safe type assertions:
//
//	// ✅ GOOD - Safe type assertion
//	if value, ok := data["key"].(string); ok {
//	    // Process value
//	} else {
//	    return fmt.Errorf("expected string, got %T", data["key"])
//	}
//
//	// ❌ BAD - Unsafe assertion (can panic)
//	value := data["key"].(string)
//
// # Documentation References
//
// For comprehensive guidance on type patterns:
//   - See specs/go-type-patterns.md for detailed type pattern documentation
//   - See AGENTS.md for dynamic type usage guidelines
//   - See frontmatter_types.go for typed configuration structs
package workflow
