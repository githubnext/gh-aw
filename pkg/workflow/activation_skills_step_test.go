//go:build !integration

package workflow

import (
	"fmt"
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
	assert.Contains(t, steps, "ensure_gh_cli_min_version.sh", "expected gh upgrade step to delegate to ensure_gh_cli_min_version.sh")
	assert.Contains(t, steps, "Install frontmatter skill 1", "expected first frontmatter skill install step in activation job")
	assert.Contains(t, steps, "Install frontmatter skill 2", "expected second frontmatter skill install step in activation job")
	assert.Contains(t, steps, "GH_AW_SKILL_DIR: \".claude/skills\"", "expected engine skill directory env var")
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6\"", "expected first skill env var")
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6\"", "expected second skill env var")
	assert.Contains(t, steps, "const { main } = require('${{ runner.temp }}/gh-aw/actions/install_frontmatter_skills.cjs');", "expected github-script runtime loader for skill install")
	assert.Contains(t, steps, "collect-skill-install-failures", "expected collect failures step in activation job")
	assert.Contains(t, steps, "collect_skill_install_failures.cjs", "expected collect failures script in activation job")
	outputs := fmt.Sprintf("%v", job.Outputs)
	assert.Contains(t, outputs, "skill_install_failure_count", "expected skill install failure count output wiring")
	assert.Contains(t, outputs, "skill_install_errors", "expected skill install errors output wiring")
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
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"${{ inputs.skill_ref }}\"", "expected first expression skill env var to preserve expression for runtime resolution")
	assert.Contains(t, steps, "GH_AW_FRONTMATTER_SKILLS: \"githubnext/skills@${{ github.sha }}\"", "expected second expression skill env var to preserve expression for runtime resolution")
	assert.NotContains(t, steps, "GH_AW_SKILL_SPEC_0", "expected per-skill env vars to be removed")
}

func TestBuildActivationJob_AddsPerSkillAuthSteps(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		Name: "skills-workflow",
		On: `"on":
  workflow_dispatch:`,
		AI: "copilot",
		SkillReferences: []SkillReference{
			{
				Skill:       "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
				GitHubToken: "${{ secrets.SKILL_PAT }}",
			},
			{
				Skill: "githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6",
				GitHubApp: &GitHubAppConfig{
					AppID:      "${{ vars.APP_ID }}",
					PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "skills.lock.yml")
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "GH_TOKEN: ${{ secrets.SKILL_PAT }}", "expected first skill install step to use per-skill github-token")
	assert.Contains(t, steps, "Generate GitHub App token for frontmatter skill 2", "expected app token mint step for second skill")
	assert.Contains(t, steps, "id: frontmatter-skill-app-token-2", "expected deterministic app token step id")
	assert.Contains(t, steps, "GH_TOKEN: ${{ steps.frontmatter-skill-app-token-2.outputs.token }}", "expected second skill install step to use minted app token")
}

func TestBuildActivationJob_AppIgnoreIfMissingFallsBackToActivationToken(t *testing.T) {
	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)

	data := &WorkflowData{
		Name: "skills-workflow",
		On: `"on":
  workflow_dispatch:`,
		AI: "copilot",
		SkillReferences: []SkillReference{
			{
				Skill: "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
				GitHubApp: &GitHubAppConfig{
					AppID:           "${{ vars.APP_ID }}",
					PrivateKey:      "${{ secrets.APP_PRIVATE_KEY }}",
					IgnoreIfMissing: true,
				},
			},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "skills.lock.yml")
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(
		t,
		steps,
		"GH_TOKEN: ${{ steps.frontmatter-skill-app-token-1.outputs.token || secrets.GITHUB_TOKEN }}",
		"expected ignore-if-missing fallback to use activation token expression",
	)
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
	assert.NotContains(t, steps, "Install frontmatter skill", "expected no skill install step without frontmatter skills")
}
