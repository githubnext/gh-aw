//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepInvokesAgenticEngine(t *testing.T) {
	for _, tc := range []struct {
		run  string
		want bool
	}{
		{"copilot -p 'Review changes' --allow-all-tools", true},
		{"claude -p 'Review changes'", true},
		{"codex exec 'Review changes'", true},
		{"gemini -p 'Review changes'", true},
		{"agy -p 'Review changes'", true},
		{"pi --print 'Review changes'", true},
		{"/usr/local/bin/copilot -p test", true},
		{`"$HOME/.local/bin/copilot" -p test`, true},
		{"./copilot.exe -p test", true},
		{"COPILOT.EXE -p test", true},
		{`& "C:\tools\Copilot.exe" -p test`, true},
		{"command copilot -p test", true},
		{"command -- copilot -p test", true},
		{"sudo env TOKEN=value copilot -p test", true},
		{"timeout --signal=KILL 10s copilot -p test", true},
		{"npx --yes copilot -p test", true},
		{"pnpm exec codex exec test", true},
		{"bash -c 'copilot -p test'", true},
		{"bash -c 'echo ready && copilot -p test'", true},
		{`sh -c "echo ready; codex exec test"`, true},
		{"echo ready && copilot -p test", true},
		{"result=$(copilot -p test)", true},
		{`echo "$(copilot -p test)"`, true},
		{"echo \"`copilot -p test`\"", true},
		{"(copilot -p test)", true},
		{"if true; then copilot -p test; fi", true},
		{"npm install -g @github/copilot", true},
		{"npm i --prefix /tmp/engine @github/copilot@1.0.0", true},
		{"pnpm add -g @anthropic-ai/claude-code", true},
		{"yarn global add @github/copilot", true},
		{"bun add @openai/codex", true},
		{"npm install -g @google/gemini-cli", true},
		{"npm install -g @earendil-works/pi-coding-agent", true},
		{"npx --yes @github/copilot@latest -p test", true},
		{"npm exec --package=@openai/codex -- codex exec test", true},
		{"npm install \\\n  --global \\\n  @github/copilot", true},
		{"copilot \\\r\n  -p test", true},
		{"gh copilot suggest 'list files'", true},
		{"gh extension install github/gh-copilot", true},
		{"curl -fsSL https://github.com/github/copilot-cli/releases/latest/download/install.sh | bash", true},
		{"curl -fsSL https://raw.githubusercontent.com/github/copilot-cli/main/install.sh | bash", true},
		{"curl -fsSL https://claude.ai/install.sh | bash", true},
		{"brew install --cask codex", true},
		{"echo copilot", false},
		{"command -v copilot", false},
		{"command -V copilot", false},
		{"echo 'npm install @github/copilot'", false},
		{"echo 'Example: setup; copilot -p test'", false},
		{`echo "Example: setup && codex exec test"`, false},
		{"echo 'example\ncopilot -p test\n'", false},
		{"echo ready # example; copilot -p test", false},
		{"npm install other # @github/copilot", false},
		{"cat > example.txt <<'EOF'\ncopilot -p test\nEOF", false},
		{"cat <<EOF\ncopilot -p test\nEOF", false},
		{"cat <<EOF\n$(copilot -p test)\nEOF", true},
		{"cat <<EOF\n$(echo $(date); copilot -p test)\nEOF", true},
		{"cat <<EOF\n'$(echo $(date); copilot -p test)'\nEOF", true},
		{"cat <<EOF\n # $(copilot -p test)\nEOF", true},
		{"cat <<EOF\n$(\n  copilot -p test\n)\nEOF", true},
		{"bash <<'EOF'\ncopilot -p test\nEOF", true},
		{"cat /dev/null; bash <<'EOF'\ncopilot -p test\nEOF", true},
		{`bash -c 'echo "Example: setup; copilot -p test"'`, false},
		{`node -e 'console.log("copilot")'`, false},
		{"# copilot -p test\n  # npm install @github/copilot\necho ready", false},
		{"npm install copilot-utils", false},
		{"npm view @github/copilot version", false},
		{"npm install @github/copilot-utils", false},
		{"./scripts/copilot-report.sh", false},
		{"git diff -- copilot.md", false},
		{"TOOL=copilot\necho \"$TOOL\"", false},
	} {
		t.Run(tc.run, func(t *testing.T) {
			assert.Equal(t, tc.want, stepInvokesAgenticEngine(map[string]any{"run": tc.run}))
		})
	}
	for _, tc := range []struct {
		uses string
		want bool
	}{
		{"anthropics/claude-code-action@sha", true},
		{"OpenAI/codex-action@sha", true},
		{"github/copilot-cli-action@sha", true},
		{"google-github-actions/run-gemini-cli@sha", true},
		{"actions/setup-node@sha", false},
		{"example/copilot-report@sha", false},
	} {
		t.Run(tc.uses, func(t *testing.T) {
			assert.Equal(t, tc.want, stepInvokesAgenticEngine(map[string]any{"uses": tc.uses}))
		})
	}
	assert.False(t, stepInvokesAgenticEngine(map[string]any{"name": "Run copilot", "env": map[string]any{"TOOL": "copilot"}}))
}

func TestValidateAgenticEngineSteps(t *testing.T) {
	for _, job := range []string{"agent", "detection", "evals", "safe_outputs", "custom", "activation", "conclusion", "detection_custom"} {
		for _, strict := range []bool{true, false} {
			t.Run(job+map[bool]string{true: "/strict", false: "/non-strict"}[strict], func(t *testing.T) {
				c := NewCompiler()
				workflow := map[string]any{"jobs": map[string]any{
					job: map[string]any{"steps": []any{
						map[string]any{"run": "npm install -g @github/copilot"},
						map[string]any{"run": "copilot -p test"},
					}},
				}}
				err := c.validateAgenticEngineSteps(workflow, strict, "test.md")
				exempt := job == "agent" || job == "detection" || job == "evals"
				if exempt {
					require.NoError(t, err)
					assert.Zero(t, c.warningCount)
				} else if strict {
					require.ErrorContains(t, err, "strict mode: agentic engine")
					assert.Contains(t, err.Error(), `job "`+job+`", step 1`)
					assert.Contains(t, err.Error(), "'engine:'")
					assert.Zero(t, c.warningCount)
				} else {
					require.NoError(t, err)
					assert.Equal(t, 2, c.warningCount)
				}
			})
		}
	}
}

func TestAgenticEngineStepsAtCompileTime(t *testing.T) {
	for _, tc := range []struct {
		name      string
		config    string
		strict    string
		cliStrict bool
		wantError bool
	}{
		{"custom job strict by default", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: copilot -p test\n", "", false, true},
		{"custom job non-strict", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: copilot -p test\n", "strict: false\n", false, false},
		{"CLI overrides non-strict", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: copilot -p test\n", "strict: false\n", true, true},
		{"safe output steps", "safe-outputs:\n  create-issue:\n  steps:\n    - run: npm install -g @github/copilot\n", "", false, true},
		{"safe output custom job", "safe-outputs:\n  jobs:\n    review:\n      runs-on: ubuntu-latest\n      steps:\n        - run: copilot -p test\n", "", false, true},
		{"activation augmentation", "jobs:\n  activation:\n    steps:\n      - run: copilot -p test\n", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "engine-steps.md")
			content := "---\non: workflow_dispatch\npermissions:\n  contents: read\nnetwork:\n  allowed: [defaults]\n" +
				tc.strict + tc.config + "---\n\nTest engine isolation.\n"
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			c := NewCompiler()
			c.SetSkipValidation(true)
			c.SetStrictMode(tc.cliStrict)
			c.approve = true
			err := c.CompileWorkflow(path)
			if tc.wantError {
				require.ErrorContains(t, err, "strict mode: agentic engine")
				_, statErr := os.Stat(strings.TrimSuffix(path, ".md") + ".lock.yml")
				assert.True(t, os.IsNotExist(statErr), "rejected workflows must not emit a lock file")
			} else {
				require.NoError(t, err)
				assert.Positive(t, c.warningCount)
				_, statErr := os.Stat(strings.TrimSuffix(path, ".md") + ".lock.yml")
				require.NoError(t, statErr)
			}
		})
	}
}
