//go:build !integration

package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

// processAndMergeSteps handles the merging of imported steps with main workflow steps
func (c *Compiler) processAndMergeSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	orchestratorWorkflowLog.Print("Processing and merging custom steps")

	workflowData.CustomSteps = c.extractTopLevelYAMLSection(frontmatter, "steps")

	// Parse copilot-setup-steps if present (these go at the start)
	var copilotSetupSteps []any
	if importsResult.CopilotSetupSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.CopilotSetupSteps), &copilotSetupSteps); err != nil {
			orchestratorWorkflowLog.Printf("Failed to unmarshal copilot-setup steps: %v", err)
		} else {
			// Convert to typed steps for action pinning
			typedCopilotSteps, err := SliceToSteps(copilotSetupSteps)
			if err != nil {
				orchestratorWorkflowLog.Printf("Failed to convert copilot-setup steps to typed steps: %v", err)
			} else {
				// Apply action pinning to copilot-setup steps
				typedCopilotSteps = ApplyActionPinsToTypedSteps(typedCopilotSteps, workflowData)
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
				orchestratorWorkflowLog.Printf("Failed to convert other imported steps to typed steps: %v", err)
			} else {
				// Apply action pinning to other imported steps
				typedOtherSteps = ApplyActionPinsToTypedSteps(typedOtherSteps, workflowData)
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
						orchestratorWorkflowLog.Printf("Failed to convert main steps to typed steps: %v", err)
					} else {
						// Apply action pinning to main steps
						typedMainSteps = ApplyActionPinsToTypedSteps(typedMainSteps, workflowData)
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
	orchestratorWorkflowLog.Print("Processing and merging pre-steps")

	mainPreStepsYAML := c.extractTopLevelYAMLSection(frontmatter, "pre-steps")

	// Parse imported pre-steps if present (these go before the main workflow's pre-steps)
	var importedPreSteps []any
	if importsResult.MergedPreSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedPreSteps), &importedPreSteps); err != nil {
			orchestratorWorkflowLog.Printf("Failed to unmarshal imported pre-steps: %v", err)
		} else {
			typedImported, err := SliceToSteps(importedPreSteps)
			if err != nil {
				orchestratorWorkflowLog.Printf("Failed to convert imported pre-steps to typed steps: %v", err)
			} else {
				typedImported = ApplyActionPinsToTypedSteps(typedImported, workflowData)
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
						orchestratorWorkflowLog.Printf("Failed to convert main pre-steps to typed steps: %v", err)
					} else {
						typedMain = ApplyActionPinsToTypedSteps(typedMain, workflowData)
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

// processAndMergePostSteps handles the processing and merging of post-steps with action pinning.
// Imported post-steps are appended after the main workflow's post-steps.
func (c *Compiler) processAndMergePostSteps(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	orchestratorWorkflowLog.Print("Processing and merging post-steps")

	mainPostStepsYAML := c.extractTopLevelYAMLSection(frontmatter, "post-steps")

	// Parse imported post-steps if present (these go after the main workflow's post-steps)
	var importedPostSteps []any
	if importsResult.MergedPostSteps != "" {
		if err := yaml.Unmarshal([]byte(importsResult.MergedPostSteps), &importedPostSteps); err != nil {
			orchestratorWorkflowLog.Printf("Failed to unmarshal imported post-steps: %v", err)
		} else {
			typedImported, err := SliceToSteps(importedPostSteps)
			if err != nil {
				orchestratorWorkflowLog.Printf("Failed to convert imported post-steps to typed steps: %v", err)
			} else {
				typedImported = ApplyActionPinsToTypedSteps(typedImported, workflowData)
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
						orchestratorWorkflowLog.Printf("Failed to convert main post-steps to typed steps: %v", err)
					} else {
						typedMain = ApplyActionPinsToTypedSteps(typedMain, workflowData)
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

// processAndMergeServices handles the merging of imported services with main workflow services
func (c *Compiler) processAndMergeServices(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	orchestratorWorkflowLog.Print("Processing and merging services")

	workflowData.Services = c.extractTopLevelYAMLSection(frontmatter, "services")

	// Merge imported services if any
	if importsResult.MergedServices != "" {
		// Parse imported services from YAML
		var importedServices map[string]any
		if err := yaml.Unmarshal([]byte(importsResult.MergedServices), &importedServices); err == nil {
			// If there are main workflow services, parse and merge them
			if workflowData.Services != "" {
				// Parse main workflow services
				var mainServicesWrapper map[string]any
				if err := yaml.Unmarshal([]byte(workflowData.Services), &mainServicesWrapper); err == nil {
					if mainServices, ok := mainServicesWrapper["services"].(map[string]any); ok {
						// Merge: main workflow services take precedence over imported
						for key, value := range importedServices {
							if _, exists := mainServices[key]; !exists {
								mainServices[key] = value
							}
						}
						// Convert back to YAML with "services:" wrapper
						servicesWrapper := map[string]any{"services": mainServices}
						servicesYAML, err := yaml.Marshal(servicesWrapper)
						if err == nil {
							workflowData.Services = string(servicesYAML)
						}
					}
				}
			} else {
				// Only imported services exist, wrap in "services:" format
				servicesWrapper := map[string]any{"services": importedServices}
				servicesYAML, err := yaml.Marshal(servicesWrapper)
				if err == nil {
					workflowData.Services = string(servicesYAML)
				}
			}
		}
	}

	// Extract service port expressions for AWF --allow-host-service-ports
	if workflowData.Services != "" {
		expressions, warnings := ExtractServicePortExpressions(workflowData.Services)
		workflowData.ServicePortExpressions = expressions
		for _, w := range warnings {
			orchestratorWorkflowLog.Printf("Warning: %s", w)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(w))
			c.IncrementWarningCount()
		}
		if expressions != "" {
			orchestratorWorkflowLog.Printf("Extracted service port expressions: %s", expressions)
		}
	}
}

// mergeJobsFromYAMLImports merges jobs from imported YAML workflows with main workflow jobs.
// Main workflow jobs take precedence over imported jobs (override behavior).
func (c *Compiler) mergeJobsFromYAMLImports(mainJobs map[string]any, mergedJobsJSON string) map[string]any {
	orchestratorWorkflowLog.Print("Merging jobs from imported YAML workflows")

	if mergedJobsJSON == "" || mergedJobsJSON == "{}" {
		orchestratorWorkflowLog.Print("No imported jobs to merge")
		return mainJobs
	}

	// Initialize result with main jobs or create empty map
	result := make(map[string]any)
	maps.Copy(result, mainJobs)

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(mergedJobsJSON, "\n")
	orchestratorWorkflowLog.Printf("Processing %d job definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to map
		var importedJobs map[string]any
		if err := json.Unmarshal([]byte(line), &importedJobs); err != nil {
			orchestratorWorkflowLog.Printf("Skipping malformed job entry: %v", err)
			continue
		}

		// Merge jobs - main workflow jobs take precedence (don't override)
		for jobName, jobConfig := range importedJobs {
			if _, exists := result[jobName]; !exists {
				orchestratorWorkflowLog.Printf("Adding imported job: %s", jobName)
				result[jobName] = jobConfig
			} else {
				orchestratorWorkflowLog.Printf("Skipping imported job %s (already defined in main workflow)", jobName)
			}
		}
	}

	orchestratorWorkflowLog.Printf("Successfully merged jobs: total=%d, imported=%d", len(result), len(result)-len(mainJobs))
	return result
}

// processOnSectionAndFilters processes the on section configuration and applies various filters
func (c *Compiler) processOnSectionAndFilters(
	frontmatter map[string]any,
	workflowData *WorkflowData,
	cleanPath string,
) error {
	orchestratorWorkflowLog.Print("Processing on section and filters")

	// Process stop-after configuration from the on: section
	if err := c.processStopAfterConfiguration(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}

	// Process skip-if-match configuration from the on: section
	if err := c.processSkipIfMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Process skip-if-no-match configuration from the on: section
	if err := c.processSkipIfNoMatchConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Process skip-if-check-failing configuration from the on: section
	if err := c.processSkipIfCheckFailingConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Process manual-approval configuration from the on: section
	if err := c.processManualApprovalConfiguration(frontmatter, workflowData); err != nil {
		return err
	}

	// Parse the "on" section for command triggers, reactions, and other events
	if err := c.parseOnSection(frontmatter, workflowData, cleanPath); err != nil {
		return err
	}

	// Apply defaults
	if err := c.applyDefaults(workflowData, cleanPath); err != nil {
		return err
	}

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, frontmatter)

	// Apply pull request fork filter if specified
	c.applyPullRequestForkFilter(workflowData, frontmatter)

	// Apply label filter if specified
	c.applyLabelFilter(workflowData, frontmatter)

	// Extract on.steps for pre-activation step injection
	onSteps, err := extractOnSteps(frontmatter)
	if err != nil {
		return err
	}

	// Apply action pinning to on.steps
	if len(onSteps) > 0 {
		anySteps := make([]any, len(onSteps))
		for i, s := range onSteps {
			anySteps[i] = s
		}
		typedSteps, convErr := SliceToSteps(anySteps)
		if convErr == nil {
			typedSteps = ApplyActionPinsToTypedSteps(typedSteps, workflowData)
			for i, s := range typedSteps {
				onSteps[i] = s.ToMap()
			}
		} else {
			orchestratorWorkflowLog.Printf("Failed to convert on.steps to typed steps for action pinning: %v", convErr)
		}
	}

	workflowData.OnSteps = onSteps

	// Extract on.permissions for pre-activation job permissions
	workflowData.OnPermissions = extractOnPermissions(frontmatter)

	return nil
}
