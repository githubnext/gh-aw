package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SampleEntry is the per-call payload consumed by apply_samples.cjs.
// Each entry corresponds to a single MCP `tools/call` invocation.
type SampleEntry struct {
	// Tool is the snake_case MCP tool name (e.g. "create_pull_request").
	Tool string `json:"tool"`
	// Arguments are passed verbatim as the MCP `tools/call` arguments.
	// Sample sidecar fields (e.g. `patch`) have already been stripped.
	Arguments map[string]any `json:"arguments"`
	// Sidecars carries fields stripped from Arguments that need out-of-band
	// pre-staging by the driver (e.g. `patch` for create_pull_request).
	Sidecars map[string]any `json:"sidecars,omitempty"`
}

// collectSampleEntries walks the safe-outputs config and flattens every
// configured `samples` entry into the order they will be sent to the MCP
// server. Iteration order is deterministic (sorted by struct field name) so
// that compiled YAML is stable across runs.
func collectSampleEntries(config *SafeOutputsConfig) []SampleEntry {
	if config == nil {
		return nil
	}

	fieldNames := make([]string, 0, len(safeOutputFieldMapping))
	for fieldName := range safeOutputFieldMapping {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	var entries []SampleEntry
	for _, fieldName := range fieldNames {
		toolName := safeOutputFieldMapping[fieldName]
		base := extractBaseSafeOutputConfig(config, fieldName)
		if base == nil || len(base.Samples) == 0 {
			continue
		}
		sidecarKeys := sampleSidecarFields[toolName]
		for _, sample := range base.Samples {
			args := make(map[string]any, len(sample))
			var sidecars map[string]any
			for k, v := range sample {
				if sidecarKeys[k] {
					if sidecars == nil {
						sidecars = make(map[string]any)
					}
					sidecars[k] = v
					continue
				}
				args[k] = v
			}
			entries = append(entries, SampleEntry{
				Tool:      toolName,
				Arguments: args,
				Sidecars:  sidecars,
			})
		}
	}
	return entries
}

// generateSamplesReplayStep emits the YAML that replaces the agentic
// `Execute coding agent` step when the hidden `gh aw compile --use-samples`
// flag is used. It spawns the safe-outputs MCP server over stdio and feeds it
// a `tools/call` for every collected sample, after pre-staging branches/patches
// for samples that carry them.
func (c *Compiler) generateSamplesReplayStep(yaml *strings.Builder, data *WorkflowData, logFile string) {
	entries := collectSampleEntries(data.SafeOutputs)
	compilerYamlLog.Printf("Generating samples replay step: entries=%d", len(entries))

	// Normalize a nil slice to an empty slice so json.Marshal emits "[]" not "null".
	// The driver rejects anything that isn't a JSON array; emitting "null" here
	// would crash the replay step with `GH_AW_SAMPLES must be a JSON array` for
	// workflows that opt into --use-samples but configure no samples (or whose
	// configured samples all live on disabled handlers).
	if entries == nil {
		entries = []SampleEntry{}
	}

	// Serialize entries to JSON for the driver. Always emit valid JSON even when
	// empty so the driver can produce a clear `no samples configured` message
	// rather than crashing on an empty env var.
	payload, err := json.Marshal(entries)
	if err != nil {
		// Should never happen for map[string]any payloads; fall back to empty
		// array so the workflow still compiles and the driver reports cleanly.
		compilerYamlLog.Printf("Warning: failed to marshal samples entries: %v", err)
		payload = []byte("[]")
	}

	yaml.WriteString("      - name: Replay safe-outputs samples (deterministic)\n")
	yaml.WriteString("        id: agentic_execution\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAMPLES: |\n")
	for line := range strings.SplitSeq(string(payload), "\n") {
		fmt.Fprintf(yaml, "            %s\n", line)
	}
	fmt.Fprintf(yaml, "          GH_AW_AGENT_STDIO_LOG: %s\n", logFile)
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS_CONFIG_PATH: ${{ runner.temp }}/gh-aw/safeoutputs/config.json\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ runner.temp }}/gh-aw/safeoutputs/outputs.jsonl\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -euo pipefail\n")
	yaml.WriteString("          mkdir -p \"$(dirname \"$GH_AW_AGENT_STDIO_LOG\")\"\n")
	yaml.WriteString("          node \"${{ runner.temp }}/gh-aw/actions/apply_samples.cjs\"\n")
}
