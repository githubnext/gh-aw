package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var skillsLog = logger.New("parser:skills")

// SkillMetadata represents metadata about a skill parsed from a SKILL.md file
type SkillMetadata struct {
	Name        string // Skill name from YAML frontmatter
	Description string // Skill description from YAML frontmatter
	Path        string // Absolute path to the skill directory or file
	IsValid     bool   // Whether the skill has valid structure
}

// IsSkillImport checks if an import path references a skill directory or SKILL.md file
// Returns true if the path:
// - Points to a directory containing a SKILL.md file
// - Points directly to a SKILL.md file
// - Is within the skills/ directory
func IsSkillImport(importPath string, baseDir string) bool {
	skillsLog.Printf("Checking if import is a skill: path=%s, baseDir=%s", importPath, baseDir)

	// Resolve the absolute path
	absPath := importPath
	if !filepath.IsAbs(importPath) {
		absPath = filepath.Join(baseDir, importPath)
	}

	// Check if path is within skills/ directory
	if strings.Contains(absPath, "/skills/") || strings.HasPrefix(filepath.Base(filepath.Dir(absPath)), "skills") {
		skillsLog.Printf("Path is in skills directory: %s", absPath)
		return true
	}

	// Check if path directly references SKILL.md
	if filepath.Base(absPath) == "SKILL.md" {
		if _, err := os.Stat(absPath); err == nil {
			skillsLog.Printf("Path directly references SKILL.md: %s", absPath)
			return true
		}
	}

	// Check if path is a directory containing SKILL.md
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		skillFile := filepath.Join(absPath, "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			skillsLog.Printf("Path is directory with SKILL.md: %s", absPath)
			return true
		}
	}

	skillsLog.Printf("Path is not a skill import: %s", absPath)
	return false
}

// ParseSkillMetadata extracts metadata from a SKILL.md file
// Returns SkillMetadata with parsed information
func ParseSkillMetadata(skillPath string) (*SkillMetadata, error) {
	skillsLog.Printf("Parsing skill metadata from: %s", skillPath)

	// Ensure we have the SKILL.md file path
	skillFilePath := skillPath
	if info, err := os.Stat(skillPath); err == nil && info.IsDir() {
		skillFilePath = filepath.Join(skillPath, "SKILL.md")
	}

	// Read the SKILL.md file
	content, err := os.ReadFile(skillFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	// Parse frontmatter using existing ExtractFrontmatterFromContent function
	result, err := ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill frontmatter: %w", err)
	}
	
	frontmatter := result.Frontmatter

	// Extract name and description from frontmatter
	metadata := &SkillMetadata{
		Path:    filepath.Dir(skillFilePath),
		IsValid: false,
	}

	if name, ok := frontmatter["name"].(string); ok {
		metadata.Name = name
	}

	if desc, ok := frontmatter["description"].(string); ok {
		metadata.Description = desc
	}

	// Skill is valid if it has both name and description
	metadata.IsValid = metadata.Name != "" && metadata.Description != ""

	if metadata.IsValid {
		skillsLog.Printf("Successfully parsed skill: name=%s, desc=%s", metadata.Name, metadata.Description)
	} else {
		skillsLog.Printf("Skill missing required fields: name=%s, desc=%s", metadata.Name, metadata.Description)
	}

	return metadata, nil
}

// DiscoverSkills finds all SKILL.md files in a directory recursively
// Returns a list of skill directories containing SKILL.md files
func DiscoverSkills(rootDir string) ([]string, error) {
	skillsLog.Printf("Discovering skills in directory: %s", rootDir)

	var skillDirs []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this is a SKILL.md file
		if !info.IsDir() && filepath.Base(path) == "SKILL.md" {
			skillDir := filepath.Dir(path)
			skillDirs = append(skillDirs, skillDir)
			skillsLog.Printf("Found skill at: %s", skillDir)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover skills: %w", err)
	}

	skillsLog.Printf("Discovered %d skills", len(skillDirs))
	return skillDirs, nil
}

// ValidateSkill checks if a skill directory has the required structure
// Returns true if the skill has a valid SKILL.md file with required frontmatter
func ValidateSkill(skillPath string) error {
	skillsLog.Printf("Validating skill at: %s", skillPath)

	metadata, err := ParseSkillMetadata(skillPath)
	if err != nil {
		return err
	}

	if !metadata.IsValid {
		return fmt.Errorf("skill is missing required frontmatter fields (name and description)")
	}

	return nil
}
