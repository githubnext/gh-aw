package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var workflowImportMergeLog = logger.New("workflow:workflow_import_merge")

// processAndMergeServices handles the merging of imported services with main workflow services
func (c *Compiler) processAndMergeServices(frontmatter map[string]any, workflowData *WorkflowData, importsResult *parser.ImportsResult) {
	workflowImportMergeLog.Print("Processing and merging services")

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
			workflowImportMergeLog.Printf("Warning: %s", w)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(w))
			c.IncrementWarningCount()
		}
		if expressions != "" {
			workflowImportMergeLog.Printf("Extracted service port expressions: %s", expressions)
		}
	}
}

// mergeJobsFromYAMLImports merges jobs from imported YAML workflows with main workflow jobs
// Main workflow jobs take precedence over imported jobs (override behavior)
func (c *Compiler) mergeJobsFromYAMLImports(mainJobs map[string]any, mergedJobsJSON string) map[string]any {
	workflowImportMergeLog.Print("Merging jobs from imported YAML workflows")

	if mergedJobsJSON == "" || mergedJobsJSON == "{}" {
		workflowImportMergeLog.Print("No imported jobs to merge")
		return mainJobs
	}

	// Initialize result with main jobs or create empty map
	result := make(map[string]any)
	maps.Copy(result, mainJobs)

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(mergedJobsJSON, "\n")
	workflowImportMergeLog.Printf("Processing %d job definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to map
		var importedJobs map[string]any
		if err := json.Unmarshal([]byte(line), &importedJobs); err != nil {
			workflowImportMergeLog.Printf("Skipping malformed job entry: %v", err)
			continue
		}

		// Merge jobs - main workflow jobs take precedence (don't override)
		for jobName, jobConfig := range importedJobs {
			if _, exists := result[jobName]; !exists {
				workflowImportMergeLog.Printf("Adding imported job: %s", jobName)
				result[jobName] = jobConfig
			} else {
				workflowImportMergeLog.Printf("Skipping imported job %s (already defined in main workflow)", jobName)
			}
		}
	}

	workflowImportMergeLog.Printf("Successfully merged jobs: total=%d, imported=%d", len(result), len(result)-len(mainJobs))
	return result
}
