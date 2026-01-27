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
		return nil, fmt.Errorf("project configuration is required for campaign orchestrator")
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

	// Copy campaign-specific fields if campaign config is present
	if config.Campaign != nil {
		// Override ID if explicitly set in campaign config
		if config.Campaign.ID != "" {
			spec.ID = config.Campaign.ID
		}

		spec.Workflows = config.Campaign.Workflows
		spec.MemoryPaths = config.Campaign.MemoryPaths
		spec.MetricsGlob = config.Campaign.MetricsGlob
		spec.CursorGlob = config.Campaign.CursorGlob
		spec.TrackerLabel = config.Campaign.TrackerLabel
		spec.Owners = config.Campaign.Owners
		spec.RiskLevel = config.Campaign.RiskLevel
		spec.State = config.Campaign.State
		spec.Tags = config.Campaign.Tags

		// Convert governance configuration
		if config.Campaign.Governance != nil {
			spec.Governance = &CampaignGovernancePolicy{
				MaxNewItemsPerRun:       config.Campaign.Governance.MaxNewItemsPerRun,
				MaxDiscoveryItemsPerRun: config.Campaign.Governance.MaxDiscoveryItemsPerRun,
				MaxDiscoveryPagesPerRun: config.Campaign.Governance.MaxDiscoveryPagesPerRun,
				OptOutLabels:            config.Campaign.Governance.OptOutLabels,
				DoNotDowngradeDoneItems: config.Campaign.Governance.DoNotDowngradeDoneItems,
				MaxProjectUpdatesPerRun: config.Campaign.Governance.MaxProjectUpdatesPerRun,
				MaxCommentsPerRun:       config.Campaign.Governance.MaxCommentsPerRun,
			}
		}

		// Convert bootstrap configuration
		if config.Campaign.Bootstrap != nil {
			spec.Bootstrap = &CampaignBootstrapConfig{
				Mode: config.Campaign.Bootstrap.Mode,
			}

			if config.Campaign.Bootstrap.SeederWorker != nil {
				spec.Bootstrap.SeederWorker = &SeederWorkerConfig{
					WorkflowID: config.Campaign.Bootstrap.SeederWorker.WorkflowID,
					Payload:    config.Campaign.Bootstrap.SeederWorker.Payload,
					MaxItems:   config.Campaign.Bootstrap.SeederWorker.MaxItems,
				}
			}

			if config.Campaign.Bootstrap.ProjectTodos != nil {
				spec.Bootstrap.ProjectTodos = &ProjectTodosConfig{
					StatusField:   config.Campaign.Bootstrap.ProjectTodos.StatusField,
					TodoValue:     config.Campaign.Bootstrap.ProjectTodos.TodoValue,
					MaxItems:      config.Campaign.Bootstrap.ProjectTodos.MaxItems,
					RequireFields: config.Campaign.Bootstrap.ProjectTodos.RequireFields,
				}
			}
		}

		// Convert worker metadata
		if len(config.Campaign.Workers) > 0 {
			spec.Workers = make([]WorkerMetadata, len(config.Campaign.Workers))
			for i, w := range config.Campaign.Workers {
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
	}

	// If no governance from campaign config, use project config defaults
	if spec.Governance == nil && config.Project.DoNotDowngradeDoneItems != nil {
		spec.Governance = &CampaignGovernancePolicy{
			DoNotDowngradeDoneItems: config.Project.DoNotDowngradeDoneItems,
		}
	}

	frontmatterConversionLog.Printf("Successfully converted to campaign spec: id=%s, workflows=%d",
		spec.ID, len(spec.Workflows))

	return spec, nil
}
