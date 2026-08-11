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
	assert.True(t, strings.HasSuffix(result, "\n"))
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

func TestPreserveRunnerGuardStepSuppressionsInlineDirectiveOnUnnamedStep(t *testing.T) {
	frontmatter := `steps:
  - uses: actions/checkout@v5 # runner-guard:ignore RGS-007 -- trusted action
`
	generated := `jobs:
  agent:
    steps:
      - uses: actions/checkout@v5
`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.Contains(t, result, "      # runner-guard:ignore RGS-007 -- trusted action\n      - uses: actions/checkout@v5")
	assert.Equal(t, 1, strings.Count(result, runnerGuardIgnorePrefix))
}

func TestPreserveRunnerGuardStepSuppressionsStandaloneDirectiveOnUnnamedStep(t *testing.T) {
	frontmatter := `steps:
  # runner-guard:ignore RGS-012 -- public read-only request
  - run: curl https://example.com/index.json
`
	generated := `      - run: curl https://example.com/index.json`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.Contains(t, result, "      # runner-guard:ignore RGS-012 -- public read-only request\n      - run: curl https://example.com/index.json")
}

func TestPreserveRunnerGuardStepSuppressionsIgnoresAmbiguousDirectives(t *testing.T) {
	frontmatter := `steps:
  # runner-guard:ignore RGS-012 -- first justification
  - name: Fetch index
    run: curl https://example.com/one.json
  # runner-guard:ignore RGS-012 -- second justification
  - name: Fetch index
    run: curl https://example.com/two.json
`
	generated := `      - name: Fetch index
        run: curl https://example.com/one.json`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.NotContains(t, result, runnerGuardIgnorePrefix)
}

func TestPreserveRunnerGuardStepSuppressionsIgnoresGeneratedScriptPayloads(t *testing.T) {
	frontmatter := `steps:
  # runner-guard:ignore RGS-012 -- public read-only request
  - name: Fetch public index
    run: curl https://example.com/index.json
`
	generated := `jobs:
  agent:
    steps:
      - name: Write fixture
        run: |
          cat <<'YAML'
          # runner-guard:ignore RGS-012 -- misplaced script comment
          - name: Fetch public index
          YAML
`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.Equal(t, generated, result)
}

func TestPreserveRunnerGuardStepSuppressionsIgnoresFrontmatterScriptStepNames(t *testing.T) {
	frontmatter := `steps:
  - name: Write fixture
    run: |
      # runner-guard:ignore RGS-012 -- misplaced script comment
      - name: Fetch public index
`
	generated := `      - name: Fetch public index`

	result := preserveRunnerGuardStepSuppressions(generated, frontmatter)

	assert.Equal(t, generated, result)
}
