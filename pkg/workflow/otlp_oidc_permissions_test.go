//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOTLPOIDCAudienceAllJobsHaveIDTokenWrite is a regression test for the bug where
// the activation, conclusion, and safe_outputs jobs were missing the id-token: write
// permission when observability.otlp.github-app was configured with audience-only
// (credential-less OIDC mode, no app-id/private-key). Each of these jobs receives the
// "Mint OTLP OIDC token" setup step (core.getIDToken) but previously lacked the
// permission required to call that API, causing:
//
//	"Unable to get ACTIONS_ID_TOKEN_REQUEST_URL env variable"
//
// The detection and evals jobs already had the fix; this test locks the invariant for
// the three that were missing it.
func TestOTLPOIDCAudienceAllJobsHaveIDTokenWrite(t *testing.T) {
	tmpDir := testutil.TempDir(t, "otlp-oidc-perms-all-jobs")
	testFile := filepath.Join(tmpDir, "otlp-oidc-audience.md")

	// Audience-only (no app-id/private-key) triggers the credential-less OIDC path.
	// safe-outputs is required to generate the conclusion and safe_outputs jobs.
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  id-token: write
observability:
  otlp:
    endpoint: ${{ secrets.OTEL_ENDPOINT }}
    github-app:
      audience: https://otel.example.com
safe-outputs:
  add-comment:
engine: copilot
---

# OTLP OIDC audience-only permissions regression test
`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(testFile), "workflow should compile successfully")

	lockBytes, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")
	lockContent := string(lockBytes)

	jobs := []string{
		string(constants.ActivationJobName),
		"conclusion",
		"safe_outputs",
	}
	for _, jobName := range jobs {
		t.Run(jobName, func(t *testing.T) {
			section := extractJobSection(lockContent, jobName)
			require.NotEmpty(t, section, "job %q should be present in the compiled output", jobName)

			// Verify the OTLP OIDC mint step is present within this job's section so that
			// the id-token: write permission is genuinely required for this specific job.
			assert.Contains(t, section, "id: mint-otlp-oidc-token",
				"job %q should contain the OTLP OIDC mint step (id: mint-otlp-oidc-token)", jobName)

			// The core assertion: every job that receives the mint step must declare id-token: write.
			assert.Contains(t, section, "id-token: write",
				"job %q must have id-token: write when OTLP OIDC audience-only auth is configured", jobName)
		})
	}
}
