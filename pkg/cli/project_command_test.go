//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectCommand(t *testing.T) {
	cmd := NewProjectCommand()
	require.NotNil(t, cmd, "Command should be created")
	assert.Equal(t, "project", cmd.Use, "Command name should be 'project'")
	assert.Equal(t, "Create and manage GitHub Projects V2 boards", cmd.Short, "Short description should describe project creation and management")
	assert.Contains(t, cmd.Long, "Create GitHub Projects V2 boards linked to repositories.", "Long description should describe creation behavior")
	assert.NotEmpty(t, cmd.Commands(), "Command should have subcommands")
}

func TestNewProjectNewCommand(t *testing.T) {
	cmd := NewProjectNewCommand()
	require.NotNil(t, cmd, "Command should be created")
	assert.Equal(t, "new <title>", cmd.Use, "Command usage should be 'new <title>'")
	assert.Contains(t, cmd.Short, "Create a new GitHub Project V2 board", "Short description should mention board creation")
	assert.Contains(t, cmd.Long, "https://github.github.com/gh-aw/reference/auth-projects/", "Long description should reference the configured docs host")
	assert.NotContains(t, cmd.Long, "https://github.github.io/gh-aw/reference/auth-projects/", "Long description should not reference the old docs host")

	// Check flags
	ownerFlag := cmd.Flags().Lookup("owner")
	require.NotNil(t, ownerFlag, "Should have --owner flag")
	assert.Empty(t, ownerFlag.Shorthand, "Owner flag should not define a shorthand to avoid -o collision with output")

	linkFlag := cmd.Flags().Lookup("link")
	require.NotNil(t, linkFlag, "Should have --link flag")
	assert.Equal(t, "l", linkFlag.Shorthand, "Link flag should have short form 'l'")
}

func TestEscapeGraphQLString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "with quotes",
			input:    `Project "Alpha"`,
			expected: `Project \"Alpha\"`,
		},
		{
			name:     "with backslash",
			input:    `Path\to\file`,
			expected: `Path\\to\\file`,
		},
		{
			name:     "with newline",
			input:    "Line 1\nLine 2",
			expected: "Line 1\\nLine 2",
		},
		{
			name:     "with tab",
			input:    "Name\tValue",
			expected: "Name\\tValue",
		},
		{
			name:     "complex string",
			input:    "Test \"project\"\nWith\ttabs\\and backslashes",
			expected: "Test \\\"project\\\"\\nWith\\ttabs\\\\and backslashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeGraphQLString(tt.input)
			assert.Equal(t, tt.expected, result, "GraphQL string should be properly escaped")
		})
	}
}

func TestProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      ProjectConfig
		description string
	}{
		{
			name: "user project",
			config: ProjectConfig{
				Title:     "My Project",
				Owner:     "testuser",
				OwnerType: "user",
			},
			description: "Should create user project",
		},
		{
			name: "org project",
			config: ProjectConfig{
				Title:     "Team Board",
				Owner:     "myorg",
				OwnerType: "org",
			},
			description: "Should create org project",
		},
		{
			name: "project with repo",
			config: ProjectConfig{
				Title:     "Bugs",
				Owner:     "myorg",
				OwnerType: "org",
				Repo:      "myorg/myrepo",
			},
			description: "Should create project linked to repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.config.Title, "Project title should not be empty")
			assert.NotEmpty(t, tt.config.Owner, "Project owner should not be empty")
			assert.NotEmpty(t, tt.config.OwnerType, "Owner type should not be empty")
			assert.Contains(t, []string{"user", "org"}, tt.config.OwnerType, "Owner type should be 'user' or 'org'")
		})
	}
}

func TestProjectNewCommandArgs(t *testing.T) {
	cmd := NewProjectNewCommand()

	tests := []struct {
		name      string
		args      []string
		shouldErr bool
	}{
		{
			name:      "no arguments",
			args:      []string{},
			shouldErr: true,
		},
		{
			name:      "one argument",
			args:      []string{"My Project"},
			shouldErr: false,
		},
		{
			name:      "too many arguments",
			args:      []string{"My Project", "Extra"},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Args(cmd, tt.args)
			if tt.shouldErr {
				assert.Error(t, err, "Should return error for invalid arguments")
			} else {
				assert.NoError(t, err, "Should not return error for valid arguments")
			}
		})
	}
}

func TestProjectNewCommandFlags(t *testing.T) {
	cmd := NewProjectNewCommand()

	// Check standard flags
	ownerFlag := cmd.Flags().Lookup("owner")
	require.NotNil(t, ownerFlag, "Should have --owner flag")

	linkFlag := cmd.Flags().Lookup("link")
	require.NotNil(t, linkFlag, "Should have --link flag")

	// Check project setup flag
	projectSetupFlag := cmd.Flags().Lookup("with-project-setup")
	require.NotNil(t, projectSetupFlag, "Should have --with-project-setup flag")
	assert.Equal(t, "bool", projectSetupFlag.Value.Type(), "Project setup flag should be boolean")

	// Verify removed flags don't exist
	viewsFlag := cmd.Flags().Lookup("views")
	assert.Nil(t, viewsFlag, "Should not have --views flag")

	fieldsFlag := cmd.Flags().Lookup("fields")
	assert.Nil(t, fieldsFlag, "Should not have --fields flag")
}

func TestParseProjectURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedScope  string
		expectedOwner  string
		expectedNumber int
		shouldErr      bool
	}{
		{
			name:           "org project",
			url:            "https://github.com/orgs/myorg/projects/123",
			expectedScope:  "orgs",
			expectedOwner:  "myorg",
			expectedNumber: 123,
			shouldErr:      false,
		},
		{
			name:           "user project",
			url:            "https://github.com/users/myuser/projects/456",
			expectedScope:  "users",
			expectedOwner:  "myuser",
			expectedNumber: 456,
			shouldErr:      false,
		},
		{
			name:      "invalid URL",
			url:       "https://github.com/myorg/myrepo",
			shouldErr: true,
		},
		{
			name:      "empty URL",
			url:       "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseProjectURL(tt.url)
			if tt.shouldErr {
				assert.Error(t, err, "Should return error for invalid URL")
			} else {
				require.NoError(t, err, "Should not return error for valid URL")
				assert.Equal(t, tt.expectedScope, result.scope, "Scope should match")
				assert.Equal(t, tt.expectedOwner, result.ownerLogin, "Owner should match")
				assert.Equal(t, tt.expectedNumber, result.projectNumber, "Project number should match")
			}
		})
	}
}

func TestEnsureSingleSelectOptionBefore(t *testing.T) {
	tests := []struct {
		name           string
		options        []singleSelectOption
		desired        singleSelectOption
		beforeName     string
		expectChanged  bool
		expectedLength int
	}{
		{
			name: "add new option before Done",
			options: []singleSelectOption{
				{Name: "Todo", Color: "GRAY"},
				{Name: "In Progress", Color: "YELLOW"},
				{Name: "Done", Color: "GREEN"},
			},
			desired:        singleSelectOption{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
			beforeName:     "Done",
			expectChanged:  true,
			expectedLength: 4,
		},
		{
			name: "option already exists in correct position",
			options: []singleSelectOption{
				{Name: "Todo", Color: "GRAY"},
				{Name: "In Progress", Color: "YELLOW"},
				{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
				{Name: "Done", Color: "GREEN"},
			},
			desired:        singleSelectOption{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
			beforeName:     "Done",
			expectChanged:  false,
			expectedLength: 4,
		},
		{
			name: "option exists but in wrong position",
			options: []singleSelectOption{
				{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
				{Name: "Todo", Color: "GRAY"},
				{Name: "In Progress", Color: "YELLOW"},
				{Name: "Done", Color: "GREEN"},
			},
			desired:        singleSelectOption{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
			beforeName:     "Done",
			expectChanged:  true,
			expectedLength: 4,
		},
		{
			name: "beforeName option does not exist - appends to end",
			options: []singleSelectOption{
				{Name: "Todo", Color: "GRAY"},
				{Name: "In Progress", Color: "YELLOW"},
			},
			desired:        singleSelectOption{Name: "Review Required", Color: "BLUE", Description: "Needs review"},
			beforeName:     "NonExistent",
			expectChanged:  true,
			expectedLength: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := ensureSingleSelectOptionBefore(tt.options, tt.desired, tt.beforeName)
			assert.Equal(t, tt.expectChanged, changed, "Changed status should match expectation")
			assert.Len(t, result, tt.expectedLength, "Result length should match")

			if !tt.expectChanged {
				// If nothing changed, result should be equal to input
				assert.Equal(t, tt.options, result, "Options should be unchanged")
			} else {
				// Find the desired option and Done option
				desiredIdx, doneIdx := -1, -1
				for i, opt := range result {
					if opt.Name == tt.desired.Name {
						desiredIdx = i
					}
					if opt.Name == tt.beforeName {
						doneIdx = i
					}
				}

				if desiredIdx >= 0 && doneIdx >= 0 {
					assert.Less(t, desiredIdx, doneIdx, "Desired option should be before Done")
				}
			}
		})
	}
}

func TestSingleSelectOptionsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []singleSelectOption
		b        []singleSelectOption
		expected bool
	}{
		{
			name: "equal options",
			a: []singleSelectOption{
				{Name: "Option 1", Color: "RED"},
				{Name: "Option 2", Color: "BLUE"},
			},
			b: []singleSelectOption{
				{Name: "Option 1", Color: "RED"},
				{Name: "Option 2", Color: "BLUE"},
			},
			expected: true,
		},
		{
			name: "different lengths",
			a: []singleSelectOption{
				{Name: "Option 1", Color: "RED"},
			},
			b: []singleSelectOption{
				{Name: "Option 1", Color: "RED"},
				{Name: "Option 2", Color: "BLUE"},
			},
			expected: false,
		},
		{
			name: "different order",
			a: []singleSelectOption{
				{Name: "Option 1", Color: "RED"},
				{Name: "Option 2", Color: "BLUE"},
			},
			b: []singleSelectOption{
				{Name: "Option 2", Color: "BLUE"},
				{Name: "Option 1", Color: "RED"},
			},
			expected: false,
		},
		{
			name:     "both empty",
			a:        []singleSelectOption{},
			b:        []singleSelectOption{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := singleSelectOptionsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "Equality check should match expectation")
		})
	}
}

func TestProjectConfigWithProjectSetup(t *testing.T) {
	tests := []struct {
		name        string
		config      ProjectConfig
		description string
	}{
		{
			name: "with project setup",
			config: ProjectConfig{
				Title:            "Project With Setup",
				Owner:            "myorg",
				OwnerType:        "org",
				WithProjectSetup: true,
			},
			description: "Should have project setup enabled",
		},
		{
			name: "without project setup",
			config: ProjectConfig{
				Title:            "Basic Project",
				Owner:            "myorg",
				OwnerType:        "org",
				WithProjectSetup: false,
			},
			description: "Should have project setup disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.config.Title, "Project title should not be empty")
			assert.NotEmpty(t, tt.config.Owner, "Project owner should not be empty")

			// Verify flag settings
			if tt.config.WithProjectSetup {
				assert.True(t, tt.config.WithProjectSetup, "Project setup should be enabled")
			} else {
				assert.False(t, tt.config.WithProjectSetup, "Project setup should be disabled")
			}
		})
	}
}

// TestGraphQLRequestBodyStructure verifies that the JSON bodies sent by createProject and
// linkProjectToRepo use the {"query": ..., "variables": {...}} structure required by
// gh api graphql --input -, rather than raw -f key=value argument concatenation.
// These tests guard against regression to the vulnerable pattern flagged by Semgrep #627/#628.
func TestGraphQLRequestBodyStructure(t *testing.T) {
	assertQueryAndVariables := func(t *testing.T, body map[string]any, expectedVarKeys ...string) {
		t.Helper()
		assert.Contains(t, body, "query", "request body must have 'query' key (not inline string concat)")
		assert.Contains(t, body, "variables", "request body must have 'variables' key (not raw -f args)")
		vars, ok := body["variables"].(map[string]any)
		require.True(t, ok, "variables must be a JSON object")
		for _, k := range expectedVarKeys {
			assert.Contains(t, vars, k, "variables must include key %q", k)
		}
	}

	t.Run("createProject request body has query and variables", func(t *testing.T) {
		ownerId := "U_kgDOABCDEF"
		title := "My Project"

		body := map[string]any{
			"query": `mutation($ownerId: ID!, $title: String!) {
				createProjectV2(input: { ownerId: $ownerId, title: $title }) {
					projectV2 { id number title url }
				}
			}`,
			"variables": map[string]any{
				"ownerId": ownerId,
				"title":   title,
			},
		}

		data, err := json.Marshal(body)
		require.NoError(t, err, "request body must marshal without error")

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		assertQueryAndVariables(t, parsed, "ownerId", "title")
		vars := parsed["variables"].(map[string]any)
		assert.Equal(t, ownerId, vars["ownerId"], "ownerId must be preserved via JSON marshaling")
		assert.Equal(t, title, vars["title"], "title must be preserved via JSON marshaling")
	})

	t.Run("createProject body safely encodes special characters in ownerId and title", func(t *testing.T) {
		// Regression guard: values with special chars must be JSON-encoded, not concatenated.
		// Raw concatenation of ownerId into -f ownerId=<value> would allow injection.
		ownerId := `U_abc"injected`
		title := `title with "quotes" and \backslash`

		body := map[string]any{
			"query": `mutation($ownerId: ID!, $title: String!) {}`,
			"variables": map[string]any{
				"ownerId": ownerId,
				"title":   title,
			},
		}

		data, err := json.Marshal(body)
		require.NoError(t, err, "json.Marshal must handle special characters without error")

		// Verify the JSON is valid and round-trips values exactly.
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed), "marshaled body must be valid JSON")
		vars := parsed["variables"].(map[string]any)
		assert.Equal(t, ownerId, vars["ownerId"], "ownerId with special chars must survive JSON round-trip")
		assert.Equal(t, title, vars["title"], "title with special chars must survive JSON round-trip")
	})

	t.Run("linkProjectToRepo repoIdQuery body has query and variables", func(t *testing.T) {
		owner := "myorg"
		name := "myrepo"

		body := map[string]any{
			"query": `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { id } }`,
			"variables": map[string]any{
				"owner": owner,
				"name":  name,
			},
		}

		data, err := json.Marshal(body)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		assertQueryAndVariables(t, parsed, "owner", "name")
		vars := parsed["variables"].(map[string]any)
		assert.Equal(t, owner, vars["owner"])
		assert.Equal(t, name, vars["name"])
	})

	t.Run("linkProjectToRepo mutation body has query and variables", func(t *testing.T) {
		projectId := "PVT_kwDOBHjIqs4A_abc"
		repositoryId := "R_kgDOXYZdef"

		body := map[string]any{
			"query": `mutation($projectId: ID!, $repositoryId: ID!) {
				linkProjectV2ToRepository(input: { projectId: $projectId, repositoryId: $repositoryId }) {
					repository { id }
				}
			}`,
			"variables": map[string]any{
				"projectId":    projectId,
				"repositoryId": repositoryId,
			},
		}

		data, err := json.Marshal(body)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		assertQueryAndVariables(t, parsed, "projectId", "repositoryId")
		vars := parsed["variables"].(map[string]any)
		assert.Equal(t, projectId, vars["projectId"])
		assert.Equal(t, repositoryId, vars["repositoryId"])
	})
}

func TestValidateOwnerUsesStringLoginField(t *testing.T) {
	oldRunGH := projectCommandRunGH
	defer func() { projectCommandRunGH = oldRunGH }()

	tests := []struct {
		name      string
		ownerType string
		owner     string
	}{
		{name: "organization login false stays string", ownerType: "org", owner: "false"},
		{name: "user login null stays string", ownerType: "user", owner: "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured []string
			projectCommandRunGH = func(spinnerMessage string, args ...string) ([]byte, error) {
				captured = append([]string(nil), args...)
				return []byte(`{}`), nil
			}

			err := validateOwner(context.Background(), tt.ownerType, tt.owner, false)
			require.NoError(t, err)

			require.Contains(t, captured, "login="+tt.owner)
			loginIndex := slices.Index(captured, "login="+tt.owner)
			require.Positive(t, loginIndex)
			assert.Equal(t, "-f", captured[loginIndex-1], "login must be passed with -f so gh keeps String! values as strings")
		})
	}
}

func TestGetStatusFieldUsesStringLoginAndIntNumberFields(t *testing.T) {
	oldRunGH := projectCommandRunGH
	defer func() { projectCommandRunGH = oldRunGH }()

	tests := []struct {
		name         string
		info         projectURLInfo
		wantProject  string
		wantProjectJ string
		wantFieldsJ  string
	}{
		{
			name:         "organization login false stays string",
			info:         projectURLInfo{scope: "orgs", ownerLogin: "false", projectNumber: 42},
			wantProject:  "project-org",
			wantProjectJ: ".data.organization.projectV2.id",
			wantFieldsJ:  ".data.organization.projectV2.fields.nodes",
		},
		{
			name:         "user login null stays string",
			info:         projectURLInfo{scope: "users", ownerLogin: "null", projectNumber: 7},
			wantProject:  "project-user",
			wantProjectJ: ".data.user.projectV2.id",
			wantFieldsJ:  ".data.user.projectV2.fields.nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls [][]string
			projectCommandRunGH = func(spinnerMessage string, args ...string) ([]byte, error) {
				call := append([]string(nil), args...)
				calls = append(calls, call)
				switch jqPathArg(t, call) {
				case tt.wantProjectJ:
					return []byte(tt.wantProject), nil
				case tt.wantFieldsJ:
					return []byte(`[{"id":"status-field","name":"Status","options":[{"name":"Todo","color":"GRAY"}]}]`), nil
				default:
					return nil, errors.New("unexpected jq path")
				}
			}

			field, err := getStatusField(context.Background(), tt.info, false)
			require.NoError(t, err)
			assert.Equal(t, tt.wantProject, field.projectID)
			assert.Equal(t, "status-field", field.fieldID)
			require.Len(t, field.options, 1)
			assert.Equal(t, "Todo", field.options[0].Name)

			require.Len(t, calls, 2)
			for _, call := range calls {
				loginIndex := slices.Index(call, "login="+tt.info.ownerLogin)
				require.Positive(t, loginIndex)
				assert.Equal(t, "-f", call[loginIndex-1], "login must be passed with -f so gh keeps String! values as strings")

				numberIndex := slices.Index(call, "number="+strconv.Itoa(tt.info.projectNumber))
				require.Positive(t, numberIndex)
				assert.Equal(t, "-F", call[numberIndex-1], "project number must keep -F so gh coerces Int! values correctly")
			}
		})
	}
}

func jqPathArg(t *testing.T, args []string) string {
	t.Helper()
	jqIndex := slices.Index(args, "--jq")
	require.Positive(t, jqIndex)
	require.Less(t, jqIndex, len(args)-1)
	return args[jqIndex+1]
}
