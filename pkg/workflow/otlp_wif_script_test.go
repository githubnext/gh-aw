package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExchangeOTLPWorkloadIdentityScriptInSync verifies that the compiler-embedded
// copy of exchange_otlp_workload_identity.cjs matches the source of truth under
// actions/setup/js, which is the file covered by the vitest suite.
func TestExchangeOTLPWorkloadIdentityScriptInSync(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine source file path")
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	sourcePath := filepath.Join(repoRoot, "actions", "setup", "js", "exchange_otlp_workload_identity.cjs")

	sourceBytes, err := os.ReadFile(sourcePath)
	require.NoError(t, err, "failed to read %s", sourcePath)

	assert.Equal(t, string(sourceBytes), exchangeOTLPWorkloadIdentityScript,
		"pkg/workflow/js/exchange_otlp_workload_identity.cjs and actions/setup/js/exchange_otlp_workload_identity.cjs must be identical; update both files together")
}

func TestGetExchangeOTLPWorkloadIdentityScript(t *testing.T) {
	script := getExchangeOTLPWorkloadIdentityScript()
	assert.Contains(t, script, "https://sts.googleapis.com/v1/token")
	assert.Contains(t, script, "GH_AW_OTLP_WIF_SERVICE_ACCOUNT")
	assert.Contains(t, script, "await main();")
}
