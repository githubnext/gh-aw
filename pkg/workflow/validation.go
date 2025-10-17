package workflow

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

// isTimeoutError checks if the error output indicates a timeout
func isTimeoutError(output string) bool {
	timeoutIndicators := []string{
		"TimeoutError",
		"Read timed out",
		"ReadTimeoutError",
		"timed out",
		"timeout",
	}

	outputLower := strings.ToLower(output)
	for _, indicator := range timeoutIndicators {
		if strings.Contains(outputLower, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
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
			actualSize := pretty.FormatFileSize(int64(len(line)))
			maxSizeFormatted := pretty.FormatFileSize(int64(maxSize))

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

	return nil
}

// validateContainerImages validates that container images specified in MCP configs exist and are accessible
func (c *Compiler) validateContainerImages(workflowData *WorkflowData) error {
	if workflowData.Tools == nil {
		return nil
	}

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

// validateNpxPackages validates that npx packages are available on npm registry
func (c *Compiler) validateNpxPackages(workflowData *WorkflowData) error {
	packages := extractNpxPackages(workflowData)
	if len(packages) == 0 {
		return nil
	}

	// Check if npm is available
	_, err := exec.LookPath("npm")
	if err != nil {
		return fmt.Errorf("npm command not found - cannot validate npx packages. Install Node.js/npm or disable validation")
	}

	var errors []string
	for _, pkg := range packages {
		// Use npm view to check if package exists
		cmd := exec.Command("npm", "view", pkg, "name")
		output, err := cmd.CombinedOutput()

		if err != nil {
			errors = append(errors, fmt.Sprintf("npx package '%s' not found on npm registry: %s", pkg, strings.TrimSpace(string(output))))
		} else if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ npm package validated: %s", pkg)))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("npx package validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validatePythonPackagesWithPip is a generic helper that validates Python packages using pip index.
// It accepts a package list, package type name for error messaging, and pip command to use.
func (c *Compiler) validatePythonPackagesWithPip(packages []string, packageType string, pipCmd string) {
	for _, pkg := range packages {
		// Extract package name without version specifier
		pkgName := pkg
		if eqIndex := strings.Index(pkg, "=="); eqIndex > 0 {
			pkgName = pkg[:eqIndex]
		}

		// Use pip index to check if package exists on PyPI
		cmd := exec.Command(pipCmd, "index", "versions", pkgName)
		output, err := cmd.CombinedOutput()

		if err != nil {
			outputStr := strings.TrimSpace(string(output))
			// Treat all pip validation errors as warnings, not compilation failures
			// The package may be experimental, not yet published, or will be installed at runtime
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s package '%s' validation failed - skipping verification. Package may or may not exist on PyPI.", packageType, pkg)))
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  Details: %s", outputStr)))
			}
		} else if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ %s package validated: %s", packageType, pkg)))
		}
	}
}

// validatePipPackages validates that pip packages are available on PyPI
func (c *Compiler) validatePipPackages(workflowData *WorkflowData) error {
	packages := extractPipPackages(workflowData)
	if len(packages) == 0 {
		return nil
	}

	// Check if pip is available
	pipCmd := "pip"
	_, err := exec.LookPath("pip")
	if err != nil {
		// Try pip3 as fallback
		_, err3 := exec.LookPath("pip3")
		if err3 != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("pip command not found - skipping pip package validation. Install Python/pip for full validation"))
			return nil
		}
		pipCmd = "pip3"
	}

	c.validatePythonPackagesWithPip(packages, "pip", pipCmd)
	return nil
}

// validateUvPackages validates that uv packages are available
func (c *Compiler) validateUvPackages(workflowData *WorkflowData) error {
	packages := extractUvPackages(workflowData)
	if len(packages) == 0 {
		return nil
	}

	// Check if uv is available
	_, err := exec.LookPath("uv")
	if err != nil {
		// uv not available, but we can still validate using pip index
		pipCmd := "pip"
		_, pipErr := exec.LookPath("pip")
		if pipErr != nil {
			// Try pip3 as fallback
			_, pip3Err := exec.LookPath("pip3")
			if pip3Err != nil {
				return fmt.Errorf("uv and pip commands not found - cannot validate uv packages. Install uv/pip or disable validation")
			}
			pipCmd = "pip3"
		}

		return c.validateUvPackagesWithPip(packages, pipCmd)
	}

	// Validate with uv
	var errors []string
	for _, pkg := range packages {
		// Extract package name without version specifier
		pkgName := pkg
		if eqIndex := strings.Index(pkg, "=="); eqIndex > 0 {
			pkgName = pkg[:eqIndex]
		}

		// Use uv pip show to check if package exists on PyPI
		cmd := exec.Command("uv", "pip", "show", pkgName, "--no-cache")
		_, err := cmd.CombinedOutput()

		if err != nil {
			// Package not installed, try to check if it's available
			errors = append(errors, fmt.Sprintf("uv package '%s' validation requires network access or local cache", pkg))
		} else if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ uv package validated: %s", pkg)))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("uv package validation requires network access:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateUvPackagesWithPip validates uv packages using pip index
func (c *Compiler) validateUvPackagesWithPip(packages []string, pipCmd string) error {
	c.validatePythonPackagesWithPip(packages, "uv", pipCmd)
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
			if config, ok := toolConfig.(map[string]any); ok {
				if command, hasCommand := config["command"]; hasCommand {
					if cmdStr, ok := command.(string); ok && cmdStr == toolCommand {
						// Extract package from args
						if args, hasArgs := config["args"]; hasArgs {
							if argsSlice, ok := args.([]any); ok && len(argsSlice) > 0 {
								if pkgStr, ok := argsSlice[0].(string); ok && !seen[pkgStr] {
									packages = append(packages, pkgStr)
									seen[pkgStr] = true
								}
							}
						}
					}
				}
			}
		}
	}

	return packages
}

// extractNpxPackages extracts npx package names from workflow data
func extractNpxPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractNpxFromCommands, "npx")
}

// extractNpxFromCommands extracts npx package names from command strings
func extractNpxFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "npx <package>" pattern
		words := strings.Fields(line)
		for i, word := range words {
			if word == "npx" && i+1 < len(words) {
				pkg := words[i+1]
				// Remove any shell operators
				pkg = strings.TrimRight(pkg, "&|;")
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}

// extractPipPackages extracts pip package names from workflow data
func extractPipPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractPipFromCommands, "")
}

// extractPipFromCommands extracts pip package names from command strings
func extractPipFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "pip install <package>" or "pip3 install <package>" patterns
		words := strings.Fields(line)
		for i, word := range words {
			if (word == "pip" || word == "pip3") && i+1 < len(words) {
				// Look for install command
				for j := i + 1; j < len(words); j++ {
					if words[j] == "install" {
						// Skip flags and find the first package name
						for k := j + 1; k < len(words); k++ {
							pkg := words[k]
							pkg = strings.TrimRight(pkg, "&|;")
							// Skip flags (start with - or --)
							if !strings.HasPrefix(pkg, "-") {
								packages = append(packages, pkg)
								break
							}
						}
						break
					}
				}
			}
		}
	}

	return packages
}

// extractUvPackages extracts uv package names from workflow data
func extractUvPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractUvFromCommands, "")
}

// extractUvFromCommands extracts uv package names from command strings
func extractUvFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "uv pip install <package>" or "uvx <package>" patterns
		words := strings.Fields(line)
		for i, word := range words {
			if word == "uvx" && i+1 < len(words) {
				pkg := words[i+1]
				pkg = strings.TrimRight(pkg, "&|;")
				packages = append(packages, pkg)
			} else if word == "uv" && i+2 < len(words) && words[i+1] == "pip" {
				// Look for install command
				for j := i + 2; j < len(words); j++ {
					if words[j] == "install" {
						// Skip flags and find the first package name
						for k := j + 1; k < len(words); k++ {
							pkg := words[k]
							pkg = strings.TrimRight(pkg, "&|;")
							// Skip flags (start with - or --)
							if !strings.HasPrefix(pkg, "-") {
								packages = append(packages, pkg)
								break
							}
						}
						break
					}
				}
			}
		}
	}

	return packages
}
