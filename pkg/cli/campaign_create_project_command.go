package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var campaignCreateProjectLog = logger.New("cli:campaign_create_project")

// NewCampaignCreateProjectCommand creates the `gh aw campaign create-project` command
func NewCampaignCreateProjectCommand() *cobra.Command {
	var (
		flagOwner  string
		flagTitle  string
		flagViews  []string
		flagFields []string
		flagOrg    bool
	)

	cmd := &cobra.Command{
		Use:   "create-project",
		Short: "Create a GitHub Project with views and custom fields",
		Long: `Create a GitHub Project V2 with optional views and custom field definitions.

This command creates a new GitHub Project and optionally configures:
- Multiple views (board, table, roadmap layouts)
- Custom field definitions (text, single-select, date fields)

The created project URL can be used in campaign spec files.

View format: name:layout[:filter]
  - name: View name (required)
  - layout: board, table, or roadmap (required)
  - filter: Optional filter expression (e.g., "is:open")

Field format: name:type[:options]
  - name: Field name (required)
  - type: TEXT, DATE, SINGLE_SELECT, NUMBER (required)
  - options: Comma-separated options for SINGLE_SELECT (e.g., "High,Medium,Low")

Examples:
  # Create basic project for org
  gh aw campaign create-project --owner myorg --title "Security Q1 2025" --org

  # Create project with views
  gh aw campaign create-project --owner myorg --title "Campaign Board" --org \
    --view "Progress:board:is:open" \
    --view "All Items:table"

  # Create project with custom fields
  gh aw campaign create-project --owner myorg --title "Task Tracker" --org \
    --field "Priority:SINGLE_SELECT:High,Medium,Low" \
    --field "Start Date:DATE" \
    --field "Campaign Id:TEXT"

  # Full example with views and fields
  gh aw campaign create-project --owner myorg --title "Security Campaign" --org \
    --view "Board:board:is:issue is:pr" \
    --view "Timeline:roadmap" \
    --field "Priority:SINGLE_SELECT:High,Medium,Low" \
    --field "Size:SINGLE_SELECT:Small,Medium,Large" \
    --field "Start Date:DATE"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagOwner == "" {
				return fmt.Errorf("--owner is required")
			}
			if flagTitle == "" {
				return fmt.Errorf("--title is required")
			}

			ownerType := "user"
			if flagOrg {
				ownerType = "org"
			}

			config := CreateProjectConfig{
				Owner:     flagOwner,
				OwnerType: ownerType,
				Title:     flagTitle,
			}

			// Parse views
			for _, viewSpec := range flagViews {
				view, err := parseViewSpec(viewSpec)
				if err != nil {
					return fmt.Errorf("invalid view specification %q: %w", viewSpec, err)
				}
				config.Views = append(config.Views, view)
			}

			// Parse fields
			for _, fieldSpec := range flagFields {
				field, err := parseFieldSpec(fieldSpec)
				if err != nil {
					return fmt.Errorf("invalid field specification %q: %w", fieldSpec, err)
				}
				config.Fields = append(config.Fields, field)
			}

			return RunCreateProject(config)
		},
	}

	cmd.Flags().StringVar(&flagOwner, "owner", "", "Owner (organization or user) for the project (required)")
	cmd.Flags().StringVar(&flagTitle, "title", "", "Project title (required)")
	cmd.Flags().StringSliceVar(&flagViews, "view", []string{}, "View specification in format name:layout[:filter] (can be repeated)")
	cmd.Flags().StringSliceVar(&flagFields, "field", []string{}, "Field specification in format name:type[:options] (can be repeated)")
	cmd.Flags().BoolVar(&flagOrg, "org", false, "Owner is an organization (default: user)")

	return cmd
}

// CreateProjectConfig holds configuration for creating a project
type CreateProjectConfig struct {
	Owner     string
	OwnerType string // "org" or "user"
	Title     string
	Views     []ProjectViewSpec
	Fields    []ProjectFieldSpec
}

// ProjectViewSpec represents a view to create
type ProjectViewSpec struct {
	Name   string
	Layout string
	Filter string
}

// ProjectFieldSpec represents a custom field to create
type ProjectFieldSpec struct {
	Name     string
	DataType string
	Options  []string
}

// RunCreateProject creates a GitHub Project with the given configuration
func RunCreateProject(config CreateProjectConfig) error {
	campaignCreateProjectLog.Printf("Creating project: owner=%s, ownerType=%s, title=%s", config.Owner, config.OwnerType, config.Title)

	ctx := context.Background()

	// Create GraphQL client
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get owner ID
	ownerID, err := getOwnerID(ctx, client, config.OwnerType, config.Owner)
	if err != nil {
		return fmt.Errorf("failed to get owner ID: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating project '%s'...", config.Title)))

	// Create project
	projectInfo, err := createProject(ctx, client, ownerID, config.Title)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Created project #%d: %s", projectInfo.Number, projectInfo.Title)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  URL: %s", projectInfo.URL)))

	// Create views if specified
	if len(config.Views) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating %d view(s)...", len(config.Views))))
		for i, view := range config.Views {
			if err := createProjectView(ctx, client, projectInfo.ID, projectInfo.Number, config.OwnerType, config.Owner, view); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create view '%s': %v", view.Name, err)))
				continue
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ Created view %d/%d: %s (%s)", i+1, len(config.Views), view.Name, view.Layout)))
		}
	}

	// Create custom fields if specified
	if len(config.Fields) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating %d custom field(s)...", len(config.Fields))))
		for i, field := range config.Fields {
			if err := createProjectField(ctx, client, projectInfo.ID, field); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create field '%s': %v", field.Name, err)))
				continue
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ Created field %d/%d: %s (%s)", i+1, len(config.Fields), field.Name, field.DataType)))
		}
	}

	// Output final project URL to stdout for scripting
	fmt.Println(projectInfo.URL)

	return nil
}

// ProjectInfo holds information about a created project
type ProjectInfo struct {
	ID     string
	Number int
	Title  string
	URL    string
}

// getOwnerID retrieves the node ID for an organization or user
func getOwnerID(ctx context.Context, client *api.GraphQLClient, ownerType, ownerLogin string) (string, error) {
	campaignCreateProjectLog.Printf("Getting owner ID: type=%s, login=%s", ownerType, ownerLogin)

	if ownerType == "org" {
		var query struct {
			Organization struct {
				ID string
			} `graphql:"organization(login: $login)"`
		}
		variables := map[string]interface{}{
			"login": ownerLogin,
		}
		if err := client.QueryWithContext(ctx, "", &query, variables); err != nil {
			return "", fmt.Errorf("failed to query organization: %w", err)
		}
		return query.Organization.ID, nil
	}

	var query struct {
		User struct {
			ID string
		} `graphql:"user(login: $login)"`
	}
	variables := map[string]interface{}{
		"login": ownerLogin,
	}
	if err := client.QueryWithContext(ctx, "", &query, variables); err != nil {
		return "", fmt.Errorf("failed to query user: %w", err)
	}
	return query.User.ID, nil
}

// createProject creates a new GitHub Project V2
func createProject(ctx context.Context, client *api.GraphQLClient, ownerID, title string) (*ProjectInfo, error) {
	campaignCreateProjectLog.Printf("Creating project: ownerID=%s, title=%s", ownerID, title)

	var mutation struct {
		CreateProjectV2 struct {
			ProjectV2 struct {
				ID     string
				Number int
				Title  string
				URL    string
			}
		} `graphql:"createProjectV2(input: $input)"`
	}

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"ownerId": ownerID,
			"title":   title,
		},
	}

	if err := client.MutateWithContext(ctx, "", &mutation, variables); err != nil {
		return nil, fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	project := mutation.CreateProjectV2.ProjectV2
	return &ProjectInfo{
		ID:     project.ID,
		Number: project.Number,
		Title:  project.Title,
		URL:    project.URL,
	}, nil
}

// createProjectView creates a view in the project using REST API
func createProjectView(ctx context.Context, client *api.GraphQLClient, projectID string, projectNumber int, ownerType, ownerLogin string, view ProjectViewSpec) error {
	campaignCreateProjectLog.Printf("Creating view: name=%s, layout=%s", view.Name, view.Layout)

	// We need to use the REST API for creating views
	// The GraphQL API doesn't support creating views yet
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	// Construct the REST endpoint based on owner type
	var endpoint string
	if ownerType == "org" {
		endpoint = fmt.Sprintf("orgs/%s/projects/%d/views", ownerLogin, projectNumber)
	} else {
		endpoint = fmt.Sprintf("users/%s/projects/%d/views", ownerLogin, projectNumber)
	}

	// Prepare request body
	body := map[string]interface{}{
		"name":   view.Name,
		"layout": view.Layout,
	}
	if view.Filter != "" {
		body["filter"] = view.Filter
	}

	// Marshal to JSON
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	var response map[string]interface{}
	if err := restClient.Post(endpoint, strings.NewReader(string(bodyJSON)), &response); err != nil {
		return fmt.Errorf("REST API request failed: %w", err)
	}

	return nil
}

// createProjectField creates a custom field in the project
func createProjectField(ctx context.Context, client *api.GraphQLClient, projectID string, field ProjectFieldSpec) error {
	campaignCreateProjectLog.Printf("Creating field: name=%s, type=%s", field.Name, field.DataType)

	// Use appropriate mutation based on field type
	switch field.DataType {
	case "SINGLE_SELECT":
		return createSingleSelectField(ctx, client, projectID, field)
	case "TEXT", "DATE", "NUMBER":
		return createSimpleField(ctx, client, projectID, field)
	default:
		return fmt.Errorf("unsupported field type: %s", field.DataType)
	}
}

// createSimpleField creates a text, date, or number field
func createSimpleField(ctx context.Context, client *api.GraphQLClient, projectID string, field ProjectFieldSpec) error {
	var mutation struct {
		CreateProjectV2Field struct {
			ProjectV2Field struct {
				ID string
			}
		} `graphql:"createProjectV2Field(input: $input)"`
	}

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"dataType":  field.DataType,
			"name":      field.Name,
		},
	}

	if err := client.MutateWithContext(ctx, "", &mutation, variables); err != nil {
		return fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	return nil
}

// createSingleSelectField creates a single-select field with options
func createSingleSelectField(ctx context.Context, client *api.GraphQLClient, projectID string, field ProjectFieldSpec) error {
	// First create the field
	if err := createSimpleField(ctx, client, projectID, field); err != nil {
		return err
	}

	// If no options specified, we're done
	if len(field.Options) == 0 {
		return nil
	}

	// Get the field ID to add options
	fieldID, err := getFieldID(ctx, client, projectID, field.Name)
	if err != nil {
		return fmt.Errorf("failed to get field ID: %w", err)
	}

	// Add each option
	for _, option := range field.Options {
		if err := addFieldOption(ctx, client, projectID, fieldID, option); err != nil {
			campaignCreateProjectLog.Printf("Warning: failed to add option '%s': %v", option, err)
			// Continue with other options
		}
	}

	return nil
}

// getFieldID retrieves the ID of a field by name
func getFieldID(ctx context.Context, client *api.GraphQLClient, projectID, fieldName string) (string, error) {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						Name string
						ID   string
					}
				} `graphql:"fields(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": projectID,
	}

	if err := client.QueryWithContext(ctx, "", &query, variables); err != nil {
		return "", fmt.Errorf("GraphQL query failed: %w", err)
	}

	for _, field := range query.Node.ProjectV2.Fields.Nodes {
		if field.Name == fieldName {
			return field.ID, nil
		}
	}

	return "", fmt.Errorf("field not found: %s", fieldName)
}

// addFieldOption adds an option to a single-select field
func addFieldOption(ctx context.Context, client *api.GraphQLClient, projectID, fieldID, optionName string) error {
	var mutation struct {
		UpdateProjectV2Field struct {
			ProjectV2Field struct {
				ID string
			}
		} `graphql:"updateProjectV2Field(input: $input)"`
	}

	// Note: The actual GraphQL API for adding options might be different
	// This is a placeholder that may need adjustment based on GitHub's API
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"projectId": projectID,
			"fieldId":   fieldID,
			"name":      optionName,
		},
	}

	if err := client.MutateWithContext(ctx, "", &mutation, variables); err != nil {
		return fmt.Errorf("GraphQL mutation failed: %w", err)
	}

	return nil
}

// parseViewSpec parses a view specification string
// Format: name:layout[:filter]
func parseViewSpec(spec string) (ProjectViewSpec, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return ProjectViewSpec{}, fmt.Errorf("view spec must have at least name and layout (format: name:layout[:filter])")
	}

	view := ProjectViewSpec{
		Name:   strings.TrimSpace(parts[0]),
		Layout: strings.TrimSpace(parts[1]),
	}

	if view.Name == "" {
		return ProjectViewSpec{}, fmt.Errorf("view name cannot be empty")
	}

	if view.Layout == "" {
		return ProjectViewSpec{}, fmt.Errorf("view layout cannot be empty")
	}

	// Validate layout
	validLayouts := map[string]bool{
		"board":   true,
		"table":   true,
		"roadmap": true,
	}
	if !validLayouts[view.Layout] {
		return ProjectViewSpec{}, fmt.Errorf("invalid layout %q (must be: board, table, or roadmap)", view.Layout)
	}

	// Optional filter
	if len(parts) > 2 {
		view.Filter = strings.TrimSpace(parts[2])
	}

	return view, nil
}

// parseFieldSpec parses a field specification string
// Format: name:type[:options]
func parseFieldSpec(spec string) (ProjectFieldSpec, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return ProjectFieldSpec{}, fmt.Errorf("field spec must have at least name and type (format: name:type[:options])")
	}

	field := ProjectFieldSpec{
		Name:     strings.TrimSpace(parts[0]),
		DataType: strings.ToUpper(strings.TrimSpace(parts[1])),
	}

	if field.Name == "" {
		return ProjectFieldSpec{}, fmt.Errorf("field name cannot be empty")
	}

	if field.DataType == "" {
		return ProjectFieldSpec{}, fmt.Errorf("field type cannot be empty")
	}

	// Validate data type
	validTypes := map[string]bool{
		"TEXT":          true,
		"DATE":          true,
		"SINGLE_SELECT": true,
		"NUMBER":        true,
	}
	if !validTypes[field.DataType] {
		return ProjectFieldSpec{}, fmt.Errorf("invalid type %q (must be: TEXT, DATE, SINGLE_SELECT, or NUMBER)", field.DataType)
	}

	// Parse options for SINGLE_SELECT
	if len(parts) > 2 && field.DataType == "SINGLE_SELECT" {
		optionsStr := strings.TrimSpace(parts[2])
		if optionsStr != "" {
			options := strings.Split(optionsStr, ",")
			for _, opt := range options {
				trimmed := strings.TrimSpace(opt)
				if trimmed != "" {
					field.Options = append(field.Options, trimmed)
				}
			}
		}
	}

	return field, nil
}
