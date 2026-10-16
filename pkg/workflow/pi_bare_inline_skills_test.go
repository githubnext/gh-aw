//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bareInlineSkillReportingHint = "Use the `reporting` skill when producing a report, issue, or PR comment to follow the formatting guidelines below."
	bareInlineSkillReportingPath = "/tmp/gh-aw/pi-agent-dir/skills/reporting/SKILL.md"
	bareInlineSkillAmbientName   = "ambient-helper"
)

// bareInlineSkillReportingImport mirrors .github/workflows/shared/reporting.md: the skill-use
// hint lives outside the inline skill block, so extraction leaves it in the prompt.
const bareInlineSkillReportingImport = "---\n# Report formatting guidelines exposed as an inline skill.\n---\n\n" +
	bareInlineSkillReportingHint + "\n\n" +
	"## skill: `reporting`\n" +
	"---\ndescription: Report formatting guidelines for gh-aw workflows\n---\n\n" +
	"- Use `###` (or lower) headers only.\n\n" +
	"## end skill: `reporting`\n"

var bareInlineSkillPiFlags = []string{"--no-context-files", "--no-skills", "--no-prompt-templates", "--no-extensions", "--no-themes"}

// setupBareInlineSkillRepo writes a repository fixture containing the reporting import and an
// unrelated ambient pi skill (negative control), then writes the workflow and returns its path.
func setupBareInlineSkillRepo(t *testing.T, workflowName, workflowContent string) string {
	t.Helper()
	root := testutil.TempDir(t, "pi-bare-inline-skills-*")
	sharedDir := filepath.Join(root, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "reporting.md"), []byte(bareInlineSkillReportingImport), 0o644))
	ambientDir := filepath.Join(root, ".pi", "skills", bareInlineSkillAmbientName)
	require.NoError(t, os.MkdirAll(ambientDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ambientDir, "SKILL.md"), []byte("---\ndescription: ambient\n---\nAmbient.\n"), 0o644))
	workflowPath := filepath.Join(root, ".github", "workflows", workflowName+".md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0o644))
	return workflowPath
}

// compileBareInlineSkillWorkflow compiles the workflow and returns the lock file content
// together with the parsed workflow data.
func compileBareInlineSkillWorkflow(t *testing.T, workflowPath string) (string, *WorkflowData) {
	t.Helper()
	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath), "workflow should compile")
	lock, err := os.ReadFile(strings.TrimSuffix(workflowPath, ".md") + ".lock.yml")
	require.NoError(t, err)
	data, err := NewCompiler().ParseWorkflowFile(workflowPath)
	require.NoError(t, err)
	return string(lock), data
}

var piCommandArgsPattern = regexp.MustCompile(` pi (--print [^']*?) --append-system-prompt`)

// compiledPiArgs returns the pi CLI argument string from the compiled execution command.
func compiledPiArgs(t *testing.T, lock string) string {
	t.Helper()
	match := piCommandArgsPattern.FindStringSubmatch(lock)
	require.NotNil(t, match, "compiled lock should invoke pi")
	return match[1]
}

// assertPromptContract checks that the runtime-imported source keeps the skill-use hint after
// inline-skill extraction and materializes exactly the reporting skill.
func assertPromptContract(t *testing.T, lock, runtimeImportPath, sourcePath string) {
	t.Helper()
	assert.Contains(t, lock, "{{#runtime-import "+runtimeImportPath+"}}", "prompt should load the skill source at runtime")
	raw, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	body, err := parser.ExtractMarkdownContent(string(raw))
	require.NoError(t, err)
	prompt, skills, err := parser.ExtractInlineSkills(body)
	require.NoError(t, err)
	assert.Contains(t, prompt, bareInlineSkillReportingHint, "effective prompt should keep the reporting hint")
	assert.NotContains(t, prompt, "## skill:", "inline skill body should be extracted from the prompt")
	require.Len(t, skills, 1)
	assert.Equal(t, "reporting", skills[0].Name, "extraction should materialize reporting/SKILL.md")
	assert.Equal(t, "/SKILL.md", engineInlineSkillExt("pi"))
}

func TestCompileBarePiInlineSkillsExplicitlyLoaded(t *testing.T) {
	tests := []struct {
		name              string
		workflow          string
		runtimeImportPath string
		sourceFile        string
	}{
		{
			name: "imported",
			workflow: `---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: pi
  bare: true
imports:
  - shared/reporting.md
tools:
  bash: false
  cli-proxy: false
---

# Bare pi imported skill

Say hello.
`,
			runtimeImportPath: ".github/workflows/shared/reporting.md",
			sourceFile:        "shared/reporting.md",
		},
		{
			name: "directly declared",
			workflow: `---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: pi
  bare: true
tools:
  bash: false
  cli-proxy: false
---

# Bare pi declared skill

` + bareInlineSkillReportingHint + `

## skill: ` + "`reporting`" + `
---
description: Report formatting guidelines for gh-aw workflows
---

- Use ` + "`###`" + ` (or lower) headers only.

## end skill: ` + "`reporting`" + `
`,
			runtimeImportPath: ".github/workflows/bare-pi-skill.md",
			sourceFile:        "bare-pi-skill.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowPath := setupBareInlineSkillRepo(t, "bare-pi-skill", tt.workflow)
			lock, data := compileBareInlineSkillWorkflow(t, workflowPath)

			args := compiledPiArgs(t, lock)
			assert.Contains(t, args, strings.Join(bareInlineSkillPiFlags, " ")+" --skill "+bareInlineSkillReportingPath,
				"bare pi should keep all five discovery-disabling flags and explicitly load the workflow skill")
			assert.Equal(t, 1, strings.Count(args, "--skill "), "only the workflow skill should be loaded explicitly")
			assert.Contains(t, lock, `"--no-themes","--skill","`+bareInlineSkillReportingPath+`"`, "pi sub-agents should inherit the explicit workflow skill")
			assert.Contains(t, lock, `GH_AW_SKILL_DIR: ".pi/skills"`, "inline skills should be restored from the activation artifact")
			assert.NotContains(t, lock, bareInlineSkillAmbientName, "ambient repository skills must not be loaded")

			require.Len(t, data.InlineSkills, 1)
			assert.Equal(t, "reporting", data.InlineSkills[0].Name)

			assertPromptContract(t, lock, tt.runtimeImportPath, filepath.Join(filepath.Dir(workflowPath), tt.sourceFile))
		})
	}
}

func TestCompilePiRoutedClaudeWithReportingSkill(t *testing.T) {
	workflowPath := setupBareInlineSkillRepo(t, "smoke-pi-routed-claude", `---
on: workflow_dispatch
permissions:
  contents: read
  copilot-requests: write
engine:
  id: pi
  bare: true
  model-routing:
    goal: cost
    mode: robust
    allowed-models: [claude-sonnet-5]
imports:
  - shared/reporting.md
max-tool-calls: 10
max-tool-denials: 3
tools:
  bash: false
  cli-proxy: false
  edit: false
safe-outputs:
  create-issue:
sandbox:
  agent:
    id: awf
---

# Smoke Test: Pi Model Routing to Claude

Answer the question "what is 6 × 7?" in one word.
`)
	lock, data := compileBareInlineSkillWorkflow(t, workflowPath)

	args := compiledPiArgs(t, lock)
	assert.Contains(t, args, "--exclude-tools bash", "bash must stay disabled")
	assert.Contains(t, args, strings.Join(bareInlineSkillPiFlags, " ")+" --skill "+bareInlineSkillReportingPath,
		"bare pi should explicitly load the reporting skill")
	assert.Contains(t, args, `--model "aw-gateway/$(cat /tmp/gh-aw/pi-routing-model)"`, "model routing should be unchanged")
	assert.Contains(t, lock, "GH_AW_MAX_TOOL_DENIALS: 3", "denial budget should be unchanged")
	assert.Contains(t, lock, "claude-sonnet-5", "routing pool should be unchanged")
	assert.NotContains(t, lock, bareInlineSkillAmbientName)
	require.Len(t, data.InlineSkills, 1)
	assert.Equal(t, "reporting", data.InlineSkills[0].Name)

	assertPromptContract(t, lock, ".github/workflows/shared/reporting.md", filepath.Join(filepath.Dir(workflowPath), "shared", "reporting.md"))
}

func TestCompileNonBarePiInlineSkillsUnchanged(t *testing.T) {
	for _, bare := range []string{"\n  bare: false", ""} {
		name := "omitted"
		if bare != "" {
			name = "false"
		}
		t.Run("bare "+name, func(t *testing.T) {
			workflowPath := setupBareInlineSkillRepo(t, "non-bare-pi-skill", `---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: pi`+bare+`
imports:
  - shared/reporting.md
tools:
  bash: false
  cli-proxy: false
---

# Non-bare pi skill

Say hello.
`)
			lock, data := compileBareInlineSkillWorkflow(t, workflowPath)
			require.Len(t, data.InlineSkills, 1, "the reporting skill should still be detected")

			args := compiledPiArgs(t, lock)
			assert.Equal(t, "--print --mode json --no-session --no-approve --exclude-tools bash --model aw-gateway/auto", args,
				"non-bare pi arguments should match the pre-fix baseline")
			assert.NotContains(t, lock, "--skill", "non-bare pi should rely on normal discovery")
			for _, flag := range bareInlineSkillPiFlags {
				assert.NotContains(t, lock, flag)
			}

			withoutSkills := *data
			withoutSkills.InlineSkills = nil
			engine := NewPiEngine()
			assert.Equal(t, engine.buildPiArgs(&withoutSkills), engine.buildPiArgs(data), "inline skills should not change non-bare arguments")

			assertPromptContract(t, lock, ".github/workflows/shared/reporting.md", filepath.Join(filepath.Dir(workflowPath), "shared", "reporting.md"))
		})
	}
}

func TestCompileBarePiWithoutInlineSkillsUnchanged(t *testing.T) {
	workflowPath := setupBareInlineSkillRepo(t, "bare-pi-no-skill", `---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: pi
  bare: true
tools:
  bash: false
  cli-proxy: false
---

# Bare pi baseline

Say hello.
`)
	lock, data := compileBareInlineSkillWorkflow(t, workflowPath)
	assert.Empty(t, data.InlineSkills)

	args := compiledPiArgs(t, lock)
	assert.Equal(t, "--print --mode json --no-session --no-approve --exclude-tools bash "+strings.Join(bareInlineSkillPiFlags, " ")+" --model aw-gateway/auto", args,
		"bare pi arguments should match the pre-fix baseline")
	assert.NotContains(t, lock, "--skill")
	assert.NotContains(t, lock, "reporting")
	assert.NotContains(t, lock, bareInlineSkillAmbientName, "ambient repository skills must not be loaded")

	raw, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	body, err := parser.ExtractMarkdownContent(string(raw))
	require.NoError(t, err)
	prompt, skills, err := parser.ExtractInlineSkills(body)
	require.NoError(t, err)
	assert.Empty(t, skills)
	assert.NotContains(t, prompt, "skill", "prompt should gain no skill-use hints")
}

func TestCompileBareInlineSkillsOtherEnginesUnchanged(t *testing.T) {
	for _, engineID := range []string{"claude", "copilot"} {
		t.Run(engineID, func(t *testing.T) {
			workflowPath := setupBareInlineSkillRepo(t, "bare-"+engineID+"-skill", `---
on: workflow_dispatch
permissions:
  contents: read
engine:
  id: `+engineID+`
  bare: true
imports:
  - shared/reporting.md
tools:
  bash: false
  cli-proxy: false
---

# Bare `+engineID+` skill

Say hello.
`)
			lock, data := compileBareInlineSkillWorkflow(t, workflowPath)
			require.Len(t, data.InlineSkills, 1)

			assert.NotContains(t, lock, "--skill", "pi skill flags must not leak into other engines")
			assert.NotContains(t, lock, "pi-agent-dir")
			assert.NotContains(t, lock, "--no-skills")
			assert.NotContains(t, lock, bareInlineSkillAmbientName)
			assert.Contains(t, lock, `GH_AW_SKILL_DIR: "`+GetEngineSkillDir(engineID)+`"`, "existing skill restoration should be unchanged")

			engine, err := NewEngineRegistry().GetEngine(engineID)
			require.NoError(t, err)
			withoutSkills := *data
			withoutSkills.InlineSkills = nil
			render := func(d *WorkflowData) string {
				var rendered []string
				for _, step := range engine.GetExecutionSteps(d, "/tmp/gh-aw/agent-stdio.log") {
					rendered = append(rendered, strings.Join(step, "\n"))
				}
				return strings.Join(rendered, "\n")
			}
			assert.Equal(t, render(&withoutSkills), render(data), "inline skills should not change %s execution", engineID)
			assert.True(t, slices.ContainsFunc(strings.Split(lock, "\n"), func(line string) bool {
				return strings.Contains(line, "{{#runtime-import .github/workflows/shared/reporting.md}}")
			}), "prompt should still runtime-import the reporting skill source")
		})
	}
}
