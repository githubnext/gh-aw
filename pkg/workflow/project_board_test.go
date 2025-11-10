package workflow

import (
	"strings"
	"testing"
)

func TestParseCampaignProjectConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *CampaignProjectConfig
	}{
		{
			name: "full configuration",
			input: map[string]any{
				"project": map[string]any{
					"name":         "Test Campaign",
					"view":         "board",
					"status-field": "Status",
					"agent-field":  "Agent",
					"fields": map[string]any{
						"campaign-id": "{{campaign.id}}",
						"started-at":  "{{run.started_at}}",
					},
					"insights": []any{
						"agent-velocity",
						"campaign-progress",
					},
					"github-token": "${{ secrets.GH_TOKEN }}",
				},
			},
			expected: &CampaignProjectConfig{
				Name:        "Test Campaign",
				View:        "board",
				StatusField: "Status",
				AgentField:  "Agent",
				Fields: map[string]string{
					"campaign-id": "{{campaign.id}}",
					"started-at":  "{{run.started_at}}",
				},
				Insights: []string{
					"agent-velocity",
					"campaign-progress",
				},
				GitHubToken: "${{ secrets.GH_TOKEN }}",
			},
		},
		{
			name: "minimal configuration with defaults",
			input: map[string]any{
				"project": map[string]any{
					"name": "Minimal Campaign",
				},
			},
			expected: &CampaignProjectConfig{
				Name:        "Minimal Campaign",
				View:        "board",  // default
				StatusField: "Status", // default
				AgentField:  "Agent",  // default
				Fields:      map[string]string{},
				Insights:    nil,
			},
		},
		{
			name: "missing name returns nil",
			input: map[string]any{
				"project": map[string]any{
					"view": "table",
				},
			},
			expected: nil,
		},
		{
			name:     "no project key returns nil",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			result := c.parseCampaignProjectConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, result.Name)
			}

			if result.View != tt.expected.View {
				t.Errorf("View: expected %q, got %q", tt.expected.View, result.View)
			}

			if result.StatusField != tt.expected.StatusField {
				t.Errorf("StatusField: expected %q, got %q", tt.expected.StatusField, result.StatusField)
			}

			if result.AgentField != tt.expected.AgentField {
				t.Errorf("AgentField: expected %q, got %q", tt.expected.AgentField, result.AgentField)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken: expected %q, got %q", tt.expected.GitHubToken, result.GitHubToken)
			}

			// Check fields map
			if len(result.Fields) != len(tt.expected.Fields) {
				t.Errorf("Fields length: expected %d, got %d", len(tt.expected.Fields), len(result.Fields))
			}
			for key, expectedVal := range tt.expected.Fields {
				if resultVal, ok := result.Fields[key]; !ok {
					t.Errorf("Fields: missing key %q", key)
				} else if resultVal != expectedVal {
					t.Errorf("Fields[%q]: expected %q, got %q", key, expectedVal, resultVal)
				}
			}

			// Check insights array
			if len(result.Insights) != len(tt.expected.Insights) {
				t.Errorf("Insights length: expected %d, got %d", len(tt.expected.Insights), len(result.Insights))
			}
			for i, expectedInsight := range tt.expected.Insights {
				if i >= len(result.Insights) {
					break
				}
				if result.Insights[i] != expectedInsight {
					t.Errorf("Insights[%d]: expected %q, got %q", i, expectedInsight, result.Insights[i])
				}
			}
		})
	}
}

func TestBuildCampaignProjectJob(t *testing.T) {
	c := &Compiler{}

	data := &WorkflowData{
		Name: "Test Workflow",
		CampaignProject: &CampaignProjectConfig{
			Name:        "Test Campaign Project",
			View:        "board",
			StatusField: "Status",
			AgentField:  "Agent",
			Fields: map[string]string{
				"campaign-id": "test-123",
			},
			Insights: []string{
				"agent-velocity",
			},
		},
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := c.buildCampaignProjectJob(data, "main_job")
	if err != nil {
		t.Fatalf("buildCampaignProjectJob failed: %v", err)
	}

	if job.Name != "campaign_project" {
		t.Errorf("Job name: expected 'campaign_project', got %q", job.Name)
	}

	if job.If != "always()" {
		t.Errorf("Job condition: expected 'always()', got %q", job.If)
	}

	if len(job.Needs) != 1 || job.Needs[0] != "main_job" {
		t.Errorf("Job needs: expected ['main_job'], got %v", job.Needs)
	}

	if job.TimeoutMinutes != 10 {
		t.Errorf("TimeoutMinutes: expected 10, got %d", job.TimeoutMinutes)
	}

	// Check that outputs are set
	if _, hasProjectNumber := job.Outputs["project_number"]; !hasProjectNumber {
		t.Error("Missing output: project_number")
	}
	if _, hasProjectURL := job.Outputs["project_url"]; !hasProjectURL {
		t.Error("Missing output: project_url")
	}
	if _, hasItemID := job.Outputs["item_id"]; !hasItemID {
		t.Error("Missing output: item_id")
	}

	// Check that permissions include projects
	if !strings.Contains(job.Permissions, "repository-projects") {
		t.Error("Permissions should include repository-projects")
	}
}
