//go:build !integration

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeModelMetadataArtifactRoundTrip(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for the runtime metadata transport regression")
	}
	scripts, err := filepath.Abs(filepath.Join("..", "..", "actions", "setup", "js"))
	require.NoError(t, err)
	for _, test := range []struct{ status, model string }{
		{"observed", "gemini-3.8-flash"},
		{"substituted", "claude-sonnet-5"},
		{"unavailable", ""},
	} {
		t.Run(test.status, func(t *testing.T) {
			root := t.TempDir()
			// Exercise the real runner writer, artifact copy, conclusion collector,
			// session projection, and Go reader without invoking GitHub APIs.
			command := exec.Command(node, "-e", `
const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const [scripts, root, status, model] = process.argv.slice(1);
global.core = { warning() {}, info() {} };
const { recordCopilotModelSelection, persistCopilotModelSelection } = require(path.join(scripts, "copilot_model_selection.cjs"));
const { persistAgentAwInfoCopy } = require(path.join(scripts, "parse_token_usage.cjs"));
const { writeUnifiedSession } = require(path.join(scripts, "unified_session.cjs"));
const runner = path.join(root, "runner");
const conclusion = path.join(root, "conclusion");
fs.mkdirSync(runner, { recursive: true });
const configured = "gemini-3.8-flash?effort=high";
const activation = { engine_id: "copilot", model: configured };
fs.writeFileSync(path.join(runner, "aw_info.json"), JSON.stringify(activation));
recordCopilotModelSelection({ status, configured_model: configured, resolved_model: "gemini-3.8-flash", effective_model: model }, { GH_AW_TMP_DIR: runner });
persistCopilotModelSelection({ GH_AW_TMP_DIR: runner });
persistAgentAwInfoCopy(runner);
// Upload/download only the runner-controlled metadata path; activation retains
// the configured snapshot and an agent file must not establish attribution.
fs.mkdirSync(path.join(conclusion, "runtime"), { recursive: true });
fs.mkdirSync(path.join(conclusion, "agent"), { recursive: true });
fs.copyFileSync(path.join(runner, "runtime/aw_info.json"), path.join(conclusion, "runtime/aw_info.json"));
fs.writeFileSync(path.join(conclusion, "aw_info.json"), JSON.stringify(activation));
fs.writeFileSync(path.join(conclusion, "agent/aw_info.json"), JSON.stringify({ model: "forged-model", model_selection: { status: "observed", effective_model: "forged-model" } }));
const collectScript = fs.readFileSync(path.join(scripts, "../sh/collect_usage_artifact_files.sh"), "utf8").replaceAll("/tmp/gh-aw", conclusion);
execFileSync("bash", ["-c", collectScript], { stdio: "pipe" });
const collected = JSON.parse(fs.readFileSync(path.join(conclusion, "usage/aw_info.json"), "utf8"));
if (collected.model !== model || collected.model_selection.status !== status) throw new Error("runtime attribution lost during artifact collection");
writeUnifiedSession({ rootDir: conclusion, engine: "copilot", warn() {} });
`, scripts, root, test.status, test.model)
			output, err := command.CombinedOutput()
			require.NoError(t, err, "%s", output)
			conclusion := filepath.Join(root, "conclusion")
			info := loadAwInfoForEngineConfig(conclusion)
			require.NotNil(t, info)
			require.Equal(t, test.status, info.ModelSelection.Status)
			require.Equal(t, test.model, info.Model)
			cachePath := filepath.Join(root, "logs.jsonl")
			require.NoError(t, newCachedLogsJSONLWriter(cachePath).AppendAudit(ProcessedRun{
				Run: WorkflowRun{DatabaseID: 42, LogsPath: conclusion},
			}))
			cachedJSON, err := os.ReadFile(cachePath)
			require.NoError(t, err)
			var record cachedLogsJSONLRecord
			require.NoError(t, json.Unmarshal(cachedJSON, &record))
			require.Equal(t, test.model, record.Run.Model)
			require.Equal(t, test.model, record.Run.AwInfo.Model)
			require.Equal(t, test.status, record.Run.AwInfo.ModelSelection.Status)
			result := resolveEffectiveModelAttribution(conclusion, &AwInfo{Model: "configured-snapshot"}, nil, nil)
			require.Equal(t, test.model, result.Model)
			require.Equal(t, "gemini-3.8-flash", result.RequestedModel)
			require.NoError(t, os.RemoveAll(filepath.Join(conclusion, "runtime")))
			require.NoError(t, os.Remove(filepath.Join(conclusion, "aw_info.json")))
			// Usage-only downloads retain selection status and intentionally empty models.
			session, found, err := readSessionModelRouting(conclusion)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, test.status, session.awInfo().ModelSelection.Status)
			merged := mergeSessionModelRoutingAwInfo(conclusion, &AwInfo{Model: "configured-snapshot"})
			require.Equal(t, test.model, merged.Model)
			require.Equal(t, test.status, merged.ModelSelection.Status)
		})
	}
}

func TestUntrustedAgentModelSelectionDoesNotEstablishAttribution(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "agent"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "agent/aw_info.json"),
		[]byte(`{"engine_id":"copilot","model":"forged","model_selection":{"status":"observed","effective_model":"forged"}}`), 0600))
	info := loadAwInfoForEngineConfig(root)
	require.NotNil(t, info)
	require.Nil(t, info.ModelSelection)
	require.Empty(t, info.Model)
}

func TestRuntimeWorkflowInfoProvenance(t *testing.T) {
	for _, test := range []struct {
		name, component, phase, path string
		trusted                      bool
	}{
		{"runner", "workflow", "agent", "runtime/aw_info.json", true},
		{"agent component", "agent", "agent", "runtime/aw_info.json", false},
		{"wrong phase", "workflow", "detection", "runtime/aw_info.json", false},
		{"agent path", "workflow", "agent", "agent/aw_info.json", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			record := fmt.Sprintf(`{"type":"workflow.info","data":{"model":"","modelSelection":{"status":"unavailable","effectiveModel":""}},"provenance":{"component":%q,"phase":%q,"path":%q}}`+"\n",
				test.component, test.phase, test.path)
			require.NoError(t, os.WriteFile(filepath.Join(root, "aw_session.jsonl"), []byte(record), 0600))
			_, found, err := readSessionModelRouting(root)
			require.NoError(t, err)
			require.Equal(t, test.trusted, found)
		})
	}
}
