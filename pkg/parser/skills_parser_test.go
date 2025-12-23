package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSkillImport(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		baseDir    string
		setupFunc  func(t *testing.T, tmpDir string) string
		want       bool
	}{
		{
			name:       "direct SKILL.md reference",
			importPath: "skills/test-skill/SKILL.md",
			baseDir:    "/tmp",
			setupFunc: func(t *testing.T, tmpDir string) string {
				skillDir := filepath.Join(tmpDir, "skills", "test-skill")
				if err := os.MkdirAll(skillDir, 0755); err != nil {
					t.Fatal(err)
				}
				skillFile := filepath.Join(skillDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte("---\nname: test\n---\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return tmpDir
			},
			want: true,
		},
		{
			name:       "skills directory path",
			importPath: "skills/test-skill",
			baseDir:    "/tmp",
			setupFunc: func(t *testing.T, tmpDir string) string {
				skillDir := filepath.Join(tmpDir, "skills", "test-skill")
				if err := os.MkdirAll(skillDir, 0755); err != nil {
					t.Fatal(err)
				}
				skillFile := filepath.Join(skillDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte("---\nname: test\n---\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return tmpDir
			},
			want: true,
		},
		{
			name:       "path contains skills directory",
			importPath: "/home/user/repo/skills/my-skill/SKILL.md",
			baseDir:    "/tmp",
			setupFunc:  nil,
			want:       true,
		},
		{
			name:       "non-skill import",
			importPath: "shared/workflow.md",
			baseDir:    "/tmp",
			setupFunc:  nil,
			want:       false,
		},
		{
			name:       "empty path",
			importPath: "",
			baseDir:    "/tmp",
			setupFunc:  nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.baseDir
			importPath := tt.importPath

			if tt.setupFunc != nil {
				tmpDir := t.TempDir()
				baseDir = tt.setupFunc(t, tmpDir)
				// Adjust import path to use temp directory
				if !filepath.IsAbs(tt.importPath) {
					importPath = filepath.Join(tmpDir, tt.importPath)
				}
			}

			got := IsSkillImport(importPath, baseDir)
			if got != tt.want {
				t.Errorf("IsSkillImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSkillMetadata(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantValid   bool
		wantErr     bool
		setupFunc   func(t *testing.T, tmpDir string, content string) string
	}{
		{
			name: "valid skill with name and description",
			content: `---
name: test-skill
description: A test skill for validation
---

# Test Skill

This is a test skill.
`,
			wantName:  "test-skill",
			wantDesc:  "A test skill for validation",
			wantValid: true,
			wantErr:   false,
			setupFunc: func(t *testing.T, tmpDir string, content string) string {
				skillFile := filepath.Join(tmpDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return skillFile
			},
		},
		{
			name: "skill missing name",
			content: `---
description: A test skill without name
---

# Test Skill
`,
			wantName:  "",
			wantDesc:  "A test skill without name",
			wantValid: false,
			wantErr:   false,
			setupFunc: func(t *testing.T, tmpDir string, content string) string {
				skillFile := filepath.Join(tmpDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return skillFile
			},
		},
		{
			name: "skill missing description",
			content: `---
name: test-skill
---

# Test Skill
`,
			wantName:  "test-skill",
			wantDesc:  "",
			wantValid: false,
			wantErr:   false,
			setupFunc: func(t *testing.T, tmpDir string, content string) string {
				skillFile := filepath.Join(tmpDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return skillFile
			},
		},
		{
			name: "skill with multiline description",
			content: `---
name: multi-line-skill
description: |
  This is a multiline description
  that spans multiple lines
---

# Multi-line Skill
`,
			wantName:  "multi-line-skill",
			wantDesc:  "This is a multiline description\nthat spans multiple lines",
			wantValid: true,
			wantErr:   false,
			setupFunc: func(t *testing.T, tmpDir string, content string) string {
				skillFile := filepath.Join(tmpDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return skillFile
			},
		},
		{
			name:      "empty frontmatter",
			content:   "---\n---\n\n# Test Skill",
			wantName:  "",
			wantDesc:  "",
			wantValid: false,
			wantErr:   false,
			setupFunc: func(t *testing.T, tmpDir string, content string) string {
				skillFile := filepath.Join(tmpDir, "SKILL.md")
				if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return skillFile
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillPath := tt.setupFunc(t, tmpDir, tt.content)

			got, err := ParseSkillMetadata(skillPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSkillMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if got.Name != tt.wantName {
					t.Errorf("ParseSkillMetadata().Name = %v, want %v", got.Name, tt.wantName)
				}
				if got.Description != tt.wantDesc {
					t.Errorf("ParseSkillMetadata().Description = %v, want %v", got.Description, tt.wantDesc)
				}
				if got.IsValid != tt.wantValid {
					t.Errorf("ParseSkillMetadata().IsValid = %v, want %v", got.IsValid, tt.wantValid)
				}
			}
		})
	}
}

func TestParseSkillMetadataDirectory(t *testing.T) {
	// Test that ParseSkillMetadata can handle directory paths
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `---
name: directory-skill
description: Skill parsed from directory path
---

# Directory Skill
`
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Pass directory path instead of file path
	metadata, err := ParseSkillMetadata(skillDir)
	if err != nil {
		t.Fatalf("ParseSkillMetadata() error = %v", err)
	}

	if metadata.Name != "directory-skill" {
		t.Errorf("Name = %v, want directory-skill", metadata.Name)
	}
	if metadata.Description != "Skill parsed from directory path" {
		t.Errorf("Description = %v, want 'Skill parsed from directory path'", metadata.Description)
	}
	if !metadata.IsValid {
		t.Errorf("IsValid = false, want true")
	}
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skill directories
	skills := []struct {
		name string
		path string
	}{
		{"skill-one", "skills/skill-one"},
		{"skill-two", "skills/skill-two"},
		{"nested-skill", "skills/category/nested-skill"},
	}

	for _, skill := range skills {
		skillDir := filepath.Join(tmpDir, skill.path)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}

		content := "---\nname: " + skill.name + "\ndescription: Test skill\n---\n\n# " + skill.name
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Discover skills
	discovered, err := DiscoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(discovered) != len(skills) {
		t.Errorf("DiscoverSkills() found %d skills, want %d", len(discovered), len(skills))
	}

	// Verify each skill was discovered
	for _, skill := range skills {
		expectedPath := filepath.Join(tmpDir, skill.path)
		found := false
		for _, discovered := range discovered {
			if discovered == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find skill at %s, but it was not discovered", expectedPath)
		}
	}
}

func TestValidateSkill(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid skill",
			content: `---
name: valid-skill
description: Valid skill for testing
---

# Valid Skill
`,
			wantErr: false,
		},
		{
			name: "invalid skill - missing name",
			content: `---
description: Missing name field
---

# Invalid Skill
`,
			wantErr: true,
		},
		{
			name: "invalid skill - missing description",
			content: `---
name: invalid-skill
---

# Invalid Skill
`,
			wantErr: true,
		},
		{
			name: "invalid skill - no frontmatter",
			content: `# Invalid Skill

No frontmatter here.
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			skillFile := filepath.Join(tmpDir, "SKILL.md")
			if err := os.WriteFile(skillFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			err := ValidateSkill(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSkill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSkillMetadataInvalidPath(t *testing.T) {
	// Test with non-existent path
	_, err := ParseSkillMetadata("/non/existent/path")
	if err == nil {
		t.Error("ParseSkillMetadata() expected error for non-existent path, got nil")
	}
}

func TestDiscoverSkillsEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	discovered, err := DiscoverSkills(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("DiscoverSkills() found %d skills in empty directory, want 0", len(discovered))
	}
}
