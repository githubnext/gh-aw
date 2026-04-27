package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var workflowBuilderLog = logger.New("workflow:workflow_builder")

// buildInitialWorkflowData creates the initial WorkflowData struct with basic fields populated
func (c *Compiler) buildInitialWorkflowData(
	result *parser.FrontmatterResult,
	toolsResult *toolsProcessingResult,
	engineSetup *engineSetupResult,
	importsResult *parser.ImportsResult,
) *WorkflowData {
	workflowBuilderLog.Print("Building initial workflow data")

	inlinedImports := resolveInlinedImports(result.Frontmatter)

	// When inlined-imports is true, agent file content is already inlined via ImportPaths → step 1b.
	// Clear AgentFile/AgentImportSpec so engines don't read it from disk separately at runtime.
	agentFile := importsResult.AgentFile
	agentImportSpec := importsResult.AgentImportSpec
	if inlinedImports {
		agentFile = ""
		agentImportSpec = ""
	}

	workflowData := &WorkflowData{
		Name:                  toolsResult.workflowName,
		FrontmatterName:       toolsResult.frontmatterName,
		FrontmatterYAML:       strings.Join(result.FrontmatterLines, "\n"),
		RawMarkdown:           result.Markdown,
		Description:           c.extractDescription(result.Frontmatter),
		Source:                c.extractSource(result.Frontmatter),
		Redirect:              c.extractRedirect(result.Frontmatter),
		TrackerID:             toolsResult.trackerID,
		ImportedFiles:         importsResult.ImportedFiles,
		ImportedMarkdown:      toolsResult.importedMarkdown, // Only imports WITH inputs
		ImportPaths:           toolsResult.importPaths,      // Import paths for runtime-import macros (imports without inputs)
		MainWorkflowMarkdown:  toolsResult.mainWorkflowMarkdown,
		IncludedFiles:         toolsResult.allIncludedFiles,
		ImportInputs:          importsResult.ImportInputs,
		Tools:                 toolsResult.tools,
		ParsedTools:           NewTools(toolsResult.tools),
		Runtimes:              toolsResult.runtimes,
		RunInstallScripts:     toolsResult.runInstallScripts,
		MarkdownContent:       toolsResult.markdownContent,
		AI:                    engineSetup.engineSetting,
		EngineConfig:          engineSetup.engineConfig,
		AgentFile:             agentFile,
		AgentImportSpec:       agentImportSpec,
		RepositoryImports:     importsResult.RepositoryImports,
		NetworkPermissions:    engineSetup.networkPermissions,
		SandboxConfig:         applySandboxDefaults(engineSetup.sandboxConfig, engineSetup.engineConfig),
		NeedsTextOutput:       toolsResult.needsTextOutput,
		ToolsTimeout:          toolsResult.toolsTimeout,
		ToolsStartupTimeout:   toolsResult.toolsStartupTimeout,
		TrialMode:             c.trialMode,
		TrialLogicalRepo:      c.trialLogicalRepoSlug,
		StrictMode:            c.strictMode,
		AllowActionRefs:       c.allowActionRefs,
		SecretMasking:         toolsResult.secretMasking,
		ParsedFrontmatter:     toolsResult.parsedFrontmatter,
		RawFrontmatter:        result.Frontmatter,
		ResolvedMCPServers:    toolsResult.resolvedMCPServers,
		HasExplicitGitHubTool: toolsResult.hasExplicitGitHubTool,
		ActionMode:            c.actionMode,
		InlinedImports:        inlinedImports,
		EngineConfigSteps:     engineSetup.configSteps,
	}

	// Populate checkout configs from parsed frontmatter.
	// Fall back to raw frontmatter parsing when full ParseFrontmatterConfig fails
	// (e.g. due to unrecognised tool config shapes like bash: ["*"]).
	if toolsResult.parsedFrontmatter != nil {
		workflowData.CheckoutConfigs = toolsResult.parsedFrontmatter.CheckoutConfigs
		workflowData.CheckoutDisabled = toolsResult.parsedFrontmatter.CheckoutDisabled
	} else if rawCheckout, ok := result.Frontmatter["checkout"]; ok {
		if checkoutValue, ok := rawCheckout.(bool); ok && !checkoutValue {
			workflowData.CheckoutDisabled = true
		} else if configs, err := ParseCheckoutConfigs(rawCheckout); err == nil {
			workflowData.CheckoutConfigs = configs
		}
	}

	// Merge checkout configs from imported shared workflows.
	// Imported configs are appended after the main workflow's configs so that the main
	// workflow's entries take precedence when CheckoutManager deduplicates by (repository, path).
	// checkout: false in the main workflow disables all checkout (including imports).
	if !workflowData.CheckoutDisabled && importsResult.MergedCheckout != "" {
		for line := range strings.SplitSeq(strings.TrimSpace(importsResult.MergedCheckout), "\n") {
			if line == "" {
				continue
			}
			var raw any
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				workflowBuilderLog.Printf("Failed to unmarshal imported checkout JSON: %v", err)
				continue
			}
			importedConfigs, err := ParseCheckoutConfigs(raw)
			if err != nil {
				workflowBuilderLog.Printf("Failed to parse imported checkout configs: %v", err)
				continue
			}
			workflowData.CheckoutConfigs = append(workflowData.CheckoutConfigs, importedConfigs...)
		}
	}

	// Populate check-for-updates flag: disabled when check-for-updates: false is set in frontmatter.
	if toolsResult.parsedFrontmatter != nil && toolsResult.parsedFrontmatter.UpdateCheck != nil {
		workflowData.UpdateCheckDisabled = !*toolsResult.parsedFrontmatter.UpdateCheck
	} else if rawVal, ok := result.Frontmatter["check-for-updates"]; ok {
		if boolVal, ok := rawVal.(bool); ok && !boolVal {
			workflowData.UpdateCheckDisabled = true
		}
	}

	// Populate stale-check flag: disabled when on.stale-check: false is set in frontmatter.
	if onVal, ok := result.Frontmatter["on"]; ok {
		if onMap, ok := onVal.(map[string]any); ok {
			if staleCheck, ok := onMap["stale-check"]; ok {
				if boolVal, ok := staleCheck.(bool); ok && !boolVal {
					workflowData.StaleCheckDisabled = true
				}
			}
		}
	}

	return workflowData
}

// resolveInlinedImports returns true if inlined-imports is enabled.
// It reads the value directly from the raw (pre-parsed) frontmatter map, which is always
// populated regardless of whether ParseFrontmatterConfig succeeded.
func resolveInlinedImports(rawFrontmatter map[string]any) bool {
	return ParseBoolFromConfig(rawFrontmatter, "inlined-imports", nil)
}

// extractYAMLSections extracts YAML configuration sections from frontmatter
func (c *Compiler) extractYAMLSections(frontmatter map[string]any, workflowData *WorkflowData) {
	workflowBuilderLog.Print("Extracting YAML sections from frontmatter")

	workflowData.On = c.extractTopLevelYAMLSection(frontmatter, "on")
	workflowData.HasDispatchItemNumber = extractDispatchItemNumber(frontmatter)
	workflowData.Permissions = c.extractPermissions(frontmatter)
	workflowData.Network = c.extractTopLevelYAMLSection(frontmatter, "network")
	workflowData.ConcurrencyJobDiscriminator = extractConcurrencyJobDiscriminator(frontmatter)
	workflowData.Concurrency = c.extractConcurrencySection(frontmatter)
	workflowData.RunName = c.extractTopLevelYAMLSection(frontmatter, "run-name")
	workflowData.Env = c.extractTopLevelYAMLSection(frontmatter, "env")
	workflowData.Features = c.extractFeatures(frontmatter)
	workflowData.If = c.extractIfCondition(frontmatter)

	// Extract timeout-minutes (canonical form)
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(frontmatter, "timeout-minutes")

	workflowData.RunsOn = c.extractTopLevelYAMLSection(frontmatter, "runs-on")
	// Extract runs-on-slim as a plain string (no YAML formatting needed)
	if v, ok := frontmatter["runs-on-slim"]; ok {
		if s, ok := v.(string); ok {
			workflowData.RunsOnSlim = s
		}
	}
	workflowData.Environment = c.extractTopLevelYAMLSection(frontmatter, "environment")
	workflowData.Container = c.extractTopLevelYAMLSection(frontmatter, "container")
	workflowData.Cache = c.extractTopLevelYAMLSection(frontmatter, "cache")
}

// extractConcurrencyJobDiscriminator reads the job-discriminator value from the
// frontmatter concurrency block without modifying the original map.
// Returns the discriminator expression string or empty string if not present.
func extractConcurrencyJobDiscriminator(frontmatter map[string]any) string {
	concurrencyRaw, ok := frontmatter["concurrency"]
	if !ok {
		return ""
	}
	concurrencyMap, ok := concurrencyRaw.(map[string]any)
	if !ok {
		return ""
	}
	discriminator, ok := concurrencyMap["job-discriminator"]
	if !ok {
		return ""
	}
	discriminatorStr, ok := discriminator.(string)
	if !ok {
		return ""
	}
	return discriminatorStr
}

// extractConcurrencySection extracts the workflow-level concurrency YAML section,
// stripping the gh-aw-specific job-discriminator field so it does not appear in
// the compiled lock file (which must be valid GitHub Actions YAML).
func (c *Compiler) extractConcurrencySection(frontmatter map[string]any) string {
	concurrencyRaw, ok := frontmatter["concurrency"]
	if !ok {
		return ""
	}
	concurrencyMap, ok := concurrencyRaw.(map[string]any)
	if !ok || len(concurrencyMap) == 0 {
		// String or empty format: serialize as-is (no job-discriminator possible)
		return c.extractTopLevelYAMLSection(frontmatter, "concurrency")
	}

	_, hasDiscriminator := concurrencyMap["job-discriminator"]
	if !hasDiscriminator {
		return c.extractTopLevelYAMLSection(frontmatter, "concurrency")
	}

	// Build a copy of the concurrency map without job-discriminator for serialization.
	// Use len(concurrencyMap) for capacity: at most one entry (job-discriminator) will be
	// omitted, so this is a slight over-allocation that avoids a subtle negative-capacity
	// edge case if job-discriminator were the only key.
	cleanMap := make(map[string]any, len(concurrencyMap))
	for k, v := range concurrencyMap {
		if k != "job-discriminator" {
			cleanMap[k] = v
		}
	}
	// When job-discriminator is the only field, there is no user-specified workflow-level
	// group to emit; return empty so the compiler can generate the default concurrency.
	if len(cleanMap) == 0 {
		return ""
	}
	// Use a minimal temporary frontmatter containing only the concurrency key to avoid
	// copying the entire (potentially large) frontmatter map.
	return c.extractTopLevelYAMLSection(map[string]any{"concurrency": cleanMap}, "concurrency")
}

// extractDispatchItemNumber reports whether the frontmatter's on.workflow_dispatch
// trigger exposes an item_number input. This is the signature produced by the label
// trigger shorthand (e.g. "on: pull_request labeled my-label"). Reading the
// structured map avoids re-parsing the rendered YAML string later.
func extractDispatchItemNumber(frontmatter map[string]any) bool {
	onVal, ok := frontmatter["on"]
	if !ok {
		return false
	}
	onMap, ok := onVal.(map[string]any)
	if !ok {
		return false
	}
	wdVal, ok := onMap["workflow_dispatch"]
	if !ok {
		return false
	}
	wdMap, ok := wdVal.(map[string]any)
	if !ok {
		return false
	}
	inputsVal, ok := wdMap["inputs"]
	if !ok {
		return false
	}
	inputsMap, ok := inputsVal.(map[string]any)
	if !ok {
		return false
	}
	_, ok = inputsMap["item_number"]
	return ok
}

// processAndMergeSteps handles the merging of imported steps with main workflow steps
func (c *Compiler) processAndMergeSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowBuilderLog.Print("Processing and merging custom steps")

	workflowData.CustomSteps = c.extractTopLevelYAMLSection(frontmatter, "steps")

	// Parse copilot-setup-steps if present (these go at the start)
	var copilotSetupSteps []any
	if importsResult.CopilotSetupSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.CopilotSetupSteps), &copilotSetupSteps); err != nil {
			workflowBuilderLog.Printf("Failed to unmarshal copilot-setup steps: %v", err)
		} else {
			// Convert to typed steps for action pinning
			typedCopilotSteps, err := SliceToSteps(copilotSetupSteps)
			if err != nil {
				workflowBuilderLog.Printf("Failed to convert copilot-setup steps to typed steps: %v", err)
			} else {
				// Apply action pinning to copilot-setup steps
				typedCopilotSteps = applyActionPinsToTypedSteps(typedCopilotSteps, workflowData)
				// Convert back to []any for YAML marshaling
				copilotSetupSteps = StepsToSlice(typedCopilotSteps)
			}
		}
	}

	// Parse other imported steps if present (these go after copilot-setup but before main steps)
	var otherImportedSteps []any
	if importsResult.MergedSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedSteps), &otherImportedSteps); err == nil {
			// Convert to typed steps for action pinning
			typedOtherSteps, err := SliceToSteps(otherImportedSteps)
			if err != nil {
				workflowBuilderLog.Printf("Failed to convert other imported steps to typed steps: %v", err)
			} else {
				// Apply action pinning to other imported steps
				typedOtherSteps = applyActionPinsToTypedSteps(typedOtherSteps, workflowData)
				// Convert back to []any for YAML marshaling
				otherImportedSteps = StepsToSlice(typedOtherSteps)
			}
		}
	}

	// If there are main workflow steps, parse them
	var mainSteps []any
	if workflowData.CustomSteps != "" {
		var mainStepsWrapper map[string]any
		if err := yaml.Unmarshal([]byte(workflowData.CustomSteps), &mainStepsWrapper); err == nil {
			if mainStepsVal, hasSteps := mainStepsWrapper["steps"]; hasSteps {
				if steps, ok := mainStepsVal.([]any); ok {
					mainSteps = steps
					// Convert to typed steps for action pinning
					typedMainSteps, err := SliceToSteps(mainSteps)
					if err != nil {
						workflowBuilderLog.Printf("Failed to convert main steps to typed steps: %v", err)
					} else {
						// Apply action pinning to main steps
						typedMainSteps = applyActionPinsToTypedSteps(typedMainSteps, workflowData)
						// Convert back to []any for YAML marshaling
						mainSteps = StepsToSlice(typedMainSteps)
					}
				}
			}
		}
	}

	// Merge steps in the correct order:
	// 1. copilot-setup-steps (at start)
	// 2. other imported steps (after copilot-setup)
	// 3. main frontmatter steps (last)
	var allSteps []any
	if len(copilotSetupSteps) > 0 || len(mainSteps) > 0 || len(otherImportedSteps) > 0 {
		allSteps = append(allSteps, copilotSetupSteps...)
		allSteps = append(allSteps, otherImportedSteps...)
		allSteps = append(allSteps, mainSteps...)

		// Convert back to YAML with "steps:" wrapper
		stepsWrapper := map[string]any{"steps": allSteps}
		stepsYAML, err := yaml.Marshal(stepsWrapper)
		if err == nil {
			// Remove quotes from uses values with version comments
			workflowData.CustomSteps = unquoteUsesWithComments(string(stepsYAML))
		}
	}
}

// processAndMergePreSteps handles the processing and merging of pre-steps with action pinning.
// Pre-steps run at the very beginning of the agent job, before checkout and the subsequent
// built-in steps, allowing users to mint tokens or perform other setup that must happen
// before the repository is checked out. Imported pre-steps are merged before the main
// workflow's pre-steps so that the main workflow can override or extend the imports.
func (c *Compiler) processAndMergePreSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowBuilderLog.Print("Processing and merging pre-steps")

	mainPreStepsYAML := c.extractTopLevelYAMLSection(frontmatter, "pre-steps")

	// Parse imported pre-steps if present (these go before the main workflow's pre-steps)
	var importedPreSteps []any
	if importsResult.MergedPreSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedPreSteps), &importedPreSteps); err != nil {
			workflowBuilderLog.Printf("Failed to unmarshal imported pre-steps: %v", err)
		} else {
			typedImported, err := SliceToSteps(importedPreSteps)
			if err != nil {
				workflowBuilderLog.Printf("Failed to convert imported pre-steps to typed steps: %v", err)
			} else {
				typedImported = applyActionPinsToTypedSteps(typedImported, workflowData)
				importedPreSteps = StepsToSlice(typedImported)
			}
		}
	}

	// Parse main workflow pre-steps if present
	var mainPreSteps []any
	if mainPreStepsYAML != "" {
		var mainWrapper map[string]any
		if err := yaml.Unmarshal([]byte(mainPreStepsYAML), &mainWrapper); err == nil {
			if mainVal, ok := mainWrapper["pre-steps"]; ok {
				if steps, ok := mainVal.([]any); ok {
					mainPreSteps = steps
					typedMain, err := SliceToSteps(mainPreSteps)
					if err != nil {
						workflowBuilderLog.Printf("Failed to convert main pre-steps to typed steps: %v", err)
					} else {
						typedMain = applyActionPinsToTypedSteps(typedMain, workflowData)
						mainPreSteps = StepsToSlice(typedMain)
					}
				}
			}
		}
	}

	// Merge in order: imported pre-steps first, then main workflow's pre-steps
	var allPreSteps []any
	if len(importedPreSteps) > 0 || len(mainPreSteps) > 0 {
		allPreSteps = append(allPreSteps, importedPreSteps...)
		allPreSteps = append(allPreSteps, mainPreSteps...)

		stepsWrapper := map[string]any{"pre-steps": allPreSteps}
		stepsYAML, err := yaml.Marshal(stepsWrapper)
		if err == nil {
			workflowData.PreSteps = unquoteUsesWithComments(string(stepsYAML))
		}
	}
}

// processAndMergePreAgentSteps handles processing and merging of pre-agent-steps with action pinning.
// Imported pre-agent-steps are prepended so main workflow pre-agent-steps run last.
func (c *Compiler) processAndMergePreAgentSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowBuilderLog.Print("Processing and merging pre-agent-steps")

	mainPreAgentStepsYAML := c.extractTopLevelYAMLSection(frontmatter, "pre-agent-steps")

	var importedPreAgentSteps []any
	if importsResult.MergedPreAgentSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedPreAgentSteps), &importedPreAgentSteps); err != nil {
			workflowBuilderLog.Printf("Failed to unmarshal imported pre-agent-steps: %v", err)
		} else {
			typedImported, err := SliceToSteps(importedPreAgentSteps)
			if err != nil {
				workflowBuilderLog.Printf("Failed to convert imported pre-agent-steps to typed steps: %v", err)
			} else {
				typedImported = applyActionPinsToTypedSteps(typedImported, workflowData)
				importedPreAgentSteps = StepsToSlice(typedImported)
			}
		}
	}

	var mainPreAgentSteps []any
	if mainPreAgentStepsYAML != "" {
		var mainWrapper map[string]any
		if err := yaml.Unmarshal([]byte(mainPreAgentStepsYAML), &mainWrapper); err == nil {
			if mainVal, ok := mainWrapper["pre-agent-steps"]; ok {
				if steps, ok := mainVal.([]any); ok {
					mainPreAgentSteps = steps
					typedMain, err := SliceToSteps(mainPreAgentSteps)
					if err != nil {
						workflowBuilderLog.Printf("Failed to convert main pre-agent-steps to typed steps: %v", err)
					} else {
						typedMain = applyActionPinsToTypedSteps(typedMain, workflowData)
						mainPreAgentSteps = StepsToSlice(typedMain)
					}
				}
			}
		}
	}

	var allPreAgentSteps []any
	if len(importedPreAgentSteps) > 0 || len(mainPreAgentSteps) > 0 {
		allPreAgentSteps = append(allPreAgentSteps, importedPreAgentSteps...)
		allPreAgentSteps = append(allPreAgentSteps, mainPreAgentSteps...)

		stepsWrapper := map[string]any{"pre-agent-steps": allPreAgentSteps}
		stepsYAML, err := yaml.Marshal(stepsWrapper)
		if err == nil {
			workflowData.PreAgentSteps = unquoteUsesWithComments(string(stepsYAML))
		}
	}
}

// processAndMergePostSteps handles the processing and merging of post-steps with action pinning.
// Imported post-steps are appended after the main workflow's post-steps.
func (c *Compiler) processAndMergePostSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowBuilderLog.Print("Processing and merging post-steps")

	mainPostStepsYAML := c.extractTopLevelYAMLSection(frontmatter, "post-steps")

	// Parse imported post-steps if present (these go after the main workflow's post-steps)
	var importedPostSteps []any
	if importsResult.MergedPostSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedPostSteps), &importedPostSteps); err != nil {
			workflowBuilderLog.Printf("Failed to unmarshal imported post-steps: %v", err)
		} else {
			typedImported, err := SliceToSteps(importedPostSteps)
			if err != nil {
				workflowBuilderLog.Printf("Failed to convert imported post-steps to typed steps: %v", err)
			} else {
				typedImported = applyActionPinsToTypedSteps(typedImported, workflowData)
				importedPostSteps = StepsToSlice(typedImported)
			}
		}
	}

	// Parse main workflow post-steps if present
	var mainPostSteps []any
	if mainPostStepsYAML != "" {
		var mainWrapper map[string]any
		if err := yaml.Unmarshal([]byte(mainPostStepsYAML), &mainWrapper); err == nil {
			if mainVal, ok := mainWrapper["post-steps"]; ok {
				if steps, ok := mainVal.([]any); ok {
					mainPostSteps = steps
					typedMain, err := SliceToSteps(mainPostSteps)
					if err != nil {
						workflowBuilderLog.Printf("Failed to convert main post-steps to typed steps: %v", err)
					} else {
						typedMain = applyActionPinsToTypedSteps(typedMain, workflowData)
						mainPostSteps = StepsToSlice(typedMain)
					}
				}
			}
		}
	}

	// Merge in order: main workflow's post-steps first, then imported post-steps
	var allPostSteps []any
	if len(mainPostSteps) > 0 || len(importedPostSteps) > 0 {
		allPostSteps = append(allPostSteps, mainPostSteps...)
		allPostSteps = append(allPostSteps, importedPostSteps...)

		stepsWrapper := map[string]any{"post-steps": allPostSteps}
		stepsYAML, err := yaml.Marshal(stepsWrapper)
		if err == nil {
			workflowData.PostSteps = unquoteUsesWithComments(string(stepsYAML))
		}
	}
}
