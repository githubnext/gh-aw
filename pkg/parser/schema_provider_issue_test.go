package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderIssueSchema(t *testing.T) {
	for _, alias := range []string{"jira-issue", "linear-issue", "azure-devops-issue"} {
		for name, test := range map[string]struct {
			config any
			valid  bool
		}{
			"empty config":  {map[string]any{}, true},
			"activities":    {map[string]any{"types": []any{"opened", "closed", "labeled"}}, true},
			"filters":       {map[string]any{"filters": map[string]any{"project": "PROJ"}}, true},
			"null":          {nil, false},
			"scalar":        {true, false},
			"unknown key":   {map[string]any{"typo": true}, false},
			"empty types":   {map[string]any{"types": []any{}}, false},
			"native action": {map[string]any{"types": []any{"created"}}, false},
			"misspelling":   {map[string]any{"types": []any{"labelled"}}, false},
			"GitHub repos":  {map[string]any{"repos": []any{"owner/repo"}}, false},
		} {
			t.Run(alias+"/"+name, func(t *testing.T) {
				config := test.config
				if alias == "azure-devops-issue" && name == "activities" {
					config = map[string]any{"types": []any{"opened", "closed", "labeled"}, "closed-states": []any{"Done"}}
				}
				err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
					"on": map[string]any{"webhook-gateway": true, alias: config},
				}, "provider.md")
				if test.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}

func TestAzureIssueClosedStatesSchema(t *testing.T) {
	for name, states := range map[string]any{
		"null": nil, "empty": []any{}, "blank": " ", "number": 123, "mixed": []any{"Done", false},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{
				"webhook-gateway": true,
				"azure-devops-issue": map[string]any{
					"types": []any{"closed"}, "closed-states": states,
				},
			}}, "azure.md")
			require.Error(t, err)
		})
	}
	for _, alias := range []string{"jira-issue", "linear-issue"} {
		err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{
			"webhook-gateway": true, alias: map[string]any{"closed-states": []any{"Done"}},
		}}, "provider.md")
		require.Error(t, err)
	}
	for _, activity := range []string{"closed", "reopened"} {
		err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{
			"webhook-gateway":    true,
			"azure-devops-issue": map[string]any{"types": []any{activity}},
		}}, "azure.md")
		require.ErrorContains(t, err, "closed-states")
	}
}
