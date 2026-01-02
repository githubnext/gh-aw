// Package workflow provides validation functions for agentic workflow compilation.
//
// # Validation Architecture
//
// The validation system for agentic workflows is organized into focused,
// domain-specific files to maintain clarity and single responsibility:
//
//   - validation.go: This file - package documentation only
//   - validation_context.go: Unified validation context for error aggregation
//   - strict_mode_validation.go: Security and strict mode validation
//   - repository_features_validation.go: Repository capability detection
//   - schema_validation.go: GitHub Actions schema validation
//   - runtime_validation.go: Runtime packages, containers, expressions
//   - agent_validation.go: Agent files and feature support
//   - pip_validation.go: Python package validation
//   - npm_validation.go: NPM package validation
//   - docker_validation.go: Docker image validation
//   - expression_safety.go: GitHub Actions expression security
//   - engine_validation.go: AI engine configuration validation
//   - mcp_config_validation.go: MCP server configuration validation
//   - template_validation.go: Template structure validation
//   - firewall_validation.go: Firewall log-level validation
//   - gateway_validation.go: Gateway port validation
//   - sandbox_validation.go: Sandbox and mounts validation
//   - bundler_safety_validation.go: JavaScript bundle safety (require/module checks)
//   - bundler_script_validation.go: JavaScript script content (execSync, GitHub globals)
//   - bundler_runtime_validation.go: JavaScript runtime mode compatibility
//
// # Validation Patterns
//
// ## Legacy Pattern (Fail-Fast)
//
// The original validation pattern returns errors immediately, stopping compilation
// on the first issue. This requires developers to fix one error at a time:
//
//	func validateSomething(data *WorkflowData) error {
//	    if /* invalid */ {
//	        return fmt.Errorf("validation failed")
//	    }
//	    return nil
//	}
//
// ## New Pattern (Error Aggregation)
//
// The new validation pattern uses ValidationContext to collect all errors before
// failing, improving developer iteration speed:
//
//	func validateSomethingWithContext(ctx *ValidationContext, data *WorkflowData) {
//	    if /* invalid */ {
//	        ctx.AddError("validator_name", fmt.Errorf("validation failed"))
//	    }
//	}
//
// ## Migration Strategy
//
// Both patterns coexist during migration:
//   - Legacy validators continue to work unchanged
//   - New validators add *WithContext variants
//   - Compiler orchestration gradually adopts ValidationContext
//   - No breaking changes to existing code
//
// ## Using ValidationContext
//
// Create a context and collect errors across multiple validators:
//
//	ctx := NewValidationContext(markdownPath, workflowData)
//	ctx.SetPhase(PhasePreCompile)
//
//	// Run validators - they add errors to context
//	validateFeaturesWithContext(ctx, workflowData)
//	validateSandboxConfigWithContext(ctx, workflowData)
//	c.validateStrictModeWithContext(ctx, frontmatter, networkPermissions)
//
//	// Check and report all errors together
//	if ctx.HasErrors() {
//	    return errors.New(ctx.Error())  // Formatted multi-error report
//	}
//
// # When to Add New Validation
//
// Add validation to existing domain files when:
//   - It fits the domain (e.g., package validation → pip_validation.go)
//   - It extends existing functionality
//
// Create a new validation file when:
//   - It represents a distinct validation domain
//   - It has multiple related validation functions
//   - It requires its own caching or state management
//
// # Validation Patterns
//
// The validation system uses several patterns:
//   - Schema validation: JSON schema validation with caching
//   - External resource validation: Docker images, npm/pip packages
//   - Size limit validation: Expression sizes, file sizes
//   - Feature detection: Repository capabilities
//   - Security validation: Permission restrictions, expression safety
//
// For detailed documentation on validation architecture, see:
// .github/instructions/developer.instructions.md#validation-architecture
package workflow
