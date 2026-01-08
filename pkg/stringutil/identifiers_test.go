package stringutil

import "testing"

func TestNormalizeWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "name without extension",
			input:    "weekly-research",
			expected: "weekly-research",
		},
		{
			name:     "name with .md extension",
			input:    "weekly-research.md",
			expected: "weekly-research",
		},
		{
			name:     "name with .lock.yml extension",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research",
		},
		{
			name:     "name with dots in filename",
			input:    "my.workflow.md",
			expected: "my.workflow",
		},
		{
			name:     "name with dots and lock.yml",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow",
		},
		{
			name:     "name with other extension",
			input:    "workflow.yaml",
			expected: "workflow.yaml",
		},
		{
			name:     "simple name",
			input:    "agent",
			expected: "agent",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just .md",
			input:    ".md",
			expected: "",
		},
		{
			name:     "just .lock.yml",
			input:    ".lock.yml",
			expected: "",
		},
		{
			name:     "multiple extensions priority",
			input:    "workflow.md.lock.yml",
			expected: "workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWorkflowName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSafeOutputIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "dash-separated to underscore",
			identifier: "create-issue",
			expected:   "create_issue",
		},
		{
			name:       "already underscore-separated",
			identifier: "create_issue",
			expected:   "create_issue",
		},
		{
			name:       "multiple dashes",
			identifier: "add-comment-to-issue",
			expected:   "add_comment_to_issue",
		},
		{
			name:       "mixed dashes and underscores",
			identifier: "update-pr_status",
			expected:   "update_pr_status",
		},
		{
			name:       "no dashes or underscores",
			identifier: "createissue",
			expected:   "createissue",
		},
		{
			name:       "single dash",
			identifier: "add-comment",
			expected:   "add_comment",
		},
		{
			name:       "trailing dash",
			identifier: "update-",
			expected:   "update_",
		},
		{
			name:       "leading dash",
			identifier: "-create",
			expected:   "_create",
		},
		{
			name:       "consecutive dashes",
			identifier: "create--issue",
			expected:   "create__issue",
		},
		{
			name:       "empty string",
			identifier: "",
			expected:   "",
		},
		{
			name:       "only dashes",
			identifier: "---",
			expected:   "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSafeOutputIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("NormalizeSafeOutputIdentifier(%q) = %q, want %q", tt.identifier, result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalizeWorkflowName(b *testing.B) {
	name := "weekly-research-workflow.lock.yml"
	for i := 0; i < b.N; i++ {
		NormalizeWorkflowName(name)
	}
}

func BenchmarkNormalizeSafeOutputIdentifier(b *testing.B) {
	identifier := "create-pull-request-review-comment"
	for i := 0; i < b.N; i++ {
		NormalizeSafeOutputIdentifier(identifier)
	}
}

func TestMarkdownToLockFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple markdown file",
			input:    "weekly-research.md",
			expected: "weekly-research.lock.yml",
		},
		{
			name:     "markdown file with path",
			input:    ".github/workflows/test.md",
			expected: ".github/workflows/test.lock.yml",
		},
		{
			name:     "already a lock file",
			input:    "workflow.lock.yml",
			expected: "workflow.lock.yml",
		},
		{
			name:     "file with dots in name",
			input:    "my.workflow.md",
			expected: "my.workflow.lock.yml",
		},
		{
			name:     "file without extension",
			input:    "workflow",
			expected: "workflow.lock.yml",
		},
		{
			name:     "absolute path",
			input:    "/home/user/.github/workflows/daily.md",
			expected: "/home/user/.github/workflows/daily.lock.yml",
		},
		{
			name:     "campaign workflow",
			input:    "test.campaign.md",
			expected: "test.campaign.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToLockFile(tt.input)
			if result != tt.expected {
				t.Errorf("MarkdownToLockFile(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLockFileToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lock file",
			input:    "weekly-research.lock.yml",
			expected: "weekly-research.md",
		},
		{
			name:     "lock file with path",
			input:    ".github/workflows/test.lock.yml",
			expected: ".github/workflows/test.md",
		},
		{
			name:     "already a markdown file",
			input:    "workflow.md",
			expected: "workflow.md",
		},
		{
			name:     "file with dots in name",
			input:    "my.workflow.lock.yml",
			expected: "my.workflow.md",
		},
		{
			name:     "absolute path",
			input:    "/home/user/.github/workflows/daily.lock.yml",
			expected: "/home/user/.github/workflows/daily.md",
		},
		{
			name:     "campaign lock file",
			input:    "test.campaign.lock.yml",
			expected: "test.campaign.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LockFileToMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("LockFileToMarkdown(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCampaignSpecToOrchestrator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple campaign spec",
			input:    "test.campaign.md",
			expected: "test.campaign.g.md",
		},
		{
			name:     "campaign spec with path",
			input:    ".github/workflows/prod.campaign.md",
			expected: ".github/workflows/prod.campaign.g.md",
		},
		{
			name:     "campaign spec with complex name",
			input:    "my-campaign.campaign.md",
			expected: "my-campaign.campaign.g.md",
		},
		{
			name:     "absolute path",
			input:    "/home/user/campaigns/test.campaign.md",
			expected: "/home/user/campaigns/test.campaign.g.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CampaignSpecToOrchestrator(tt.input)
			if result != tt.expected {
				t.Errorf("CampaignSpecToOrchestrator(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCampaignOrchestratorToLockFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple orchestrator",
			input:    "test.campaign.g.md",
			expected: "test.campaign.lock.yml",
		},
		{
			name:     "orchestrator with path",
			input:    ".github/workflows/prod.campaign.g.md",
			expected: ".github/workflows/prod.campaign.lock.yml",
		},
		{
			name:     "orchestrator with complex name",
			input:    "my-campaign.campaign.g.md",
			expected: "my-campaign.campaign.lock.yml",
		},
		{
			name:     "absolute path",
			input:    "/home/user/campaigns/test.campaign.g.md",
			expected: "/home/user/campaigns/test.campaign.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CampaignOrchestratorToLockFile(tt.input)
			if result != tt.expected {
				t.Errorf("CampaignOrchestratorToLockFile(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCampaignSpecToLockFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple campaign spec",
			input:    "test.campaign.md",
			expected: "test.campaign.lock.yml",
		},
		{
			name:     "campaign spec with path",
			input:    ".github/workflows/prod.campaign.md",
			expected: ".github/workflows/prod.campaign.lock.yml",
		},
		{
			name:     "campaign spec with complex name",
			input:    "my-campaign.campaign.md",
			expected: "my-campaign.campaign.lock.yml",
		},
		{
			name:     "absolute path",
			input:    "/home/user/campaigns/test.campaign.md",
			expected: "/home/user/campaigns/test.campaign.lock.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CampaignSpecToLockFile(tt.input)
			if result != tt.expected {
				t.Errorf("CampaignSpecToLockFile(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTripConversions(t *testing.T) {
	// Test that converting back and forth preserves the base name
	t.Run("markdown to lock and back", func(t *testing.T) {
		original := "workflow.md"
		lockFile := MarkdownToLockFile(original)
		backToMd := LockFileToMarkdown(lockFile)
		if backToMd != original {
			t.Errorf("Round trip failed: %q -> %q -> %q", original, lockFile, backToMd)
		}
	})

	t.Run("lock to markdown and back", func(t *testing.T) {
		original := "workflow.lock.yml"
		mdFile := LockFileToMarkdown(original)
		backToLock := MarkdownToLockFile(mdFile)
		if backToLock != original {
			t.Errorf("Round trip failed: %q -> %q -> %q", original, mdFile, backToLock)
		}
	})

	t.Run("campaign spec to orchestrator to lock", func(t *testing.T) {
		spec := "test.campaign.md"
		orchestrator := CampaignSpecToOrchestrator(spec)
		lockFile := CampaignOrchestratorToLockFile(orchestrator)
		expectedLock := "test.campaign.lock.yml"
		if lockFile != expectedLock {
			t.Errorf("Campaign chain failed: %q -> %q -> %q (expected %q)", spec, orchestrator, lockFile, expectedLock)
		}
	})

	t.Run("campaign spec direct to lock", func(t *testing.T) {
		spec := "test.campaign.md"
		lockFile := CampaignSpecToLockFile(spec)
		expectedLock := "test.campaign.lock.yml"
		if lockFile != expectedLock {
			t.Errorf("Direct campaign conversion failed: %q -> %q (expected %q)", spec, lockFile, expectedLock)
		}
	})
}
