// This file provides runtime validation for packages, containers, and expressions.
//
// # Runtime Validation
//
// This file validates runtime dependencies and configuration for agentic workflows.
// It ensures that:
//   - Container images exist and are accessible
//   - Runtime packages (npm, pip, uv) are available
//   - Expression sizes don't exceed GitHub Actions limits
//
// # Validation Functions
//
//   - validateExpressionSizes() - Validates expression size limits (21KB max)
//   - validateContainerImages() - Validates Docker images exist
//   - validateRuntimePackages() - Validates npm, pip, uv packages
//   - collectPackagesFromWorkflow() - Generic package collection helper
//
// # Validation Patterns
//
// This file uses several patterns:
//   - External resource validation: Docker images, npm/pip packages
//   - Size limit validation: Expression sizes, file sizes
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
//   - It requires external resource checking
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var runtimeValidationLog = logger.New("workflow:runtime_validation")

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits.
//
// GitHub Actions enforces a 21,000-character limit on YAML string values that contain
// template expressions (${{ }}). This check covers two cases:
//
//  1. Single-line values: each YAML line is checked individually.
//
//  2. Multi-line block scalars: YAML literal-block (|) and folded-block (>) scalars span
//     many lines but are parsed by GitHub Actions as a single string value.  When such a
//     block contains at least one ${{ }} expression AND its total length exceeds 21,000
//     characters, GitHub Actions rejects the workflow with "Exceeded max expression length".
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
	lines := strings.Split(yamlContent, "\n")
	runtimeValidationLog.Printf("Validating expression sizes: yaml_lines=%d, max_size=%d", len(lines), MaxExpressionSize)
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
				errorMsg = fmt.Sprintf("expression value for %q (%s) exceeds maximum allowed size (%s) at line %d. GitHub Actions has a 21KB limit for expression values including environment variables. Consider chunking the content or using artifacts instead.",
					key, actualSize, maxSizeFormatted, lineNum+1)
			} else {
				errorMsg = fmt.Sprintf("line %d (%s) exceeds maximum allowed expression size (%s). GitHub Actions has a 21KB limit for expression values.",
					lineNum+1, actualSize, maxSizeFormatted)
			}

			return errors.New(errorMsg)
		}
	}

	// Check multi-line YAML block scalars that contain template expressions.
	// A run: | or any other block-scalar value is a single string from GitHub Actions'
	// perspective; if it contains ${{ }} AND is longer than 21,000 characters the
	// runner rejects it with "Exceeded max expression length".
	if err := validateBlockScalarExpressionSizes(lines, maxSize); err != nil {
		return err
	}

	return nil
}

type blockScalarExpressionState struct {
	inBlock            bool
	blockKey           string
	blockStartLine     int
	blockIndent        int
	blockSize          int
	blockHasExpression bool
	maxSize            int
}

// validateBlockScalarExpressionSizes scans the YAML lines for multi-line block scalars
// and returns an error when any expression-containing block exceeds maxSize bytes.
func validateBlockScalarExpressionSizes(lines []string, maxSize int) error {
	state := blockScalarExpressionState{blockIndent: -1, maxSize: maxSize}
	for i, line := range lines {
		if err := state.processLine(i, line); err != nil {
			return err
		}
	}
	return state.checkBlock()
}

func (s *blockScalarExpressionState) processLine(i int, line string) error {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if s.inBlock {
		if s.lineContinuesBlock(line, indent) {
			s.addBlockLine(line)
			return nil
		}
		if err := s.checkBlock(); err != nil {
			return err
		}
		s.resetBlock()
	}
	if key, ok := detectBlockScalarStart(trimmed); ok {
		s.inBlock = true
		s.blockKey = key
		s.blockStartLine = i
		s.blockIndent = indent
		s.blockSize = 0
		s.blockHasExpression = false
	}
	return nil
}

func (s *blockScalarExpressionState) lineContinuesBlock(line string, indent int) bool {
	return strings.TrimSpace(line) == "" || indent > s.blockIndent
}

func (s *blockScalarExpressionState) addBlockLine(line string) {
	s.blockSize += len(line) + 1
	if strings.Contains(line, "${{") {
		s.blockHasExpression = true
	}
}

func (s *blockScalarExpressionState) checkBlock() error {
	if s.inBlock && s.blockHasExpression && s.blockSize > s.maxSize {
		actualSize := console.FormatFileSize(int64(s.blockSize))
		maxSizeFormatted := console.FormatFileSize(int64(s.maxSize))
		return fmt.Errorf("expression value for %q (%s) exceeds maximum allowed size (%s) starting at line %d. "+
			"GitHub Actions has a 21KB limit for YAML values that contain template expressions (${{ }}). "+
			"Split the step into separate run: blocks so that no single block containing ${{ }} expressions exceeds the limit",
			s.blockKey, actualSize, maxSizeFormatted, s.blockStartLine+1)
	}
	return nil
}

func (s *blockScalarExpressionState) resetBlock() {
	s.inBlock = false
	s.blockKey = ""
	s.blockSize = 0
	s.blockHasExpression = false
	s.blockIndent = -1
}

func detectBlockScalarStart(trimmed string) (string, bool) {
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx <= 0 {
		return "", false
	}
	afterColon := strings.TrimSpace(trimmed[colonIdx+1:])
	if afterColon == "|" || afterColon == ">" || strings.HasPrefix(afterColon, "|-") || strings.HasPrefix(afterColon, ">-") {
		return strings.TrimSpace(trimmed[:colonIdx]), true
	}
	return "", false
}

// validateContainerImages validates that container images specified in MCP configs exist and are accessible
func (c *Compiler) validateContainerImages(workflowData *WorkflowData) error {
	if workflowData.Tools == nil {
		runtimeValidationLog.Print("No tools configured, skipping container validation")
		return nil
	}
	runtimeValidationLog.Printf("Validating container images for %d tools", len(workflowData.Tools))
	daemonWasAvailable := isDockerDaemonRunning()
	validationErrors := c.collectContainerImageValidationErrors(workflowData)
	c.warnIfDockerDaemonStopped(daemonWasAvailable)
	if len(validationErrors) > 0 {
		return containerImageValidationError(validationErrors)
	}
	runtimeValidationLog.Print("Container image validation passed")
	return nil
}

func (c *Compiler) collectContainerImageValidationErrors(workflowData *WorkflowData) []string {
	var validationErrors []string
	for toolName, toolConfig := range workflowData.Tools {
		containerImage, ok := containerImageForValidation(toolName, toolConfig)
		if !ok {
			continue
		}
		if err := validateDockerImage(containerImage, c.verbose, c.requireDocker); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("tool '%s': %v", toolName, err))
		}
	}
	return validationErrors
}

func containerImageForValidation(toolName string, toolConfig any) (string, bool) {
	config, ok := toolConfig.(map[string]any)
	if !ok {
		return "", false
	}
	mcpConfig, err := getMCPConfig(config, toolName)
	if err != nil {
		return "", false
	}
	containerName, hasContainer := config["container"]
	if !hasContainer || mcpConfig.Type != "stdio" {
		return "", false
	}
	containerStr, ok := containerName.(string)
	if !ok {
		return "", false
	}
	containerImage := containerStr
	if version, hasVersion := config["version"]; hasVersion {
		if versionStr, ok := version.(string); ok && versionStr != "" {
			containerImage += ":" + versionStr
		}
	}
	return containerImage, true
}

func (c *Compiler) warnIfDockerDaemonStopped(daemonWasAvailable bool) {
	if daemonWasAvailable && !isDockerDaemonRunning() && !c.requireDocker {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Docker daemon is not running — skipping container image validation"))
		c.IncrementWarningCount()
	}
}

func containerImageValidationError(validationErrors []string) error {
	return NewValidationError(
		"container.images",
		fmt.Sprintf("%d images failed validation", len(validationErrors)),
		"container image validation failed",
		fmt.Sprintf("Fix the following container image issues:\n\n%s\n\nEnsure:\n1. Container images exist and are accessible\n2. Registry URLs are correct\n3. Image tags are specified\n4. You have pull permissions for private images", strings.Join(validationErrors, "\n")),
	)
}

// validateRuntimePackages validates that packages required by npx, pip, and uv are available
func (c *Compiler) validateRuntimePackages(workflowData *WorkflowData) error {
	// Detect runtime requirements
	requirements := detectRuntimeRequirementsCached(workflowData)
	runtimeValidationLog.Printf("Validating runtime packages: found %d runtime requirements", len(requirements))

	var errors []string
	if err := c.validateMCPScriptDependencies(workflowData); err != nil {
		errors = append(errors, err.Error())
	}

	for _, req := range requirements {
		switch req.Runtime.ID {
		case "node":
			// Validate npx packages used in the workflow
			runtimeValidationLog.Print("Validating npx packages")
			if err := c.validateNpxPackages(workflowData); err != nil {
				if isErrNpmNotAvailable(err) {
					// npm is not installed on this system — treat as a warning, not an error.
					// The workflow may still compile and run successfully in environments
					// that have npm (e.g., GitHub Actions).
					runtimeValidationLog.Print("npm not available, skipping npx package validation")
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("npm not found, skipping npx package validation"))
					c.IncrementWarningCount()
				} else {
					runtimeValidationLog.Printf("Npx package validation failed: %v", err)
					errors = append(errors, err.Error())
				}
			}
		case "python":
			// Validate pip packages used in the workflow
			runtimeValidationLog.Print("Validating pip packages")
			if err := c.validatePipPackages(workflowData); err != nil {
				runtimeValidationLog.Printf("Pip package validation failed: %v", err)
				errors = append(errors, err.Error())
			}
		case "uv":
			// Validate uv packages used in the workflow
			runtimeValidationLog.Print("Validating uv packages")
			if err := c.validateUvPackages(workflowData); err != nil {
				runtimeValidationLog.Printf("Uv package validation failed: %v", err)
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		runtimeValidationLog.Printf("Runtime package validation completed with %d errors", len(errors))
		return NewValidationError(
			"runtime.packages",
			fmt.Sprintf("%d package validation errors", len(errors)),
			"runtime package validation failed",
			fmt.Sprintf("Fix the following package issues:\n\n%s\n\nEnsure:\n1. Package names are spelled correctly\n2. Packages exist in their respective registries (npm, PyPI)\n3. Package managers (npm, pip, uv) are installed\n4. Network access is available for registry checks", strings.Join(errors, "\n")),
		)
	}

	runtimeValidationLog.Print("Runtime package validation passed")
	return nil
}
