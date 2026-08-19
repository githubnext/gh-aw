//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeJobRunnerCodemod(t *testing.T) {
	codemod := getSafeJobRunnerCodemod()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "safe-job-runner-to-runs-on", codemod.ID)
		assert.Equal(t, "Rename safe-outputs.jobs runner to runs-on", codemod.Name)
		assert.Equal(t, "Renames deprecated safe-outputs.jobs.<job>.runner fields to runs-on.", codemod.Description)
		assert.Equal(t, "1.5.0", codemod.IntroducedIn)
		require.NotNil(t, codemod.Apply)
	})

	tests := []struct {
		name        string
		content     string
		want        string
		wantApplied bool
	}{
		{
			name: "renames scalar runner",
			content: `---
safe-outputs:
  jobs:
    notify:
      runner: ubuntu-latest
      steps:
        - run: echo hi
---`,
			want: `---
safe-outputs:
  jobs:
    notify:
      runs-on: ubuntu-latest
      steps:
        - run: echo hi
---`,
			wantApplied: true,
		},
		{
			name: "preserves runner group block",
			content: `---
safe-outputs:
  jobs:
    notify:
      runner: # runner group
        group: larger-runners
        labels: [linux]
---`,
			want: `---
safe-outputs:
  jobs:
    notify:
      runs-on: # runner group
        group: larger-runners
        labels: [linux]
---`,
			wantApplied: true,
		},
		{
			name: "skips job with canonical field",
			content: `---
safe-outputs:
  jobs:
    notify:
      runner: old-runner
      runs-on: ubuntu-latest
---
`,
			want: `---
safe-outputs:
  jobs:
    notify:
      runner: old-runner
      runs-on: ubuntu-latest
---
`,
			wantApplied: false,
		},
		{
			name: "ignores runner outside safe jobs",
			content: `---
runner: top-level
safe-outputs:
  create-issue: {}
jobs:
  build:
    runner: custom
---
`,
			want: `---
runner: top-level
safe-outputs:
  create-issue: {}
jobs:
  build:
    runner: custom
---
`,
			wantApplied: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applied, err := codemod.Apply(tt.content, map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantApplied, applied)
			assert.Equal(t, tt.want, result)
		})
	}
}
