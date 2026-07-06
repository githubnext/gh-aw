package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectRuntimeRequirementsCached_ReturnsDefensiveCopy(t *testing.T) {
	workflowData := &WorkflowData{
		CustomSteps: "run: node --version",
	}

	first := detectRuntimeRequirementsCached(workflowData)
	require.NotEmpty(t, first)
	require.True(t, workflowData.CachedRuntimeRequirementsSet)

	first[0].Version = "mutated-version"

	second := detectRuntimeRequirementsCached(workflowData)
	require.NotEmpty(t, second)
	require.NotEqual(t, "mutated-version", second[0].Version)
}
