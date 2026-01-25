package campaign

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
	LinkRepo     string // Optional: owner/name to link the project to
	NoLinkRepo   bool   // Disable best-effort repo linking
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

	// Create standard views (board/table/roadmap)
	if err := createProjectViews(config, projectURL); err != nil {
		return nil, fmt.Errorf("failed to create project views: %w", err)
	}

	console.LogVerbose(config.Verbose, "Created project views")

	// Best-effort: link the project to the current repository.
	// This should not block campaign creation if linking fails due to permissions.
	if err := linkProjectToRepo(config, projectURL); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"Could not link project to repository automatically: "+err.Error(),
		))
	}

	// Ensure the Progress Board's typical grouping field (Status) includes a
	// "Review Required" option, which effectively becomes a new column.
	if err := ensureStatusOption(config, projectURL, "Review Required"); err != nil {
		return nil, fmt.Errorf("failed to update project status options: %w", err)
	}

	result := &ProjectCreationResult{
		ProjectURL:    projectURL,
		ProjectNumber: projectNumber,
	}

	return result, nil
}

func linkProjectToRepo(config ProjectCreationConfig, projectURL string) error {
	if config.NoLinkRepo {
		console.LogVerbose(config.Verbose, "Skipping project-to-repo linking (--no-link-repo)")
		return nil
	}

	nameWithOwner := strings.TrimSpace(config.LinkRepo)
	if nameWithOwner == "" {
		var err error
		nameWithOwner, err = getCurrentRepoNameWithOwner()
		if err != nil {
			return err
		}
	}

	owner, repo, err := parseRepoNameWithOwner(nameWithOwner)
	if err != nil {
		return err
	}

	info, err := parseProjectURL(projectURL)
	if err != nil {
		return err
	}

	// GitHub only allows linking when the project and repository share the same owner.
	// Avoid calling the mutation to prevent a noisy (and expected) GraphQL validation error.
	if !strings.EqualFold(info.ownerLogin, owner) {
		return fmt.Errorf(
			"project is owned by %q but current repository is %q; GitHub only allows linking projects to repositories owned by the same account/org. Re-run with --owner %s (or --owner @me for personal repos) from the repo you want linked",
			info.ownerLogin,
			nameWithOwner,
			owner,
		)
	}

	projectID, err := getProjectID(info)
	if err != nil {
		return err
	}

	repoID, err := getRepositoryID(owner, repo)
	if err != nil {
		return err
	}

	if err := linkProjectV2ToRepository(projectID, repoID); err != nil {
		return err
	}

	console.LogVerbose(config.Verbose, fmt.Sprintf("Linked project to repository: %s", nameWithOwner))
	return nil
}

func parseRepoNameWithOwner(nameWithOwner string) (string, string, error) {
	trimmed := strings.TrimSpace(nameWithOwner)
	owner, repo, ok := strings.Cut(trimmed, "/")
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	owner = strings.TrimPrefix(owner, "@")
	if !ok || owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository %q; expected format owner/name", nameWithOwner)
	}
	return owner, repo, nil
}

func getCurrentRepoNameWithOwner() (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh repo view failed: %w\nOutput: %s", err, string(out))
	}
	nameWithOwner := strings.TrimSpace(string(out))
	if nameWithOwner == "" {
		return "", fmt.Errorf("failed to determine current repository (empty nameWithOwner)")
	}
	return nameWithOwner, nil
}

func getRepositoryID(owner, name string) (string, error) {
	query := `query($owner: String!, $name: String!) {
		repository(owner: $owner, name: $name) { id }
	}`

	cmd := exec.Command(
		"gh",
		"api",
		"graphql",
		"-F",
		"owner="+owner,
		"-F",
		"name="+name,
		"-f",
		"query="+query,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api graphql failed: %w\nOutput: %s", err, string(out))
	}

	var resp struct {
		Data struct {
			Repository struct {
				ID string `json:"id"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response: %w\nOutput: %s", err, string(out))
	}

	if resp.Data.Repository.ID == "" {
		return "", fmt.Errorf("failed to find repository ID for %s/%s", owner, name)
	}

	return resp.Data.Repository.ID, nil
}

func getProjectID(info projectURLInfo) (string, error) {
	query := ""
	if info.scope == "orgs" {
		query = `query($login: String!, $number: Int!) {
			organization(login: $login) {
				projectV2(number: $number) { id }
			}
		}`
	} else {
		query = `query($login: String!, $number: Int!) {
			user(login: $login) {
				projectV2(number: $number) { id }
			}
		}`
	}

	cmd := exec.Command(
		"gh",
		"api",
		"graphql",
		"-F",
		"login="+info.ownerLogin,
		"-F",
		fmt.Sprintf("number=%d", info.projectNumber),
		"-f",
		"query="+query,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api graphql failed: %w\nOutput: %s", err, string(out))
	}

	if info.scope == "orgs" {
		var resp struct {
			Data struct {
				Organization struct {
					ProjectV2 struct {
						ID string `json:"id"`
					} `json:"projectV2"`
				} `json:"organization"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			return "", fmt.Errorf("failed to parse GraphQL response: %w\nOutput: %s", err, string(out))
		}
		if resp.Data.Organization.ProjectV2.ID == "" {
			return "", fmt.Errorf("failed to find project ID in GraphQL response")
		}
		return resp.Data.Organization.ProjectV2.ID, nil
	}

	var resp struct {
		Data struct {
			User struct {
				ProjectV2 struct {
					ID string `json:"id"`
				} `json:"projectV2"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response: %w\nOutput: %s", err, string(out))
	}
	if resp.Data.User.ProjectV2.ID == "" {
		return "", fmt.Errorf("failed to find project ID in GraphQL response")
	}
	return resp.Data.User.ProjectV2.ID, nil
}

func linkProjectV2ToRepository(projectID, repositoryID string) error {
	mutation := `mutation($input: LinkProjectV2ToRepositoryInput!) {
		linkProjectV2ToRepository(input: $input) {
			clientMutationId
		}
	}`

	requestBody := map[string]any{
		"query": mutation,
		"variables": map[string]any{
			"input": map[string]any{
				"projectId":    projectID,
				"repositoryId": repositoryID,
			},
		},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request body: %w", err)
	}

	cmd := exec.Command("gh", "api", "graphql", "--input", "-")
	cmd.Stdin = bytes.NewReader(requestJSON)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// If it's already linked, treat it as success.
		msg := string(out)
		if strings.Contains(strings.ToLower(msg), "already") && strings.Contains(strings.ToLower(msg), "link") {
			return nil
		}
		return fmt.Errorf("gh api graphql link failed: %w\nOutput: %s", err, msg)
	}

	return nil
}

type projectURLInfo struct {
	scope         string // "users" or "orgs"
	ownerLogin    string
	projectNumber int
}

var projectURLRegexp = regexp.MustCompile(`github\.com\/(users|orgs)\/([^/]+)\/projects\/(\d+)`)

func parseProjectURL(projectURL string) (projectURLInfo, error) {
	match := projectURLRegexp.FindStringSubmatch(projectURL)
	if match == nil {
		return projectURLInfo{}, fmt.Errorf("invalid project URL: %q. Expected format: https://github.com/orgs/myorg/projects/123", projectURL)
	}

	projectNumber, err := strconv.Atoi(match[3])
	if err != nil {
		return projectURLInfo{}, fmt.Errorf("invalid project number in URL %q: %w", projectURL, err)
	}

	return projectURLInfo{
		scope:         match[1],
		ownerLogin:    match[2],
		projectNumber: projectNumber,
	}, nil
}

func createProjectViews(config ProjectCreationConfig, projectURL string) error {
	projectLog.Printf("Creating standard views for project URL: %s", projectURL)

	info, err := parseProjectURL(projectURL)
	if err != nil {
		return err
	}

	views := []struct {
		name   string
		layout string
	}{
		{name: "Progress Board", layout: "board"},
		{name: "Task Tracker", layout: "table"},
		{name: "Campaign Roadmap", layout: "roadmap"},
	}

	for _, view := range views {
		if err := createProjectView(info, view.name, view.layout); err != nil {
			return fmt.Errorf("failed to create view %q (%s): %w", view.name, view.layout, err)
		}
		console.LogVerbose(config.Verbose, fmt.Sprintf("Created view: %s (%s)", view.name, view.layout))
	}

	return nil
}

func createProjectView(info projectURLInfo, name, layout string) error {
	path := ""
	if info.scope == "orgs" {
		path = fmt.Sprintf("/orgs/%s/projectsV2/%d/views", info.ownerLogin, info.projectNumber)
	} else {
		path = fmt.Sprintf("/users/%s/projectsV2/%d/views", info.ownerLogin, info.projectNumber)
	}

	cmd := exec.Command(
		"gh",
		"api",
		"--method",
		"POST",
		path,
		"-H",
		"Accept: application/vnd.github+json",
		"-H",
		"X-GitHub-Api-Version: 2022-11-28",
		"-f",
		"name="+name,
		"-f",
		"layout="+layout,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

type singleSelectOption struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type statusFieldLookup struct {
	ProjectID string
	FieldID   string
	Options   []singleSelectOption
}

func ensureStatusOption(config ProjectCreationConfig, projectURL string, optionName string) error {
	projectLog.Printf("Ensuring Status option %q exists for project URL: %s", optionName, projectURL)

	info, err := parseProjectURL(projectURL)
	if err != nil {
		return err
	}

	lookup, err := getStatusField(config, info)
	if err != nil {
		return err
	}

	updatedOptions, changed := ensureSingleSelectOptionBefore(
		lookup.Options,
		singleSelectOption{Name: optionName, Color: "BLUE", Description: "Needs review before moving to Done"},
		"Done",
	)
	if !changed {
		console.LogVerbose(config.Verbose, fmt.Sprintf("Status option already present and ordered: %s", optionName))
		return nil
	}

	if err := updateSingleSelectFieldOptions(lookup.FieldID, updatedOptions); err != nil {
		return err
	}

	console.LogVerbose(config.Verbose, fmt.Sprintf("Ensured Status option is ordered before Done: %s", optionName))
	return nil
}

func ensureSingleSelectOptionBefore(options []singleSelectOption, desired singleSelectOption, beforeName string) ([]singleSelectOption, bool) {
	var existing *singleSelectOption
	without := make([]singleSelectOption, 0, len(options))
	for _, opt := range options {
		if opt.Name == desired.Name {
			if existing == nil {
				copyOpt := opt
				existing = &copyOpt
			}
			continue
		}
		without = append(without, opt)
	}

	toInsert := desired
	if existing != nil {
		toInsert = *existing
		toInsert.Color = desired.Color
		if desired.Description != "" {
			toInsert.Description = desired.Description
		}
	}

	insertAt := len(without)
	for i, opt := range without {
		if opt.Name == beforeName {
			insertAt = i
			break
		}
	}

	withInserted := make([]singleSelectOption, 0, len(without)+1)
	withInserted = append(withInserted, without[:insertAt]...)
	withInserted = append(withInserted, toInsert)
	withInserted = append(withInserted, without[insertAt:]...)

	return withInserted, !singleSelectOptionsEqual(options, withInserted)
}

func singleSelectOptionsEqual(a, b []singleSelectOption) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func getStatusField(config ProjectCreationConfig, info projectURLInfo) (statusFieldLookup, error) {
	// Query the project ID and the built-in Status single-select field.
	// We use the REST URL components (org/user + number) to locate the project.
	query := ""
	if info.scope == "orgs" {
		query = `query($login: String!, $number: Int!) {
			organization(login: $login) {
				projectV2(number: $number) {
					id
					fields(first: 100) {
						nodes {
							... on ProjectV2SingleSelectField {
								id
								name
								options { name color description }
							}
						}
					}
				}
			}
		}`
	} else {
		query = `query($login: String!, $number: Int!) {
			user(login: $login) {
				projectV2(number: $number) {
					id
					fields(first: 100) {
						nodes {
							... on ProjectV2SingleSelectField {
								id
								name
								options { name color description }
							}
						}
					}
				}
			}
		}`
	}

	cmd := exec.Command(
		"gh",
		"api",
		"graphql",
		"-F",
		"login="+info.ownerLogin,
		"-F",
		fmt.Sprintf("number=%d", info.projectNumber),
		"-f",
		"query="+query,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return statusFieldLookup{}, fmt.Errorf("gh api graphql failed: %w\nOutput: %s", err, string(out))
	}

	// Parse response
	if info.scope == "orgs" {
		var resp struct {
			Data struct {
				Organization struct {
					ProjectV2 struct {
						ID     string `json:"id"`
						Fields struct {
							Nodes []struct {
								ID      string               `json:"id"`
								Name    string               `json:"name"`
								Options []singleSelectOption `json:"options"`
							} `json:"nodes"`
						} `json:"fields"`
					} `json:"projectV2"`
				} `json:"organization"`
			} `json:"data"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return statusFieldLookup{}, fmt.Errorf("failed to parse GraphQL response: %w\nOutput: %s", err, string(out))
		}

		projectID := resp.Data.Organization.ProjectV2.ID
		if projectID == "" {
			return statusFieldLookup{}, fmt.Errorf("failed to find project ID in GraphQL response")
		}

		for _, node := range resp.Data.Organization.ProjectV2.Fields.Nodes {
			if node.Name == "Status" {
				return statusFieldLookup{ProjectID: projectID, FieldID: node.ID, Options: node.Options}, nil
			}
		}
	} else {
		var resp struct {
			Data struct {
				User struct {
					ProjectV2 struct {
						ID     string `json:"id"`
						Fields struct {
							Nodes []struct {
								ID      string               `json:"id"`
								Name    string               `json:"name"`
								Options []singleSelectOption `json:"options"`
							} `json:"nodes"`
						} `json:"fields"`
					} `json:"projectV2"`
				} `json:"user"`
			} `json:"data"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return statusFieldLookup{}, fmt.Errorf("failed to parse GraphQL response: %w\nOutput: %s", err, string(out))
		}

		projectID := resp.Data.User.ProjectV2.ID
		if projectID == "" {
			return statusFieldLookup{}, fmt.Errorf("failed to find project ID in GraphQL response")
		}

		for _, node := range resp.Data.User.ProjectV2.Fields.Nodes {
			if node.Name == "Status" {
				return statusFieldLookup{ProjectID: projectID, FieldID: node.ID, Options: node.Options}, nil
			}
		}
	}

	return statusFieldLookup{}, fmt.Errorf("failed to locate Status field on project")
}

func updateSingleSelectFieldOptions(fieldID string, options []singleSelectOption) error {
	mutation := `mutation($input: UpdateProjectV2FieldInput!) {
		updateProjectV2Field(input: $input) {
			projectV2Field {
				... on ProjectV2SingleSelectField {
					name
					options { name }
				}
			}
		}
	}`

	input := map[string]any{
		"fieldId":             fieldID,
		"singleSelectOptions": options,
	}

	requestBody := map[string]any{
		"query": mutation,
		"variables": map[string]any{
			"input": input,
		},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request body: %w", err)
	}

	cmd := exec.Command("gh", "api", "graphql", "--input", "-")
	cmd.Stdin = bytes.NewReader(requestJSON)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api graphql update failed: %w\nOutput: %s", err, string(out))
	}

	return nil
}

// isGHCLIAvailable checks if the gh CLI is installed and available
func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

func normalizeProjectOwner(owner string) string {
	trimmed := strings.TrimSpace(owner)
	if strings.EqualFold(trimmed, "@me") {
		return "@me"
	}
	trimmed = strings.TrimPrefix(trimmed, "@")
	return trimmed
}

// createProject creates a new GitHub Project and returns its URL and number
func createProject(config ProjectCreationConfig) (string, int, error) {
	projectLog.Printf("Creating project with title: %s", config.CampaignName)

	owner := normalizeProjectOwner(config.Owner)

	// Create project using gh CLI
	cmd := exec.Command("gh", "project", "create",
		"--owner", owner,
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

	owner := normalizeProjectOwner(config.Owner)

	args := []string{
		"project", "field-create", fmt.Sprintf("%d", projectNumber),
		"--owner", owner,
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
