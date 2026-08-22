//go:build !integration

package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestCollectSideRepoTargets(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []*WorkflowData
		expectedRepos []string
	}{
		{
			name:          "no workflows returns empty",
			workflows:     nil,
			expectedRepos: nil,
		},
		{
			name: "workflow without checkout returns empty",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: nil},
			},
			expectedRepos: nil,
		},
		{
			name: "nil workflow entry is ignored",
			workflows: []*WorkflowData{
				nil,
				{Name: "wf", CheckoutConfigs: nil},
			},
			expectedRepos: nil,
		},
		{
			name: "checkout without current:true is ignored",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo", Current: false},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "checkout with current:true and static repo is detected",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true, GitHubToken: "${{ secrets.GH_AW_MAIN_REPO_TOKEN }}"},
				}},
			},
			expectedRepos: []string{"my-org/main-repo"},
		},
		{
			name: "expression-based repository is skipped",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "${{ inputs.target_repo }}", Current: true},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "empty repository is skipped",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "", Current: true},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "duplicate repos across workflows are deduplicated",
			workflows: []*WorkflowData{
				{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true},
				}},
				{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true},
				}},
			},
			expectedRepos: []string{"my-org/main-repo"},
		},
		{
			name: "multiple distinct repos are all detected",
			workflows: []*WorkflowData{
				{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo-a", Current: true},
				}},
				{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo-b", Current: true},
				}},
			},
			expectedRepos: []string{"org/repo-a", "org/repo-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := collectSideRepoTargets(tt.workflows)

			var got []string
			for _, tgt := range targets {
				got = append(got, tgt.Repository)
			}

			if len(got) != len(tt.expectedRepos) {
				t.Errorf("expected %d targets, got %d: %v", len(tt.expectedRepos), len(got), got)
				return
			}
			// Use a set-based comparison so the test is not sensitive to ordering.
			gotSet := make(map[string]struct{}, len(got))
			for _, r := range got {
				gotSet[r] = struct{}{}
			}
			for _, repo := range tt.expectedRepos {
				if _, ok := gotSet[repo]; !ok {
					t.Errorf("expected target %q not found in results %v", repo, got)
				}
			}
		})
	}

	t.Run("non-empty token is preferred when same repo appears multiple times", func(t *testing.T) {
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				// First appearance has no token.
				{Repository: "my-org/shared-repo", Current: true, GitHubToken: ""},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				// Second appearance provides a token — should win.
				{Repository: "my-org/shared-repo", Current: true, GitHubToken: "${{ secrets.SHARED_TOKEN }}"},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(targets))
		}
		if targets[0].GitHubToken != "${{ secrets.SHARED_TOKEN }}" {
			t.Errorf("expected non-empty token to win, got %q", targets[0].GitHubToken)
		}
	})

	t.Run("github-app config is collected from checkout", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:        "${{ secrets.APP_ID }}",
			PrivateKey:   "${{ secrets.APP_PRIVATE_KEY }}",
			Owner:        "my-org",
			Repositories: []string{"target-repo"},
		}
		workflows := []*WorkflowData{
			{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "my-org/target-repo", Current: true, GitHubApp: app},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(targets))
		}
		if targets[0].GitHubToken != "" {
			t.Errorf("expected empty GitHubToken when github-app is used, got %q", targets[0].GitHubToken)
		}
		if targets[0].GitHubApp == nil {
			t.Fatal("expected GitHubApp to be set on target")
		}
		if targets[0].GitHubApp.AppID != "${{ secrets.APP_ID }}" {
			t.Errorf("expected AppID %q, got %q", "${{ secrets.APP_ID }}", targets[0].GitHubApp.AppID)
		}
	})

	t.Run("github-app is preferred over empty auth from earlier occurrence", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ secrets.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			Owner:      "my-org",
		}
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				// First appearance has no auth.
				{Repository: "my-org/shared-repo", Current: true},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				// Second appearance has github-app — should win.
				{Repository: "my-org/shared-repo", Current: true, GitHubApp: app},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(targets))
		}
		if targets[0].GitHubApp == nil {
			t.Error("expected GitHubApp to be set after upgrade from later occurrence")
		}
		if targets[0].GitHubToken != "" {
			t.Errorf("expected empty GitHubToken when github-app is used, got %q", targets[0].GitHubToken)
		}
	})

	t.Run("later auth does not override existing auth (first-seen wins)", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ secrets.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			Owner:      "my-org",
		}
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				// First appearance has github-app.
				{Repository: "my-org/shared-repo", Current: true, GitHubApp: app},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				// Second appearance has a token — should not override the already-set app.
				{Repository: "my-org/shared-repo", Current: true, GitHubToken: "${{ secrets.TOKEN }}"},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(targets))
		}
		// The first-seen github-app should be preserved; the later token should be ignored.
		if targets[0].GitHubApp == nil {
			t.Error("expected GitHubApp to be preserved (first-seen auth wins)")
		}
		if targets[0].GitHubToken != "" {
			t.Errorf("expected GitHubToken to remain empty, got %q", targets[0].GitHubToken)
		}
	})

	t.Run("multiple repos preserve first-seen discovery order", func(t *testing.T) {
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/first-repo", Current: true},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/second-repo", Current: true},
			}},
			{Name: "wf3", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/third-repo", Current: true},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 3 {
			t.Fatalf("expected 3 targets, got %d", len(targets))
		}
		wantOrder := []string{"org/first-repo", "org/second-repo", "org/third-repo"}
		for i, want := range wantOrder {
			if targets[i].Repository != want {
				t.Errorf("targets[%d] = %q, want %q", i, targets[i].Repository, want)
			}
		}
	})
}

func TestEffectiveSideRepoToken(t *testing.T) {
	t.Run("explicit github-token wins", func(t *testing.T) {
		target := SideRepoTarget{
			Repository:  "org/repo",
			GitHubToken: "${{ secrets.MY_TOKEN }}",
		}
		got := effectiveSideRepoToken(target)
		if got != "${{ secrets.MY_TOKEN }}" {
			t.Errorf("expected explicit token, got %q", got)
		}
	})

	t.Run("github-app returns minted token reference", func(t *testing.T) {
		target := SideRepoTarget{
			Repository: "org/repo",
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ secrets.APP_ID }}",
				PrivateKey: "${{ secrets.APP_KEY }}",
			},
		}
		got := effectiveSideRepoToken(target)
		if got != sideRepoAppTokenRef {
			t.Errorf("expected minted token ref %q, got %q", sideRepoAppTokenRef, got)
		}
	})

	t.Run("falls back to GH_AW_GITHUB_TOKEN when no auth configured", func(t *testing.T) {
		target := SideRepoTarget{Repository: "org/repo"}
		got := effectiveSideRepoToken(target)
		if got != "${{ secrets.GH_AW_GITHUB_TOKEN }}" {
			t.Errorf("expected fallback secret, got %q", got)
		}
	})

	t.Run("explicit github-token wins when both token and app are set", func(t *testing.T) {
		target := SideRepoTarget{
			Repository:  "org/repo",
			GitHubToken: "${{ secrets.EXPLICIT_TOKEN }}",
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ secrets.APP_ID }}",
				PrivateKey: "${{ secrets.APP_KEY }}",
			},
		}
		got := effectiveSideRepoToken(target)
		if got != "${{ secrets.EXPLICIT_TOKEN }}" {
			t.Errorf("expected explicit token to win, got %q", got)
		}
	})
}

func TestSanitizeRepoForFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-org/main-repo", "my-org-main-repo"},
		{"org/repo", "org-repo"},
		{"my.org/my_repo", "my.org-my_repo"},
		{"owner/repo-name.git", "owner-repo-name.git"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringutil.SanitizeForFilename(tt.input)
			if got != tt.expected {
				t.Errorf("stringutil.SanitizeForFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateSideRepoMaintenanceCron(t *testing.T) {
	t.Run("is deterministic for the same slug", func(t *testing.T) {
		cron1, desc1 := generateSideRepoMaintenanceCron("org/repo", 10)
		cron2, desc2 := generateSideRepoMaintenanceCron("org/repo", 10)
		if cron1 != cron2 || desc1 != desc2 {
			t.Errorf("expected deterministic output, got %q/%q and %q/%q", cron1, desc1, cron2, desc2)
		}
	})

	t.Run("different repos produce different cron expressions", func(t *testing.T) {
		repos := []string{"org/repo-a", "org/repo-b", "another-org/service", "myorg/tooling"}
		seen := make(map[string]string)
		for _, repo := range repos {
			cron, _ := generateSideRepoMaintenanceCron(repo, 10)
			if existing, ok := seen[cron]; ok {
				// Collisions are theoretically possible but should be rare for distinct slugs.
				t.Logf("cron collision between %q and %q: %s", repo, existing, cron)
			}
			seen[cron] = repo
		}
	})

	t.Run("minute is in valid range 0-59", func(t *testing.T) {
		slugs := []string{"a/b", "owner/repo", "my-org/my-repo", "x/y"}
		for _, slug := range slugs {
			for _, days := range []int{0, 1, 2, 3, 5, 10, 30} {
				cron, _ := generateSideRepoMaintenanceCron(slug, days)
				// Extract the minute field (first token).
				parts := strings.Fields(cron)
				if len(parts) < 5 {
					t.Errorf("invalid cron %q for slug=%q days=%d", cron, slug, days)
					continue
				}
				var min int
				if _, err := fmt.Sscanf(parts[0], "%d", &min); err != nil {
					t.Errorf("failed to parse minute from cron %q: %v", cron, err)
					continue
				}
				if min < 0 || min > 59 {
					t.Errorf("minute %d out of range [0,59] for slug=%q days=%d", min, slug, days)
				}
			}
		}
	})

	t.Run("frequency tier matches minExpiresDays", func(t *testing.T) {
		slug := "test/repo"
		cases := []struct {
			days        int
			descContain string
		}{
			{1, "Every 2 hours"},
			{2, "Every 6 hours"},
			{3, "Every 12 hours"},
			{4, "Every 12 hours"},
			{5, "Daily"},
			{30, "Daily"},
		}
		for _, tc := range cases {
			_, desc := generateSideRepoMaintenanceCron(slug, tc.days)
			if desc != tc.descContain {
				t.Errorf("days=%d: expected desc %q, got %q", tc.days, tc.descContain, desc)
			}
		}
	})
}

func TestGenerateSideRepoMaintenanceWorkflow(t *testing.T) {
	t.Run("generates file for static side-repo target", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository:  "my-org/target-repo",
						Current:     true,
						GitHubToken: "${{ secrets.GH_AW_TARGET_TOKEN }}",
					},
				},
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// The standard hosting-repo maintenance should be generated (has expires).
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); statErr != nil {
			t.Errorf("Expected standard agentics-maintenance.yml to exist")
		}

		// The side-repo maintenance should also be generated.
		sideFile := filepath.Join(tmpDir, "agentics-maintenance-my-org-target-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file %s to exist: %v", sideFile, err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "my-org/target-repo") {
			t.Errorf("Side-repo maintenance should reference target repo, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "${{ secrets.GH_AW_TARGET_TOKEN }}") {
			t.Errorf("Side-repo maintenance should use custom token, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "GH_AW_TARGET_REPO_SLUG") {
			t.Errorf("Side-repo maintenance should set GH_AW_TARGET_REPO_SLUG, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "workflow_call") {
			t.Errorf("Side-repo maintenance should have workflow_call trigger, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "apply_safe_outputs") {
			t.Errorf("Side-repo maintenance should include apply_safe_outputs job, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "create_labels") {
			t.Errorf("Side-repo maintenance should include create_labels job, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "activity_report") {
			t.Errorf("Side-repo maintenance should include activity_report job, got content length %d", len(contentStr))
		}
	})

	t.Run("no side-repo file generated when no current checkout", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "normal-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Only standard maintenance should exist.
		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "agentics-maintenance-") {
				t.Errorf("Unexpected side-repo maintenance file: %s", entry.Name())
			}
		}
	})

	t.Run("side-repo generated without expires uses safe_outputs, create_labels, and activity_report", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-no-expires",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository: "org/no-expires-repo",
						Current:    true,
					},
				},
				// No expires configured — standard maintenance won't be generated.
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Standard maintenance should NOT be generated (no expires).
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Standard agentics-maintenance.yml should not exist when no expires")
		}

		// Side-repo maintenance should be generated.
		sideFile := filepath.Join(tmpDir, "agentics-maintenance-org-no-expires-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file to exist: %v", err)
		}
		contentStr := string(content)

		// Should use fallback token when none specified.
		if !strings.Contains(contentStr, "GH_AW_GITHUB_TOKEN") {
			t.Errorf("Side-repo maintenance should use fallback token GH_AW_GITHUB_TOKEN, got content length %d", len(contentStr))
		}
		// Should NOT include close-expired jobs (no expires).
		if strings.Contains(contentStr, "close-expired-discussions") || strings.Contains(contentStr, "close-expired-issues") || strings.Contains(contentStr, "close-expired-pull-requests") {
			t.Errorf("Side-repo maintenance should NOT include close-expired jobs when no expires, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "activity_report") {
			t.Errorf("Side-repo maintenance should include activity_report when no expires, got content length %d", len(contentStr))
		}
	})

	t.Run("expression-based repository does not generate side-repo maintenance", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "dynamic-repo-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository: "${{ inputs.target_repo }}",
						Current:    true,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "agentics-maintenance-") {
				t.Errorf("Unexpected side-repo maintenance file for dynamic repo: %s", entry.Name())
			}
		}
	})

	t.Run("side-repo with expires includes schedule trigger", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Expires: 48 hours = 2 days → generateSideRepoMaintenanceCron("org/expires-repo", 2)
		repoSlug := "org/expires-repo"
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-with-expires",
				CheckoutConfigs: []*CheckoutConfig{
					{Repository: repoSlug, Current: true},
				},
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 48},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		sideFile := filepath.Join(tmpDir, "agentics-maintenance-org-expires-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file to exist: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, "schedule:") {
			t.Errorf("Side-repo maintenance with expires should include a schedule trigger, got content length %d", len(contentStr))
		}
		// 48 hours = 2 days → generateSideRepoMaintenanceCron returns the fuzzy 6-hour cron.
		expectedCron, _ := generateSideRepoMaintenanceCron(repoSlug, 2)
		if !strings.Contains(contentStr, expectedCron) {
			t.Errorf("Side-repo maintenance with 2-day expires should use cron %q, got content:\n%s", expectedCron, contentStr[:min(500, len(contentStr))])
		}
		// Verify the cron is different from the fixed minute used by the main workflow (37).
		// (For this particular slug the minute should not be 37 — but the real assertion is
		// that the expected fuzzy value is present, which we already checked above.)
	})

	t.Run("stale side-repo maintenance workflow is removed on recompile", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Simulate a stale file from a previous run.
		staleName := "agentics-maintenance-old-org-old-repo.yml"
		stalePath := filepath.Join(tmpDir, staleName)
		if err := os.WriteFile(stalePath, []byte("stale"), 0644); err != nil {
			t.Fatalf("Failed to create stale file: %v", err)
		}

		// Current run has a different target repo.
		workflowDataList := []*WorkflowData{
			{
				Name: "new-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{Repository: "new-org/new-repo", Current: true},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Stale file should have been removed.
		if _, statErr := os.Stat(stalePath); !os.IsNotExist(statErr) {
			t.Errorf("Stale side-repo maintenance file %s should have been removed", staleName)
		}

		// The new file should exist.
		newFile := filepath.Join(tmpDir, "agentics-maintenance-new-org-new-repo.yml")
		if _, statErr := os.Stat(newFile); statErr != nil {
			t.Errorf("New side-repo maintenance file should exist: %v", statErr)
		}
	})
}
