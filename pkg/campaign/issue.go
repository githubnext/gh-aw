package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// IssueFormData holds parsed data from a campaign issue form
type IssueFormData struct {
	CampaignName      string
	CampaignID        string
	CampaignVersion   string
	ProjectURL        string
	CampaignType      string
	Scope             string
	Constraints       string
	PriorLearnings    string
	Description       string
	AdditionalContext string
}

// ParseIssueForm extracts campaign form fields from an issue body
func ParseIssueForm(issueBody string) (*IssueFormData, error) {
	data := &IssueFormData{}

	// Extract form fields using regex
	// Match: ### Label\n\nContent (until next ### section or end of string)
	extractField := func(body, label string) string {
		// Split on ### to find sections, then look for our label
		sections := regexp.MustCompile(`(?m)^###\s+`).Split(body, -1)
		for _, section := range sections {
			// Check if this section starts with our label
			if strings.HasPrefix(strings.TrimSpace(section), label) {
				// Remove the label and return the rest
				content := strings.TrimPrefix(strings.TrimSpace(section), label)
				return strings.TrimSpace(content)
			}
		}
		return ""
	}

	data.CampaignName = extractField(issueBody, "Campaign Name")
	data.CampaignID = extractField(issueBody, "Campaign Identifier")
	data.CampaignVersion = extractField(issueBody, "Campaign Version")
	data.ProjectURL = extractField(issueBody, "Project Board URL")
	data.CampaignType = extractField(issueBody, "Campaign Type / Playbook")
	data.Scope = extractField(issueBody, "Scope")
	data.Constraints = extractField(issueBody, "Constraints")
	data.PriorLearnings = extractField(issueBody, "Notes from Prior Learnings")
	data.Description = extractField(issueBody, "Campaign Description")
	data.AdditionalContext = extractField(issueBody, "Additional Context")

	// Validate required fields
	if data.CampaignID == "" {
		return nil, fmt.Errorf("campaign identifier is required")
	}

	// Validate ID format first (security: prevent path traversal)
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(data.CampaignID) {
		return nil, fmt.Errorf("campaign identifier must use only lowercase letters, digits, and hyphens")
	}

	if data.ProjectURL == "" {
		return nil, fmt.Errorf("project board URL is required")
	}

	// Set defaults
	if data.CampaignVersion == "" {
		data.CampaignVersion = "v1"
	}

	return data, nil
}

// CreateSpecFromIssue creates a campaign spec file from parsed issue form data
func CreateSpecFromIssue(rootDir string, data *IssueFormData, force bool) (string, error) {
	workflowsDir := filepath.Join(rootDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	fileName := data.CampaignID + ".campaign.md"
	fullPath := filepath.Join(workflowsDir, fileName)
	relPath := filepath.ToSlash(filepath.Join(".github", "workflows", fileName))

	if _, err := os.Stat(fullPath); err == nil && !force {
		return "", fmt.Errorf("campaign spec already exists at %s (use --force to overwrite)", relPath)
	}

	// Build the campaign spec
	spec := CampaignSpec{
		ID:          data.CampaignID,
		Name:        data.CampaignName,
		Version:     data.CampaignVersion,
		Description: data.Description,
		ProjectURL:  data.ProjectURL,
		Workflows:   []string{},
		MemoryPaths: []string{
			fmt.Sprintf("memory/campaigns/%s-*/**", data.CampaignID),
		},
		Owners:            []string{},
		ExecutiveSponsors: []string{},
		RiskLevel:         "medium",
		State:             "active",
		Tags:              []string{},
		TrackerLabel:      fmt.Sprintf("campaign:%s", data.CampaignID),
		MetricsGlob:       fmt.Sprintf("memory/campaigns/%s-*/metrics/*.json", data.CampaignID),
		AllowedSafeOutputs: []string{
			"create-issue",
			"add-comment",
			"upload-assets",
			"update-project",
		},
		ApprovalPolicy: &CampaignApprovalPolicy{
			RequiredApprovals: 1,
			RequiredRoles:     []string{},
			ChangeControl:     false,
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal campaign spec: %w", err)
	}

	// Build the markdown content
	var buf strings.Builder
	buf.WriteString("---\n")
	buf.Write(yamlData)
	buf.WriteString("---\n\n")
	buf.WriteString(fmt.Sprintf("# %s\n\n", spec.Name))
	buf.WriteString(fmt.Sprintf("%s\n\n", spec.Description))

	if data.CampaignType != "" {
		buf.WriteString("## Campaign Type\n\n")
		buf.WriteString(fmt.Sprintf("%s\n\n", data.CampaignType))
	}

	if data.Scope != "" {
		buf.WriteString("## Scope\n\n")
		buf.WriteString(fmt.Sprintf("%s\n\n", data.Scope))
	}

	if data.Constraints != "" {
		buf.WriteString("## Constraints\n\n")
		buf.WriteString(fmt.Sprintf("%s\n\n", data.Constraints))
	}

	if data.PriorLearnings != "" {
		buf.WriteString("## Prior Learnings\n\n")
		buf.WriteString(fmt.Sprintf("%s\n\n", data.PriorLearnings))
	}

	if data.AdditionalContext != "" {
		buf.WriteString("## Additional Context\n\n")
		buf.WriteString(fmt.Sprintf("%s\n", data.AdditionalContext))
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(buf.String()), 0o644); err != nil {
		return "", fmt.Errorf("failed to write campaign spec file '%s': %w", relPath, err)
	}

	return relPath, nil
}

// ReadIssueBodyFromFile reads issue body from a file
func ReadIssueBodyFromFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read from file %s: %w", filename, err)
	}
	return string(content), nil
}
