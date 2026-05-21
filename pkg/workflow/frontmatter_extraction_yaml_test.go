//go:build !integration

package workflow

import "testing"

func TestIsGitHubAppNestedField(t *testing.T) {
	t.Run("supports ignore-if-missing field", func(t *testing.T) {
		if !isGitHubAppNestedField("ignore-if-missing: true") {
			t.Fatal("expected ignore-if-missing to be treated as on.github-app nested field")
		}
	})
}
