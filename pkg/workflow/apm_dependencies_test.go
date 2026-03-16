//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAPMDependenciesFromFrontmatter(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      map[string]any
		expectedDeps     []string
		expectedIsolated bool
		hasDefaultApp    bool
		perPackageApps   int // number of entries that have their own github-app
	}{
		{
			name: "No dependencies field",
			frontmatter: map[string]any{
				"engine": "copilot",
			},
			expectedDeps: nil,
		},
		{
			name: "Single dependency (array format)",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package"},
		},
		{
			name: "Multiple dependencies (array format)",
			frontmatter: map[string]any{
				"dependencies": []any{
					"microsoft/apm-sample-package",
					"github/awesome-copilot/skills/review-and-refactor",
					"anthropics/skills/skills/frontend-design",
				},
			},
			expectedDeps: []string{
				"microsoft/apm-sample-package",
				"github/awesome-copilot/skills/review-and-refactor",
				"anthropics/skills/skills/frontend-design",
			},
		},
		{
			name: "Empty array",
			frontmatter: map[string]any{
				"dependencies": []any{},
			},
			expectedDeps: nil,
		},
		{
			name: "Non-array, non-object value is ignored",
			frontmatter: map[string]any{
				"dependencies": "microsoft/apm-sample-package",
			},
			expectedDeps: nil,
		},
		{
			name: "Empty string items are skipped",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package", "", "github/awesome-copilot"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
		},
		{
			name: "Object format with packages only",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{
						"microsoft/apm-sample-package",
						"github/awesome-copilot",
					},
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
			expectedIsolated: false,
		},
		{
			name: "Object format with isolated true",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": true,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: true,
		},
		{
			name: "Object format with isolated false",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": false,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: false,
		},
		{
			name: "Object format with empty packages",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{},
				},
			},
			expectedDeps: nil,
		},
		// New: github-app support
		{
			name: "Object format with default github-app",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
					"packages": []any{
						"acme-platform-org/acme-skills/plugins/dev-tools",
					},
				},
			},
			expectedDeps:  []string{"acme-platform-org/acme-skills/plugins/dev-tools"},
			hasDefaultApp: true,
		},
		{
			name: "Object format with default github-app and repositories",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"github-app": map[string]any{
						"app-id":       "${{ vars.APP_ID }}",
						"private-key":  "${{ secrets.APP_PRIVATE_KEY }}",
						"repositories": []any{"*"},
					},
					"packages": []any{
						"acme-platform-org/acme-skills/plugins/dev-tools",
						"acme-platform-org/another-package",
					},
				},
			},
			expectedDeps:  []string{"acme-platform-org/acme-skills/plugins/dev-tools", "acme-platform-org/another-package"},
			hasDefaultApp: true,
		},
		{
			name: "Object format with per-package github-app override",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
					"packages": []any{
						"acme-platform-org/acme-skills/plugins/dev-tools",
						map[string]any{
							"source": "partner-org/partner-package",
							"github-app": map[string]any{
								"app-id":      "${{ vars.PARTNER_APP_ID }}",
								"private-key": "${{ secrets.PARTNER_APP_PRIVATE_KEY }}",
							},
						},
					},
				},
			},
			expectedDeps:   []string{"acme-platform-org/acme-skills/plugins/dev-tools", "partner-org/partner-package"},
			hasDefaultApp:  true,
			perPackageApps: 1,
		},
		{
			name: "Object format with only per-package github-app (no default)",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{
						map[string]any{
							"source": "partner-org/partner-package",
							"github-app": map[string]any{
								"app-id":      "${{ vars.PARTNER_APP_ID }}",
								"private-key": "${{ secrets.PARTNER_APP_PRIVATE_KEY }}",
							},
						},
					},
				},
			},
			expectedDeps:   []string{"partner-org/partner-package"},
			hasDefaultApp:  false,
			perPackageApps: 1,
		},
		{
			name: "Object entry without source is skipped",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{
						map[string]any{
							"github-app": map[string]any{
								"app-id":      "${{ vars.APP_ID }}",
								"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
							},
						},
					},
				},
			},
			expectedDeps: nil, // empty source → skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAPMDependenciesFromFrontmatter(tt.frontmatter)
			if tt.expectedDeps == nil {
				assert.Nil(t, result, "Should return nil for no dependencies")
			} else {
				require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
				assert.Equal(t, tt.expectedDeps, result.Packages, "Extracted packages should match expected")
				assert.Equal(t, tt.expectedIsolated, result.Isolated, "Isolated flag should match expected")
				if tt.hasDefaultApp {
					assert.NotNil(t, result.GitHubApp, "Default github-app should be present")
				} else {
					assert.Nil(t, result.GitHubApp, "Default github-app should be absent")
				}
				// Count entries with a per-package GitHubApp
				perPkgCount := 0
				for _, e := range result.Entries {
					if e.GitHubApp != nil {
						perPkgCount++
					}
				}
				assert.Equal(t, tt.perPackageApps, perPkgCount, "Per-package github-app count should match")
			}
		})
	}
}

func TestAPMDependenciesHasGitHubApp(t *testing.T) {
	tests := []struct {
		name     string
		deps     *APMDependenciesInfo
		expected bool
	}{
		{
			name:     "No github-app configured",
			deps:     &APMDependenciesInfo{Packages: []string{"pkg1"}, Entries: []APMPackageEntry{{Source: "pkg1"}}},
			expected: false,
		},
		{
			name: "Default github-app configured",
			deps: &APMDependenciesInfo{
				Packages:  []string{"pkg1"},
				Entries:   []APMPackageEntry{{Source: "pkg1"}},
				GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key"},
			},
			expected: true,
		},
		{
			name: "Per-package github-app configured",
			deps: &APMDependenciesInfo{
				Packages: []string{"pkg1"},
				Entries: []APMPackageEntry{
					{Source: "pkg1", GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key"}},
				},
			},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.deps.HasGitHubApp(), "HasGitHubApp should match expected")
		})
	}
}

func TestGroupAPMEntriesByApp(t *testing.T) {
	tests := []struct {
		name           string
		deps           *APMDependenciesInfo
		expectedGroups int
		groupSizes     []int // expected number of packages per group (in order)
	}{
		{
			name: "Simple case - no github-app gives single group",
			deps: &APMDependenciesInfo{
				Packages: []string{"pkg1", "pkg2"},
				Entries:  []APMPackageEntry{{Source: "pkg1"}, {Source: "pkg2"}},
			},
			expectedGroups: 1,
			groupSizes:     []int{2},
		},
		{
			name: "Default github-app only - single group",
			deps: &APMDependenciesInfo{
				Packages:  []string{"pkg1", "pkg2"},
				Entries:   []APMPackageEntry{{Source: "pkg1"}, {Source: "pkg2"}},
				GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key"},
			},
			expectedGroups: 1,
			groupSizes:     []int{2},
		},
		{
			name: "Default github-app + per-package override with different app - two groups",
			deps: &APMDependenciesInfo{
				Packages: []string{"pkg1", "pkg2"},
				Entries: []APMPackageEntry{
					{Source: "pkg1"}, // uses default app
					{Source: "pkg2", GitHubApp: &GitHubAppConfig{AppID: "456", PrivateKey: "key2"}},
				},
				GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key1"},
			},
			expectedGroups: 2,
			groupSizes:     []int{1, 1},
		},
		{
			name: "Two packages with same per-package github-app - merged into one group",
			deps: &APMDependenciesInfo{
				Packages: []string{"pkg1", "pkg2"},
				Entries: []APMPackageEntry{
					{Source: "pkg1", GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key"}},
					{Source: "pkg2", GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key"}},
				},
			},
			expectedGroups: 1,
			groupSizes:     []int{2},
		},
		{
			name: "Three packages across three different apps",
			deps: &APMDependenciesInfo{
				Packages: []string{"pkg1", "pkg2", "pkg3"},
				Entries: []APMPackageEntry{
					{Source: "pkg1"}, // uses default app
					{Source: "pkg2", GitHubApp: &GitHubAppConfig{AppID: "456", PrivateKey: "key2"}},
					{Source: "pkg3", GitHubApp: &GitHubAppConfig{AppID: "789", PrivateKey: "key3"}},
				},
				GitHubApp: &GitHubAppConfig{AppID: "123", PrivateKey: "key1"},
			},
			expectedGroups: 3,
			groupSizes:     []int{1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := groupAPMEntriesByApp(tt.deps)
			require.Len(t, groups, tt.expectedGroups, "Number of groups should match")
			for i, g := range groups {
				assert.Equal(t, i, g.Index, "Group index should match position")
				assert.Len(t, g.Packages, tt.groupSizes[i], "Group %d package count should match", i)
			}
		})
	}
}

func TestEngineGetAPMTarget(t *testing.T) {
	tests := []struct {
		name     string
		engine   CodingAgentEngine
		expected string
	}{
		{name: "copilot engine returns copilot", engine: NewCopilotEngine(), expected: "copilot"},
		{name: "claude engine returns claude", engine: NewClaudeEngine(), expected: "claude"},
		{name: "codex engine returns all", engine: NewCodexEngine(), expected: "all"},
		{name: "gemini engine returns all", engine: NewGeminiEngine(), expected: "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.engine.GetAPMTarget()
			assert.Equal(t, tt.expected, result, "APM target should match for engine %s", tt.engine.GetID())
		})
	}
}

func TestGenerateAPMPackStep(t *testing.T) {
	tests := []struct {
		name             string
		apmDeps          *APMDependenciesInfo
		target           string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:    "Single dependency with copilot target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Entries: []APMPackageEntry{{Source: "microsoft/apm-sample-package"}}},
			target:  "copilot",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"dependencies: |",
				"- microsoft/apm-sample-package",
				"isolated: 'true'",
				"pack: 'true'",
				"archive: 'true'",
				"target: copilot",
				"working-directory: /tmp/gh-aw/apm-workspace",
			},
		},
		{
			name:    "Multiple dependencies with claude target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package", "github/skills/review"}, Entries: []APMPackageEntry{{Source: "microsoft/apm-sample-package"}, {Source: "github/skills/review"}}},
			target:  "claude",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"- microsoft/apm-sample-package",
				"- github/skills/review",
				"target: claude",
			},
		},
		{
			name:    "All target for non-copilot/claude engine",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Entries: []APMPackageEntry{{Source: "microsoft/apm-sample-package"}}},
			target:  "all",
			expectedContains: []string{
				"target: all",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMPackStep(tt.apmDeps, tt.target, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

			var sb strings.Builder
			for _, line := range step {
				sb.WriteString(line + "\n")
			}
			combined := sb.String()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
		})
	}
}

func TestGenerateAPMPackStepForGroup(t *testing.T) {
	tests := []struct {
		name             string
		group            APMPackageGroup
		target           string
		tokenStepID      string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Empty group returns empty step",
			group:         APMPackageGroup{Packages: []string{}, Index: 0},
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:        "Single package group without token",
			group:       APMPackageGroup{Packages: []string{"microsoft/apm-sample-package"}, Index: 0},
			target:      "copilot",
			tokenStepID: "",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack_0",
				"microsoft/apm-action",
				"- microsoft/apm-sample-package",
				"target: copilot",
				"working-directory: /tmp/gh-aw/apm-workspace-0",
			},
		},
		{
			name:        "Group with GitHub App token",
			group:       APMPackageGroup{Packages: []string{"acme-org/acme-skills"}, Index: 1},
			target:      "claude",
			tokenStepID: "apm-app-token-1",
			expectedContains: []string{
				"id: apm_pack_1",
				"env:",
				"GITHUB_TOKEN: ${{ steps.apm-app-token-1.outputs.token }}",
				"- acme-org/acme-skills",
				"target: claude",
				"working-directory: /tmp/gh-aw/apm-workspace-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMPackStepForGroup(tt.group, tt.target, tt.tokenStepID, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty group")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

			var sb strings.Builder
			for _, line := range step {
				sb.WriteString(line + "\n")
			}
			combined := sb.String()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
		})
	}
}

func TestGenerateAPMRestoreStep(t *testing.T) {
	tests := []struct {
		name                string
		apmDeps             *APMDependenciesInfo
		expectedContains    []string
		expectedNotContains []string
		expectedEmpty       bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			expectedEmpty: true,
		},
		{
			name:    "Non-isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: false},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
			},
			expectedNotContains: []string{"isolated"},
		},
		{
			name:    "Isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: true},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
				"isolated: 'true'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMRestoreStep(tt.apmDeps, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")

			var sb strings.Builder
			for _, line := range step {
				sb.WriteString(line + "\n")
			}
			combined := sb.String()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, combined, notExpected, "Step should not contain: %s", notExpected)
			}
		})
	}
}
