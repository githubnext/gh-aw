package parser

import (
	"testing"
)

func TestIsSkillFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		want     bool
	}{
		{
			name:     "Valid SKILL file with proper frontmatter",
			filePath: "/path/to/developer/SKILL.md",
			content: `---
name: developer
description: Developer instructions for the project
---

# Developer Instructions

Some content here.
`,
			want: true,
		},
		{
			name:     "SKILL.md with lowercase name",
			filePath: "/path/to/skill.md",
			content: `---
name: test-skill
description: A test skill
---

Content
`,
			want: true,
		},
		{
			name:     "SKILL file missing name field",
			filePath: "/path/to/SKILL.md",
			content: `---
description: A skill without name
---

Content
`,
			want: false,
		},
		{
			name:     "SKILL file missing description field",
			filePath: "/path/to/SKILL.md",
			content: `---
name: test-skill
---

Content
`,
			want: false,
		},
		{
			name:     "SKILL file with no frontmatter",
			filePath: "/path/to/SKILL.md",
			content: `# Just a regular markdown file

With no frontmatter.
`,
			want: false,
		},
		{
			name:     "Regular markdown file not named SKILL.md",
			filePath: "/path/to/README.md",
			content: `---
name: something
description: something else
---

Content
`,
			want: false,
		},
		{
			name:     "File with SKILL in path but not in filename",
			filePath: "/path/to/skills/guide.md",
			content: `---
name: guide
description: A guide
---

Content
`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSkillFile(tt.filePath, tt.content)
			if got != tt.want {
				t.Errorf("isSkillFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
