package campaign

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var frontmatterConversionLog = logger.New("campaign:frontmatter_conversion")

// ConvertFromFrontmatter converts a workflow frontmatter configuration with project
// into a CampaignSpec suitable for orchestrator generation
func ConvertFromFrontmatter(config *workflow.FrontmatterConfig, workflowPath string) (*CampaignSpec, error) {
	frontmatterConversionLog.Printf("Converting workflow frontmatter to campaign spec: %s", workflowPath)

	if config == nil {
		return nil, fmt.Errorf("frontmatter config is nil")
	}

	if config.Project == nil {
		return nil, fmt.Errorf("project configuration is required for campaign orchestrator generation")
	}

	// Derive campaign ID from workflow filename (basename without .md extension)
	basename := filepath.Base(workflowPath)
	campaignID := strings.TrimSuffix(basename, ".md")
	frontmatterConversionLog.Printf("Derived campaign ID: %s", campaignID)

	// Build campaign spec from frontmatter configuration
	spec := &CampaignSpec{
		ID:          campaignID,
		Name:        config.Name,
		Description: config.Description,
		ProjectURL:  config.Project.URL,
		Version:     config.Version,
		ConfigPath:  workflowPath,
	}

	// Set engine from frontmatter
	if config.Engine != "" {
		spec.Engine = config.Engine
	}

	// Set scope from project configuration
	if len(config.Project.Scope) > 0 {
		spec.Scope = config.Project.Scope
	}

	// Copy campaign-specific fields from project configuration
	// All campaign fields are now nested within the project config
	if config.Project.ID != "" {
		spec.ID = config.Project.ID
	}

	spec.Workflows = config.Project.Workflows
	spec.MemoryPaths = config.Project.MemoryPaths
	spec.MetricsGlob = config.Project.MetricsGlob
	spec.CursorGlob = config.Project.CursorGlob
	spec.TrackerLabel = config.Project.TrackerLabel
	spec.Owners = config.Project.Owners
	spec.RiskLevel = config.Project.RiskLevel
	spec.State = config.Project.State
	spec.Tags = config.Project.Tags

	// Convert governance configuration
	if config.Project.Governance != nil {
		spec.Governance = &CampaignGovernancePolicy{
			MaxNewItemsPerRun:       config.Project.Governance.MaxNewItemsPerRun,
			MaxDiscoveryItemsPerRun: config.Project.Governance.MaxDiscoveryItemsPerRun,
			MaxDiscoveryPagesPerRun: config.Project.Governance.MaxDiscoveryPagesPerRun,
			OptOutLabels:            config.Project.Governance.OptOutLabels,
			DoNotDowngradeDoneItems: config.Project.Governance.DoNotDowngradeDoneItems,
			MaxProjectUpdatesPerRun: config.Project.Governance.MaxProjectUpdatesPerRun,
			MaxCommentsPerRun:       config.Project.Governance.MaxCommentsPerRun,
		}
	}

	// Convert bootstrap configuration
	if config.Project.Bootstrap != nil {
		spec.Bootstrap = &CampaignBootstrapConfig{
			Mode: config.Project.Bootstrap.Mode,
		}

		if config.Project.Bootstrap.SeederWorker != nil {
			spec.Bootstrap.SeederWorker = &SeederWorkerConfig{
				WorkflowID: config.Project.Bootstrap.SeederWorker.WorkflowID,
				Payload:    config.Project.Bootstrap.SeederWorker.Payload,
				MaxItems:   config.Project.Bootstrap.SeederWorker.MaxItems,
			}
		}

		if config.Project.Bootstrap.ProjectTodos != nil {
			spec.Bootstrap.ProjectTodos = &ProjectTodosConfig{
				StatusField:   config.Project.Bootstrap.ProjectTodos.StatusField,
				TodoValue:     config.Project.Bootstrap.ProjectTodos.TodoValue,
				MaxItems:      config.Project.Bootstrap.ProjectTodos.MaxItems,
				RequireFields: config.Project.Bootstrap.ProjectTodos.RequireFields,
			}
		}
	}

	// Convert worker metadata
	if len(config.Project.Workers) > 0 {
		spec.Workers = make([]WorkerMetadata, len(config.Project.Workers))
		for i, w := range config.Project.Workers {
			spec.Workers[i] = WorkerMetadata{
				ID:                  w.ID,
				Name:                w.Name,
				Description:         w.Description,
				Capabilities:        w.Capabilities,
				IdempotencyStrategy: w.IdempotencyStrategy,
				Priority:            w.Priority,
			}

			// Convert payload schema
			if len(w.PayloadSchema) > 0 {
				spec.Workers[i].PayloadSchema = make(map[string]WorkerPayloadField)
				for key, field := range w.PayloadSchema {
					spec.Workers[i].PayloadSchema[key] = WorkerPayloadField{
						Type:        field.Type,
						Description: field.Description,
						Required:    field.Required,
						Example:     field.Example,
					}
				}
			}

			// Convert output labeling
			spec.Workers[i].OutputLabeling = WorkerOutputLabeling{
				Labels:         w.OutputLabeling.Labels,
				KeyInTitle:     w.OutputLabeling.KeyInTitle,
				KeyFormat:      w.OutputLabeling.KeyFormat,
				MetadataFields: w.OutputLabeling.MetadataFields,
			}
		}
	}

	// Note: DoNotDowngradeDoneItems is already part of project governance if set,
	// so no additional inheritance is needed

	frontmatterConversionLog.Printf("Successfully converted to campaign spec: id=%s, workflows=%d",
		spec.ID, len(spec.Workflows))

	return spec, nil
}
