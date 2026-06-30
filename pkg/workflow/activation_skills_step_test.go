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
	assert.Contains(t, steps, "GH_AW_SKILLS_SUMMARY: '[\"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6\",\"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6\"]'", "expected summary env var for requested skills")
	assert.Contains(t, steps, "GH_AW_SKILL_SPEC_0: \"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6\"", "expected first skill env var")
	assert.Contains(t, steps, "GH_AW_SKILL_SPEC_1: \"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6\"", "expected second skill env var")
	assert.Contains(t, steps, "skill_spec=\"${GH_AW_SKILL_SPEC_0}\"", "expected runtime install loop to read first skill from env")
	assert.Contains(t, steps, "install_args+=(--all)", "expected runtime repository-scope detection")
	assert.Contains(t, steps, "gh skill install \"${skill_spec}\" \"${install_args[@]}\" --dir \"${SKILLS_DST}\" --force", "expected runtime install command to use quoted env values")
	assert.Contains(t, steps, "### Frontmatter skills installed", "expected step summary output")
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
	assert.Contains(t, steps, "GH_AW_SKILL_SPEC_0: \"${{ inputs.skill_ref }}\"", "expected whole-expression skill env var")
	assert.Contains(t, steps, "GH_AW_SKILL_SPEC_1: \"githubnext/skills@${{ github.sha }}\"", "expected expression-ref skill env var")
	assert.NotContains(t, steps, "echo \"Installing skill reference: ${{ inputs.skill_ref }}\"", "expression should not be interpolated directly into the run script")
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
