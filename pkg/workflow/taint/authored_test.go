package taint

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthoredAWExecutionScopes(t *testing.T) {
	for _, tt := range []struct {
		name, fields, prompt string
		paths                []string
	}{
		{"root defaults", "defaults:\n  run:\n    shell: bash\nenv:\n  TOKEN: '${{ secrets.TOKEN }}'\nsteps:\n  - run: echo \"$TOKEN\"", "", []string{"steps[0].run"}},
		{"pre-step prompt env", "steps:\n  - shell: bash\n    run: echo 'TOKEN=${{ secrets.TOKEN }}' >> \"$GITHUB_ENV\"", "${{ env.TOKEN }}", []string{"markdown"}},
		{"post-step env is too late for prompt", "post-steps:\n  - shell: bash\n    run: echo 'TOKEN=${{ secrets.TOKEN }}' >> \"$GITHUB_ENV\"", "${{ env.TOKEN }}", nil},
		{"job scope isolation", "jobs:\n  first:\n    env:\n      TOKEN: '${{ secrets.TOKEN }}'\n    steps:\n      - shell: bash\n        run: echo \"$TOKEN\"\n  second:\n    steps:\n      - shell: bash\n        run: echo \"$TOKEN\"", "", []string{"jobs.first.steps[0].run"}},
		{"display and job output", "name: '${{ secrets.TOKEN }}'\njobs:\n  first:\n    outputs:\n      token: '${{ secrets.TOKEN }}'", "", []string{"name", "jobs.first.outputs.token"}},
		{"reusable workflow input", "jobs:\n  reuse:\n    uses: owner/repo/.github/workflows/test.yml@main\n    with:\n      value: '${{ secrets.TOKEN }}'\n    secrets:\n      TOKEN: '${{ secrets.TOKEN }}'", "", []string{"jobs.reuse.with.value"}},
		{"unknown shell", "steps:\n  - run: echo '${{ secrets.TOKEN }}'", "", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			document, err := aw.Parse("---\n" + tt.fields + "\n---\n" + tt.prompt)
			require.NoError(t, err)
			result, err := AnalyzeAuthoredDocument(context.Background(), document, SecretLeakRule())
			require.NoError(t, err)
			var paths []string
			for _, finding := range result.Findings {
				paths = append(paths, finding.Site.Path)
			}
			assert.ElementsMatch(t, tt.paths, paths)
			assert.NotEmpty(t, result.Gaps, "authored ASTs must not imply generated execution coverage")
		})
	}
}
