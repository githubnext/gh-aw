package lint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareGraderShellFixture(t testing.TB, input Input, source string) Input {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".grader-shell-lint-test-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	root, err := filepath.Abs(dir)
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".github", "workflows", "shared"), 0o700))
	input.SourcePath = filepath.Join(root, ".github", "workflows", "shared", "grader.md")
	require.NoError(t, os.WriteFile(input.SourcePath, []byte("---\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(input.SourcePath), "check.sh"), []byte(source), 0o600))
	return input
}

func graderShellInput(t testing.TB, fields string) Input {
	t.Helper()
	document, err := aw.Parse("---\ngraders:\n  operational-value:\n" + fields + "---\nPrompt\n")
	require.NoError(t, err)
	return Input{Document: document}
}

func TestGraderShellFileWriteRule(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "enabled", true: "disabled"}[disabled], func(t *testing.T) {
			fields := "    run: ./check.sh\n"
			if disabled {
				fields += "    enabled: false\n"
			}
			input := prepareGraderShellFixture(t, graderShellInput(t, fields), "#!/bin/bash\njq '.' \"$GH_AW_SESSION\"\ntouch PRIVATE_FILENAME\n")
			runner, err := Default().Select("grader-shell-file-write")
			require.NoError(t, err)
			diagnostics, err := runner.Run(t.Context(), input)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			assert.Equal(t, "grader-shell-file-write", diagnostics[0].RuleID)
			assert.Equal(t, Error, diagnostics[0].Severity)
			assert.Equal(t, "graders.operational-value.run", diagnostics[0].Path)
			assert.Equal(t, 3, diagnostics[0].Line)
			assert.NotContains(t, diagnostics[0].Message, "PRIVATE_FILENAME")
			require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(input.SourcePath), "check.sh"), []byte("#!/bin/bash\njq '.events | length' \"$GH_AW_SESSION\"\n"), 0o600))
			diagnostics, err = runner.Run(t.Context(), input)
			require.NoError(t, err)
			assert.Empty(t, diagnostics, "do not cache stale source across lint runs")
		})
	}
}

func TestGraderShellFileWriteRuleFailures(t *testing.T) {
	for _, test := range []struct {
		name, fields, source, want string
	}{
		{"missing file", "    run: ./missing.sh\n", "#!/bin/bash\n", "existing regular"},
		{"invalid syntax", "    run: ./check.sh\n", "#!/bin/bash\nif PRIVATE\n", "invalid Bash syntax"},
		{"invalid UTF-8", "    run: ./check.sh\n", "#!/bin/bash\n# \xff\n", "UTF-8"},
		{"no shebang", "    run: ./check.sh\n", "printf PRIVATE\n", "Bash shebang"},
		{"expression", "    run: '${{ inputs.PRIVATE }}.sh'\n", "#!/bin/bash\n", "literal relative .sh path"},
		{"traversal", "    run: ../PRIVATE.sh\n", "#!/bin/bash\n", "literal relative .sh path"},
		{"wrong type", "    run: [PRIVATE]\n", "#!/bin/bash\n", "literal .sh path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := prepareGraderShellFixture(t, graderShellInput(t, test.fields), test.source)
			runner, err := New(GraderShellFileWriteRule())
			require.NoError(t, err)
			_, err = runner.Run(t.Context(), input)
			require.ErrorContains(t, err, test.want)
			assert.NotContains(t, err.Error(), "PRIVATE")
		})
	}
	input := graderShellInput(t, "    run: ./check.sh\n")
	_, err := GraderShellFileWriteRule().Check(t.Context(), input)
	require.ErrorContains(t, err, "declaring workflow source path")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = GraderShellFileWriteRule().Check(ctx, input)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGraderShellFileWriteRuleScope(t *testing.T) {
	for _, fields := range []string{
		"graders:\n  operational-value: false\n",
		"graders:\n  operational-value:\n    script: return 1;\n",
		"graders:\n  custom:\n    script: return 1;\n",
		"graders:\n  other:\n    run: ./missing.sh\n",
		"on: workflow_dispatch\n",
	} {
		t.Run(strings.ReplaceAll(fields, "\n", " "), func(t *testing.T) {
			document, err := aw.Parse("---\n" + fields + "---\nPrompt\n")
			require.NoError(t, err)
			diagnostics, err := GraderShellFileWriteRule().Check(t.Context(), Input{Document: document})
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	}
	diagnostics, err := GraderShellFileWriteRule().Check(t.Context(), Input{})
	require.ErrorContains(t, err, "requires an Actions or AW AST")
	assert.Empty(t, diagnostics)
}
