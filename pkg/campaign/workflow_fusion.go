package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var workflowFusionLog = logger.New("campaign:workflow_fusion")

// FusionResult contains the result of fusing a workflow for campaign use
type FusionResult struct {
	OriginalWorkflowID string // Original workflow ID
	CampaignWorkflowID string // New workflow ID in campaign folder
	OutputPath         string // Path to the fused workflow file
	WorkflowDispatch   bool   // Whether workflow_dispatch was added
}

// FuseWorkflowForCampaign takes an existing workflow and adapts it for campaign use
// by adding workflow_dispatch trigger and storing it in a campaign-specific folder
func FuseWorkflowForCampaign(rootDir string, workflowID string, campaignID string) (*FusionResult, error) {
	workflowFusionLog.Printf("Fusing workflow %s for campaign %s", workflowID, campaignID)

	// Read original workflow
	originalPath := filepath.Join(rootDir, ".github", "workflows", workflowID+".md")
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse frontmatter
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	frontmatter := result.Frontmatter
	bodyContent := result.Markdown

	// Check if workflow_dispatch already exists
	hasWorkflowDispatch := checkWorkflowDispatch(frontmatter)

	// Add workflow_dispatch if not present
	if !hasWorkflowDispatch {
		workflowFusionLog.Printf("Adding workflow_dispatch trigger to %s", workflowID)
		frontmatter = addWorkflowDispatch(frontmatter)
	}

	// Add campaign metadata
	frontmatter["campaign-worker"] = true
	frontmatter["campaign-id"] = campaignID
	frontmatter["source-workflow"] = workflowID

	// Marshal frontmatter back to YAML
	frontmatterYAML, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Reconstruct workflow content
	newContent := fmt.Sprintf("---\n%s---\n%s", string(frontmatterYAML), bodyContent)

	// Create campaign folder structure
	campaignDir := filepath.Join(rootDir, ".github", "workflows", "campaigns", campaignID)
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create campaign directory: %w", err)
	}

	// Write fused workflow to campaign folder
	campaignWorkflowID := fmt.Sprintf("%s-worker", workflowID)
	outputPath := filepath.Join(campaignDir, campaignWorkflowID+".md")
	if err := os.WriteFile(outputPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write fused workflow: %w", err)
	}

	workflowFusionLog.Printf("Fused workflow written to %s", outputPath)

	return &FusionResult{
		OriginalWorkflowID: workflowID,
		CampaignWorkflowID: campaignWorkflowID,
		OutputPath:         outputPath,
		WorkflowDispatch:   !hasWorkflowDispatch,
	}, nil
}

// checkWorkflowDispatch checks if the workflow already has workflow_dispatch trigger
func checkWorkflowDispatch(frontmatter map[string]any) bool {
	onField, ok := frontmatter["on"]
	if !ok {
		return false
	}

	// Handle string format: "on: workflow_dispatch"
	if onStr, ok := onField.(string); ok {
		return strings.Contains(onStr, "workflow_dispatch")
	}

	// Handle map format
	if onMap, ok := onField.(map[string]any); ok {
		_, hasDispatch := onMap["workflow_dispatch"]
		return hasDispatch
	}

	return false
}

// addWorkflowDispatch adds workflow_dispatch trigger to the frontmatter
func addWorkflowDispatch(frontmatter map[string]any) map[string]any {
	onField, ok := frontmatter["on"]
	if !ok {
		// No trigger defined, add workflow_dispatch
		frontmatter["on"] = "workflow_dispatch"
		return frontmatter
	}

	// Handle string format
	if onStr, ok := onField.(string); ok {
		// Parse existing triggers
		triggers := strings.Fields(onStr)
		triggers = append(triggers, "workflow_dispatch")
		frontmatter["on"] = strings.Join(triggers, "\n  ")
		return frontmatter
	}

	// Handle map format
	if onMap, ok := onField.(map[string]any); ok {
		onMap["workflow_dispatch"] = nil // Add workflow_dispatch
		frontmatter["on"] = onMap
		return frontmatter
	}

	// Fallback: replace with workflow_dispatch
	frontmatter["on"] = "workflow_dispatch"
	return frontmatter
}

// FuseMultipleWorkflows fuses multiple workflows for a campaign
func FuseMultipleWorkflows(rootDir string, workflowIDs []string, campaignID string) ([]FusionResult, error) {
	workflowFusionLog.Printf("Fusing %d workflows for campaign %s", len(workflowIDs), campaignID)

	var results []FusionResult
	for _, workflowID := range workflowIDs {
		result, err := FuseWorkflowForCampaign(rootDir, workflowID, campaignID)
		if err != nil {
			workflowFusionLog.Printf("Failed to fuse workflow %s: %v", workflowID, err)
			continue
		}
		results = append(results, *result)
	}

	workflowFusionLog.Printf("Successfully fused %d workflows", len(results))
	return results, nil
}
