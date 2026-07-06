//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectRuntimeRequirementsCached_NilWorkflowData(t *testing.T) {
	require.Nil(t, detectRuntimeRequirementsCached(nil), "nil workflow data should produce no runtime requirements")
}

func TestDetectRuntimeRequirementsCached_EmptyRequirements(t *testing.T) {
	workflowData := &WorkflowData{}

	result := detectRuntimeRequirementsCached(workflowData)

	require.Empty(t, result, "workflow without runtime inputs should produce no runtime requirements")
	require.True(t, workflowData.CachedRuntimeRequirementsSet, "empty detection result should still mark the cache as populated")
	require.Nil(t, workflowData.CachedRuntimeRequirements, "empty detection result should be cached as nil requirements")
}

func TestDetectRuntimeRequirementsCached_CacheMissReturnsIndependentCopy(t *testing.T) {
	workflowData := &WorkflowData{
		CustomSteps: "run: node --version",
	}

	first := detectRuntimeRequirementsCached(workflowData)
	require.NotEmpty(t, first, "cache miss should still return detected runtime requirements")
	require.True(t, workflowData.CachedRuntimeRequirementsSet, "cache miss should populate the runtime requirement cache")

	first[0].Version = "mutated-version"
	require.NotEqual(t, "mutated-version", workflowData.CachedRuntimeRequirements[0].Version, "mutating the first cache-miss result must not corrupt the cached requirements")

	second := detectRuntimeRequirementsCached(workflowData)
	require.NotEmpty(t, second, "subsequent cache hits should still return runtime requirements")
	require.NotEqual(t, "mutated-version", second[0].Version, "cache-hit result should not observe mutation from the cache-miss return value")
}

func TestDetectRuntimeRequirementsCached_CacheHitReturnsDeepCopy(t *testing.T) {
	workflowData := &WorkflowData{
		CachedRuntimeRequirements: []RuntimeRequirement{
			{
				Runtime: findRuntimeByID("node"),
				Version: "24",
				ExtraFields: map[string]any{
					"cache": "npm",
				},
			},
		},
		CachedRuntimeRequirementsSet: true,
	}

	first := detectRuntimeRequirementsCached(workflowData)
	require.Len(t, first, 1, "cache hit should return the cached runtime requirements")

	first[0].Version = "mutated-version"
	first[0].ExtraFields["cache"] = "mutated-cache"

	require.Equal(t, "24", workflowData.CachedRuntimeRequirements[0].Version, "mutating a cache-hit result must not change the cached version")
	require.Equal(t, "npm", workflowData.CachedRuntimeRequirements[0].ExtraFields["cache"], "mutating a cache-hit result must not change cached extra fields")

	second := detectRuntimeRequirementsCached(workflowData)
	require.Len(t, second, 1, "subsequent cache hits should continue to return one cached runtime requirement")
	require.Equal(t, "24", second[0].Version, "subsequent cache hits should return the original cached version")
	require.Equal(t, "npm", second[0].ExtraFields["cache"], "subsequent cache hits should return a deep-cloned extra-fields map")
}
