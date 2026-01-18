package campaign

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		ID:         id,
		Name:       name,
		ProjectURL: "https://github.com/orgs/ORG/projects/1",
		Version:    "v1",
		State:      "planned",
		// Define the repositories this campaign can operate on (REQUIRED)
		// Examples: ["myorg/backend", "myorg/frontend", "myorg/api"]
		AllowedRepos: []string{"ORG/REPO"},
		// Optionally define organizations for org-wide scope
		// AllowedOrgs: []string{"myorg"},
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
	buf.WriteString("1. **Update allowed-repos**: Add repositories this campaign should operate on\n")
	buf.WriteString("2. **Define workflows**: List workflows to execute (e.g., `vulnerability-scanner`)\n")
	buf.WriteString("3. **Add objective & KPIs**: Define measurable success criteria\n")
	buf.WriteString("4. **Set owners**: Specify who is responsible for this campaign\n")
	buf.WriteString("5. **Compile**: Run `gh aw compile` to generate the orchestrator\n\n")
	buf.WriteString("## Example Configuration\n\n")
	buf.WriteString("```yaml\n")
	buf.WriteString("# Add to the frontmatter above:\n")
	buf.WriteString("objective: \"Reduce security vulnerabilities across all repositories\"\n")
	buf.WriteString("workflows:\n")
	buf.WriteString("  - vulnerability-scanner\n")
	buf.WriteString("  - dependency-updater\n")
	buf.WriteString("owners:\n")
	buf.WriteString("  - @security-team\n")
	buf.WriteString("kpis:\n")
	buf.WriteString("  - name: \"Critical vulnerabilities resolved\"\n")
	buf.WriteString("    priority: primary\n")
	buf.WriteString("    unit: count\n")
	buf.WriteString("    baseline: 0\n")
	buf.WriteString("    target: 50\n")
	buf.WriteString("    time-window-days: 30\n")
	buf.WriteString("    direction: increase\n")
	buf.WriteString("    source: code_security\n")
	buf.WriteString("```\n")

	// Use restrictive permissions (0600) to follow security best practices
	if err := os.WriteFile(fullPath, []byte(buf.String()), 0o600); err != nil {
		return "", fmt.Errorf("failed to write campaign spec file '%s': %w", relPath, err)
	}

	return relPath, nil
}

func boolPtr(v bool) *bool {
	return &v
}
