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
	assert.Contains(t, steps, "Upgrade gh CLI for frontmatter skills", "expected gh upgrade step in activation job")
	assert.Contains(t, steps, "Install frontmatter skills", "expected frontmatter skills install step in activation job")
	assert.Contains(t, steps, "GH_AW_SKILL_DIR: \".claude/skills\"", "expected engine skill directory env var")
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6\\ngithubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6\"", "expected skills env var")
	assert.Contains(t, steps, "const { main } = require('${{ runner.temp }}/gh-aw/actions/install_frontmatter_skills.cjs');", "expected github-script runtime loader for skill install")
	assert.NotContains(t, steps, "GH_AW_SKILL_SPEC_0", "expected per-skill env vars to be removed")
}

func TestBuildActivationJob_AddsExpressionSkillInstallSteps(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		Name: "skills-workflow",
		On: `"on":
  workflow_dispatch:`,
		AI: "copilot",
		Skills: []string{
			"${{ inputs.skill_ref }}",
			"githubnext/skills@${{ github.sha }}",
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "skills.lock.yml")
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"${{ inputs.skill_ref }}\\ngithubnext/skills@${{ github.sha }}\"", "expected skills env var to preserve expressions for runtime resolution")
	assert.NotContains(t, steps, "GH_AW_SKILL_SPEC_0", "expected per-skill env vars to be removed")
}

func TestBuildActivationJob_NoSkillsStepsWhenSkillsAbsent(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		Name: "no-skills-workflow",
		On: `"on":
  workflow_dispatch:`,
		AI: "copilot",
	}

	job, err := compiler.buildActivationJob(data, false, "", "no-skills.lock.yml")
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.NotContains(t, steps, "Upgrade gh CLI for frontmatter skills", "expected no gh upgrade step without frontmatter skills")
	assert.NotContains(t, steps, "Install frontmatter skills", "expected no skill install step without frontmatter skills")
}
