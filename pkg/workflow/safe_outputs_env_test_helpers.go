package workflow

import (
	"strings"
	"testing"
)

// assertEnvVarsInSteps checks that all expected environment variables are present in the job steps.
// This is a helper function to reduce duplication in safe outputs env tests.
func assertEnvVarsInSteps(t *testing.T, steps []string, expectedEnvVars []string) {
	t.Helper()
	stepsStr := strings.Join(steps, "")
	for _, expectedEnvVar := range expectedEnvVars {
		if !strings.Contains(stepsStr, expectedEnvVar) {
			t.Errorf("Expected env var %q not found in job YAML", expectedEnvVar)
		}
	}
}
