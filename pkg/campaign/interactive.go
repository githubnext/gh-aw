package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var interactiveLog = logger.New("campaign:interactive")

// InteractiveCampaignConfig holds the configuration for interactive campaign creation
type InteractiveCampaignConfig struct {
	ID             string
	Name           string
	Description    string
	Scope          string // "current", "multiple", "org-wide"
	ScopeSelectors []string
	Workflows      []string
	Owners         []string
	RiskLevel      string
	CreateProject  bool
	ProjectOwner   string
	LinkRepo       string
	Force          bool
}

// RunInteractiveCampaignCreation runs an interactive wizard to create a campaign spec
func RunInteractiveCampaignCreation(rootDir string, force bool, verbose bool) error {
	interactiveLog.Print("Starting interactive campaign creation")

	// Assert this function is not running in automated unit tests
	if os.Getenv("GO_TEST_MODE") == "true" || os.Getenv("CI") != "" {
		return fmt.Errorf("interactive mode cannot be used in automated tests or CI environments")
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("🎯 Let's create your agentic campaign!"))
	fmt.Fprintln(os.Stderr, "")

	config := &InteractiveCampaignConfig{
		Force: force,
	}

	// Step 1: Campaign ID
	if err := promptForCampaignID(config); err != nil {
		return err
	}

	// Step 2: Campaign objective/description
	if err := promptForObjective(config); err != nil {
		return err
	}

	// Step 3: Repository scope
	if err := promptForRepositoryScope(config); err != nil {
		return err
	}

	// Step 4: Workflow selection
	if err := promptForWorkflows(config); err != nil {
		return err
	}

	// Step 5: Owners/stakeholders
	if err := promptForOwners(config); err != nil {
		return err
	}

	// Step 6: Risk level
	if err := promptForRiskLevel(config); err != nil {
		return err
	}

	// Step 7: Project board creation
	if err := promptForProjectCreation(config); err != nil {
		return err
	}

	// Generate the campaign spec
	if err := generateCampaignFromConfig(rootDir, config, verbose); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✅ Campaign spec created successfully!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Next steps:"))
	fmt.Fprintln(os.Stderr, "  1. Review and edit: .github/workflows/"+config.ID+".campaign.md")
	fmt.Fprintln(os.Stderr, "  2. Compile the orchestrator: gh aw compile")
	if config.CreateProject {
		fmt.Fprintln(os.Stderr, "  3. Project board will be created during compilation")
	}
	fmt.Fprintln(os.Stderr, "")

	return nil
}

func promptForCampaignID(config *InteractiveCampaignConfig) error {
	var id string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Campaign ID").
				Description("Use lowercase letters, digits, and hyphens (e.g., security-q1-2025)").
				Placeholder("my-campaign").
				Value(&id).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("campaign ID is required")
					}
					for _, ch := range s {
						if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
							continue
						}
						return fmt.Errorf("campaign ID must use only lowercase letters, digits, and hyphens")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("campaign ID input failed: %w", err)
	}

	config.ID = strings.TrimSpace(id)
	config.Name = formatCampaignName(config.ID)
	interactiveLog.Printf("Campaign ID: %s", config.ID)
	return nil
}

func promptForObjective(config *InteractiveCampaignConfig) error {
	var objective string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("What is the main objective of this campaign?").
				Description("Describe what you want to achieve (e.g., 'Reduce critical vulnerabilities across all repositories')").
				Placeholder("Enter campaign objective...").
				Value(&objective).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("objective is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("objective input failed: %w", err)
	}

	config.Description = strings.TrimSpace(objective)
	interactiveLog.Printf("Campaign description: %s", config.Description)
	return nil
}

func promptForRepositoryScope(config *InteractiveCampaignConfig) error {
	var scopeType string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What is the repository scope for this campaign?").
				Options(
					huh.NewOption("Current repository only", "current"),
					huh.NewOption("Specific repositories", "multiple"),
					huh.NewOption("Organization-wide", "org-wide"),
				).
				Value(&scopeType),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("scope selection failed: %w", err)
	}

	config.Scope = scopeType

	switch scopeType {
	case "current":
		currentRepo, err := getCurrentRepository()
		if err != nil {
			interactiveLog.Printf("Warning: could not determine current repository for scope: %v", err)
			break
		}
		config.ScopeSelectors = []string{currentRepo}
	case "multiple":
		var reposInput string
		reposForm := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Allowed repositories").
					Description("Enter repositories this campaign can operate on (comma-separated, e.g., 'owner/repo1, owner/repo2')").
					Placeholder("owner/repo1, owner/repo2").
					Value(&reposInput).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("at least one repository is required")
						}
						return nil
					}),
			),
		)

		if err := reposForm.Run(); err != nil {
			return fmt.Errorf("allowed repos input failed: %w", err)
		}

		repos := strings.Split(reposInput, ",")
		for _, repo := range repos {
			repo = strings.TrimSpace(repo)
			if repo != "" {
				config.ScopeSelectors = append(config.ScopeSelectors, repo)
			}
		}
	case "org-wide":
		var orgsInput string
		orgsForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Organization name").
					Description("Enter the organization name (e.g., 'myorg')").
					Placeholder("myorg").
					Value(&orgsInput).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("organization name is required")
						}
						return nil
					}),
			),
		)

		if err := orgsForm.Run(); err != nil {
			return fmt.Errorf("allowed org input failed: %w", err)
		}

		org := strings.TrimSpace(orgsInput)
		if org != "" {
			config.ScopeSelectors = append(config.ScopeSelectors, "org:"+org)
		}
	}

	interactiveLog.Printf("Scope: %s, selectors: %v", config.Scope, config.ScopeSelectors)
	return nil
}

func promptForWorkflows(config *InteractiveCampaignConfig) error {
	var workflowsInput string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Which workflows should this campaign use?").
				Description("Enter workflow names (comma-separated, e.g., 'vulnerability-scanner, dependency-updater'). Leave empty to configure later.").
				Placeholder("workflow-1, workflow-2").
				Value(&workflowsInput),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("workflows input failed: %w", err)
	}

	if strings.TrimSpace(workflowsInput) != "" {
		workflows := strings.Split(workflowsInput, ",")
		for _, workflow := range workflows {
			workflow = strings.TrimSpace(workflow)
			if workflow != "" {
				config.Workflows = append(config.Workflows, workflow)
			}
		}
	}

	interactiveLog.Printf("Workflows: %v", config.Workflows)
	return nil
}

func promptForOwners(config *InteractiveCampaignConfig) error {
	var ownersInput string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Who are the campaign owners?").
				Description("Enter GitHub usernames (comma-separated, with @ prefix, e.g., '@alice, @bob'). Leave empty to configure later.").
				Placeholder("@username").
				Value(&ownersInput),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("owners input failed: %w", err)
	}

	if strings.TrimSpace(ownersInput) != "" {
		owners := strings.Split(ownersInput, ",")
		for _, owner := range owners {
			trimmedOwner := strings.TrimSpace(owner)
			if trimmedOwner != "" {
				if !strings.HasPrefix(trimmedOwner, "@") {
					trimmedOwner = "@" + trimmedOwner
				}
				config.Owners = append(config.Owners, trimmedOwner)
			}
		}
	}

	interactiveLog.Printf("Owners: %v", config.Owners)
	return nil
}

func promptForRiskLevel(config *InteractiveCampaignConfig) error {
	var riskLevel string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What is the risk level for this campaign?").
				Description("Risk level determines approval requirements").
				Options(
					huh.NewOption("Low - Read-only, single repo", "low"),
					huh.NewOption("Medium - Cross-repo, automated changes", "medium"),
					huh.NewOption("High - Multi-repo, sensitive data, breaking changes", "high"),
				).
				Value(&riskLevel),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("risk level selection failed: %w", err)
	}

	config.RiskLevel = riskLevel
	interactiveLog.Printf("Risk level: %s", config.RiskLevel)
	return nil
}

func promptForProjectCreation(config *InteractiveCampaignConfig) error {
	var createProject bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create a GitHub Project board for this campaign?").
				Description("This will set up project views and fields for tracking").
				Value(&createProject),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("project creation prompt failed: %w", err)
	}

	config.CreateProject = createProject

	if createProject {
		var projectOwner string
		ownerForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Project owner").
					Description("GitHub organization or user (use '@me' for personal projects)").
					Placeholder("@me or org-name").
					Value(&projectOwner).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("project owner is required")
						}
						return nil
					}),
			),
		)

		if err := ownerForm.Run(); err != nil {
			return fmt.Errorf("project owner input failed: %w", err)
		}

		config.ProjectOwner = strings.TrimSpace(projectOwner)
	}

	interactiveLog.Printf("Create project: %v, owner: %s", config.CreateProject, config.ProjectOwner)
	return nil
}

func generateCampaignFromConfig(rootDir string, config *InteractiveCampaignConfig, verbose bool) error {
	interactiveLog.Print("Generating campaign spec from interactive config")

	workflowsDir := filepath.Join(rootDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	fileName := config.ID + ".campaign.md"
	fullPath := filepath.Join(workflowsDir, fileName)
	relPath := filepath.ToSlash(filepath.Join(".github", "workflows", fileName))

	if _, err := os.Stat(fullPath); err == nil && !config.Force {
		return fmt.Errorf("campaign spec already exists at %s (use --force to overwrite)", relPath)
	}

	// Build the spec
	spec := CampaignSpec{
		ID:          config.ID,
		Name:        config.Name,
		Description: config.Description,
		ProjectURL:  "https://github.com/orgs/ORG/projects/1", // Placeholder
		Version:     "v1",
		State:       "planned",
		Workflows:   config.Workflows,
		Scope:       config.ScopeSelectors,
		Owners:      config.Owners,
		RiskLevel:   config.RiskLevel,
		MemoryPaths: []string{"memory/campaigns/" + config.ID + "/**"},
		MetricsGlob: "memory/campaigns/" + config.ID + "/metrics/*.json",
		CursorGlob:  "memory/campaigns/" + config.ID + "/cursor.json",
		Governance: &CampaignGovernancePolicy{
			MaxNewItemsPerRun:       25,
			MaxDiscoveryItemsPerRun: 200,
			MaxDiscoveryPagesPerRun: 10,
			OptOutLabels:            []string{"no-campaign", "no-bot"},
			DoNotDowngradeDoneItems: boolPtr(true),
			MaxProjectUpdatesPerRun: 10,
			MaxCommentsPerRun:       10,
		},
	}

	data, err := yaml.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("failed to marshal campaign spec: %w", err)
	}

	var buf strings.Builder
	buf.WriteString("---\n")
	buf.Write(data)
	buf.WriteString("---\n\n")
	buf.WriteString("# " + config.Name + "\n\n")
	buf.WriteString(config.Description + "\n\n")

	buf.WriteString("## Workflows\n\n")
	if len(config.Workflows) > 0 {
		for _, workflow := range config.Workflows {
			buf.WriteString("### " + workflow + "\n")
			buf.WriteString("Description of what this workflow does in the context of this campaign.\n\n")
		}
	} else {
		buf.WriteString("Add workflow descriptions here.\n\n")
	}

	buf.WriteString("## Timeline\n\n")
	buf.WriteString("- **Start**: TBD\n")
	buf.WriteString("- **Target**: Ongoing\n\n")

	buf.WriteString("## Governance\n\n")
	buf.WriteString("Describe risk mitigation, approval process, and stakeholder communication.\n")

	// Use restrictive permissions (0644) for proper git tracking
	if err := os.WriteFile(fullPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write campaign spec file '%s': %w", relPath, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created campaign spec at %s", relPath)))

	// Create project if requested
	if config.CreateProject {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Creating GitHub Project..."))

		projectConfig := ProjectCreationConfig{
			CampaignID:   config.ID,
			CampaignName: config.Name,
			Owner:        config.ProjectOwner,
			LinkRepo:     config.LinkRepo,
			NoLinkRepo:   false,
			Verbose:      verbose,
		}

		result, err := CreateCampaignProject(projectConfig)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created project: %s", result.ProjectURL)))

		// Update the spec file with the project URL
		if err := UpdateSpecWithProjectURL(fullPath, result.ProjectURL); err != nil {
			return fmt.Errorf("failed to update spec with project URL: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Updated campaign spec with project URL"))
	}

	return nil
}

func formatCampaignName(id string) string {
	name := strings.ReplaceAll(id, "-", " ")
	if name != "" {
		words := strings.Fields(name)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
		name = strings.Join(words, " ")
	}
	return name
}
