//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func workflowHeaderLogoBlock() string {
	var block strings.Builder
	for line := range strings.SplitSeq(strings.TrimRight(headerAsciiLogo, "\n"), "\n") {
		block.WriteString("# " + strings.TrimRight(line, " \t") + "\n")
	}
	block.WriteString("#\n#\n")
	return block.String()
}

func TestGenerateWorkflowHeader_Logo(t *testing.T) {
	withLogo := GenerateWorkflowHeader("test.md", "gh-aw", "Custom instructions.", true)
	withoutLogo := GenerateWorkflowHeader("test.md", "gh-aw", "Custom instructions.", false)

	require.Equal(t, withLogo, GenerateWorkflowHeader("test.md", "gh-aw", "Custom instructions."))
	require.Contains(t, withLogo, workflowHeaderLogoBlock())
	require.NotContains(t, withoutLogo, "#    ___")
	require.Equal(t, strings.Replace(withLogo, workflowHeaderLogoBlock(), "", 1), withoutLogo)
	require.Contains(t, withoutLogo, "DO NOT EDIT.")
	require.Contains(t, withoutLogo, "# To update this file, edit test.md and run:\n#   gh aw compile\n")
	require.Contains(t, withoutLogo, "# Custom instructions.\n")
}

func TestCompilerLockFileLogo(t *testing.T) {
	for _, tt := range []struct {
		name       string
		config     string
		logoOption *bool
		wantLogo   bool
	}{
		{name: "default", wantLogo: true},
		{name: "empty config", config: `{}`, wantLogo: true},
		{name: "config enabled", config: `{"lock_file_logo": true}`, wantLogo: true},
		{name: "config disabled", config: `{"lock_file_logo": false}`},
		{name: "setter disabled", logoOption: new(false)},
		{name: "flag overrides config", config: `{"lock_file_logo": true}`, logoOption: new(false)},
		{name: "config overrides setter", config: `{"lock_file_logo": false}`, logoOption: new(true)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.config != "" {
				writeAWJSON(t, dir, tt.config)
			}
			compiler := NewCompiler()
			compiler.gitRoot = dir
			if tt.logoOption != nil {
				compiler.SetLockFileLogo(*tt.logoOption)
			}
			data := &WorkflowData{Source: "test.md", Description: "Test workflow"}
			var header strings.Builder
			require.NoError(t, compiler.generateWorkflowHeader(&header, data, "frontmatter-hash", "body-hash", []string{"TEST_SECRET"}, []string{"actions/checkout@v6"}))

			text := header.String()
			require.Equal(t, tt.wantLogo, strings.Contains(text, workflowHeaderLogoBlock()))
			require.True(t, strings.HasPrefix(text, "# gh-aw-metadata: "))
			require.Contains(t, text, "# gh-aw-manifest: ")
			require.Contains(t, text, "DO NOT EDIT.")
			require.Contains(t, text, "# Test workflow")
			require.Contains(t, text, "TEST_SECRET")
			require.Contains(t, text, "actions/checkout@v6")

			baseline := NewCompiler()
			baseline.gitRoot = t.TempDir()
			var defaultHeader strings.Builder
			require.NoError(t, baseline.generateWorkflowHeader(&defaultHeader, data, "frontmatter-hash", "body-hash", []string{"TEST_SECRET"}, []string{"actions/checkout@v6"}))
			expected := defaultHeader.String()
			if !tt.wantLogo {
				expected = strings.Replace(expected, workflowHeaderLogoBlock(), "", 1)
			}
			require.Equal(t, expected, text, "only the logo block should change")
		})
	}
}

func TestCompanionWorkflowLockFileLogo(t *testing.T) {
	t.Setenv("GH_AW_ACTION_MODE", "dev")
	dir := t.TempDir()
	data := []*WorkflowData{{
		WorkflowID:         "test",
		Command:            []string{"test"},
		CommandEvents:      []string{"issue_comment"},
		CommandCentralized: true,
		Tools:              map[string]any{"work-queue": map[string]any{"issues": true}},
		CheckoutConfigs:    []*CheckoutConfig{{Repository: "my-org/target-repo", Current: true}},
		SafeOutputs:        &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{Expires: 48}},
	}}
	config := &RepoConfig{AutoUpgrade: new(true)}
	paths := []string{
		"agentics-maintenance.yml",
		"agentics-maintenance-my-org-target-repo.yml",
		AutoUpdateWorkflowFileName,
		centralSlashCommandWorkflowFilename,
		workQueueEventWorkflowFilename,
	}
	baseline := make(map[string]string)
	for _, logo := range []bool{true, false} {
		if !logo {
			config.LockFileLogo = new(false)
		}
		require.NoError(t, GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: data,
			WorkflowDir:      dir,
			ActionMode:       ActionModeDev,
			RepoConfig:       config,
		}))
		require.NoError(t, GenerateCentralSlashCommandWorkflow(context.Background(), data, dir, config))
		require.NoError(t, GenerateWorkQueueEventWorkflow(context.Background(), data, dir, config))
		for _, path := range paths {
			content, err := os.ReadFile(filepath.Join(dir, path))
			require.NoError(t, err)
			text := string(content)
			require.Equal(t, logo, strings.Contains(text, workflowHeaderLogoBlock()), path)
			require.Contains(t, text, "DO NOT EDIT.", path)
			if logo {
				baseline[path] = text
			} else {
				require.Equal(t, strings.Replace(baseline[path], workflowHeaderLogoBlock(), "", 1), text, path)
			}
		}
	}
}

func TestGenerateWorkflowHeader_VersionInDevBuild(t *testing.T) {
	// Save original version
	originalVersion := compilerVersion
	defer func() { compilerVersion = originalVersion }()

	// Test with dev version - should not include version in header
	SetVersion("dev")
	header := GenerateWorkflowHeader("test.md", "gh-aw", "")

	if strings.Contains(header, "(dev)") {
		t.Error("Expected header to NOT include version for dev builds")
	}
	if !strings.Contains(header, "This file was automatically generated by gh-aw. DO NOT EDIT.") {
		t.Error("Expected header to contain disclaimer without version")
	}
	// Verify auto-generated notice is on the first line
	firstLine := strings.SplitN(header, "\n", 2)[0]
	if !strings.HasPrefix(firstLine, "# This file was automatically generated") {
		t.Errorf("Expected auto-generated notice on first line, got: %s", firstLine)
	}
	if !strings.Contains(firstLine, "https://github.com/github/gh-aw/blob/main/debug.md") {
		t.Errorf("Expected skill URL on first line, got: %s", firstLine)
	}
}

func TestGenerateWorkflowHeader_VersionInReleaseBuild(t *testing.T) {
	// Save original state
	originalVersion := compilerVersion
	originalIsRelease := isReleaseBuild
	defer func() {
		compilerVersion = originalVersion
		isReleaseBuild = originalIsRelease
	}()

	// Test with release version - should include version in header when isRelease is true
	SetVersion("1.0.0")
	SetIsRelease(true)
	header := GenerateWorkflowHeader("test.md", "gh-aw", "")

	if !strings.Contains(header, "This file was automatically generated by gh-aw (1.0.0). DO NOT EDIT.") {
		t.Errorf("Expected header to contain disclaimer with version, got:\n%s", header)
	}
}

func TestGenerateWorkflowHeader_DirtyVersion(t *testing.T) {
	// Save original version
	originalVersion := compilerVersion
	defer func() { compilerVersion = originalVersion }()

	// Test with dirty version (git describe output) - should NOT include version
	// IsReleasedVersion excludes versions containing "dirty"
	SetVersion("c610c2a-dirty")
	header := GenerateWorkflowHeader("test.md", "gh-aw", "")

	if strings.Contains(header, "(c610c2a-dirty)") {
		t.Error("Expected header to NOT include version for dirty builds")
	}
	if !strings.Contains(header, "This file was automatically generated by gh-aw. DO NOT EDIT.") {
		t.Error("Expected header to contain disclaimer without version")
	}
}

func TestGenerateWorkflowHeader_GitHashVersion(t *testing.T) {
	// Save original version
	originalVersion := compilerVersion
	defer func() { compilerVersion = originalVersion }()

	// Test with git hash version - should NOT include version
	// IsReleasedVersion requires proper semver format
	SetVersion("c610c2a")
	header := GenerateWorkflowHeader("test.md", "gh-aw", "")

	if strings.Contains(header, "(c610c2a)") {
		t.Error("Expected header to NOT include version for git hash builds")
	}
	if !strings.Contains(header, "This file was automatically generated by gh-aw. DO NOT EDIT.") {
		t.Error("Expected header to contain disclaimer without version")
	}
}

func TestGenerateWorkflowHeader_EmptyVersion(t *testing.T) {
	// Save original version
	originalVersion := compilerVersion
	defer func() { compilerVersion = originalVersion }()

	// Test with empty version - should not include version in header
	SetVersion("")
	header := GenerateWorkflowHeader("test.md", "gh-aw", "")

	if strings.Contains(header, "()") {
		t.Error("Expected header to NOT include empty parentheses")
	}
	if !strings.Contains(header, "This file was automatically generated by gh-aw. DO NOT EDIT.") {
		t.Error("Expected header to contain disclaimer without version")
	}
}

func TestGenerateWorkflowHeader_NoGeneratedBy(t *testing.T) {
	// Save original version
	originalVersion := compilerVersion
	defer func() { compilerVersion = originalVersion }()

	// Test with empty generatedBy - should use default message
	SetVersion("1.0.0")
	header := GenerateWorkflowHeader("test.md", "", "")

	if !strings.Contains(header, "This file was automatically generated. DO NOT EDIT.") {
		t.Error("Expected header to contain default disclaimer")
	}
	// Should not include version when generatedBy is empty
	if strings.Contains(header, "(1.0.0)") {
		t.Error("Expected header to NOT include version when generatedBy is empty")
	}
}

func TestGenerateWorkflowHeader_WithReleaseFlag(t *testing.T) {
	// Save original state
	originalVersion := compilerVersion
	originalIsRelease := isReleaseBuild
	defer func() {
		compilerVersion = originalVersion
		isReleaseBuild = originalIsRelease
	}()

	tests := []struct {
		name              string
		version           string
		isRelease         bool
		shouldHaveVersion bool
		description       string
	}{
		{
			name:              "Release build with valid version",
			version:           "1.0.0",
			isRelease:         true,
			shouldHaveVersion: true,
			description:       "Release build should include version in header",
		},
		{
			name:              "Dev build with valid version",
			version:           "1.0.0",
			isRelease:         false,
			shouldHaveVersion: false,
			description:       "Non-release build should not include version",
		},
		{
			name:              "Dev build with dev version",
			version:           "dev",
			isRelease:         false,
			shouldHaveVersion: false,
			description:       "Non-release build with 'dev' version should not include version",
		},
		{
			name:              "Release build with dev version",
			version:           "dev",
			isRelease:         true,
			shouldHaveVersion: true,
			description:       "Release build should include version even if it says 'dev'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetVersion(tt.version)
			SetIsRelease(tt.isRelease)
			header := GenerateWorkflowHeader("test.md", "gh-aw", "")

			hasVersion := strings.Contains(header, "("+tt.version+")")
			if hasVersion != tt.shouldHaveVersion {
				t.Errorf("Header version inclusion = %v, want %v\n%s\nHeader:\n%s",
					hasVersion, tt.shouldHaveVersion, tt.description, header)
			}
		})
	}
}
