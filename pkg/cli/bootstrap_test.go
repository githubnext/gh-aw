//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

func TestNewBootstrapCommand(t *testing.T) {
	cmd := NewBootstrapCommand(func(string) error { return nil })
	if cmd == nil {
		t.Fatal("NewBootstrapCommand returned nil")
	}
	if cmd.Use != "bootstrap [source]..." {
		t.Fatalf("unexpected use: %s", cmd.Use)
	}
	if cmd.Flags().Lookup("repo") == nil {
		t.Fatal("expected --repo flag")
	}
	if cmd.Flags().Lookup("create-repo") == nil {
		t.Fatal("expected --create-repo flag")
	}
	if cmd.Flags().Lookup("visibility") == nil {
		t.Fatal("expected --visibility flag")
	}
	if cmd.Flags().Lookup("plan") == nil {
		t.Fatal("expected --plan flag")
	}
	if cmd.Flags().Lookup("no-compile") == nil {
		t.Fatal("expected --no-compile flag")
	}
	if cmd.Flags().Lookup("engine") == nil {
		t.Fatal("expected --engine flag")
	}
	if cmd.Flags().Lookup("visibility").DefValue != "private" {
		t.Fatalf("unexpected visibility default: %s", cmd.Flags().Lookup("visibility").DefValue)
	}
	if cmd.Flags().Lookup("require-owner-type").DefValue != "any" {
		t.Fatalf("unexpected require-owner-type default: %s", cmd.Flags().Lookup("require-owner-type").DefValue)
	}
	if cmd.GroupID != "" {
		t.Fatalf("group should be assigned by main, got %q", cmd.GroupID)
	}
}

func TestNewBootstrapCommand_RequiresRepoFlagOnExecute(t *testing.T) {
	cmd := NewBootstrapCommand(func(string) error { return nil })
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing --repo error")
	}
	if err.Error() != "--repo is required\n\nRun 'bootstrap --help' for usage information" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildBootstrapPlan_AttachedCheckoutNeedsInit(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	plan, err := buildBootstrapPlan(context.Background(), normalizeBootstrapOptions(BootstrapOptions{
		Repo:             "octo/platform-ops",
		Dir:              repoDir,
		Visibility:       "private",
		RequireOwnerType: "any",
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
	}, repoDir)
	if err != nil {
		t.Fatalf("buildBootstrapPlan returned error: %v", err)
	}
	if !plan.AttachedCheckout {
		t.Fatal("expected attached checkout")
	}
	if plan.CloneRepo {
		t.Fatal("did not expect clone plan")
	}
	if !plan.InitNeeded {
		t.Fatal("expected init to be needed")
	}
	if len(plan.InitMissingMarkers) == 0 {
		t.Fatal("expected missing init markers")
	}
}

func TestBuildBootstrapPlan_EnforcesOwnerTypeRequirement(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	_, err := buildBootstrapPlan(context.Background(), normalizeBootstrapOptions(BootstrapOptions{
		Repo:             "octo/platform-ops",
		Dir:              repoDir,
		RequireOwnerType: "user",
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
	}, repoDir)
	if err == nil {
		t.Fatal("expected owner type mismatch error")
	}
	if err.Error() != "owner octo is org, but --require-owner-type=user was requested" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBootstrapWithRuntime_CreateCloneInitAddCompile(t *testing.T) {
	tempDir := testutil.TempDir(t, "bootstrap-*")
	checkoutDir := filepath.Join(tempDir, "platform-ops")

	createCalls := 0
	cloneCalls := 0
	initCalls := 0
	addCalls := 0
	compileCalls := 0

	err := runBootstrapWithRuntime(normalizeBootstrapOptions(BootstrapOptions{
		Ctx:        context.Background(),
		Repo:       "octo/platform-ops",
		Dir:        checkoutDir,
		CreateRepo: true,
		Yes:        true,
		Sources:    []string{"github/central-agentic-ops/readiness"},
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:  func(context.Context) error { return nil },
			repoExists: func(context.Context, string) (bool, error) { return false, nil },
			createRepo: func(context.Context, string, string) error {
				createCalls++
				return nil
			},
			cloneRepo: func(_ context.Context, _ string, dir string) error {
				cloneCalls++
				return os.MkdirAll(dir, 0o755)
			},
			checkCleanWorktree: func(bool) error { return nil },
		},
		confirmAction: func(string, string, string) (bool, error) { return false, nil },
		initRepo:      func(InitOptions) error { initCalls++; return nil },
		addWorkflows: func(context.Context, []string, AddOptions) (*AddWorkflowsResult, error) {
			addCalls++
			return &AddWorkflowsResult{}, nil
		},
		compileWorkflows: func(context.Context, CompileConfig) ([]*workflow.WorkflowData, error) {
			compileCalls++
			return nil, nil
		},
	}, tempDir)
	if err != nil {
		t.Fatalf("runBootstrapWithRuntime returned error: %v", err)
	}
	if createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", createCalls)
	}
	if cloneCalls != 1 {
		t.Fatalf("expected 1 clone call, got %d", cloneCalls)
	}
	if initCalls != 1 {
		t.Fatalf("expected 1 init call, got %d", initCalls)
	}
	if addCalls != 1 {
		t.Fatalf("expected 1 add call, got %d", addCalls)
	}
	if compileCalls != 1 {
		t.Fatalf("expected 1 compile call, got %d", compileCalls)
	}
}

func TestRunBootstrapWithRuntime_RequiresYesInCIWhenMutationPending(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	t.Setenv("CI", "true")

	confirmCalls := 0
	err := runBootstrapWithRuntime(normalizeBootstrapOptions(BootstrapOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  repoDir,
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
		confirmAction: func(string, string, string) (bool, error) {
			confirmCalls++
			return true, nil
		},
	}, repoDir)
	if err == nil {
		t.Fatal("expected CI confirmation error")
	}
	if err.Error() != "--yes is required in CI when bootstrap would make changes" {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmCalls != 0 {
		t.Fatalf("confirmAction should not be called in CI, got %d calls", confirmCalls)
	}
}

func TestRunBootstrapWithRuntime_PropagatesCleanWorktreeError(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	wantErr := errors.New("working directory has uncommitted changes, please commit or stash them first")

	err := runBootstrapWithRuntime(normalizeBootstrapOptions(BootstrapOptions{
		Ctx:  context.Background(),
		Repo: "octo/platform-ops",
		Dir:  repoDir,
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:     func(context.Context) error { return nil },
			repoExists:    func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo: func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error {
				return wantErr
			},
		},
	}, repoDir)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected clean worktree error, got %v", err)
	}
}

func TestRunBootstrapWithRuntime_SkipsExistingSourcedWorkflow(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	writeBootstrapMarkers(t, repoDir, "")
	workflowPath := filepath.Join(repoDir, ".github", "workflows", "readiness.md")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	content := "---\nsource: github/central-agentic-ops/readiness@main\n---\n\n# Readiness\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	plan, err := buildBootstrapPlan(context.Background(), normalizeBootstrapOptions(BootstrapOptions{
		Repo:    "octo/platform-ops",
		Dir:     repoDir,
		Sources: []string{"github/central-agentic-ops/readiness"},
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
	}, repoDir)
	if err != nil {
		t.Fatalf("buildBootstrapPlan returned error: %v", err)
	}
	if plan.NeedsMutation {
		t.Fatal("expected no-op bootstrap plan when sourced workflow is already present")
	}
	if len(plan.ResolvedSources) != 0 {
		t.Fatalf("expected no pending sources, got %d", len(plan.ResolvedSources))
	}
	if len(plan.SkippedSources) != 1 || plan.SkippedSources[0] != "readiness" {
		t.Fatalf("expected skipped readiness workflow, got %#v", plan.SkippedSources)
	}

	initCalls := 0
	addCalls := 0
	compileCalls := 0

	err = runBootstrapWithRuntime(normalizeBootstrapOptions(BootstrapOptions{
		Ctx:     context.Background(),
		Repo:    "octo/platform-ops",
		Dir:     repoDir,
		Yes:     true,
		Sources: []string{"github/central-agentic-ops/readiness"},
	}), bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return true, nil },
			dirOriginRepo:      func(string) (string, error) { return "octo/platform-ops", nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
		initRepo: func(InitOptions) error { initCalls++; return nil },
		addWorkflows: func(context.Context, []string, AddOptions) (*AddWorkflowsResult, error) {
			addCalls++
			return &AddWorkflowsResult{}, nil
		},
		compileWorkflows: func(context.Context, CompileConfig) ([]*workflow.WorkflowData, error) {
			compileCalls++
			return nil, nil
		},
	}, repoDir)
	if err != nil {
		t.Fatalf("runBootstrapWithRuntime returned error: %v", err)
	}
	if initCalls != 0 {
		t.Fatalf("expected init to be skipped, got %d calls", initCalls)
	}
	if addCalls != 0 {
		t.Fatalf("expected add to be skipped, got %d calls", addCalls)
	}
	if compileCalls != 0 {
		t.Fatalf("expected compile to be skipped, got %d calls", compileCalls)
	}
}

func initBootstrapGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := testutil.TempDir(t, "bootstrap-repo-*")
	cmd := exec.Command("git", "init", repoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git not available: %v (%s)", err, output)
	}
	return repoDir
}

func writeBootstrapMarkers(t *testing.T, repoDir string, engineOverride string) {
	t.Helper()
	for _, marker := range expectedBootstrapInitMarkers(engineOverride) {
		path := filepath.Join(repoDir, filepath.FromSlash(marker))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create marker dir for %s: %v", marker, err)
		}
		if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
			t.Fatalf("failed to create marker %s: %v", marker, err)
		}
	}
}
