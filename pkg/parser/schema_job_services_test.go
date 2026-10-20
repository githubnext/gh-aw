//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomJobServiceSchema(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		service any
		valid   bool
	}{
		{
			name: "admission service",
			service: map[string]any{
				"image":   "wiremock/wiremock:3.13.1",
				"ports":   []any{"8080:8080"},
				"options": "--health-cmd curl",
				"env":     map[string]any{"MODE": "smoke"},
			},
			valid: true,
		},
		{name: "missing image", service: map[string]any{"ports": []any{"8080:8080"}}},
		{name: "unknown property", service: map[string]any{"image": "redis:7", "unknown": true}},
		{name: "invalid ports", service: map[string]any{"image": "redis:7", "ports": true}},
		{name: "unsupported shorthand", service: "redis:7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": "workflow_dispatch",
				"jobs": map[string]any{
					"probe": map[string]any{
						"runs-on":  "ubuntu-latest",
						"services": map[string]any{"admission": tc.service},
						"steps":    []any{map[string]any{"run": "echo smoke"}},
					},
				},
			}, "smoke-admission.md")
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
