package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseViewSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		expected    ProjectViewSpec
		shouldError bool
	}{
		{
			name: "valid board view",
			spec: "Progress:board",
			expected: ProjectViewSpec{
				Name:   "Progress",
				Layout: "board",
			},
			shouldError: false,
		},
		{
			name: "valid table view with filter",
			spec: "All Items:table:is:open",
			expected: ProjectViewSpec{
				Name:   "All Items",
				Layout: "table",
				Filter: "is:open",
			},
			shouldError: false,
		},
		{
			name: "valid roadmap view",
			spec: "Timeline:roadmap",
			expected: ProjectViewSpec{
				Name:   "Timeline",
				Layout: "roadmap",
			},
			shouldError: false,
		},
		{
			name:        "missing layout",
			spec:        "Progress",
			shouldError: true,
		},
		{
			name:        "empty name",
			spec:        ":board",
			shouldError: true,
		},
		{
			name:        "invalid layout",
			spec:        "Progress:invalid",
			shouldError: true,
		},
		{
			name: "view with complex filter",
			spec: "Active:board:is:issue is:pr is:open",
			expected: ProjectViewSpec{
				Name:   "Active",
				Layout: "board",
				Filter: "is:issue is:pr is:open",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseViewSpec(tt.spec)

			if tt.shouldError {
				assert.Error(t, err, "Expected error for spec: %s", tt.spec)
			} else {
				require.NoError(t, err, "Unexpected error for spec: %s", tt.spec)
				assert.Equal(t, tt.expected.Name, result.Name, "Name mismatch")
				assert.Equal(t, tt.expected.Layout, result.Layout, "Layout mismatch")
				assert.Equal(t, tt.expected.Filter, result.Filter, "Filter mismatch")
			}
		})
	}
}

func TestParseFieldSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		expected    ProjectFieldSpec
		shouldError bool
	}{
		{
			name: "text field",
			spec: "Campaign Id:TEXT",
			expected: ProjectFieldSpec{
				Name:     "Campaign Id",
				DataType: "TEXT",
			},
			shouldError: false,
		},
		{
			name: "date field",
			spec: "Start Date:DATE",
			expected: ProjectFieldSpec{
				Name:     "Start Date",
				DataType: "DATE",
			},
			shouldError: false,
		},
		{
			name: "single select field with options",
			spec: "Priority:SINGLE_SELECT:High,Medium,Low",
			expected: ProjectFieldSpec{
				Name:     "Priority",
				DataType: "SINGLE_SELECT",
				Options:  []string{"High", "Medium", "Low"},
			},
			shouldError: false,
		},
		{
			name: "single select field without options",
			spec: "Status:SINGLE_SELECT",
			expected: ProjectFieldSpec{
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options:  nil,
			},
			shouldError: false,
		},
		{
			name: "number field",
			spec: "Score:NUMBER",
			expected: ProjectFieldSpec{
				Name:     "Score",
				DataType: "NUMBER",
			},
			shouldError: false,
		},
		{
			name: "lowercase type gets uppercased",
			spec: "Field:text",
			expected: ProjectFieldSpec{
				Name:     "Field",
				DataType: "TEXT",
			},
			shouldError: false,
		},
		{
			name:        "missing type",
			spec:        "Field",
			shouldError: true,
		},
		{
			name:        "empty name",
			spec:        ":TEXT",
			shouldError: true,
		},
		{
			name:        "invalid type",
			spec:        "Field:INVALID",
			shouldError: true,
		},
		{
			name: "single select with spaces in options",
			spec: "Priority:SINGLE_SELECT: High , Medium , Low ",
			expected: ProjectFieldSpec{
				Name:     "Priority",
				DataType: "SINGLE_SELECT",
				Options:  []string{"High", "Medium", "Low"},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFieldSpec(tt.spec)

			if tt.shouldError {
				assert.Error(t, err, "Expected error for spec: %s", tt.spec)
			} else {
				require.NoError(t, err, "Unexpected error for spec: %s", tt.spec)
				assert.Equal(t, tt.expected.Name, result.Name, "Name mismatch")
				assert.Equal(t, tt.expected.DataType, result.DataType, "DataType mismatch")
				assert.Equal(t, tt.expected.Options, result.Options, "Options mismatch")
			}
		})
	}
}

func TestNewCampaignCreateProjectCommand(t *testing.T) {
	cmd := NewCampaignCreateProjectCommand()

	require.NotNil(t, cmd, "Command should be created")
	assert.Equal(t, "create-project", cmd.Use, "Command name should be create-project")
	assert.Contains(t, cmd.Short, "GitHub Project", "Short description should mention GitHub Project")
	assert.NotEmpty(t, cmd.Long, "Long description should not be empty")

	// Check required flags exist
	ownerFlag := cmd.Flags().Lookup("owner")
	require.NotNil(t, ownerFlag, "owner flag should exist")

	titleFlag := cmd.Flags().Lookup("title")
	require.NotNil(t, titleFlag, "title flag should exist")

	viewFlag := cmd.Flags().Lookup("view")
	require.NotNil(t, viewFlag, "view flag should exist")

	fieldFlag := cmd.Flags().Lookup("field")
	require.NotNil(t, fieldFlag, "field flag should exist")

	orgFlag := cmd.Flags().Lookup("org")
	require.NotNil(t, orgFlag, "org flag should exist")
}

func TestCreateProjectConfig(t *testing.T) {
	config := CreateProjectConfig{
		Owner:     "myorg",
		OwnerType: "org",
		Title:     "Test Project",
		Views: []ProjectViewSpec{
			{Name: "Board", Layout: "board"},
		},
		Fields: []ProjectFieldSpec{
			{Name: "Priority", DataType: "SINGLE_SELECT", Options: []string{"High", "Low"}},
		},
	}

	assert.Equal(t, "myorg", config.Owner, "Owner should be set")
	assert.Equal(t, "org", config.OwnerType, "OwnerType should be set")
	assert.Equal(t, "Test Project", config.Title, "Title should be set")
	assert.Len(t, config.Views, 1, "Should have one view")
	assert.Len(t, config.Fields, 1, "Should have one field")
}
