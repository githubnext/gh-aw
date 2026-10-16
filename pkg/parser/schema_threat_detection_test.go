//go:build !integration

package parser

import "testing"

func TestMainWorkflowSchemaThreatDetectionCACert(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		value any
		valid bool
	}{
		{name: "certificate path", value: "/tmp/detection-ca.pem", valid: true},
		{name: "empty path", value: "", valid: true},
		{name: "boolean rejected", value: true},
		{name: "number rejected", value: 42},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on":     "push",
				"engine": "claude",
				"safe-outputs": map[string]any{
					"create-issue":     nil,
					"threat-detection": map[string]any{"ca-cert": tt.value},
				},
			}, "/tmp/gh-aw/threat-detection-ca-cert.md")
			if (err == nil) != tt.valid {
				t.Errorf("validation error = %v, want valid=%v", err, tt.valid)
			}
		})
	}
}

func TestMainWorkflowSchemaThreatDetectionArtifactBaseURL(t *testing.T) {
	t.Parallel()

	frontmatter := func(baseURL string) map[string]any {
		return map[string]any{
			"on":     "push",
			"engine": "copilot",
			"safe-outputs": map[string]any{
				"create-issue": nil,
				"threat-detection": map[string]any{
					"artifact-base-url": baseURL,
				},
			},
		}
	}

	if err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(
		frontmatter("https://artifacts.example.com/threat-detect/releases/download"),
		"/tmp/gh-aw/threat-detection-artifact-mirror-valid.md",
	); err != nil {
		t.Fatalf("expected HTTPS artifact mirror to validate: %v", err)
	}

	for _, invalid := range []string{
		"http://artifacts.example.com/threat-detect",
		"file:///tmp/threat-detect",
		"artifacts.example.com/threat-detect",
	} {
		if err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(
			frontmatter(invalid),
			"/tmp/gh-aw/threat-detection-artifact-mirror-invalid.md",
		); err == nil {
			t.Errorf("expected unsafe artifact mirror %q to fail schema validation", invalid)
		}
	}
}
