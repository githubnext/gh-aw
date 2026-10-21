package lint

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTLPSecretLeakRuleChecksBothAuthoredAttributeMaps(t *testing.T) {
	document, err := aw.Parse(`---
observability:
  otlp:
    attributes:
      account: "${{ secrets.API_TOKEN }}"
    resource-attributes:
      tenant: "${{ github.token }}"
---
Public prompt.
`)
	require.NoError(t, err)

	runner, err := Default().Select("otlp-secret-leak")
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), Input{Document: document})
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, "observability.otlp.attributes.account", diagnostics[0].Path)
	assert.Equal(t, "observability.otlp.resource-attributes.tenant", diagnostics[1].Path)
	for _, diagnostic := range diagnostics {
		assert.Equal(t, "CTR-017", diagnostic.ThreatRuleID)
		assert.NotContains(t, diagnostic.Message, "API_TOKEN")
		assert.NotContains(t, diagnostic.Message, "github.token")
		assert.Contains(t, diagnostic.Message, "OTLP backend")
	}
}

func TestOTLPSecretLeakRuleFollowsEnvironmentBindings(t *testing.T) {
	workflow, err := actions.Parse([]byte(`
env:
  PRIVATE_VALUE: ${{ secrets.API_TOKEN }}
jobs:
  export:
    runs-on: ubuntu-latest
    env:
      GH_AW_OTLP_ATTRIBUTES: '{"account":"${{ env.PRIVATE_VALUE }}"}'
    steps:
      - env:
          OTEL_RESOURCE_ATTRIBUTES: 'tenant=${{ env.PRIVATE_VALUE }}'
        run: echo ready
`))
	require.NoError(t, err)

	runner, err := Default().Select("otlp-secret-leak")
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), Input{Workflow: workflow})
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, "jobs.export.env.GH_AW_OTLP_ATTRIBUTES", diagnostics[0].Path)
	assert.Equal(t, "jobs.export.steps[0].env.OTEL_RESOURCE_ATTRIBUTES", diagnostics[1].Path)
	for _, diagnostic := range diagnostics {
		assert.NotContains(t, diagnostic.Message, "API_TOKEN")
	}
}

func TestDefaultRulesReportOTLPSecretLeakOnce(t *testing.T) {
	workflow, err := actions.Parse([]byte(`
env:
  GH_AW_OTLP_ATTRIBUTES: '{"account":"${{ secrets.API_TOKEN }}"}'
jobs:
  export:
    runs-on: ubuntu-latest
    steps:
      - run: echo ready
`))
	require.NoError(t, err)

	diagnostics, err := Default().Run(context.Background(), Input{Workflow: workflow})
	require.NoError(t, err)
	var otlpFindings, genericOTLPFindings int
	for _, diagnostic := range diagnostics {
		switch diagnostic.RuleID {
		case "otlp-secret-leak":
			otlpFindings++
		case "secret-leak":
			if diagnostic.Path == "env.GH_AW_OTLP_ATTRIBUTES" {
				genericOTLPFindings++
			}
		}
	}
	assert.Equal(t, 1, otlpFindings)
	assert.Zero(t, genericOTLPFindings)
}

func TestOTLPSecretLeakRuleIgnoresQuotedMentionsAndPublicContexts(t *testing.T) {
	workflow, err := actions.Parse([]byte(`
env:
  GH_AW_OTLP_ATTRIBUTES: '{"note":"${{ ''secrets.API_TOKEN'' }}","repository":"${{ github.repository }}"}'
  OTEL_RESOURCE_ATTRIBUTES: 'service.name=${{ vars.SERVICE_NAME }}'
jobs:
  export:
    runs-on: ubuntu-latest
    steps:
      - run: echo ready
`))
	require.NoError(t, err)

	runner, err := Default().Select("otlp-secret-leak")
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), Input{Workflow: workflow})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}
