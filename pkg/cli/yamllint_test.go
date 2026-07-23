//go:build !integration

package cli

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYamllintLine(t *testing.T) {
	t.Run("parses error line", func(t *testing.T) {
		issue, err := parseYamllintLine("./.github/workflows/test.lock.yml:7:9: [error] wrong indentation: expected 8 but found 10 (indentation)")

		require.NoError(t, err)
		assert.Equal(t, yamllintIssue{
			File:    "./.github/workflows/test.lock.yml",
			Line:    7,
			Column:  9,
			Level:   "error",
			Message: "wrong indentation: expected 8 but found 10",
			Rule:    "indentation",
		}, issue)
	})

	t.Run("parses warning line", func(t *testing.T) {
		issue, err := parseYamllintLine("./test.lock.yml:1:1: [warning] missing document start \"---\" (document-start)")

		require.NoError(t, err)
		assert.Equal(t, yamllintIssue{
			File:    "./test.lock.yml",
			Line:    1,
			Column:  1,
			Level:   "warning",
			Message: "missing document start \"---\"",
			Rule:    "document-start",
		}, issue)
	})

	t.Run("rejects malformed line", func(t *testing.T) {
		_, err := parseYamllintLine("not parsable output")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match yamllint parsable format")
	})
}

func TestParseAndDisplayYamllintOutput(t *testing.T) {
	stdout, stderr := captureOutput(t, func() error {
		issues, err := parseAndDisplayYamllintOutput(strings.Join([]string{
			"./test.lock.yml:1:1: [warning] missing document start \"---\" (document-start)",
			"malformed output",
			"./test.lock.yml:2:3: [error] syntax error: expected <block end>, but found '-' (syntax)",
		}, "\n"))
		require.NoError(t, err)
		assert.Equal(t, 2, issues)
		return nil
	})

	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "test.lock.yml:1:1")
	assert.Contains(t, stderr, "[warning] missing document start \"---\" (document-start)")
	assert.Contains(t, stderr, "Failed to parse yamllint output line: malformed output")
	assert.Contains(t, stderr, "test.lock.yml:2:3")
	assert.Contains(t, stderr, "[error] syntax error: expected <block end>, but found '-' (syntax)")
}

func TestParseAndDisplayYamllintOutput_AllMalformed(t *testing.T) {
	stdout, stderr := captureOutput(t, func() error {
		issues, err := parseAndDisplayYamllintOutput("bad line one\nbad line two")
		require.NoError(t, err)
		assert.Equal(t, 0, issues)
		return nil
	})

	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "Failed to parse yamllint output line: bad line one")
	assert.Contains(t, stderr, "Failed to parse yamllint output line: bad line two")
}

func TestBuildYamllintContainerPaths(t *testing.T) {
	t.Run("normalizes in-repo path", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(gitRoot, ".github", "workflows", "static-analysis-report.lock.yml")

		paths, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.NoError(t, err)
		assert.Equal(t, []string{"./.github/workflows/static-analysis-report.lock.yml"}, paths)
	})

	t.Run("rejects path outside repository", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(filepath.Dir(gitRoot), "outside.lock.yml")

		_, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "outside repository root")
	})

	t.Run("prefixes option-looking root file", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(gitRoot, "-workflow.lock.yml")

		paths, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.NoError(t, err)
		assert.Equal(t, []string{"./-workflow.lock.yml"}, paths)
	})
}

func TestBuildYamllintDockerArgs(t *testing.T) {
	args := buildYamllintDockerArgs("/repo", []string{"./test.lock.yml"}, true)

	assert.Equal(t, []string{
		"run",
		"--rm",
		"-v", "/repo:/workdir",
		"-w", "/workdir",
		YamllintImage,
		"-d", yamllintDefaultConfig,
		"--format", "parsable",
		"--strict",
		"./test.lock.yml",
	}, args)
	assert.Equal(t,
		"docker run --rm -v /repo:/workdir -w /workdir "+YamllintImage+" -d "+strconv.Quote(yamllintDefaultConfig)+" --format parsable --strict ./test.lock.yml",
		buildYamllintVerboseCommand("/repo", []string{"./test.lock.yml"}, true),
	)
	assert.Equal(t,
		"docker run --rm -v "+strconv.Quote("/repo root:/workdir")+" -w /workdir "+YamllintImage+" -d "+strconv.Quote(yamllintDefaultConfig)+" --format parsable ./workflow.lock.yml",
		buildYamllintVerboseCommand("/repo root", []string{"./workflow.lock.yml"}, false),
	)
}

func TestClassifyYamllintExit(t *testing.T) {
	t.Run("non-strict exit code 1 is tolerated", func(t *testing.T) {
		assert.NoError(t, classifyYamllintExit(1, false, 2, "workflows"))
	})

	t.Run("strict exit code 1 fails", func(t *testing.T) {
		err := classifyYamllintExit(1, true, 2, "test.lock.yml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "strict mode: yamllint found 2 issue(s) in test.lock.yml")
	})

	t.Run("strict warning-only exit code 2 fails", func(t *testing.T) {
		err := classifyYamllintExit(2, true, 1, "workflows")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "strict mode: yamllint found 1 issue(s) in workflows")
	})

	t.Run("unexpected exit code is returned", func(t *testing.T) {
		err := classifyYamllintExit(3, false, 0, "workflows")
		require.Error(t, err)
		assert.EqualError(t, err, "yamllint failed with exit code 3 on workflows")
	})
}
