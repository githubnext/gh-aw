package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var initRepoSetupLog = logger.New("cli:init_repo_setup")

// RepoSetupStatus represents the primary outcome of a repo setup execution.
type RepoSetupStatus string

const (
	// RepoSetupStatusAttached means the repo exists and the local dir already
	// matches the requested repo; no clone was needed.
	RepoSetupStatusAttached RepoSetupStatus = "attached"
	// RepoSetupStatusCreated means the remote repository was created.
	RepoSetupStatusCreated RepoSetupStatus = "created"
	// RepoSetupStatusCloned means the repository was cloned into the target dir.
	RepoSetupStatusCloned RepoSetupStatus = "cloned"
	// RepoSetupStatusInitialized means gh aw init markers were applied.
	RepoSetupStatusInitialized RepoSetupStatus = "initialized"
	// RepoSetupStatusBlocked means a dirty worktree prevents mutations.
	RepoSetupStatusBlocked RepoSetupStatus = "blocked"
	// RepoSetupStatusNoop means everything was already up to date; nothing changed.
	RepoSetupStatusNoop RepoSetupStatus = "noop"
)

// OwnerType restricts the kind of GitHub account that may own the repository.
type OwnerType string

const (
	OwnerTypeOrg  OwnerType = "org"
	OwnerTypeUser OwnerType = "user"
	OwnerTypeAny  OwnerType = "any"
)

// RepoSetupOptions holds all inputs for the planner/executor.
type RepoSetupOptions struct {
	// Repo is the required "OWNER/REPO" slug.
	Repo string
	// Dir is the optional target directory for the local checkout.
	// Defaults to "./<repo-name>" when empty and a clone is needed.
	Dir string
	// Create creates the remote repository when it does not exist.
	Create bool
	// Private sets the new repository to private (only relevant when Create is true).
	Private bool
	// RequireOwnerType enforces that the owner is an org or user account.
	// Valid values: "org", "user", "any" (default).
	RequireOwnerType OwnerType
	// Plan prints the planned actions without executing any mutation.
	Plan bool
	// Yes skips the interactive confirmation prompt before mutations.
	Yes bool
	// JSON emits a machine-readable JSON result to stdout.
	JSON bool
	// Verbose enables verbose logging.
	Verbose bool
}

// RepoSetupPlanAction is a single step in the plan.
type RepoSetupPlanAction struct {
	Action      string `json:"action"`
	Description string `json:"description"`
}

// RepoSetupPlan describes all steps that will be taken before any mutation.
type RepoSetupPlan struct {
	Repo         string                `json:"repo"`
	Dir          string                `json:"dir,omitempty"`
	Actions      []RepoSetupPlanAction `json:"actions"`
	HasMutations bool                  `json:"has_mutations"`
}

// RepoSetupResult is the structured result after execution (or plan-only).
type RepoSetupResult struct {
	Status      RepoSetupStatus `json:"status"`
	Repo        string          `json:"repo"`
	Dir         string          `json:"dir,omitempty"`
	Messages    []string        `json:"messages,omitempty"`
	Created     bool            `json:"created"`
	Cloned      bool            `json:"cloned"`
	Initialized bool            `json:"initialized"`
	Error       string          `json:"error,omitempty"`
}

// PlanAndExecuteRepoSetup validates inputs, builds a plan, optionally confirms
// with the user, and runs the plan.  When opts.Plan is true the function prints
// the plan and returns without mutating anything.
func PlanAndExecuteRepoSetup(opts RepoSetupOptions) (*RepoSetupResult, error) {
	// --- Input validation ---
	if opts.Repo == "" {
		return nil, errors.New("--repo is required (format: OWNER/REPO)")
	}
	owner, repoName, err := validateRepoSlug(opts.Repo)
	if err != nil {
		return nil, err
	}

	if err := validateOwnerType(owner, opts.RequireOwnerType); err != nil {
		return nil, err
	}

	// --- Prerequisites ---
	if !isGHCLIAvailable() {
		return nil, errors.New("GitHub CLI (gh) is required but not found on PATH\n  Install: https://cli.github.com/")
	}
	if err := checkGHAuthStatusShared(opts.Verbose); err != nil {
		return nil, err
	}

	// --- Remote repository resolution ---
	repoExists, err := remoteRepoExists(opts.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to check whether %s exists: %w", opts.Repo, err)
	}

	if !repoExists && !opts.Create {
		return nil, fmt.Errorf(
			"repository %s does not exist\n  To create it, add --create",
			opts.Repo,
		)
	}

	// --- Local checkout resolution ---
	targetDir, err := resolveTargetDir(opts.Dir, repoName)
	if err != nil {
		return nil, err
	}

	// --- Plan computation ---
	plan, dirState, initState, err := buildPlan(opts, repoExists, targetDir)
	if err != nil {
		return nil, err
	}

	// --- Plan mode: print and return ---
	if opts.Plan {
		printPlan(plan)
		if opts.JSON {
			return &RepoSetupResult{
				Status: RepoSetupStatusNoop,
				Repo:   opts.Repo,
				Dir:    targetDir,
			}, nil
		}
		return &RepoSetupResult{
			Status: RepoSetupStatusNoop,
			Repo:   opts.Repo,
			Dir:    targetDir,
		}, nil
	}

	// --- Blocked? ---
	if dirState == dirStateExistsDirty && plan.HasMutations {
		result := &RepoSetupResult{
			Status: RepoSetupStatusBlocked,
			Repo:   opts.Repo,
			Dir:    targetDir,
			Error:  "working directory has uncommitted changes; commit or stash before continuing",
		}
		emitResult(result, opts.JSON)
		return result, fmt.Errorf(
			"directory %s has uncommitted changes\n  Commit or stash before running init:\n    git -C %s stash",
			targetDir, targetDir,
		)
	}

	// --- Interactive confirmation ---
	if plan.HasMutations && !opts.Yes {
		printPlan(plan)
		fmt.Fprintln(os.Stderr, "")
		if !promptYesNo("Proceed with the above actions?") {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Aborted."))
			return &RepoSetupResult{Status: RepoSetupStatusNoop, Repo: opts.Repo, Dir: targetDir}, nil
		}
	}

	// --- Execution ---
	return executePlan(opts, plan, repoExists, targetDir, dirState, initState)
}

// ------------------------------ internals ----------------------------------

// dirStateType classifies the local directory.
type dirStateType int

const (
	dirStateUnknown     dirStateType = iota // not yet evaluated
	dirStateNotExist                        // directory does not exist
	dirStateNotGit                          // exists but not a git repo
	dirStateWrongRemote                     // git repo but origin points elsewhere
	dirStateMatchClean                      // correct remote, clean working tree
	dirStateExistsDirty                     // correct remote, dirty working tree
)

// initStateType classifies how much of gh aw init has been applied.
type initStateType int

const (
	initStateUnknown    initStateType = iota // cannot determine (dir not accessible yet)
	initStateComplete                        // all markers present
	initStateIncomplete                      // markers missing; init needs to run
)

func buildPlan(
	opts RepoSetupOptions,
	repoExists bool,
	targetDir string,
) (*RepoSetupPlan, dirStateType, initStateType, error) {
	plan := &RepoSetupPlan{
		Repo: opts.Repo,
		Dir:  targetDir,
	}

	var ds dirStateType
	var is initStateType

	// 1. Remote repository
	if !repoExists {
		visibility := "public"
		if opts.Private {
			visibility = "private"
		}
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "create-repo",
			Description: fmt.Sprintf("Create %s remote repository %s", visibility, opts.Repo),
		})
		plan.HasMutations = true
	}

	// 2. Local checkout
	ds = classifyDir(targetDir, opts.Repo)
	switch ds {
	case dirStateNotExist:
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "clone",
			Description: fmt.Sprintf("Clone %s into %s", opts.Repo, targetDir),
		})
		plan.HasMutations = true
		is = initStateIncomplete // will need init after clone

	case dirStateNotGit:
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "clone",
			Description: fmt.Sprintf("Clone %s into %s (directory exists but is not a git repo)", opts.Repo, targetDir),
		})
		plan.HasMutations = true
		is = initStateIncomplete

	case dirStateWrongRemote:
		existingSlug := getOriginSlugForDir(targetDir)
		return nil, ds, is, fmt.Errorf(
			"directory %s is a git repository but its origin points to %s, not %s\n  "+
				"Use a different --dir to avoid the conflict",
			targetDir, existingSlug, opts.Repo,
		)

	case dirStateMatchClean, dirStateExistsDirty:
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "attach",
			Description: "Attach to existing checkout at " + targetDir,
		})
		// Check init markers only when the checkout already exists
		is = detectInitMarkers(targetDir)
	}

	// 3. Init markers (skip when we cannot know yet – dir will be cloned)
	switch is {
	case initStateIncomplete:
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "init",
			Description: "Run gh aw init to apply initialization markers",
		})
		plan.HasMutations = true
	case initStateComplete:
		plan.Actions = append(plan.Actions, RepoSetupPlanAction{
			Action:      "noop-init",
			Description: "Repository already initialized (markers present)",
		})
	}

	return plan, ds, is, nil
}

// executePlan runs the steps computed by buildPlan.
func executePlan(
	opts RepoSetupOptions,
	plan *RepoSetupPlan,
	repoExists bool,
	targetDir string,
	ds dirStateType,
	is initStateType,
) (*RepoSetupResult, error) {
	result := &RepoSetupResult{
		Repo: opts.Repo,
		Dir:  targetDir,
	}

	// 1. Create remote repo if needed
	if !repoExists {
		if err := createRemoteRepo(opts.Repo, opts.Private, opts.Verbose); err != nil {
			result.Error = err.Error()
			result.Status = RepoSetupStatusBlocked
			emitResult(result, opts.JSON)
			return result, err
		}
		result.Created = true
		result.Messages = append(result.Messages, "Created remote repository "+opts.Repo)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created repository "+opts.Repo))
	}

	// 2. Clone if needed
	if ds == dirStateNotExist || ds == dirStateNotGit {
		if err := cloneRepo(opts.Repo, targetDir, opts.Verbose); err != nil {
			result.Error = err.Error()
			result.Status = RepoSetupStatusBlocked
			emitResult(result, opts.JSON)
			return result, err
		}
		result.Cloned = true
		result.Messages = append(result.Messages, "Cloned "+opts.Repo+" into "+targetDir)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Cloned "+opts.Repo+" into "+targetDir))
		// Re-evaluate init state now that the directory exists
		is = detectInitMarkers(targetDir)
	} else {
		result.Messages = append(result.Messages, "Attached to existing checkout at "+targetDir)
	}

	// 3. Run gh aw init when markers are missing
	switch is {
	case initStateIncomplete:
		if err := runInitInDir(targetDir, opts.Verbose); err != nil {
			result.Error = err.Error()
			result.Status = RepoSetupStatusBlocked
			emitResult(result, opts.JSON)
			return result, err
		}
		result.Initialized = true
		result.Messages = append(result.Messages, "Applied gh aw init markers")
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Repository initialized for agentic workflows"))
	case initStateComplete:
		result.Messages = append(result.Messages, "Init markers already present (skipped)")
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Init markers already present, skipping gh aw init"))
		}
	}

	// 4. Compute final status
	result.Status = computeStatus(result)

	emitResult(result, opts.JSON)
	return result, nil
}

// computeStatus derives the primary status from the individual action flags.
func computeStatus(r *RepoSetupResult) RepoSetupStatus {
	if r.Initialized {
		return RepoSetupStatusInitialized
	}
	if r.Cloned {
		return RepoSetupStatusCloned
	}
	if r.Created {
		return RepoSetupStatusCreated
	}
	if len(r.Messages) > 0 {
		return RepoSetupStatusAttached
	}
	return RepoSetupStatusNoop
}

// ------------------------------ helpers ------------------------------------

// validateRepoSlug checks that s is in OWNER/REPO form and returns the parts.
func validateRepoSlug(s string) (owner, repo string, err error) {
	owner, repo, err = repoutil.SplitRepoSlug(s)
	if err != nil {
		return "", "", fmt.Errorf(
			"invalid --repo value %q: must be OWNER/REPO (e.g. myorg/myrepo)",
			s,
		)
	}
	return owner, repo, nil
}

// validateOwnerType checks the owner name against the requested owner-type policy.
// It resolves the actual type via the GitHub API only when policy is not "any".
func validateOwnerType(owner string, required OwnerType) error {
	if required == "" || required == OwnerTypeAny {
		return nil
	}

	initRepoSetupLog.Printf("Checking owner type for %s (required: %s)", owner, required)

	out, err := workflow.RunGH("Resolving owner type...", "api", "/users/"+owner, "--jq", ".type")
	if err != nil {
		initRepoSetupLog.Printf("Could not resolve owner type for %s: %v", owner, err)
		// Non-fatal: skip type enforcement when API is unreachable
		return nil
	}

	actualType := strings.ToLower(strings.TrimSpace(string(out)))

	switch required {
	case OwnerTypeOrg:
		if actualType != "organization" {
			return fmt.Errorf(
				"owner %q is a user account, but --require-owner-type=org requires an organization",
				owner,
			)
		}
	case OwnerTypeUser:
		if actualType != "user" {
			return fmt.Errorf(
				"owner %q is an organization, but --require-owner-type=user requires a user account",
				owner,
			)
		}
	}

	initRepoSetupLog.Printf("Owner %s type check passed (actual: %s)", owner, actualType)
	return nil
}

// remoteRepoExists returns true when OWNER/REPO is accessible via the GitHub API.
func remoteRepoExists(repoSlug string) (bool, error) {
	initRepoSetupLog.Printf("Checking whether remote repo %s exists", repoSlug)
	_, err := workflow.RunGH("Checking repository...", "api", "/repos/"+repoSlug, "--jq", ".full_name")
	if err != nil {
		// 404 → repo does not exist (or no access)
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "could not resolve") {
			initRepoSetupLog.Printf("Remote repo %s does not exist (404/not-found)", repoSlug)
			return false, nil
		}
		// Any other error (auth, network) is a hard failure
		return false, err
	}
	return true, nil
}

// createRemoteRepo creates a new GitHub repository.
func createRemoteRepo(repoSlug string, private, verbose bool) error {
	initRepoSetupLog.Printf("Creating remote repository %s (private=%v)", repoSlug, private)

	_, repoName, err := repoutil.SplitRepoSlug(repoSlug)
	if err != nil {
		return err
	}

	owner := strings.SplitN(repoSlug, "/", 2)[0]

	args := []string{"repo", "create", repoName, "--source", "."}
	if private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}

	// Determine whether owner is an org and use --org flag accordingly
	orgOut, orgErr := workflow.RunGH("", "api", "/users/"+owner, "--jq", ".type")
	if orgErr == nil && strings.ToLower(strings.TrimSpace(string(orgOut))) == "organization" {
		args = append(args, "--org", owner)
	}

	spinner := fmt.Sprintf("Creating repository %s...", repoSlug)
	out, err := workflow.RunGHCombined(spinner, args...)
	if err != nil {
		return fmt.Errorf("failed to create repository %s: %w\n%s", repoSlug, err, string(out))
	}

	initRepoSetupLog.Printf("Created remote repository %s", repoSlug)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("gh repo create output: "+strings.TrimSpace(string(out))))
	}
	return nil
}

// resolveTargetDir returns the absolute path to the target directory.
// When dir is empty and a clone is needed, defaults to "./<repoName>".
func resolveTargetDir(dir, repoName string) (string, error) {
	if dir == "" {
		dir = repoName
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory %q: %w", dir, err)
	}
	return abs, nil
}

// classifyDir categorises the local directory relative to the expected repo.
func classifyDir(dir, repoSlug string) dirStateType {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return dirStateNotExist
	}
	if err != nil || !info.IsDir() {
		return dirStateNotExist
	}

	// Check whether it is a git repository
	if _, gitErr := gitutil.FindGitRootFrom(dir); gitErr != nil {
		return dirStateNotGit
	}

	// Check whether the origin points to the expected repo
	originSlug := getOriginSlugForDir(dir)
	if !slugsMatch(originSlug, repoSlug) {
		return dirStateWrongRemote
	}

	// Check cleanliness
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return dirStateMatchClean // cannot determine; assume clean
	}
	if strings.TrimSpace(string(out)) != "" {
		return dirStateExistsDirty
	}
	return dirStateMatchClean
}

// getOriginSlugForDir returns the "owner/repo" slug from the origin remote of a
// git repo at the given directory, or "" if it cannot be determined.
func getOriginSlugForDir(dir string) string {
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return parseGitHubRepoSlugFromURL(strings.TrimSpace(string(out)))
}

// slugsMatch compares two owner/repo slugs case-insensitively.
func slugsMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSuffix(a, ".git"), strings.TrimSuffix(b, ".git"))
}

// detectInitMarkers checks whether the key gh aw init artifacts are already present.
func detectInitMarkers(dir string) initStateType {
	initRepoSetupLog.Printf("Detecting init markers in %s", dir)

	gitRoot, err := gitutil.FindGitRootFrom(dir)
	if err != nil {
		return initStateIncomplete
	}

	// Primary marker: .gitattributes must contain the lock yml entry
	gitAttrPath := filepath.Join(gitRoot, ".gitattributes")
	data, err := os.ReadFile(gitAttrPath)
	if err != nil || !strings.Contains(string(data), constants.WorkflowsLockYmlGitAttributesEntry) {
		initRepoSetupLog.Printf("Init marker missing: .gitattributes entry not found in %s", gitRoot)
		return initStateIncomplete
	}

	// Secondary marker: dispatcher skill or VSCode settings (either suffices)
	skillPath := filepath.Join(gitRoot, ".github", "skills", "agentic-workflows", "SKILL.md")
	vscodePath := filepath.Join(gitRoot, ".vscode", "settings.json")

	_, skillErr := os.Stat(skillPath)
	_, vscodeErr := os.Stat(vscodePath)
	if os.IsNotExist(skillErr) && os.IsNotExist(vscodeErr) {
		initRepoSetupLog.Printf("Init marker missing: neither skill nor vscode settings found in %s", gitRoot)
		return initStateIncomplete
	}

	initRepoSetupLog.Printf("Init markers present in %s", gitRoot)
	return initStateComplete
}

// cloneRepo runs git clone for repoSlug into targetDir.
func cloneRepo(repoSlug, targetDir string, verbose bool) error {
	initRepoSetupLog.Printf("Cloning %s into %s", repoSlug, targetDir)

	// Build the HTTPS clone URL using the configured GitHub host
	host := getGitHubHost()
	cloneURL := fmt.Sprintf("%s/%s.git", strings.TrimSuffix(host, "/"), repoSlug)

	cmd := exec.Command("git", "clone", cloneURL, targetDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(strings.TrimSpace(string(out))))
	}
	return nil
}

// runInitInDir changes into dir and calls InitRepository with default options.
func runInitInDir(dir string, verbose bool) error {
	initRepoSetupLog.Printf("Running gh aw init in %s", dir)

	// Save and restore working directory
	prev, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() { _ = os.Chdir(prev) }()

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to enter directory %s: %w", dir, err)
	}

	return InitRepository(InitOptions{
		Verbose: verbose,
		Skill:   true,
		Agent:   true,
		MCP:     false,
	})
}

// printPlan renders the plan to stderr for human consumption.
func printPlan(plan *RepoSetupPlan) {
	fmt.Fprintln(os.Stderr, console.FormatSectionHeaderStderr("Planned actions"))
	fmt.Fprintln(os.Stderr, "")
	for i, a := range plan.Actions {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, a.Description)
	}
	fmt.Fprintln(os.Stderr, "")
	if !plan.HasMutations {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No mutations — this is a read-only plan."))
	}
}

// promptYesNo asks the user to confirm; returns true when the user answers yes.
func promptYesNo(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", console.FormatPromptMessage(question))
	var answer string
	if _, err := fmt.Fscan(os.Stdin, &answer); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(answer), "y") ||
		strings.EqualFold(strings.TrimSpace(answer), "yes")
}

// emitResult writes the result as JSON to stdout when jsonMode is true.
func emitResult(result *RepoSetupResult, jsonMode bool) {
	if !jsonMode {
		return
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		initRepoSetupLog.Printf("Failed to marshal result to JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}
