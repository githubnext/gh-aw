package campaign

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var launcherLog = logger.New("campaign:launcher")

// BuildLauncher constructs a minimal agentic workflow representation for a
// given CampaignSpec. The launcher is intended to keep the campaign's GitHub
// Project board in sync with campaign-labeled issues/PRs.
//
// The resulting WorkflowData is compiled via the standard
// CompileWorkflowDataWithValidation pipeline, and the launcherPath determines
// the emitted .lock.yml name.
func BuildLauncher(spec *CampaignSpec, campaignFilePath string) (*workflow.WorkflowData, string) {
	launcherLog.Printf("Building launcher for campaign: id=%s, file=%s", spec.ID, campaignFilePath)

	base := strings.TrimSuffix(campaignFilePath, ".campaign.md")
	launcherPath := base + ".campaign.launcher.g.md"
	launcherLog.Printf("Generated launcher path: %s", launcherPath)

	name := spec.Name
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Campaign: %s", spec.ID)
	}
	launcherName := fmt.Sprintf("%s (launcher)", name)

	description := spec.Description
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Launcher workflow for campaign '%s' (tracker: %s)", spec.ID, spec.TrackerLabel)
	}

	// Default triggers: daily schedule plus manual workflow_dispatch.
	onSection := "on:\n  schedule:\n    - cron: \"0 17 * * *\"\n  workflow_dispatch:\n"

	// Prevent overlapping runs. This reduces sustained automated traffic on GitHub's
	// infrastructure by ensuring only one launcher run executes at a time per ref.
	concurrency := fmt.Sprintf("concurrency:\n  group: \"campaign-%s-launcher-${{ github.ref }}\"\n  cancel-in-progress: false", spec.ID)

	markdownBuilder := &strings.Builder{}
	markdownBuilder.WriteString("# Campaign Launcher\n\n")
	markdownBuilder.WriteString(fmt.Sprintf("This workflow keeps the '%s' campaign dashboard in sync.\n\n", name))

	hasDetails := false
	if spec.TrackerLabel != "" {
		markdownBuilder.WriteString(fmt.Sprintf("- Tracker label: `%s`\n", spec.TrackerLabel))
		hasDetails = true
	}
	if strings.TrimSpace(spec.ProjectURL) != "" {
		markdownBuilder.WriteString(fmt.Sprintf("- Project URL: %s\n", strings.TrimSpace(spec.ProjectURL)))
		hasDetails = true
	}
	if spec.CursorGlob != "" {
		markdownBuilder.WriteString(fmt.Sprintf("- Cursor glob: `%s`\n", spec.CursorGlob))
		hasDetails = true
	}
	if spec.Governance != nil {
		if spec.Governance.MaxNewItemsPerRun > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: max new items per run: %d\n", spec.Governance.MaxNewItemsPerRun))
			hasDetails = true
		}
		if spec.Governance.MaxDiscoveryItemsPerRun > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: max discovery items per run: %d\n", spec.Governance.MaxDiscoveryItemsPerRun))
			hasDetails = true
		}
		if spec.Governance.MaxDiscoveryPagesPerRun > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: max discovery pages per run: %d\n", spec.Governance.MaxDiscoveryPagesPerRun))
			hasDetails = true
		}
		if len(spec.Governance.OptOutLabels) > 0 {
			markdownBuilder.WriteString("- Governance: opt-out labels: ")
			markdownBuilder.WriteString(strings.Join(spec.Governance.OptOutLabels, ", "))
			markdownBuilder.WriteString("\n")
			hasDetails = true
		}
		if spec.Governance.DoNotDowngradeDoneItems != nil {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: do not downgrade done items: %t\n", *spec.Governance.DoNotDowngradeDoneItems))
			hasDetails = true
		}
		if spec.Governance.MaxProjectUpdatesPerRun > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: max project updates per run: %d\n", spec.Governance.MaxProjectUpdatesPerRun))
			hasDetails = true
		}
		if spec.Governance.MaxCommentsPerRun > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("- Governance: max comments per run: %d\n", spec.Governance.MaxCommentsPerRun))
			hasDetails = true
		}
	}

	if !hasDetails {
		launcherLog.Printf("Campaign '%s' has no meaningful details, skipping launcher build", spec.ID)
		return nil, ""
	}

	promptData := CampaignPromptData{ProjectURL: strings.TrimSpace(spec.ProjectURL)}
	promptData.TrackerLabel = strings.TrimSpace(spec.TrackerLabel)
	promptData.CursorGlob = strings.TrimSpace(spec.CursorGlob)
	if spec.Governance != nil {
		promptData.MaxDiscoveryItemsPerRun = spec.Governance.MaxDiscoveryItemsPerRun
		promptData.MaxDiscoveryPagesPerRun = spec.Governance.MaxDiscoveryPagesPerRun
	}
	launcherInstructions := RenderLauncherInstructions(promptData)
	markdownBuilder.WriteString("\n" + launcherInstructions + "\n")

	maxComments := 10
	maxProjectUpdates := 10
	if spec.Governance != nil {
		if spec.Governance.MaxCommentsPerRun > 0 {
			maxComments = spec.Governance.MaxCommentsPerRun
		}
		if spec.Governance.MaxProjectUpdatesPerRun > 0 {
			maxProjectUpdates = spec.Governance.MaxProjectUpdatesPerRun
		}
	}

	safeOutputs := &workflow.SafeOutputsConfig{}
	safeOutputs.AddComments = &workflow.AddCommentsConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxComments}}
	updateProjectConfig := &workflow.UpdateProjectConfig{BaseSafeOutputConfig: workflow.BaseSafeOutputConfig{Max: maxProjectUpdates}}
	if strings.TrimSpace(spec.ProjectGitHubToken) != "" {
		updateProjectConfig.GitHubToken = strings.TrimSpace(spec.ProjectGitHubToken)
		launcherLog.Printf("Campaign launcher '%s' configured with custom GitHub token for update-project", spec.ID)
	}
	safeOutputs.UpdateProjects = updateProjectConfig

	data := &workflow.WorkflowData{
		Name:            launcherName,
		Description:     description,
		MarkdownContent: markdownBuilder.String(),
		On:              onSection,
		Concurrency:     concurrency,
		RunsOn:          "runs-on: ubuntu-latest",
		Roles:           []string{"admin", "maintainer", "write"},
		SafeOutputs:     safeOutputs,
	}

	launcherLog.Printf("Campaign launcher '%s' built successfully with safe outputs enabled", spec.ID)
	return data, launcherPath
}
