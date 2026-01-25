package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var projectLog = logger.New("campaign:project")

// ProjectCreationConfig holds configuration for creating a campaign project
type ProjectCreationConfig struct {
	CampaignID   string
	CampaignName string
	Owner        string // GitHub org or user
	Verbose      bool
}

// ProjectCreationResult holds the result of project creation
type ProjectCreationResult struct {
	ProjectURL    string
	ProjectNumber int
}

// CreateCampaignProject creates a GitHub Project with required views and fields for a campaign
func CreateCampaignProject(config ProjectCreationConfig) (*ProjectCreationResult, error) {
	projectLog.Printf("Creating campaign project for campaign ID: %s", config.CampaignID)

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return nil, fmt.Errorf("GitHub CLI (gh) is not available. Install it from https://cli.github.com/")
	}

	// Create the project
	projectURL, projectNumber, err := createProject(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	console.LogVerbose(config.Verbose, fmt.Sprintf("Created project: %s", projectURL))

	// Create required fields
	if err := createProjectFields(config, projectNumber); err != nil {
		return nil, fmt.Errorf("failed to create project fields: %w", err)
	}

	console.LogVerbose(config.Verbose, "Created project fields")

	result := &ProjectCreationResult{
		ProjectURL:    projectURL,
		ProjectNumber: projectNumber,
	}

	return result, nil
}

// isGHCLIAvailable checks if the gh CLI is installed and available
func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

// createProject creates a new GitHub Project and returns its URL and number
func createProject(config ProjectCreationConfig) (string, int, error) {
	projectLog.Printf("Creating project with title: %s", config.CampaignName)

	// Create project using gh CLI
	cmd := exec.Command("gh", "project", "create",
		"--owner", config.Owner,
		"--title", config.CampaignName,
		"--format", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create project: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON output to get project URL and number
	var result struct {
		URL    string `json:"url"`
		Number int    `json:"number"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", 0, fmt.Errorf("failed to parse project creation output: %w\nOutput: %s", err, string(output))
	}

	projectLog.Printf("Project created: URL=%s, Number=%d", result.URL, result.Number)
	return result.URL, result.Number, nil
}

// createProjectFields creates the required fields for a campaign project
func createProjectFields(config ProjectCreationConfig, projectNumber int) error {
	projectLog.Printf("Creating fields for project number: %d", projectNumber)

	// Define required fields
	fields := []struct {
		name     string
		dataType string
		options  []string // For SINGLE_SELECT fields
	}{
		{"Campaign Id", "TEXT", nil},
		{"Worker Workflow", "TEXT", nil},
		{"Priority", "SINGLE_SELECT", []string{"High", "Medium", "Low"}},
		{"Size", "SINGLE_SELECT", []string{"Small", "Medium", "Large"}},
		{"Start Date", "DATE", nil},
		{"End Date", "DATE", nil},
	}

	// Create each field
	for _, field := range fields {
		if err := createField(config, projectNumber, field.name, field.dataType, field.options); err != nil {
			return fmt.Errorf("failed to create field '%s': %w", field.name, err)
		}
		console.LogVerbose(config.Verbose, fmt.Sprintf("Created field: %s", field.name))
	}

	return nil
}

// createField creates a single field in the project
func createField(config ProjectCreationConfig, projectNumber int, name, dataType string, options []string) error {
	projectLog.Printf("Creating field: name=%s, type=%s", name, dataType)

	args := []string{
		"project", "field-create", fmt.Sprintf("%d", projectNumber),
		"--owner", config.Owner,
		"--name", name,
		"--data-type", dataType,
	}

	// Add options for SINGLE_SELECT fields
	if dataType == "SINGLE_SELECT" && len(options) > 0 {
		args = append(args, "--single-select-options", strings.Join(options, ","))
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create field: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// UpdateSpecWithProjectURL updates a campaign spec file with the project URL
func UpdateSpecWithProjectURL(specPath, projectURL string) error {
	projectLog.Printf("Updating spec file %s with project URL: %s", specPath, projectURL)

	// Read the spec file
	content, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}

	specContent := string(content)

	// Replace the placeholder project URL with the actual one
	placeholderURL := "https://github.com/orgs/ORG/projects/1"
	if !strings.Contains(specContent, placeholderURL) {
		// If placeholder doesn't exist, the spec might have been manually edited
		projectLog.Print("Placeholder project URL not found, spec may have been edited")
		return nil
	}

	updatedContent := strings.Replace(specContent, placeholderURL, projectURL, 1)

	// Write the updated content back
	if err := os.WriteFile(specPath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("failed to write updated spec file: %w", err)
	}

	projectLog.Print("Successfully updated spec file with project URL")
	return nil
}
