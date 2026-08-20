// Package parser provides functions for parsing and processing workflow markdown files.
// import_field_extractor_engine.go implements extraction of engine and scalar/builder
// configuration fields (engine, max-turns, mcp-servers, safe-outputs, steps, runtimes,
// services, network, sandbox, permissions, secret-masking) from imported frontmatter.
package parser

import (
	"encoding/json"
	"strings"
)

// extractEngineConfig extracts engine-related settings from the imported frontmatter map
// and accumulates them. Engine configs with only `mcp` sub-keys (no `id` or `runtime`)
// are not counted as engine specifications — they carry MCP gateway settings only.
//
// Side effects: acc.engines, acc.mergedEngineMCPToolTimeout,
// acc.mergedEngineMCPSessionTimeout, acc.mergedEngineModel.
func (acc *importAccumulator) extractEngineConfig(fm map[string]any, fullPath string) {
	if modelStr, ok := fm["model"].(string); ok && modelStr != "" && acc.mergedEngineModel == "" {
		acc.mergedEngineModel = modelStr
		parserLog.Printf("Extracted top-level model preference from import %s: %s", fullPath, modelStr)
	}

	engineVal, hasEngine := fm["engine"]
	if !hasEngine {
		return
	}
	parserLog.Printf("Found engine config in import: %s", fullPath)

	switch v := engineVal.(type) {
	case string:
		// String engine (e.g. "copilot") — always counts as an engine spec.
		if engineJSON, merr := json.Marshal(v); merr == nil {
			acc.engines = append(acc.engines, string(engineJSON))
		}
	case map[string]any:
		// Object engine — extract engine.mcp.* settings first, then decide
		// whether to add to engines based on whether an engine ID is present.
		if mcpVal, hasMCP := v["mcp"]; hasMCP {
			acc.extractEngineMCPSettings(mcpVal, fullPath)
		}
		// Only add to engines list if this config specifies an actual engine
		// (i.e. it carries an 'id' or 'runtime' field). Configs with only
		// 'model' or 'mcp' settings are preferences, not engine selections,
		// and must not trigger the "multiple engine fields" validation error.
		_, hasID := v["id"]
		_, hasRuntime := v["runtime"]
		if hasID || hasRuntime {
			if engineJSON, merr := json.Marshal(v); merr == nil {
				acc.engines = append(acc.engines, string(engineJSON))
			}
		} else {
			// No engine ID or runtime — this is a model/MCP-only preference.
			// Extract the model hint (first-wins) so it can be applied to the
			// resolved engine after all imports are processed.
			if modelStr, ok := v["model"].(string); ok && modelStr != "" {
				if acc.mergedEngineModel == "" {
					acc.mergedEngineModel = modelStr
					parserLog.Printf("Extracted engine.model preference from import %s: %s", fullPath, modelStr)
				}
			}
		}
	default:
		// Unexpected type — marshal and add to preserve existing behavior.
		if engineJSON, merr := json.Marshal(engineVal); merr == nil {
			acc.engines = append(acc.engines, string(engineJSON))
		}
	}
}

// extractEngineMCPSettings extracts engine.mcp.tool-timeout and engine.mcp.session-timeout
// from mcpVal (first-wins across all imports).
func (acc *importAccumulator) extractEngineMCPSettings(mcpVal any, fullPath string) {
	mcpMap, ok := mcpVal.(map[string]any)
	if !ok {
		return
	}
	if acc.mergedEngineMCPToolTimeout == "" {
		if ttStr, ok := mcpMap["tool-timeout"].(string); ok && ttStr != "" {
			acc.mergedEngineMCPToolTimeout = ttStr
			parserLog.Printf("Extracted engine.mcp.tool-timeout from import %s: %s", fullPath, ttStr)
		}
	}
	if acc.mergedEngineMCPSessionTimeout == "" {
		if stStr, ok := mcpMap["session-timeout"].(string); ok && stStr != "" {
			acc.mergedEngineMCPSessionTimeout = stStr
			parserLog.Printf("Extracted engine.mcp.session-timeout from import %s: %s", fullPath, stStr)
		}
	}
}

// extractConfigFields extracts scalar and builder-based configuration fields from the
// frontmatter map and writes them into the appropriate accumulator builders and slices.
//
// Side effects: acc.mergedMaxTurns, acc.mergedMaxToolDenials, acc.mergedMaxRuns, acc.mergedMaxAICredits,
// acc.mergedMaxDailyAICredits, acc.mcpServersBuilder,
// acc.safeOutputs, acc.mcpScripts, acc.stepsBuilder, acc.runtimesBuilder,
// acc.servicesBuilder, acc.networkBuilder, acc.permissionsBuilder,
// acc.secretMaskingBuilder.
func (acc *importAccumulator) extractConfigFields(fm map[string]any, fullPath string) {
	acc.extractFirstWinsJSONField(fm, fullPath, "max-turns", &acc.mergedMaxTurns)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-tool-denials", &acc.mergedMaxToolDenials)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-runs", &acc.mergedMaxRuns)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-turn-cache-misses", &acc.mergedMaxTurnCacheMisses)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-ai-credits", &acc.mergedMaxAICredits)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-daily-ai-credits", &acc.mergedMaxDailyAICredits)

	acc.appendJSONBuilderField(fm, "mcp-servers", "{}", &acc.mcpServersBuilder)
	acc.plugins = append(acc.plugins, parseStringSliceField(fm["plugins"], false)...)
	acc.appendJSONSliceField(fm, "safe-outputs", "{}", &acc.safeOutputs)
	acc.appendJSONSliceField(fm, "mcp-scripts", "{}", &acc.mcpScripts)
	acc.appendYAMLBuilderField(fm, "steps", &acc.stepsBuilder)
	acc.appendJSONBuilderField(fm, "runtimes", "{}", &acc.runtimesBuilder)
	acc.appendYAMLBuilderField(fm, "services", &acc.servicesBuilder)
	acc.appendJSONBuilderField(fm, "network", "{}", &acc.networkBuilder)
	acc.mergeSandboxAgentMounts(fm)
	acc.mergeSandboxAgentRuntimeInstall(fm)
	acc.appendJSONBuilderField(fm, "permissions", "{}", &acc.permissionsBuilder)
	acc.appendJSONBuilderField(fm, "secret-masking", "{}", &acc.secretMaskingBuilder)
}

func (acc *importAccumulator) mergeSandboxAgentMounts(fm map[string]any) {
	sandboxVal, hasSandbox := fm["sandbox"]
	if !hasSandbox {
		return
	}

	sandboxMap, ok := sandboxVal.(map[string]any)
	if !ok {
		return
	}

	agentVal, hasAgent := sandboxMap["agent"]
	if !hasAgent {
		return
	}

	agentMap, ok := agentVal.(map[string]any)
	if !ok {
		return
	}

	mountsVal, hasMounts := agentMap["mounts"]
	if !hasMounts {
		return
	}

	mounts, ok := mountsVal.([]any)
	if !ok {
		return
	}

	for _, mountVal := range mounts {
		mount, ok := mountVal.(string)
		if !ok || mount == "" {
			continue
		}
		if !acc.sandboxAgentMountsSet[mount] {
			acc.sandboxAgentMountsSet[mount] = true
			acc.sandboxAgentMounts = append(acc.sandboxAgentMounts, mount)
		}
	}
}

// mergeSandboxAgentRuntimeInstall extracts sandbox.agent.runtime-install from an
// imported workflow's frontmatter. False wins: if any import sets runtime-install
// to false the accumulated value becomes false and stays false.
func (acc *importAccumulator) mergeSandboxAgentRuntimeInstall(fm map[string]any) {
	// Already locked to false — no need to inspect further imports.
	if acc.sandboxAgentRuntimeInstall != nil && !*acc.sandboxAgentRuntimeInstall {
		return
	}

	sandboxVal, hasSandbox := fm["sandbox"]
	if !hasSandbox {
		return
	}
	sandboxMap, ok := sandboxVal.(map[string]any)
	if !ok {
		return
	}
	agentVal, hasAgent := sandboxMap["agent"]
	if !hasAgent {
		return
	}
	agentMap, ok := agentVal.(map[string]any)
	if !ok {
		return
	}
	riVal, hasRI := agentMap["runtime-install"]
	if !hasRI {
		return
	}
	ri, ok := riVal.(bool)
	if !ok {
		return
	}
	if !ri {
		f := false
		acc.sandboxAgentRuntimeInstall = &f
	} else if acc.sandboxAgentRuntimeInstall == nil {
		t := true
		acc.sandboxAgentRuntimeInstall = &t
	}
}

func (acc *importAccumulator) extractFirstWinsJSONField(fm map[string]any, fullPath, field string, target *string) {
	if *target != "" {
		return
	}
	fieldJSON, err := extractFieldJSONFromMap(fm, field, "")
	if err != nil || fieldJSON == "" || fieldJSON == "null" {
		return
	}
	*target = fieldJSON
	parserLog.Printf("Extracted %s from import: %s", field, fullPath)
}

func (acc *importAccumulator) appendJSONBuilderField(fm map[string]any, field, emptyValue string, builder *strings.Builder) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	builder.WriteString(content + "\n")
}

func (acc *importAccumulator) appendJSONSliceField(fm map[string]any, field, emptyValue string, target *[]string) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	*target = append(*target, content)
}

func (acc *importAccumulator) appendYAMLBuilderField(fm map[string]any, field string, builder *strings.Builder) {
	content, err := extractYAMLFieldFromMap(fm, field)
	if err != nil || content == "" {
		return
	}
	builder.WriteString(content + "\n")
}
