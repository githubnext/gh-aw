package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreserveRunnerGuardStepSuppressions(t *testing.T) {
	frontmatter := `steps:
  # runner-guard:ignore RGS-012 -- public read-only request
  - name: Fetch public index
    run: curl https://example.com/index.json
`
	generated := `jobs:
  agent:
    steps:
      - name: Fetch public index
        run: curl https://example.com/index.json
`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.Contains(t, result, "      # runner-guard:ignore RGS-012 -- public read-only request\n      - name: Fetch public index")
	assert.Equal(t, 1, strings.Count(result, runnerGuardIgnorePrefix))
}

func TestPreserveRunnerGuardStepSuppressionsIgnoresScriptComments(t *testing.T) {
	frontmatter := `steps:
  - name: Fetch public index
    run: |
      # runner-guard:ignore RGS-012 -- misplaced script comment
      curl https://example.com/index.json
`

	result := preserveRunnerGuardStepSuppressions("      - name: Fetch public index", frontmatter)

	assert.NotContains(t, result, runnerGuardIgnorePrefix)
}

func TestPreserveRunnerGuardStepSuppressionsIgnoresDuplicateNames(t *testing.T) {
	frontmatter := `steps:
  # runner-guard:ignore RGS-012 -- public read-only request
  - name: Fetch index
    run: curl https://example.com/one.json
  - name: Fetch index
    run: curl https://example.com/two.json
`
	generated := `      - name: Fetch index
        run: curl https://example.com/one.json
      - name: Fetch index
        run: curl https://example.com/two.json`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.NotContains(t, result, runnerGuardIgnorePrefix)
}
