package campaign

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var log = logger.New("campaign:loader")

// LoadSpecs scans the repository for campaign spec files and returns
// a slice of CampaignSpec. Campaign specs are stored as .campaign.md files
// in .github/workflows/. If the workflows directory does not exist, it
// returns an empty slice and no error.
func LoadSpecs(rootDir string) ([]CampaignSpec, error) {
	log.Printf("Loading campaign specs from rootDir=%s", rootDir)

	workflowsDir := filepath.Join(rootDir, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Print("No .github/workflows directory found; returning empty list")
			return []CampaignSpec{}, nil
		}
		return nil, fmt.Errorf("failed to read .github/workflows directory '%s': %w", workflowsDir, err)
	}

	var specs []CampaignSpec

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".campaign.md") {
			continue
		}

		fullPath := filepath.Join(workflowsDir, name)
		log.Printf("Found campaign spec file: %s", fullPath)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read campaign spec '%s': %w", fullPath, err)
		}

		// Use parser package's frontmatter extraction helper
		result, err := parser.ExtractFrontmatterFromContent(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse campaign spec frontmatter '%s': %w", fullPath, err)
		}

		if len(result.Frontmatter) == 0 {
			return nil, fmt.Errorf("campaign spec '%s' must start with YAML frontmatter delimited by '---'", filepath.ToSlash(filepath.Join(".github", "workflows", name)))
		}

		// Marshal frontmatter map to YAML and unmarshal to CampaignSpec
		frontmatterYAML, err := yaml.Marshal(result.Frontmatter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter for '%s': %w", fullPath, err)
		}

		var spec CampaignSpec
		if err := yaml.Unmarshal(frontmatterYAML, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse campaign spec frontmatter '%s': %w", fullPath, err)
		}

		if strings.TrimSpace(spec.ID) == "" {
			base := strings.TrimSuffix(name, ".campaign.md")
			spec.ID = base
		}

		if strings.TrimSpace(spec.Name) == "" {
			spec.Name = spec.ID
		}

		// Default state to "active" if not specified
		if strings.TrimSpace(spec.State) == "" {
			spec.State = "active"
			log.Printf("Defaulted state to 'active' for campaign '%s'", spec.ID)
		}

		// Default tracker-label based on campaign ID if not specified
		if strings.TrimSpace(spec.TrackerLabel) == "" {
			spec.TrackerLabel = fmt.Sprintf("z_campaign_%s", spec.ID)
			log.Printf("Defaulted tracker-label to '%s' for campaign '%s'", spec.TrackerLabel, spec.ID)
		}

		// Default memory-paths based on campaign ID if not specified
		if len(spec.MemoryPaths) == 0 {
			spec.MemoryPaths = []string{fmt.Sprintf("memory/campaigns/%s/**", spec.ID)}
			log.Printf("Defaulted memory-paths to '%s' for campaign '%s'", spec.MemoryPaths[0], spec.ID)
		}

		// Default metrics-glob based on campaign ID if not specified
		if strings.TrimSpace(spec.MetricsGlob) == "" {
			spec.MetricsGlob = fmt.Sprintf("memory/campaigns/%s/metrics/*.json", spec.ID)
			log.Printf("Defaulted metrics-glob to '%s' for campaign '%s'", spec.MetricsGlob, spec.ID)
		}

		// Default cursor-glob based on campaign ID if not specified
		if strings.TrimSpace(spec.CursorGlob) == "" {
			spec.CursorGlob = fmt.Sprintf("memory/campaigns/%s/cursor.json", spec.ID)
			log.Printf("Defaulted cursor-glob to '%s' for campaign '%s'", spec.CursorGlob, spec.ID)
		}

		// Default scope to current repository if not specified
		if len(spec.Scope) == 0 {
			currentRepo, err := getCurrentRepository()
			if err != nil {
				log.Printf("Warning: Could not determine current repository for campaign '%s': %v. Campaign will require explicit scope.", spec.ID, err)
			} else {
				spec.Scope = []string{currentRepo}
				log.Printf("Defaulted scope to current repository for campaign '%s': %s", spec.ID, currentRepo)
			}
		}

		spec.ConfigPath = filepath.ToSlash(filepath.Join(".github", "workflows", name))
		specs = append(specs, spec)
	}

	log.Printf("Loaded %d campaign specs", len(specs))
	return specs, nil
}

// FilterSpecs filters campaigns by a simple substring match on ID or
// Name (case-insensitive). When pattern is empty, all campaigns are returned.
func FilterSpecs(specs []CampaignSpec, pattern string) []CampaignSpec {
	if pattern == "" {
		return specs
	}

	var filtered []CampaignSpec
	lowerPattern := strings.ToLower(pattern)

	for _, spec := range specs {
		if strings.Contains(strings.ToLower(spec.ID), lowerPattern) || strings.Contains(strings.ToLower(spec.Name), lowerPattern) {
			filtered = append(filtered, spec)
		}
	}

	return filtered
}

// CreateSpecSkeleton creates a new campaign spec YAML file under
// .github/workflows/ with a minimal skeleton definition. It returns the
// relative file path created.
func CreateSpecSkeleton(rootDir, id string, force bool) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("campaign id is required")
	}

	// Reuse the same simple rules as ValidateSpec for IDs
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return "", fmt.Errorf("campaign id must use only lowercase letters, digits, and hyphens (%s)", id)
	}

	workflowsDir := filepath.Join(rootDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	fileName := id + ".campaign.md"
	fullPath := filepath.Join(workflowsDir, fileName)
	relPath := filepath.ToSlash(filepath.Join(".github", "workflows", fileName))

	if _, err := os.Stat(fullPath); err == nil && !force {
		return "", fmt.Errorf("campaign spec already exists at %s (use --force to overwrite)", relPath)
	}

	name := strings.ReplaceAll(id, "-", " ")
	if name != "" {
		first := strings.ToUpper(name[:1])
		if len(name) > 1 {
			name = first + name[1:]
		} else {
			name = first
		}
	}

	spec := CampaignSpec{
		ID:          id,
		Name:        name,
		ProjectURL:  "https://github.com/orgs/ORG/projects/1",
		Version:     "v1",
		State:       "planned",
		MemoryPaths: []string{"memory/campaigns/" + id + "/**"},
		MetricsGlob: "memory/campaigns/" + id + "/metrics/*.json",
		CursorGlob:  "memory/campaigns/" + id + "/cursor.json",
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
		return "", fmt.Errorf("failed to marshal campaign spec: %w", err)
	}

	var buf strings.Builder
	buf.WriteString("---\n")
	buf.Write(data)
	buf.WriteString("---\n\n")
	if name != "" {
		buf.WriteString("# " + name + "\n\n")
	} else {
		buf.WriteString("# " + id + "\n\n")
	}
	buf.WriteString("Describe this campaign's goals, guardrails, stakeholders, and playbook.\n\n")
	buf.WriteString("## Quick Start\n\n")
	buf.WriteString("By default, this campaign will target the current repository. To target additional repositories:\n\n")
	buf.WriteString("1. **Add scope** (optional): Specify repositories and/or organizations to target\n")
	buf.WriteString("2. **Define workflows**: List workflows to execute (e.g., `vulnerability-scanner`)\n")
	buf.WriteString("3. **Add narrative context**: Define campaign goals, workflows, and timeline in the markdown body\n")
	buf.WriteString("4. **Set owners**: Specify who is responsible for this campaign\n")
	buf.WriteString("5. **Compile**: Run `gh aw compile` to generate the orchestrator\n\n")
	buf.WriteString("## Example Configuration\n\n")
	buf.WriteString("```yaml\n")
	buf.WriteString("# Add to the frontmatter above:\n")
	buf.WriteString("\n")
	buf.WriteString("# Optional: Target specific repositories and/or organizations (defaults to current repo)\n")
	buf.WriteString("# scope:\n")
	buf.WriteString("#   - myorg/backend\n")
	buf.WriteString("#   - myorg/frontend\n")
	buf.WriteString("#   - org:myorg\n")
	buf.WriteString("\n")
	buf.WriteString("# Campaigns with workflows MUST be scoped via scope\n")
	buf.WriteString("\n")
	buf.WriteString("workflows:\n")
	buf.WriteString("  - vulnerability-scanner\n")
	buf.WriteString("  - dependency-updater\n")
	buf.WriteString("owners:\n")
	buf.WriteString("  - @security-team\n")
	buf.WriteString("```\n")

	// Use restrictive permissions (0644) for proper git tracking
	if err := os.WriteFile(fullPath, []byte(buf.String()), 0o644); err != nil {
		return "", fmt.Errorf("failed to write campaign spec file '%s': %w", relPath, err)
	}

	return relPath, nil
}

func boolPtr(v bool) *bool {
	return &v
}

// getCurrentRepository gets the current repository from git context
// This function mirrors the logic from pkg/workflow/repository_features_validation.go
func getCurrentRepository() (string, error) {
	// Use native repository.Current() to get the current repository
	// This works when in a git repository with GitHub remote and respects GH_REPO
	repo, err := repository.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current repository: %w", err)
	}

	// Validate that owner and name are not empty
	if repo.Owner == "" || repo.Name == "" {
		return "", fmt.Errorf("repository owner or name is empty (owner: %q, name: %q)", repo.Owner, repo.Name)
	}

	return fmt.Sprintf("%s/%s", repo.Owner, repo.Name), nil
}
