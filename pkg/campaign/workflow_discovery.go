package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var workflowDiscoveryLog = logger.New("campaign:workflow_discovery")

// WorkflowMatch represents a discovered workflow that matches campaign goals
type WorkflowMatch struct {
	ID          string   // Workflow ID (basename without .md)
	FilePath    string   // Relative path to workflow file
	Name        string   // Workflow name from frontmatter
	Description string   // Workflow description
	Keywords    []string // Matching keywords
	Score       int      // Match score (higher is better)
}

// CampaignGoalKeywords maps campaign types to relevant keywords
var CampaignGoalKeywords = map[string][]string{
	"security": {
		"security", "vulnerability", "vulnerabilities", "scan", "scanning",
		"cve", "audit", "compliance", "threat", "detection",
	},
	"dependencies": {
		"dependency", "dependencies", "upgrade", "update", "npm", "pip",
		"package", "packages", "version", "outdated",
	},
	"documentation": {
		"doc", "docs", "documentation", "guide", "guides", "readme",
		"wiki", "reference", "tutorial",
	},
	"quality": {
		"quality", "test", "testing", "lint", "linting", "coverage",
		"code-quality", "static-analysis", "sonar",
	},
	"cicd": {
		"ci", "cd", "build", "deploy", "deployment", "release",
		"pipeline", "automation", "continuous",
	},
}

// DiscoverWorkflows scans the repository for existing workflows that match campaign goals
func DiscoverWorkflows(rootDir string, campaignGoals []string) ([]WorkflowMatch, error) {
	workflowDiscoveryLog.Printf("Discovering workflows for goals: %v", campaignGoals)

	workflowsDir := filepath.Join(rootDir, ".github", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		workflowDiscoveryLog.Print("Workflows directory does not exist")
		return []WorkflowMatch{}, nil
	}

	// Scan for .md files (agentic workflows)
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var matches []WorkflowMatch
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Skip campaign files and generated files
		if strings.HasSuffix(entry.Name(), ".campaign.md") || strings.HasSuffix(entry.Name(), ".g.md") {
			continue
		}

		filePath := filepath.Join(workflowsDir, entry.Name())
		match, err := matchWorkflow(filePath, campaignGoals)
		if err != nil {
			workflowDiscoveryLog.Printf("Failed to match workflow %s: %v", entry.Name(), err)
			continue
		}

		if match != nil {
			matches = append(matches, *match)
			workflowDiscoveryLog.Printf("Found matching workflow: %s (score: %d)", match.ID, match.Score)
		}
	}

	// Sort by score (highest first)
	sortWorkflowMatches(matches)

	workflowDiscoveryLog.Printf("Discovered %d matching workflows", len(matches))
	return matches, nil
}

// matchWorkflow checks if a workflow matches the campaign goals
func matchWorkflow(filePath string, campaignGoals []string) (*WorkflowMatch, error) {
	// Read workflow file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Extract frontmatter
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Get workflow name and description
	name := getStringField(result.Frontmatter, "name")
	description := getStringField(result.Frontmatter, "description")

	// Build searchable text (lowercase)
	searchText := strings.ToLower(name + " " + description)

	// Calculate match score
	score := 0
	matchedKeywords := []string{}

	for _, goal := range campaignGoals {
		keywords := CampaignGoalKeywords[strings.ToLower(goal)]
		for _, keyword := range keywords {
			if strings.Contains(searchText, keyword) {
				score += 10
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}
	}

	// No match if score is 0
	if score == 0 {
		return nil, nil
	}

	// Extract workflow ID from filename
	filename := filepath.Base(filePath)
	workflowID := strings.TrimSuffix(filename, ".md")

	return &WorkflowMatch{
		ID:          workflowID,
		FilePath:    filePath,
		Name:        name,
		Description: description,
		Keywords:    matchedKeywords,
		Score:       score,
	}, nil
}

// sortWorkflowMatches sorts workflow matches by score (descending)
func sortWorkflowMatches(matches []WorkflowMatch) {
	// Simple bubble sort (good enough for small lists)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

// getStringField safely extracts a string field from frontmatter
func getStringField(frontmatter map[string]any, field string) string {
	if val, ok := frontmatter[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
