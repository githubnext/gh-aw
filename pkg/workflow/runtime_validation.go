// Package workflow provides runtime validation for packages, containers, and expressions.
//
// # Runtime Validation
//
// This file validates runtime dependencies and configuration for agentic workflows.
// It ensures that:
//   - Container images exist and are accessible
//   - Runtime packages (npm, pip, uv) are available
//   - Expression sizes don't exceed GitHub Actions limits
//   - Secret references are valid
//   - Cache IDs are unique
//
// # Validation Functions
//
//   - validateExpressionSizes() - Validates expression size limits (21KB max)
//   - validateContainerImages() - Validates Docker images exist
//   - validateRuntimePackages() - Validates npm, pip, uv packages
//   - collectPackagesFromWorkflow() - Generic package collection helper
//   - validateNoDuplicateCacheIDs() - Validates unique cache-memory IDs
//   - validateSecretReferences() - Validates secret name format
//
// # Validation Patterns
//
// This file uses several patterns:
//   - External resource validation: Docker images, npm/pip packages
//   - Size limit validation: Expression sizes, file sizes
//   - Format validation: Secret names, cache IDs
//   - Collection and deduplication: Package extraction
//
// # Size Limits
//
// GitHub Actions has a 21KB limit for expression values including environment variables.
// This validation prevents compilation of workflows that will fail at runtime.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates runtime dependencies (packages, containers)
//   - It checks expression or content size limits
//   - It validates configuration format (secrets, cache IDs)
//   - It requires external resource checking
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var runtimeValidationLog = logger.New("workflow:runtime_validation")

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
	runtimeValidationLog.Print("Validating expression sizes in generated YAML")
	lines := strings.Split(yamlContent, "\n")
	maxSize := MaxExpressionSize

	for lineNum, line := range lines {
		// Check the line length (actual content that will be in the YAML)
		if len(line) > maxSize {
			// Extract the key/value for better error message
			trimmed := strings.TrimSpace(line)
			key := ""
			if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
				key = strings.TrimSpace(trimmed[:colonIdx])
			}

			// Format sizes for display
			actualSize := console.FormatFileSize(int64(len(line)))
			maxSizeFormatted := console.FormatFileSize(int64(maxSize))

			var errorMsg string
			if key != "" {
				errorMsg = fmt.Sprintf("📝 The expression value for %q is too large (%s).\n\nWhy this matters: GitHub Actions has a 21KB limit for expression values including environment variables. This prevents workflows from failing at runtime.\n\nCurrent size: %s\nMaximum allowed: %s\nFound at line: %d\n\nTo fix, consider:\n  • Breaking the content into smaller chunks\n  • Using GitHub Actions artifacts for large data\n  • Storing data in files instead of environment variables",
					key, actualSize, actualSize, maxSizeFormatted, lineNum+1)
			} else {
				errorMsg = fmt.Sprintf("📝 Line %d is too large (%s).\n\nGitHub Actions has a 21KB limit for expression values.\n\nMaximum allowed: %s\n\nConsider breaking this into smaller pieces or using artifacts.",
					lineNum+1, actualSize, maxSizeFormatted)
			}

			return errors.New(errorMsg)
		}
	}

	return nil
}

// validateContainerImages validates that container images specified in MCP configs exist and are accessible
func (c *Compiler) validateContainerImages(workflowData *WorkflowData) error {
	if workflowData.Tools == nil {
		return nil
	}

	runtimeValidationLog.Printf("Validating container images for %d tools", len(workflowData.Tools))
	var errors []string
	for toolName, toolConfig := range workflowData.Tools {
		if config, ok := toolConfig.(map[string]any); ok {
			// Get the MCP configuration to extract container info
			mcpConfig, err := getMCPConfig(config, toolName)
			if err != nil {
				// If we can't parse the MCP config, skip validation (will be caught elsewhere)
				continue
			}

			// Check if this tool originally had a container field (before transformation)
			if containerName, hasContainer := config["container"]; hasContainer && mcpConfig.Type == "stdio" {
				// Build the full container image name with version
				containerStr, ok := containerName.(string)
				if !ok {
					continue
				}

				containerImage := containerStr
				if version, hasVersion := config["version"]; hasVersion {
					if versionStr, ok := version.(string); ok && versionStr != "" {
						containerImage = containerImage + ":" + versionStr
					}
				}

				// Validate the container image exists using docker
				if err := validateDockerImage(containerImage, c.verbose); err != nil {
					errors = append(errors, fmt.Sprintf("tool '%s': %v", toolName, err))
				} else if c.verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ Container image validated: %s", containerImage)))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("container image validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateRuntimePackages validates that packages required by npx, pip, and uv are available
func (c *Compiler) validateRuntimePackages(workflowData *WorkflowData) error {
	// Detect runtime requirements
	requirements := DetectRuntimeRequirements(workflowData)
	runtimeValidationLog.Printf("Validating runtime packages: found %d runtime requirements", len(requirements))

	var errors []string
	for _, req := range requirements {
		switch req.Runtime.ID {
		case "node":
			// Validate npx packages used in the workflow
			if err := c.validateNpxPackages(workflowData); err != nil {
				errors = append(errors, err.Error())
			}
		case "python":
			// Validate pip packages used in the workflow
			if err := c.validatePipPackages(workflowData); err != nil {
				errors = append(errors, err.Error())
			}
		case "uv":
			// Validate uv packages used in the workflow
			if err := c.validateUvPackages(workflowData); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("runtime package validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// collectPackagesFromWorkflow is a generic helper that collects packages from workflow data
// using the provided extractor function. It deduplicates packages and optionally extracts
// from MCP tool configurations when toolCommand is provided.
func collectPackagesFromWorkflow(
	workflowData *WorkflowData,
	extractor func(string) []string,
	toolCommand string,
) []string {
	var packages []string
	seen := make(map[string]bool)

	// Extract from custom steps
	if workflowData.CustomSteps != "" {
		pkgs := extractor(workflowData.CustomSteps)
		for _, pkg := range pkgs {
			if !seen[pkg] {
				packages = append(packages, pkg)
				seen[pkg] = true
			}
		}
	}

	// Extract from engine steps
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			if run, hasRun := step["run"]; hasRun {
				if runStr, ok := run.(string); ok {
					pkgs := extractor(runStr)
					for _, pkg := range pkgs {
						if !seen[pkg] {
							packages = append(packages, pkg)
							seen[pkg] = true
						}
					}
				}
			}
		}
	}

	// Extract from MCP server configurations (if toolCommand is provided)
	if toolCommand != "" && workflowData.Tools != nil {
		for _, toolConfig := range workflowData.Tools {
			// Handle structured MCP config with command and args fields
			if config, ok := toolConfig.(map[string]any); ok {
				if command, hasCommand := config["command"]; hasCommand {
					if cmdStr, ok := command.(string); ok && cmdStr == toolCommand {
						// Extract package from args, skipping flags
						if args, hasArgs := config["args"]; hasArgs {
							if argsSlice, ok := args.([]any); ok {
								for _, arg := range argsSlice {
									if pkgStr, ok := arg.(string); ok {
										// Skip flags (arguments starting with - or --)
										if !strings.HasPrefix(pkgStr, "-") && !seen[pkgStr] {
											packages = append(packages, pkgStr)
											seen[pkgStr] = true
											break // Only take the first non-flag argument
										}
									}
								}
							}
						}
					}
				}
			} else if cmdStr, ok := toolConfig.(string); ok {
				// Handle string-format MCP tool (e.g., "npx -y package")
				// Use the extractor function to parse the command string
				pkgs := extractor(cmdStr)
				for _, pkg := range pkgs {
					if !seen[pkg] {
						packages = append(packages, pkg)
						seen[pkg] = true
					}
				}
			}
		}
	}

	return packages
}

// validateNoDuplicateCacheIDs checks for duplicate cache IDs and returns an error if found
func validateNoDuplicateCacheIDs(caches []CacheMemoryEntry) error {
	seen := make(map[string]bool)
	for _, cache := range caches {
		if seen[cache.ID] {
			return fmt.Errorf("💡 Duplicate cache-memory ID '%s' found.\n\nWhy this matters: Each cache needs a unique ID so we can track it separately.\n\nTo fix: Give each cache a unique ID.\n\nExample:\n  tools:\n    cache-memory:\n      - id: user-preferences\n      - id: session-data", cache.ID)
		}
		seen[cache.ID] = true
	}
	return nil
}

// validateSecretReferences validates that secret references are valid
func validateSecretReferences(secrets []string) error {
	// Secret names must be valid environment variable names
	secretNamePattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

	for _, secret := range secrets {
		if !secretNamePattern.MatchString(secret) {
			return fmt.Errorf("🔒 The secret name '%s' doesn't follow the required format.\n\nWhy? GitHub secret names must be valid environment variable names for security and compatibility.\n\nRules:\n  • Start with an uppercase letter\n  • Contain only uppercase letters, numbers, and underscores\n\nExample: MY_SECRET_KEY\n\nLearn more: https://docs.github.com/en/actions/security-guides/encrypted-secrets", secret)
		}
	}

	return nil
}

// validateFirewallConfig validates firewall configuration including log-level
func (c *Compiler) validateFirewallConfig(workflowData *WorkflowData) error {
	if workflowData.NetworkPermissions == nil || workflowData.NetworkPermissions.Firewall == nil {
		return nil
	}

	config := workflowData.NetworkPermissions.Firewall
	if config.LogLevel != "" {
		if err := ValidateLogLevel(config.LogLevel); err != nil {
			return err
		}
	}

	return nil
}
