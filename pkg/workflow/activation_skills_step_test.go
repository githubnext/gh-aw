//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildActivationJob_AddsFrontmatterSkillsInstallSteps(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		Name: "skills-workflow",
		On: `"on":
  workflow_dispatch:`,
		AI: "copilot",
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
		Skills: []string{
			"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
			"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6",
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "skills.lock.yml")
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "Upgrade gh CLI for frontmatter skills")
	assert.Contains(t, steps, "Install frontmatter skills")
	assert.Contains(t, steps, "GH_AW_SKILL_DIR: \".claude/skills\"")
	assert.Contains(t, steps, "gh skill install \"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6\" --all --dir \"${SKILLS_DST}\" --force")
	assert.Contains(t, steps, "gh skill install \"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6\" --dir \"${SKILLS_DST}\" --force")
	assert.Contains(t, steps, "### Frontmatter skills installed")
}
