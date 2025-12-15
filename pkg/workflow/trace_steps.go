package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"gopkg.in/yaml.v3"
)

var traceStepsLog = logger.New("workflow:trace_steps")

// GenerateTraceSetupStep generates the step to initialize trace capture
func GenerateTraceSetupStep(yaml *strings.Builder) {
	traceStepsLog.Print("Generating trace setup step")
	
	yaml.WriteString("      - name: Setup Trace Capture\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p trace/{diffs,tools,summaries}\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Initialize manifest.json\n")
	yaml.WriteString("          cat > trace/manifest.json <<'EOF_MANIFEST'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"run_id\": \"${{ github.run_id }}\",\n")
	yaml.WriteString("            \"workflow\": \"${{ github.workflow }}\",\n")
	yaml.WriteString("            \"engine\": \"${ENGINE:-copilot}\",\n")
	yaml.WriteString("            \"repo_sha\": \"${{ github.sha }}\",\n")
	yaml.WriteString("            \"created_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\n")
	yaml.WriteString("            \"trace_version\": \"v1\"\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF_MANIFEST\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Initialize empty checkpoints.jsonl\n")
	yaml.WriteString("          : > trace/checkpoints.jsonl\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          echo \"✅ Trace capture initialized\"\n")
}

// GenerateReplaySetupStep generates the step to download and restore from replay
func GenerateReplaySetupStep(yaml *strings.Builder) {
	traceStepsLog.Print("Generating replay setup step")
	
	yaml.WriteString("      - name: Replay Setup\n")
	yaml.WriteString("        if: ${{ inputs.replay_run_id != '' }}\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"📥 Downloading trace from run: ${{ inputs.replay_run_id }}\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Download trace artifact from previous run\n")
	yaml.WriteString("          gh run download \"${{ inputs.replay_run_id }}\" -n trace -D replay_in || {\n")
	yaml.WriteString("            echo \"❌ Failed to download trace artifact\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Copy prior trace into current trace folder\n")
	yaml.WriteString("          cp -r replay_in/trace/* trace/ || {\n")
	yaml.WriteString("            echo \"❌ Failed to copy replay trace\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          echo \"✅ Downloaded replay trace\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Restore repo SHA from manifest\n")
	yaml.WriteString("          REPO_SHA=$(python3 -c \"import json; print(json.load(open('trace/manifest.json'))['repo_sha'])\")\n")
	yaml.WriteString("          echo \"🔄 Restoring repo to SHA: $REPO_SHA\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          git checkout \"$REPO_SHA\" || {\n")
	yaml.WriteString("            echo \"❌ Failed to checkout SHA: $REPO_SHA\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          echo \"✅ Repository restored to checkpoint state\"\n")
}

// GenerateTraceJobSummaryStep generates the step to render checkpoint timeline
func GenerateTraceJobSummaryStep(yaml *strings.Builder) {
	traceStepsLog.Print("Generating trace job summary step")
	
	yaml.WriteString("      - name: Render Checkpoint Timeline\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          # Render checkpoint timeline to Job Summary\n")
	yaml.WriteString("          python3 ./scripts/render_trace_summary.py trace > \"$GITHUB_STEP_SUMMARY\"\n")
}

// GenerateTraceArtifactUploadStep generates the step to upload trace artifact
func GenerateTraceArtifactUploadStep(yaml *strings.Builder) {
	traceStepsLog.Print("Generating trace artifact upload step")
	
	yaml.WriteString("      - name: Upload Trace Artifact\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: trace\n")
	yaml.WriteString("          path: trace\n")
	yaml.WriteString("          retention-days: 30\n")
	yaml.WriteString("          if-no-files-found: warn\n")
}

// InjectReplayInputsIntoOn injects replay workflow_dispatch inputs into the on: section
func InjectReplayInputsIntoOn(onYAML string) (string, error) {
	traceStepsLog.Print("Injecting replay inputs into on: section")
	
	// Parse the on section
	var onMap map[string]any
	if err := yaml.Unmarshal([]byte(onYAML), &onMap); err != nil {
		return onYAML, fmt.Errorf("failed to parse on section: %w", err)
	}
	
	// Get the "on" key
	onTriggers, ok := onMap["on"].(map[string]any)
	if !ok {
		// Try without "on" wrapper
		onTriggers = onMap
	}
	
	// Check if workflow_dispatch exists
	workflowDispatch, hasWorkflowDispatch := onTriggers["workflow_dispatch"]
	
	// Create replay inputs
	replayInputs := map[string]any{
		"replay_run_id": map[string]any{
			"description": "Run ID to replay from (leave empty for normal run)",
			"required":    false,
			"type":        "string",
		},
		"start_checkpoint": map[string]any{
			"description": "Checkpoint ID to start from (e.g., c005)",
			"required":    false,
			"type":        "string",
		},
		"tool_mode": map[string]any{
			"description": "Tool execution mode (cached or live)",
			"required":    false,
			"type":        "choice",
			"options":     []string{"cached", "live"},
			"default":     "cached",
		},
	}
	
	if !hasWorkflowDispatch {
		// Add workflow_dispatch with replay inputs
		onTriggers["workflow_dispatch"] = map[string]any{
			"inputs": replayInputs,
		}
	} else {
		// Merge replay inputs into existing workflow_dispatch
		switch wd := workflowDispatch.(type) {
		case map[string]any:
			// workflow_dispatch has configuration
			if inputs, hasInputs := wd["inputs"].(map[string]any); hasInputs {
				// Merge with existing inputs
				for k, v := range replayInputs {
					if _, exists := inputs[k]; !exists {
						inputs[k] = v
					}
				}
			} else {
				// No inputs yet, add them
				wd["inputs"] = replayInputs
			}
		case nil:
			// workflow_dispatch: null, replace with inputs
			onTriggers["workflow_dispatch"] = map[string]any{
				"inputs": replayInputs,
			}
		}
	}
	
	// Convert back to YAML
	var result map[string]any
	if _, hasOnKey := onMap["on"]; hasOnKey {
		result = map[string]any{"on": onTriggers}
	} else {
		result = map[string]any{"on": onTriggers}
	}
	
	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return onYAML, fmt.Errorf("failed to marshal modified on section: %w", err)
	}
	
	return string(yamlBytes), nil
}

// ShouldEnableTraceCapture determines if trace capture should be enabled for a workflow
func ShouldEnableTraceCapture(data *WorkflowData) bool {
	// Enable trace capture if:
	// 1. Workflow has an agent engine (copilot, claude, codex, custom)
	// 2. Trace is not explicitly disabled in frontmatter
	
	if data == nil {
		return false
	}
	
	// Check if trace is explicitly disabled via Features map
	if data.Features != nil {
		if trace, ok := data.Features["trace"]; ok && !trace {
			traceStepsLog.Print("Trace capture explicitly disabled in frontmatter")
			return false
		}
	}
	
	// Enable for all agent engines (AI field contains the engine name)
	if data.AI != "" {
		traceStepsLog.Printf("Enabling trace capture for engine: %s", data.AI)
		return true
	}
	
	// Also check EngineConfig for custom engines
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		traceStepsLog.Printf("Enabling trace capture for custom engine: %s", data.EngineConfig.ID)
		return true
	}
	
	return false
}
