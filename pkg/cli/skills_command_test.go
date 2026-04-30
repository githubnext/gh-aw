//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUserSkills(t *testing.T) {
	skills, err := listUserSkills()
	require.NoError(t, err, "listUserSkills should succeed")
	assert.NotEmpty(t, skills, "listUserSkills should return at least one skill")

	expectedSkills := []string{
		"audit-workflows",
		"compile-workflows",
		"debug-workflow-run",
		"discover-workflows",
		"install-workflow",
	}
	assert.Equal(t, expectedSkills, skills, "listUserSkills should return all expected skills in alphabetical order")
}

func TestSkillDescription(t *testing.T) {
	tests := []struct {
		name     string
		skill    string
		contains string
	}{
		{
			name:     "discover-workflows has description",
			skill:    "discover-workflows",
			contains: "discover",
		},
		{
			name:     "install-workflow has description",
			skill:    "install-workflow",
			contains: "install",
		},
		{
			name:     "compile-workflows has description",
			skill:    "compile-workflows",
			contains: "compile",
		},
		{
			name:     "audit-workflows has description",
			skill:    "audit-workflows",
			contains: "audit",
		},
		{
			name:     "debug-workflow-run has description",
			skill:    "debug-workflow-run",
			contains: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := skillDescription(tt.skill)
			assert.NotEmpty(t, desc, "Skill %q should have a non-empty description", tt.skill)
			assert.Contains(t, strings.ToLower(desc), tt.contains,
				"Skill %q description %q should contain %q",
				tt.skill, desc, tt.contains)
		})
	}
}

func TestSkillDescriptionUnknownSkill(t *testing.T) {
	desc := skillDescription("nonexistent-skill")
	assert.Empty(t, desc, "Unknown skill should return empty description")
}

func TestAgentSkillsDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "Should be able to get home directory")

	tests := []struct {
		name        string
		agent       AgentType
		expectedDir string
		shouldErr   bool
	}{
		{
			name:        "claude-code returns ~/.claude/agents",
			agent:       AgentClaudeCode,
			expectedDir: filepath.Join(homeDir, ".claude", "agents"),
		},
		{
			name:        "copilot returns ~/.config/gh-copilot/agents",
			agent:       AgentCopilot,
			expectedDir: filepath.Join(homeDir, ".config", "gh-copilot", "agents"),
		},
		{
			name:        "codex returns ~/.codex/agents",
			agent:       AgentCodex,
			expectedDir: filepath.Join(homeDir, ".codex", "agents"),
		},
		{
			name:      "unknown agent returns error",
			agent:     AgentType("unknown-agent"),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := AgentSkillsDir(tt.agent)
			if tt.shouldErr {
				assert.Error(t, err, "AgentSkillsDir(%q) should return an error", tt.agent)
				return
			}
			require.NoError(t, err, "AgentSkillsDir(%q) should not return an error", tt.agent)
			assert.Equal(t, tt.expectedDir, dir, "AgentSkillsDir(%q) should return expected directory", tt.agent)
		})
	}
}

func TestInstallSkill(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("installs skill file to target directory", func(t *testing.T) {
		err := installSkill("discover-workflows", tmpDir, false)
		require.NoError(t, err, "installSkill should succeed for valid skill")

		destPath := filepath.Join(tmpDir, "discover-workflows.md")
		assert.FileExists(t, destPath, "Skill file should be created at expected path")

		data, err := os.ReadFile(destPath)
		require.NoError(t, err, "Should be able to read installed skill file")
		assert.Contains(t, string(data), "discover-workflows", "Installed skill should contain skill name")
	})

	t.Run("skips existing file without force flag", func(t *testing.T) {
		destPath := filepath.Join(tmpDir, "discover-workflows.md")
		originalContent := []byte("original content")
		require.NoError(t, os.WriteFile(destPath, originalContent, 0600))

		err := installSkill("discover-workflows", tmpDir, false)
		require.NoError(t, err, "installSkill without force should not return error for existing file")

		data, err := os.ReadFile(destPath)
		require.NoError(t, err, "Should be able to read file")
		assert.Equal(t, originalContent, data, "File content should not change without --force")
	})

	t.Run("overwrites existing file with force flag", func(t *testing.T) {
		destPath := filepath.Join(tmpDir, "install-workflow.md")
		require.NoError(t, os.WriteFile(destPath, []byte("old content"), 0600))

		err := installSkill("install-workflow", tmpDir, true)
		require.NoError(t, err, "installSkill with force should succeed")

		data, err := os.ReadFile(destPath)
		require.NoError(t, err, "Should be able to read file after force-overwrite")
		assert.NotEqual(t, []byte("old content"), data, "File content should change with --force")
	})

	t.Run("returns error for unknown skill", func(t *testing.T) {
		err := installSkill("nonexistent-skill", tmpDir, false)
		assert.Error(t, err, "installSkill should return error for unknown skill")
	})
}

func TestNewSkillsCommand(t *testing.T) {
	cmd := NewSkillsCommand()
	require.NotNil(t, cmd, "NewSkillsCommand should return a command")
	assert.Equal(t, "skills", cmd.Name(), "Command name should be 'skills'")

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	assert.True(t, subcommands["list"], "skills command should have 'list' subcommand")
	assert.True(t, subcommands["install"], "skills command should have 'install' subcommand")
	assert.True(t, subcommands["path"], "skills command should have 'path' subcommand")
}

func TestSkillsInstallSubcommandFlags(t *testing.T) {
	cmd := NewSkillsCommand()
	installCmd := findCmd(cmd, "install")
	require.NotNil(t, installCmd, "install subcommand should exist")

	agentFlag := installCmd.Flags().Lookup("agent")
	require.NotNil(t, agentFlag, "install should have --agent flag")
	assert.Equal(t, string(AgentClaudeCode), agentFlag.DefValue, "--agent should default to claude-code")

	forceFlag := installCmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag, "install should have --force flag")
}

func TestSkillsPathSubcommandFlags(t *testing.T) {
	cmd := NewSkillsCommand()
	pathCmd := findCmd(cmd, "path")
	require.NotNil(t, pathCmd, "path subcommand should exist")

	agentFlag := pathCmd.Flags().Lookup("agent")
	require.NotNil(t, agentFlag, "path should have --agent flag")
	assert.Equal(t, string(AgentClaudeCode), agentFlag.DefValue, "--agent should default to claude-code")
}

// findCmd finds a subcommand by name within a parent command.
func findCmd(parent interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
