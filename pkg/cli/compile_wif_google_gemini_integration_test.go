//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileGeminiWIFGoogle verifies that a workflow using Google
// Workload Identity Federation (engine.auth.type=github-oidc with
// provider=google) compiles successfully without requiring GEMINI_API_KEY,
// and that the WIF fields are correctly emitted as env vars in the lock file.
func TestCompileGeminiWIFGoogle(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the canonical Google WIF workflow fixture into the test's .github/workflows dir
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-gemini-wif-google.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-gemini-wif-google.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "Failed to read source workflow file %s", srcPath)
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "Failed to write workflow to test dir")

	// Compile the workflow - it must succeed (exit 0) without GEMINI_API_KEY.
	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Gemini WIF Google workflow must compile without error:\n%s", string(output))

	// Verify the lock file was created and contains the expected WIF env vars.
	lockFilePath := filepath.Join(setup.workflowsDir, "test-gemini-wif-google.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Expected lock file %s to be created", lockFilePath)
	lockStr := string(lockContent)

	// All WIF fields from the fixture must be emitted as env vars in the compiled lock
	// file. Checking for "KEY: value" pairs ensures both the key and the value round-trip
	// correctly through the schema → parser → compiler pipeline.
	assert.Contains(t, lockStr, "AWF_AUTH_PROVIDER: gcp", "lock file should contain AWF_AUTH_PROVIDER=gcp")
	assert.Contains(t, lockStr, "AWF_AUTH_GCP_WORKLOAD_IDENTITY_PROVIDER: projects/123456789/locations/global/workloadIdentityPools/github-pool/providers/github",
		"lock file should contain AWF_AUTH_GCP_WORKLOAD_IDENTITY_PROVIDER")
	assert.Contains(t, lockStr, "AWF_AUTH_GCP_SERVICE_ACCOUNT: my-sa@my-project.iam.gserviceaccount.com",
		"lock file should contain AWF_AUTH_GCP_SERVICE_ACCOUNT")

	// Vertex AI backend env var and project/location must be set
	assert.Contains(t, lockStr, "GOOGLE_GENAI_USE_VERTEXAI: true", "lock file should set GOOGLE_GENAI_USE_VERTEXAI=true")
	assert.Contains(t, lockStr, "GOOGLE_CLOUD_PROJECT: my-project", "lock file should set GOOGLE_CLOUD_PROJECT")
	assert.Contains(t, lockStr, "GOOGLE_CLOUD_LOCATION: us-central1", "lock file should set GOOGLE_CLOUD_LOCATION")

	// GEMINI_API_KEY must NOT appear in the lock file when WIF is configured
	assert.NotContains(t, lockStr, "GEMINI_API_KEY", "lock file must not contain GEMINI_API_KEY when Google WIF is configured")

	t.Logf("Google WIF workflow compiled successfully to %s", lockFilePath)
}
