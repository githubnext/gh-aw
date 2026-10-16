//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectionDefaultRepoConfig(t *testing.T) {
	for _, value := range []string{"true", "false", `"false"`, "null", "0"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, RepoConfigFileName)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `{"default_detection_continue_on_error":%s}`, value), 0600))
			cfg, err := LoadRepoConfig(root)
			if value != "true" && value != "false" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg.DefaultDetectionContinueOnError)
			assert.Equal(t, value == "true", *cfg.DefaultDetectionContinueOnError)
		})
	}
	cfg, err := LoadRepoConfig(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, cfg.DefaultDetectionContinueOnError)
}

func TestDetectionDefaultPrecedence(t *testing.T) {
	t.Setenv("GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR", "invalid")
	for _, external := range []bool{false, true} {
		for _, tc := range []struct {
			name     string
			repo     *bool
			workflow string
			imported string
			want     string
			runtime  bool
		}{
			{name: "builtin/runtime", want: "${{ steps.detection_mode.outputs.continue_on_error == 'true' }}", runtime: true},
			{name: "repo false", repo: new(false), want: `"false"`},
			{name: "repo true", repo: new(true), want: `"true"`},
			{name: "explicit false", repo: new(true), workflow: "false", want: `"false"`},
			{name: "explicit true", repo: new(false), workflow: "true", want: `"true"`},
			{name: "import false", repo: new(true), imported: "false", want: `"false"`},
			{name: "import true", repo: new(false), imported: "true", want: `"true"`},
			{name: "workflow wins import", repo: new(false), workflow: "true", imported: "false", want: `"true"`},
			{name: "workflow false wins import", repo: new(true), workflow: "false", imported: "true", want: `"false"`},
			{name: "expression wins", repo: new(false), workflow: "${{ inputs.warn }}", want: "${{ inputs.warn }}"},
			{name: "imported expression wins", repo: new(false), imported: "${{ inputs.warn }}", want: "${{ inputs.warn }}"},
		} {
			t.Run(fmt.Sprintf("%t/%s", external, tc.name), func(t *testing.T) {
				dir := testutil.TempDir(t, "detection-default")
				source := "---\non:\n  workflow_dispatch:\n    inputs:\n      warn:\n        type: boolean\n        default: true\npermissions:\n  contents: read\nengine: claude\nstrict: false\n"
				if tc.imported != "" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\nsafe-outputs:\n  threat-detection:\n    continue-on-error: "+tc.imported+"\n---\n"), 0600))
					source += "imports:\n  - shared.md\n"
				}
				source += fmt.Sprintf("features:\n  gh-aw-detection: %t\nsafe-outputs:\n  create-issue:\n", external)
				if tc.workflow != "" {
					source += "  threat-detection:\n    continue-on-error: " + tc.workflow + "\n"
				}
				source += "---\nCreate an issue.\n"
				path := filepath.Join(dir, "test.md")
				require.NoError(t, os.WriteFile(path, []byte(source), 0600))
				c := NewCompiler()
				c.repoConfigLoaded = true
				c.repoConfig = &RepoConfig{DefaultDetectionContinueOnError: tc.repo}
				require.NoError(t, c.CompileWorkflow(path))
				content, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
				require.NoError(t, err)
				detection := extractJobSection(string(content), "detection")
				require.NotEmpty(t, detection)
				assert.Contains(t, detection, "GH_AW_DETECTION_CONTINUE_ON_ERROR: "+tc.want)
				assert.Equal(t, tc.runtime, strings.Contains(detection, "id: detection_mode"))
				assert.Equal(t, tc.runtime, strings.Contains(detection, "vars.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR"))
				assert.Equal(t, tc.runtime, strings.Contains(detection, "steps.detection_mode.outcome != 'failure'"))
				if !tc.runtime {
					assert.NotContains(t, detection, "steps.detection_mode")
				}
				if actionlint, err := exec.LookPath("actionlint"); err == nil {
					// Match gh aw lint's existing compatibility allowance for Actions concurrency queues.
					output, err := exec.Command(actionlint, "-shellcheck=", "-pyflakes=", "-ignore", `unexpected key "queue" for "concurrency" section`, stringutil.MarkdownToLockFile(path)).CombinedOutput()
					require.NoError(t, err, "%s", output)
				} else {
					t.Logf("actionlint unavailable; structural step-reference assertions still run: %v", err)
				}
				// All consumers must receive the same mode, not independent defaults.
				for line := range strings.SplitSeq(detection, "\n") {
					if strings.Contains(line, "GH_AW_DETECTION_CONTINUE_ON_ERROR:") {
						assert.Equal(t, "GH_AW_DETECTION_CONTINUE_ON_ERROR: "+tc.want, strings.TrimSpace(line))
					}
				}
			})
		}
	}
}

func TestDetectionDefaultDisabled(t *testing.T) {
	for _, setting := range []string{"false", "{engine: false}"} {
		dir := testutil.TempDir(t, "detection-disabled-default")
		path := filepath.Join(dir, "test.md")
		source := "---\non: workflow_dispatch\nengine: claude\nstrict: false\nsafe-outputs:\n  create-issue:\n  threat-detection: " + setting + "\n---\nTest.\n"
		require.NoError(t, os.WriteFile(path, []byte(source), 0600))
		c := NewCompiler()
		c.repoConfigLoaded = true
		c.repoConfig = &RepoConfig{DefaultDetectionContinueOnError: new(false)}
		require.NoError(t, c.CompileWorkflow(path))
		content, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
		require.NoError(t, err)
		assert.Empty(t, extractJobSection(string(content), "detection"))
		assert.NotContains(t, string(content), "GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR")
	}
}

// Smoke-test blocking mode with the real detector conclusion and a gated write.
// Resolver edge cases are covered by resolve_detection_continue_on_error.test.cjs.
func TestDetectionDefaultExternalPublicationGate(t *testing.T) {
	binary := os.Getenv("GH_AW_TEST_THREAT_DETECT_BINARY")
	if binary == "" {
		t.Skip("set GH_AW_TEST_THREAT_DETECT_BINARY to the verified pinned native threat-detect binary")
	}
	binDir := t.TempDir()
	require.NoError(t, os.Symlink(binary, filepath.Join(binDir, "threat-detect")))
	dir := testutil.TempDir(t, "detection-publication-default")
	path := filepath.Join(dir, "test.md")
	source := "---\non: workflow_dispatch\nengine: claude\nstrict: false\nfeatures:\n  gh-aw-detection: true\nsafe-outputs:\n  create-issue:\n---\nTest.\n"
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	c := NewCompiler()
	c.repoConfigLoaded = true
	c.repoConfig = &RepoConfig{}
	require.NoError(t, c.CompileWorkflow(path))
	content, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
	require.NoError(t, err)
	var compiled struct {
		Jobs map[string]struct {
			If    string `yaml:"if"`
			Steps []struct {
				ID              string `yaml:"id"`
				ContinueOnError string `yaml:"continue-on-error"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(content, &compiled))
	gate := compiled.Jobs["safe_outputs"].If
	require.NotEmpty(t, gate)
	var conclusionContinueOnError string
	for _, step := range compiled.Jobs["detection"].Steps {
		if step.ID == "detection_conclusion" {
			conclusionContinueOnError = step.ContinueOnError
		}
	}
	require.NotEmpty(t, conclusionContinueOnError)
	jsDir, err := filepath.Abs("../../actions/setup/js")
	require.NoError(t, err)
	wrapper, err := filepath.Abs("../../actions/setup/sh/conclude_threat_detection.sh")
	require.NoError(t, err)
	for _, threat := range []bool{false, true} {
		t.Run(fmt.Sprintf("threat=%t", threat), func(t *testing.T) {
			runDir := t.TempDir()
			result := filepath.Join(runDir, "detection_result.json")
			marker := filepath.Join(runDir, "published")
			require.NoError(t, os.WriteFile(result, fmt.Appendf(nil, `{"prompt_injection":%t,"secret_leak":false,"malicious_patch":false,"reasons":[]}`, threat), 0600))
			script := `
const assert = require('assert');
const fs = require('fs');
const [jsDir, wrapper, result, marker, gate, continueOnError, expectedStatus] = process.argv.slice(1);
const { resolveDetectionContinueOnError } = require(jsDir + '/resolve_detection_continue_on_error.cjs');
const coe = resolveDetectionContinueOnError(process.env.GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR);
assert.strictEqual(coe, false);
process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = String(coe);
const execution = require('child_process').spawnSync('bash', [wrapper, result], {env: process.env, encoding:'utf8'});
if (execution.error) throw execution.error;
assert.strictEqual(execution.status, Number(expectedStatus), execution.stdout + execution.stderr);
const steps = {detection_mode:{outputs:{continue_on_error:String(coe)}}};
const evaluate = expression => eval(expression.replace(/^\s*\$\{\{\s*/, '').replace(/\s*\}\}\s*$/, ''));
const failed = execution.status !== 0 && !evaluate(continueOnError);
const needs = {agent:{result:'success'}, detection:{result:failed?'failure':'success'}};
const always = () => true, cancelled = () => false, success = () => true;
if (evaluate(gate)) fs.writeFileSync(marker, 'published');
`
			status := "0"
			if threat {
				status = "1"
			}
			cmd := exec.Command("node", "-e", script, jsDir, wrapper, result, marker, gate, conclusionContinueOnError, status)
			cmd.Env = append(os.Environ(),
				"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GH_AW_DEFAULT_DETECTION_CONTINUE_ON_ERROR=false",
				"RUN_DETECTION=true",
				"DETECTION_AGENTIC_EXECUTION_OUTCOME=success",
				"THREAT_DETECT_INSTALL_OUTCOME=success",
				"GITHUB_OUTPUT="+filepath.Join(runDir, "output"),
				"GITHUB_ENV="+filepath.Join(runDir, "env"),
				"GITHUB_STEP_SUMMARY="+filepath.Join(runDir, "summary"),
			)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", out)
			if threat {
				assert.NoFileExists(t, marker, "blocking detection must prevent publication")
			} else {
				assert.FileExists(t, marker, "clean detection must allow publication")
			}
		})
	}
}
