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

// collectSampleRepoTokens walks the workflow's checkout configs and returns
// a map of "owner/repo" -> token expression so that apply_samples.cjs can
// pick the right token when calling the GitHub REST API to resolve a PR head
// ref for a cross-repo sample.
//
// The keys are repository slugs as written in the workflow frontmatter; the
// value is the GitHub Actions expression that resolves to the token at
// runtime — for plain `github-token: ${{ secrets.X }}` checkouts this is the
// expression itself, and for `github-app:` checkouts it is the
// `${{ steps.checkout-app-token-N.outputs.token }}` reference for the same
// app-token step that the agent job's checkout uses. Entries whose checkout
// declares no auth are omitted; the driver falls back to GITHUB_TOKEN for
// those.
func collectSampleRepoTokens(configs []*CheckoutConfig) map[string]string {
	if len(configs) == 0 {
		return nil
	}
	cm := NewCheckoutManager(configs)
	tokens := make(map[string]string)
	for i, entry := range cm.ordered {
		repo := entry.key.repository
		if repo == "" {
			// The repo slug is unknown at compile time; emit a GitHub Actions
			// expression. The Actions runner expands ${{ github.repository }} to
			// "owner/repo" before apply_samples.cjs reads GH_AW_REPO_TOKENS, so
			// the runtime key matches the slug the driver looks up.
			repo = "${{ github.repository }}"
		}
		var token string
		switch {
		case entry.githubApp != nil:
			//nolint:gosec // G101: GitHub Actions expression template, not a hardcoded credential
			token = fmt.Sprintf("${{ steps.checkout-app-token-%d.outputs.token }}", i)
		case entry.token != "":
			token = entry.token
		default:
			continue
		}
		// First-seen wins so the per-repo entry from the user's frontmatter
		// takes precedence over later imported configs (CheckoutManager
		// already enforces this for merged entries; this guards against
		// distinct entries that share a repo but differ in path).
		if _, exists := tokens[repo]; !exists {
			tokens[repo] = token
		}
	}
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

// marshalRepoTokens returns a compact JSON object encoding of m, or nil for
// empty/nil maps so callers can skip emission. encoding/json sorts string-
// keyed map entries lexicographically, so the output is byte-stable across
// runs without an explicit sort step.
func marshalRepoTokens(m map[string]string) []byte {
	if len(m) == 0 {
		return nil
	}
	out, _ := json.Marshal(m) //nolint:jsonmarshalignoredeerror // marshaling a string map cannot fail
	return out
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

	// Build the per-repo token map so apply_samples.cjs can reach repos that
	// require a non-default token (cross-repo `checkout:` entries with their
	// own `github-token:` or `github-app:`).
	repoTokens := collectSampleRepoTokens(data.CheckoutConfigs)
	repoTokensPayload := marshalRepoTokens(repoTokens)

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
	// GITHUB_TOKEN is the fallback used by apply_samples.cjs when resolving a
	// pull-request head ref via the REST API for issue_comment / slash_command
	// events. For cross-repo samples whose target repository has its own
	// `checkout:` entry with `github-token:` or `github-app:`, the driver
	// prefers the matching token from GH_AW_REPO_TOKENS below.
	yaml.WriteString("          GITHUB_TOKEN: ${{ github.token }}\n")
	if repoTokensPayload != nil {
		yaml.WriteString("          GH_AW_REPO_TOKENS: |\n")
		for line := range strings.SplitSeq(string(repoTokensPayload), "\n") {
			fmt.Fprintf(yaml, "            %s\n", line)
		}
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -euo pipefail\n")
	yaml.WriteString("          mkdir -p \"$(dirname \"$GH_AW_AGENT_STDIO_LOG\")\"\n")
	yaml.WriteString("          node \"${{ runner.temp }}/gh-aw/actions/apply_samples.cjs\"\n")
}
