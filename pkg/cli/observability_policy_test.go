//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateObservabilityPolicy(t *testing.T) {
	policy := ObservabilityPolicy{
		SchemaVersion: "1.0.0",
		Rules: []ObservabilityPolicyRule{
			{
				ID:      "block-unapproved-domain",
				Action:  "fail",
				Message: "Blocked domain is not allowed",
				Match: ObservabilityPolicyMatch{
					BlockedDomains: []string{"evil.example.com"},
				},
			},
			{
				ID:      "gate-high-risk-write",
				Action:  "gate",
				Message: "High severity write-capable run requires approval",
				Match: ObservabilityPolicyMatch{
					InsightSeverities: []string{"high", "critical"},
					ActuationModes:    []string{"write_capable", "mixed"},
				},
			},
			{
				ID:      "warn-control-plane-failure",
				Action:  "warn",
				Message: "GitHub MCP failed during the run",
				Match: ObservabilityPolicyMatch{
					MCPFailureServers: []string{"github"},
				},
			},
		},
	}

	payload := ObservabilityPayload{
		Network: &ObservabilityPayloadNetwork{
			BlockedRequests: 3,
			BlockedDomains:  []string{"evil.example.com", "unknown.example.com"},
		},
		Actuation: &ObservabilityPayloadActuation{
			Mode: "write_capable",
			CreatedItems: []ObservabilityCreatedItem{
				{Type: "create_pull_request"},
			},
		},
		Tooling: &ObservabilityPayloadTooling{
			MCPFailures: []ObservabilityPolicyMCPFailure{
				{ServerName: "github"},
			},
		},
		Insights: []ObservabilityInsight{
			{Severity: "high", Title: "Network friction detected"},
		},
	}

	result := EvaluateObservabilityPolicy(policy, payload)
	require.Len(t, result.Violations, 3, "expected all three rules to match")

	assert.Equal(t, "block-unapproved-domain", result.Violations[0].RuleID)
	assert.Equal(t, "fail", result.Violations[0].Action)
	assert.Contains(t, result.Violations[0].Evidence, "blocked_domain=evil.example.com")

	assert.Equal(t, "gate-high-risk-write", result.Violations[1].RuleID)
	assert.Equal(t, "gate", result.Violations[1].Action)
	assert.Contains(t, result.Violations[1].Evidence, "insight_severity=high")
	assert.Contains(t, result.Violations[1].Evidence, "actuation_mode=write_capable")

	assert.Equal(t, "warn-control-plane-failure", result.Violations[2].RuleID)
	assert.Equal(t, "warn", result.Violations[2].Action)
	assert.Contains(t, result.Violations[2].Evidence, "mcp_failure_server=github")
}

func TestEvaluateObservabilityPolicy_NoMatch(t *testing.T) {
	policy := ObservabilityPolicy{
		SchemaVersion: "1.0.0",
		Rules: []ObservabilityPolicyRule{
			{
				ID:      "no-match",
				Action:  "fail",
				Message: "Should not trigger",
				Match: ObservabilityPolicyMatch{
					BlockedDomains: []string{"evil.example.com"},
				},
			},
		},
	}

	payload := ObservabilityPayload{
		Network: &ObservabilityPayloadNetwork{
			BlockedDomains: []string{"safe.example.com"},
		},
	}

	result := EvaluateObservabilityPolicy(policy, payload)
	assert.Empty(t, result.Violations, "unexpected violations for non-matching payload")
}

func TestObservabilityPolicySchemaParsesAndHasRules(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "observability-policy.json")
	schemaContent, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should read observability policy schema")

	var schema map[string]any
	require.NoError(t, json.Unmarshal(schemaContent, &schema), "schema should parse as JSON")

	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schema["$schema"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "root properties should exist")
	assert.Contains(t, properties, "rules")

	defs, ok := schema["$defs"].(map[string]any)
	require.True(t, ok, "schema defs should exist")
	assert.Contains(t, defs, "Rule")
	assert.Contains(t, defs, "Match")
}
