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
				source := "---\non: workflow_dispatch\npermissions:\n  contents: read\nengine: claude\nstrict: false\n"
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

// Execute the real inline conclusion with isolated in-process artifact reads,
// then evaluate the generated write-job gate against the resulting job status.
func TestDetectionDefaultPublicationGate(t *testing.T) {
	testDetectionDefaultPublicationGate(t, "")
}

func TestDetectionDefaultExternalPublicationGate(t *testing.T) {
	binary := os.Getenv("GH_AW_TEST_THREAT_DETECT_BINARY")
	if binary == "" {
		t.Skip("set GH_AW_TEST_THREAT_DETECT_BINARY to the verified pinned native threat-detect binary")
	}
	binDir := t.TempDir()
	require.NoError(t, os.Symlink(binary, filepath.Join(binDir, "threat-detect")))
	testDetectionDefaultPublicationGate(t, binDir)
}

func testDetectionDefaultPublicationGate(t *testing.T, binDir string) {
	t.Helper()
	dir := testutil.TempDir(t, "detection-publication-default")
	path := filepath.Join(dir, "test.md")
	source := fmt.Sprintf("---\non: workflow_dispatch\nengine: claude\nstrict: false\nfeatures:\n  gh-aw-detection: %t\nsafe-outputs:\n  create-issue:\n---\nTest.\n", binDir != "")
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	require.NoError(t, NewCompiler().CompileWorkflow(path))
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
	for _, mode := range []string{"", "true", "false", "invalid"} {
		for _, outcome := range []string{"clean", "threat", "invalid", "failure", "cancelled", "failed_clean", "missing", "skipped"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				wantPublish := outcome == "skipped" || (mode != "invalid" && (mode != "false" || outcome == "clean"))
				script := `
const assert = require('assert');
const fs = require('fs');
const jsDir = process.argv[1], raw = process.argv[2], scenario = process.argv[3], gate = process.argv[4];
const { resolveDetectionContinueOnError } = require(jsDir + '/resolve_detection_continue_on_error.cjs');
let failed = false, configurationFailed = false, outputs = {};
global.core = { info(){}, warning(){}, error(){}, exportVariable(){}, setOutput(k,v){outputs[k]=v}, setFailed(){failed=true} };
let coe = false;
if (scenario !== 'skipped') {
  try { coe = resolveDetectionContinueOnError(raw); } catch (err) { assert.match(err.message, /must be "true" or "false"/); configurationFailed = true; }
}
process.env.RUN_DETECTION = scenario === 'skipped' ? 'false' : 'true';
process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = String(coe);
process.env.DETECTION_AGENTIC_EXECUTION_OUTCOME = scenario==='failed_clean'?'failure':(['failure','cancelled'].includes(scenario) ? scenario : 'success');
const verdict = JSON.stringify({prompt_injection:scenario==='threat',secret_leak:false,malicious_patch:false,reasons:[]});
const exists = fs.existsSync, read = fs.readFileSync;
fs.existsSync = p => String(p).startsWith('/tmp/gh-aw/threat-detection/') ? (!['missing','failure'].includes(scenario) && String(p).endsWith('detection.log')) : exists(p);
fs.readFileSync = (p,...args) => String(p).endsWith('/threat-detection/detection.log') ? (scenario==='invalid'?'invalid verdict':'THREAT_DETECTION_RESULT:'+verdict) : read(p,...args);
(async () => {
  if (process.argv[6]) {
    const dir = process.argv[7], result = dir + '/detection_result.json', output = dir + '/output';
    if (!['missing','failure'].includes(scenario)) fs.writeFileSync(result, scenario === 'invalid' ? '{invalid' : verdict);
    const execution = require('child_process').spawnSync('bash', [process.argv[8], result], {env: {...process.env,
      PATH:process.argv[6]+':'+process.env.PATH, GITHUB_OUTPUT:output, GITHUB_ENV:dir+'/env', GITHUB_STEP_SUMMARY:dir+'/summary',
      THREAT_DETECT_INSTALL_OUTCOME:scenario==='skipped'?'skipped':'success'
    }, encoding:'utf8'});
    if (execution.error) throw execution.error;
    failed = execution.status !== 0;
    outputs = Object.fromEntries(fs.readFileSync(output, 'utf8').trim().split('\n').map(line => {
      const idx = line.indexOf('='); return [line.slice(0,idx), line.slice(idx+1)];
    }));
  } else {
    await require(jsDir + '/parse_threat_detection_results.cjs').main();
  }
  const steps = {detection_mode:{outputs:{continue_on_error:String(coe)}}};
  const unwrap = value => value.replace(/^\s*\$\{\{\s*/, '').replace(/\s*\}\}\s*$/, '');
  const stepContinueOnError = eval(unwrap(process.argv[9]));
  const jobFailed = configurationFailed || (failed && !stepContinueOnError);
  const needs = {agent:{result:'success'}, detection:{result:jobFailed?'failure':'success'}};
  const always = () => true, cancelled = () => false, success = () => true;
  const condition = gate.replace(/^\s*\$\{\{\s*/, '').replace(/\s*\}\}\s*$/, '');
  let writes = 0;
  if (eval(condition)) { writes++; }
  assert.equal(writes, Number(process.argv[5]));
  if (scenario==='skipped') assert.equal(outputs.conclusion, 'skipped');
  if (modeIsBlocking() && ['failure','cancelled','failed_clean','missing'].includes(scenario)) assert.equal(outputs.reason, 'agent_failure');
  if (scenario==='threat') assert.equal(outputs.reason, 'threat_detected');
  if (scenario==='invalid') assert.equal(outputs.reason, 'parse_error');
  function modeIsBlocking() { return raw === 'false'; }
})().catch(err => {console.error(err);process.exitCode=1});
`
				wantWrites := "0"
				if wantPublish {
					wantWrites = "1"
				}
				cmd := exec.Command("node", "-e", script, jsDir, mode, outcome, gate, wantWrites, binDir, t.TempDir(), wrapper, conclusionContinueOnError)
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", out)
			})
		}
	}
}
