package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var validationLog = logger.New("workflow:validation")

// validateExpressionSizes validates that no expression values in the generated YAML exceed GitHub Actions limits
func (c *Compiler) validateExpressionSizes(yamlContent string) error {
	validationLog.Print("Validating expression sizes in generated YAML")
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

	validationLog.Printf("Validating container images for %d tools", len(workflowData.Tools))
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
	validationLog.Printf("Validating runtime packages: found %d runtime requirements", len(requirements))

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

// validateGitHubActionsSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateGitHubActionsSchema(yamlContent string) error {
	// Convert YAML to any for JSON conversion
	var workflowData any
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse YAML for schema validation: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON for validation: %w", err)
	}

	// Parse the embedded schema
	var schemaDoc any
	if err := json.Unmarshal([]byte(githubWorkflowSchema), &schemaDoc); err != nil {
		return fmt.Errorf("failed to parse embedded GitHub Actions schema: %w", err)
	}

	// Create compiler and add the schema as a resource
	loader := jsonschema.NewCompiler()
	schemaURL := "https://json.schemastore.org/github-workflow.json"
	if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := loader.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile GitHub Actions schema: %w", err)
	}

	// Validate the JSON data against the schema
	var jsonObj any
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("GitHub Actions schema validation failed: %w", err)
	}

	return nil
}

// validateNoDuplicateCacheIDs checks for duplicate cache IDs and returns an error if found
func validateNoDuplicateCacheIDs(caches []CacheMemoryEntry) error {
	seen := make(map[string]bool)
	for _, cache := range caches {
		if seen[cache.ID] {
			return fmt.Errorf("duplicate cache-memory ID '%s' found. Each cache must have a unique ID", cache.ID)
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
			return fmt.Errorf("invalid secret name: %s", secret)
		}
	}

	return nil
}

// validateRepositoryFeatures validates that required repository features are enabled
// when safe-outputs are configured that depend on them (discussions, issues)
func (c *Compiler) validateRepositoryFeatures(workflowData *WorkflowData) error {
	if workflowData.SafeOutputs == nil {
		return nil
	}

	validationLog.Print("Validating repository features for safe-outputs")

	// Get the repository from the current git context
	// This will work when running in a git repository
	repo, err := getCurrentRepository()
	if err != nil {
		validationLog.Printf("Could not determine repository: %v", err)
		// Don't fail if we can't determine the repository (e.g., not in a git repo)
		// This allows validation to pass in non-git environments
		return nil
	}

	validationLog.Printf("Checking repository features for: %s", repo)

	// Check if discussions are enabled when create-discussion is configured
	if workflowData.SafeOutputs.CreateDiscussions != nil {
		hasDiscussions, err := checkRepositoryHasDiscussions(repo)
		if err != nil {
			// If we can't check, log but don't fail
			// This could happen due to network issues or auth problems
			validationLog.Printf("Warning: Could not check if discussions are enabled: %v", err)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("Could not verify if discussions are enabled: %v", err)))
			}
			return nil
		}

		if !hasDiscussions {
			return fmt.Errorf("workflow uses safe-outputs.create-discussion but repository %s does not have discussions enabled. Enable discussions in repository settings or remove create-discussion from safe-outputs", repo)
		}

		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
				fmt.Sprintf("✓ Repository %s has discussions enabled", repo)))
		}
	}

	// Check if issues are enabled when create-issue is configured
	if workflowData.SafeOutputs.CreateIssues != nil {
		hasIssues, err := checkRepositoryHasIssues(repo)
		if err != nil {
			// If we can't check, log but don't fail
			validationLog.Printf("Warning: Could not check if issues are enabled: %v", err)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("Could not verify if issues are enabled: %v", err)))
			}
			return nil
		}

		if !hasIssues {
			return fmt.Errorf("workflow uses safe-outputs.create-issue but repository %s does not have issues enabled. Enable issues in repository settings or remove create-issue from safe-outputs", repo)
		}

		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
				fmt.Sprintf("✓ Repository %s has issues enabled", repo)))
		}
	}

	return nil
}

// getCurrentRepository gets the current repository from git context
func getCurrentRepository() (string, error) {
	// Use gh CLI to get the current repository
	// This works when in a git repository with GitHub remote
	stdOut, _, err := gh.Exec("repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to get current repository: %w", err)
	}

	repo := strings.TrimSpace(stdOut.String())
	if repo == "" {
		return "", fmt.Errorf("repository name is empty")
	}

	return repo, nil
}

// checkRepositoryHasDiscussions checks if a repository has discussions enabled
func checkRepositoryHasDiscussions(repo string) (bool, error) {
	// Use GitHub GraphQL API to check if discussions are enabled
	// The hasDiscussionsEnabled field is the canonical way to check this
	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) {
			hasDiscussionsEnabled
		}
	}`

	// Split repo into owner and name
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid repository format: %s (expected owner/repo)", repo)
	}
	owner, name := parts[0], parts[1]

	// Execute GraphQL query using gh CLI
	type GraphQLResponse struct {
		Data struct {
			Repository struct {
				HasDiscussionsEnabled bool `json:"hasDiscussionsEnabled"`
			} `json:"repository"`
		} `json:"data"`
	}

	stdOut, _, err := gh.Exec("api", "graphql", "-f", fmt.Sprintf("query=%s", query),
		"-f", fmt.Sprintf("owner=%s", owner), "-f", fmt.Sprintf("name=%s", name))
	if err != nil {
		return false, fmt.Errorf("failed to query discussions status: %w", err)
	}

	var response GraphQLResponse
	if err := json.Unmarshal(stdOut.Bytes(), &response); err != nil {
		return false, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	return response.Data.Repository.HasDiscussionsEnabled, nil
}

// checkRepositoryHasIssues checks if a repository has issues enabled
func checkRepositoryHasIssues(repo string) (bool, error) {
	// Use GitHub REST API to check if issues are enabled
	// The has_issues field indicates if issues are enabled
	type RepositoryResponse struct {
		HasIssues bool `json:"has_issues"`
	}

	stdOut, _, err := gh.Exec("api", fmt.Sprintf("repos/%s", repo))
	if err != nil {
		return false, fmt.Errorf("failed to query repository: %w", err)
	}

	var response RepositoryResponse
	if err := json.Unmarshal(stdOut.Bytes(), &response); err != nil {
		return false, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return response.HasIssues, nil
}
