package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var validationLog = logger.New("workflow:validation")

// RepositoryFeatures holds cached information about repository capabilities
type RepositoryFeatures struct {
	HasDiscussions bool
	HasIssues      bool
}

// Global cache for repository features and current repository info
var (
	repositoryFeaturesCache  = sync.Map{} // sync.Map is thread-safe and efficient for read-heavy workloads
	getCurrentRepositoryOnce sync.Once
	currentRepositoryResult  string
	currentRepositoryError   error
)

// ClearRepositoryFeaturesCache clears the repository features cache
// This is useful for testing or when repository settings might have changed
func ClearRepositoryFeaturesCache() {
	// Clear the features cache
	repositoryFeaturesCache.Range(func(key, value any) bool {
		repositoryFeaturesCache.Delete(key)
		return true
	})

	// Reset the current repository cache
	getCurrentRepositoryOnce = sync.Once{}
	currentRepositoryResult = ""
	currentRepositoryError = nil

	validationLog.Print("Repository features and current repository caches cleared")
}

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
// Cached compiled schema to avoid recompiling on every validation
var (
	compiledSchemaOnce sync.Once
	compiledSchema     *jsonschema.Schema
	schemaCompileError error
)

// getCompiledSchema returns the compiled GitHub Actions schema, compiling it once and caching
func getCompiledSchema() (*jsonschema.Schema, error) {
	compiledSchemaOnce.Do(func() {
		// Parse the embedded schema
		var schemaDoc any
		if err := json.Unmarshal([]byte(githubWorkflowSchema), &schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to parse embedded GitHub Actions schema: %w", err)
			return
		}

		// Create compiler and add the schema as a resource
		loader := jsonschema.NewCompiler()
		schemaURL := "https://json.schemastore.org/github-workflow.json"
		if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
			schemaCompileError = fmt.Errorf("failed to add schema resource: %w", err)
			return
		}

		// Compile the schema once
		schema, err := loader.Compile(schemaURL)
		if err != nil {
			schemaCompileError = fmt.Errorf("failed to compile GitHub Actions schema: %w", err)
			return
		}

		compiledSchema = schema
	})

	return compiledSchema, schemaCompileError
}

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

	// Get the cached compiled schema
	schema, err := getCompiledSchema()
	if err != nil {
		return err
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

	// Check if discussions are enabled when create-discussion or add-comment with discussion: true is configured
	needsDiscussions := workflowData.SafeOutputs.CreateDiscussions != nil ||
		(workflowData.SafeOutputs.AddComments != nil &&
			workflowData.SafeOutputs.AddComments.Discussion != nil &&
			*workflowData.SafeOutputs.AddComments.Discussion)

	if needsDiscussions {
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
			if workflowData.SafeOutputs.CreateDiscussions != nil {
				return fmt.Errorf("workflow uses safe-outputs.create-discussion but repository %s does not have discussions enabled. Enable discussions in repository settings or remove create-discussion from safe-outputs", repo)
			}
			// For add-comment with discussion: true
			return fmt.Errorf("workflow uses safe-outputs.add-comment with discussion: true but repository %s does not have discussions enabled. Enable discussions in repository settings or change add-comment configuration", repo)
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

// getCurrentRepository gets the current repository from git context (with caching)
func getCurrentRepository() (string, error) {
	getCurrentRepositoryOnce.Do(func() {
		currentRepositoryResult, currentRepositoryError = getCurrentRepositoryUncached()
	})

	if currentRepositoryError != nil {
		return "", currentRepositoryError
	}

	validationLog.Printf("Using cached current repository: %s", currentRepositoryResult)
	return currentRepositoryResult, nil
}

// getCurrentRepositoryUncached fetches the current repository from gh CLI (no caching)
func getCurrentRepositoryUncached() (string, error) {
	validationLog.Print("Fetching current repository from gh CLI")

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

	validationLog.Printf("Cached current repository: %s", repo)
	return repo, nil
}

// getRepositoryFeatures gets repository features with caching to amortize API calls
func getRepositoryFeatures(repo string) (*RepositoryFeatures, error) {
	// Check cache first using sync.Map
	if cached, exists := repositoryFeaturesCache.Load(repo); exists {
		features := cached.(*RepositoryFeatures)
		validationLog.Printf("Using cached repository features for: %s", repo)
		return features, nil
	}

	validationLog.Printf("Fetching repository features from API for: %s", repo)

	// Fetch from API
	features := &RepositoryFeatures{}

	// Check discussions
	hasDiscussions, err := checkRepositoryHasDiscussionsUncached(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check discussions: %w", err)
	}
	features.HasDiscussions = hasDiscussions

	// Check issues
	hasIssues, err := checkRepositoryHasIssuesUncached(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check issues: %w", err)
	}
	features.HasIssues = hasIssues

	// Cache the result using sync.Map's LoadOrStore for atomic caching
	// This handles the race condition where multiple goroutines might fetch the same repo
	actual, _ := repositoryFeaturesCache.LoadOrStore(repo, features)
	actualFeatures := actual.(*RepositoryFeatures)

	validationLog.Printf("Cached repository features for: %s (discussions: %v, issues: %v)", repo, actualFeatures.HasDiscussions, actualFeatures.HasIssues)

	return actualFeatures, nil
}

// checkRepositoryHasDiscussions checks if a repository has discussions enabled (with caching)
func checkRepositoryHasDiscussions(repo string) (bool, error) {
	features, err := getRepositoryFeatures(repo)
	if err != nil {
		return false, err
	}
	return features.HasDiscussions, nil
}

// checkRepositoryHasDiscussionsUncached checks if a repository has discussions enabled (no caching)
func checkRepositoryHasDiscussionsUncached(repo string) (bool, error) {
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

// checkRepositoryHasIssues checks if a repository has issues enabled (with caching)
func checkRepositoryHasIssues(repo string) (bool, error) {
	features, err := getRepositoryFeatures(repo)
	if err != nil {
		return false, err
	}
	return features.HasIssues, nil
}

// checkRepositoryHasIssuesUncached checks if a repository has issues enabled (no caching)
func checkRepositoryHasIssuesUncached(repo string) (bool, error) {
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

// validateHTTPTransportSupport validates that HTTP MCP servers are only used with engines that support HTTP transport
func (c *Compiler) validateHTTPTransportSupport(tools map[string]any, engine CodingAgentEngine) error {
	if engine.SupportsHTTPTransport() {
		// Engine supports HTTP transport, no validation needed
		return nil
	}

	// Engine doesn't support HTTP transport, check for HTTP MCP servers
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(config); hasMcp && mcpType == "http" {
				return fmt.Errorf("tool '%s' uses HTTP transport which is not supported by engine '%s' (only stdio transport is supported)", toolName, engine.GetID())
			}
		}
	}

	return nil
}

// validateMaxTurnsSupport validates that max-turns is only used with engines that support this feature
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine CodingAgentEngine) error {
	// Check if max-turns is specified in the engine config
	engineSetting, engineConfig := c.ExtractEngineConfig(frontmatter)
	_ = engineSetting // Suppress unused variable warning

	hasMaxTurns := engineConfig != nil && engineConfig.MaxTurns != ""

	if !hasMaxTurns {
		// No max-turns specified, no validation needed
		return nil
	}

	// max-turns is specified, check if the engine supports it
	if !engine.SupportsMaxTurns() {
		return fmt.Errorf("max-turns not supported: engine '%s' does not support the max-turns feature", engine.GetID())
	}

	// Engine supports max-turns - additional validation could be added here if needed
	// For now, we rely on JSON schema validation for format checking

	return nil
}

// validateWebSearchSupport validates that web-search tool is only used with engines that support this feature
func (c *Compiler) validateWebSearchSupport(tools map[string]any, engine CodingAgentEngine) {
	// Check if web-search tool is requested
	_, hasWebSearch := tools["web-search"]

	if !hasWebSearch {
		// No web-search specified, no validation needed
		return
	}

	// web-search is specified, check if the engine supports it
	if !engine.SupportsWebSearch() {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Engine '%s' does not support the web-search tool. See https://githubnext.github.io/gh-aw/guides/web-search/ for alternatives.", engine.GetID())))
		c.IncrementWarningCount()
	}
}
