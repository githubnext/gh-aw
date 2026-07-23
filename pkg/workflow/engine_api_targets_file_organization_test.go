//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAWFHelpersDoesNotContainEngineAPITargetHelpers(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("awf_helpers.go"))
	require.NoError(t, err)

	forbiddenFunctionSignatures := []string{
		"func extractAPITargetHost(",
		"func extractAPIBasePath(",
		"func extractAPITargetAuthHeader(",
		"func GetCopilotAPITarget(",
		"func extractLiteralEngineEnvHost(",
		"func GetCopilotAllowlistTargets(",
		"func GetAntigravityAPITarget(",
		"func GetGeminiAPITarget(",
		"func getEngineAPIHosts(",
		"const DefaultAntigravityAPITarget",
		"const DefaultGeminiAPITarget",
	}

	for _, signature := range forbiddenFunctionSignatures {
		require.NotContains(t, string(content), signature, "awf_helpers.go must not define %s", signature)
	}
}
