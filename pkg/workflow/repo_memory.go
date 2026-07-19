// This file provides repository memory configuration and generation.
//
// This file handles:
//   - Repo-memory configuration structures and defaults
//   - Repo-memory tool configuration extraction and parsing
//   - Generation of per-memory GitHub token secrets
//
// See repo_memory_validation.go for validation functions.

package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var repoMemoryLog = logger.New("workflow:repo_memory")

const (
	// defaultRepoMemoryMaxFileSize is the default maximum file size in bytes (100KB).
	defaultRepoMemoryMaxFileSize = 102400
	// defaultRepoMemoryMaxPatchSize is the default maximum total patch size in bytes (10KB).
	defaultRepoMemoryMaxPatchSize = 10240
	// maxRepoMemoryPatchSize is the maximum allowed value for max-patch-size (1MB).
	maxRepoMemoryPatchSize = 1048576
)

// Pre-compiled regexes for performance (avoid recompilation in hot paths)
var (
	// branchPrefixValidPattern matches valid branch prefix characters (alphanumeric, hyphens, underscores)
	branchPrefixValidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// RepoMemoryConfig holds configuration for repo-memory functionality
type RepoMemoryConfig struct {
	BranchPrefix string            `yaml:"branch-prefix,omitempty"` // branch prefix (default: "memory")
	Memories     []RepoMemoryEntry `yaml:"memories,omitempty"`      // repo-memory configurations
}

// RepoMemoryEntry represents a single repo-memory configuration
type RepoMemoryEntry struct {
	ID                string   `yaml:"id"`                           // memory identifier (required for array notation)
	TargetRepo        string   `yaml:"target-repo,omitempty"`        // target repository (default: current repo)
	BranchName        string   `yaml:"branch-name,omitempty"`        // branch name (default: memory/{memory-id})
	FileGlob          []string `yaml:"file-glob,omitempty"`          // file glob patterns for allowed files
	MaxFileSize       int      `yaml:"max-file-size,omitempty"`      // maximum size per file in bytes (default: 100KB)
	MaxFileCount      int      `yaml:"max-file-count,omitempty"`     // maximum file count per commit (default: 100)
	MaxPatchSize      int      `yaml:"max-patch-size,omitempty"`     // maximum total patch size in bytes (default: 10KB, max: 1MB)
	Description       string   `yaml:"description,omitempty"`        // optional description for this memory
	CreateOrphan      bool     `yaml:"create-orphan,omitempty"`      // create orphaned branch if missing (default: true)
	AllowedExtensions []string `yaml:"allowed-extensions,omitempty"` // allowed file extensions (default: [".json", ".jsonl", ".txt", ".md", ".csv"])
	Wiki              bool     `yaml:"wiki,omitempty"`               // use the GitHub Wiki git repository instead of the regular repo
	FormatJSON        bool     `yaml:"format-json,omitempty"`        // pretty-print all .json files before committing (default: false)
}

// RepoMemoryToolConfig represents the configuration for repo-memory in tools
type RepoMemoryToolConfig struct {
	// Can be boolean, object, or array - handled by this file
	Raw any `yaml:"-"`
}

// generateDefaultBranchName generates a default branch name for a given memory ID and prefix
func generateDefaultBranchName(memoryID string, branchPrefix string) string {
	if branchPrefix == "" {
		branchPrefix = "memory"
	}
	return fmt.Sprintf("%s/%s", branchPrefix, memoryID)
}

// extractRepoMemoryConfig extracts repo-memory configuration from tools section.
// workflowID is used to qualify the default branch name (e.g. "memory/{workflowID}").
func (c *Compiler) extractRepoMemoryConfig(toolsConfig *ToolsConfig, workflowID string) (*RepoMemoryConfig, error) {
	// Check if repo-memory tool is configured
	if toolsConfig == nil || toolsConfig.RepoMemory == nil {
		return nil, nil
	}

	repoMemoryLog.Print("Extracting repo-memory configuration from ToolsConfig")

	config := &RepoMemoryConfig{
		BranchPrefix: "memory", // Default branch prefix
	}
	repoMemoryValue := toolsConfig.RepoMemory.Raw

	// Handle nil value (simple enable with defaults) - same as true
	if repoMemoryValue == nil {
		repoMemoryLog.Print("Using default repo-memory configuration (nil value)")
		config.Memories = []RepoMemoryEntry{defaultRepoMemoryEntry(defaultMemoryBranchID(workflowID), config.BranchPrefix)}
		return config, nil
	}

	// Handle boolean value (simple enable/disable)
	if boolValue, ok := repoMemoryValue.(bool); ok {
		if boolValue {
			repoMemoryLog.Print("Using default repo-memory configuration (boolean true)")
			config.Memories = []RepoMemoryEntry{defaultRepoMemoryEntry(defaultMemoryBranchID(workflowID), config.BranchPrefix)}
		} else {
			repoMemoryLog.Print("Repo-memory disabled (boolean false)")
		}
		// If false, return empty config (empty array means disabled)
		return config, nil
	}

	// Handle array of memory configurations
	if memoryArray, ok := repoMemoryValue.([]any); ok {
		if err := parseRepoMemoryArray(memoryArray, config, workflowID); err != nil {
			return nil, err
		}
		return config, nil
	}

	// Handle object configuration (single memory, backward compatible)
	// Convert to array with single entry
	if configMap, ok := repoMemoryValue.(map[string]any); ok {
		repoMemoryLog.Print("Processing object-style repo-memory configuration (backward compatible)")
		entry, err := parseRepoMemoryObject(configMap, config, workflowID)
		if err != nil {
			return nil, err
		}
		config.Memories = []RepoMemoryEntry{entry}
		return config, nil
	}

	return nil, nil
}

func defaultMemoryBranchID(workflowID string) string {
	if workflowID != "" {
		return workflowID
	}
	return "default"
}

func defaultRepoMemoryEntry(branchID, branchPrefix string) RepoMemoryEntry {
	return RepoMemoryEntry{
		ID:                "default",
		BranchName:        generateDefaultBranchName(branchID, branchPrefix),
		MaxFileSize:       defaultRepoMemoryMaxFileSize,
		MaxFileCount:      100,
		MaxPatchSize:      defaultRepoMemoryMaxPatchSize,
		CreateOrphan:      true,
		AllowedExtensions: constants.DefaultAllowedMemoryExtensions,
	}
}

func parseRepoMemoryArray(memoryArray []any, config *RepoMemoryConfig, workflowID string) error {
	repoMemoryLog.Printf("Processing memory array with %d entries", len(memoryArray))
	config.Memories = make([]RepoMemoryEntry, 0, len(memoryArray))
	if err := parseRepoMemoryArrayBranchPrefix(memoryArray, config); err != nil {
		return err
	}
	for _, item := range memoryArray {
		memoryMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry, err := parseRepoMemoryEntry(memoryMap, config.BranchPrefix, workflowID, true, true)
		if err != nil {
			return err
		}
		config.Memories = append(config.Memories, entry)
	}
	return validateNoDuplicateMemoryIDs(config.Memories)
}

func parseRepoMemoryArrayBranchPrefix(memoryArray []any, config *RepoMemoryConfig) error {
	if len(memoryArray) == 0 {
		return nil
	}
	firstItem, ok := memoryArray[0].(map[string]any)
	if !ok {
		return nil
	}
	return parseRepoMemoryBranchPrefix(firstItem, config)
}

func parseRepoMemoryBranchPrefix(configMap map[string]any, config *RepoMemoryConfig) error {
	branchPrefix, exists := configMap["branch-prefix"]
	if !exists {
		return nil
	}
	prefixStr, ok := branchPrefix.(string)
	if !ok {
		return nil
	}
	if err := validateBranchPrefix(prefixStr); err != nil {
		return err
	}
	config.BranchPrefix = prefixStr
	repoMemoryLog.Printf("Using custom branch-prefix: %s", prefixStr)
	return nil
}

func parseRepoMemoryObject(configMap map[string]any, config *RepoMemoryConfig, workflowID string) (RepoMemoryEntry, error) {
	if err := parseRepoMemoryBranchPrefix(configMap, config); err != nil {
		return RepoMemoryEntry{}, err
	}
	return parseRepoMemoryEntry(configMap, config.BranchPrefix, workflowID, false, false)
}

func parseRepoMemoryEntry(memoryMap map[string]any, branchPrefix, workflowID string, qualifyDefaultBranch, parseID bool) (RepoMemoryEntry, error) {
	entry := RepoMemoryEntry{
		ID:           "default",
		MaxFileSize:  defaultRepoMemoryMaxFileSize,
		MaxFileCount: 100,
		MaxPatchSize: defaultRepoMemoryMaxPatchSize,
		CreateOrphan: true,
	}
	explicitID, explicitBranchName := parseRepoMemoryIdentity(memoryMap, &entry, parseID)
	if entry.BranchName == "" {
		branchID := defaultMemoryBranchID(workflowID)
		if qualifyDefaultBranch && explicitID {
			branchID = entry.ID
		}
		entry.BranchName = generateDefaultBranchName(branchID, branchPrefix)
	}
	if err := parseRepoMemoryEntryFields(memoryMap, &entry); err != nil {
		return RepoMemoryEntry{}, err
	}
	applyRepoMemoryWikiDefaults(&entry, explicitBranchName)
	return entry, nil
}

func parseRepoMemoryIdentity(memoryMap map[string]any, entry *RepoMemoryEntry, parseID bool) (bool, bool) {
	explicitID := false
	if id, exists := memoryMap["id"]; parseID && exists {
		if idStr, ok := id.(string); ok {
			entry.ID = idStr
			explicitID = true
		}
	}
	if entry.ID == "" {
		entry.ID = "default"
	}
	explicitBranchName := false
	if branchName, exists := memoryMap["branch-name"]; exists {
		if branchStr, ok := branchName.(string); ok {
			entry.BranchName = branchStr
			explicitBranchName = true
		}
	}
	return explicitID, explicitBranchName
}

func parseRepoMemoryEntryFields(memoryMap map[string]any, entry *RepoMemoryEntry) error {
	parseRepoMemoryStringsAndBools(memoryMap, entry)
	if err := parseRepoMemoryFileGlob(memoryMap, entry); err != nil {
		return err
	}
	if err := parseRepoMemoryLimits(memoryMap, entry); err != nil {
		return err
	}
	parseRepoMemoryAllowedExtensions(memoryMap, entry)
	if len(entry.AllowedExtensions) == 0 {
		entry.AllowedExtensions = constants.DefaultAllowedMemoryExtensions
	}
	return nil
}

func parseRepoMemoryStringsAndBools(memoryMap map[string]any, entry *RepoMemoryEntry) {
	if targetRepo, exists := memoryMap["target-repo"]; exists {
		if repoStr, ok := targetRepo.(string); ok {
			entry.TargetRepo = repoStr
		}
	}
	if description, exists := memoryMap["description"]; exists {
		if descStr, ok := description.(string); ok {
			entry.Description = descStr
		}
	}
	if createOrphan, exists := memoryMap["create-orphan"]; exists {
		if orphanBool, ok := createOrphan.(bool); ok {
			entry.CreateOrphan = orphanBool
		}
	}
	if wiki, exists := memoryMap["wiki"]; exists {
		if wikiBool, ok := wiki.(bool); ok {
			entry.Wiki = wikiBool
		}
	}
	if formatJSON, exists := memoryMap["format-json"]; exists {
		if formatJSONBool, ok := formatJSON.(bool); ok {
			entry.FormatJSON = formatJSONBool
		}
	}
}

func parseRepoMemoryFileGlob(memoryMap map[string]any, entry *RepoMemoryEntry) error {
	fileGlob, exists := memoryMap["file-glob"]
	if !exists {
		return nil
	}
	if globArray, ok := fileGlob.([]any); ok {
		entry.FileGlob = make([]string, 0, len(globArray))
		for _, item := range globArray {
			if str, ok := item.(string); ok {
				entry.FileGlob = append(entry.FileGlob, str)
			}
		}
	} else if globStr, ok := fileGlob.(string); ok {
		entry.FileGlob = []string{globStr}
	}
	return validateFileGlobPatterns(entry.FileGlob)
}

func parseRepoMemoryLimits(memoryMap map[string]any, entry *RepoMemoryEntry) error {
	var err error
	if entry.MaxFileSize, err = parseRepoMemoryIntField(memoryMap, "max-file-size", entry.MaxFileSize); err != nil {
		return err
	}
	if err := validateIntRange(entry.MaxFileSize, 1, 104857600, "max-file-size"); err != nil {
		return err
	}
	if entry.MaxFileCount, err = parseRepoMemoryIntField(memoryMap, "max-file-count", entry.MaxFileCount); err != nil {
		return err
	}
	if err := validateIntRange(entry.MaxFileCount, 1, 1000, "max-file-count"); err != nil {
		return err
	}
	if entry.MaxPatchSize, err = parseRepoMemoryIntField(memoryMap, "max-patch-size", entry.MaxPatchSize); err != nil {
		return err
	}
	return validateIntRange(entry.MaxPatchSize, 1, maxRepoMemoryPatchSize, "max-patch-size")
}

func parseRepoMemoryIntField(memoryMap map[string]any, key string, current int) (int, error) {
	value, exists := memoryMap[key]
	if !exists {
		return current, nil
	}
	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case uint64:
		return int(v), nil
	default:
		return current, nil
	}
}

func parseRepoMemoryAllowedExtensions(memoryMap map[string]any, entry *RepoMemoryEntry) {
	allowedExts, exists := memoryMap["allowed-extensions"]
	if !exists {
		return
	}
	extArray, ok := allowedExts.([]any)
	if !ok {
		return
	}
	entry.AllowedExtensions = make([]string, 0, len(extArray))
	for _, ext := range extArray {
		if extStr, ok := ext.(string); ok {
			entry.AllowedExtensions = append(entry.AllowedExtensions, extStr)
		}
	}
}

func applyRepoMemoryWikiDefaults(entry *RepoMemoryEntry, explicitBranchName bool) {
	if !entry.Wiki {
		return
	}
	if !explicitBranchName {
		entry.BranchName = "master"
	}
	entry.CreateOrphan = false
}

// generateRepoMemoryArtifactUpload generates steps to upload repo-memory directories as artifacts.
// This runs at the end of the agent job (always condition) to save the state.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func generateRepoMemoryArtifactUpload(builder *strings.Builder, data *WorkflowData, pinAction func(string) string) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return
	}

	repoMemoryLog.Printf("Generating repo-memory artifact upload steps for %d memories", len(data.RepoMemoryConfig.Memories))

	// In workflow_call context, apply the per-invocation prefix to avoid artifact name clashes.
	prefix := artifactPrefixExprForDownstreamJob(data)

	builder.WriteString("      # Upload repo memory as artifacts for push job\n")

	for _, memory := range data.RepoMemoryConfig.Memories {
		// Determine the memory directory
		memoryDir := constants.TmpRepoMemoryDir + memory.ID

		// Sanitize memory ID for artifact naming (remove hyphens, lowercase)
		sanitizedID := SanitizeWorkflowIDForCacheKey(memory.ID)

		// Determine the label for step names
		memoryLabel := "repo-memory"
		if memory.Wiki {
			memoryLabel = "wiki-memory"
		}

		// Step: Sanitize filenames before upload to prevent artifact upload failures.
		// GitHub Actions artifacts are stored on NTFS-compatible filesystems, so filenames
		// must not contain: ? : * | < > " (among other characters).
		// The agent may create files with these characters (e.g. "Can-we-have-a-PR?.md"),
		// which causes the upload-artifact action to fail with a hard error.
		// The script uses git commands (git mv for tracked files, mv for untracked) since
		// repo-memory is backed by a git working tree.
		fmt.Fprintf(builder, "      - name: Sanitize %s filenames (%s)\n", memoryLabel, memory.ID)
		builder.WriteString("        if: always()\n")
		builder.WriteString("        continue-on-error: true\n")
		builder.WriteString("        env:\n")
		fmt.Fprintf(builder, "          MEMORY_DIR: %s\n", memoryDir)
		builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/sanitize_repo_memory_filenames.sh\"\n")

		// Step: Upload repo-memory directory as artifact
		fmt.Fprintf(builder, "      - name: Upload %s artifact (%s)\n", memoryLabel, memory.ID)
		builder.WriteString("        if: always()\n")
		fmt.Fprintf(builder, "        uses: %s\n", pinAction("actions/upload-artifact"))
		builder.WriteString("        with:\n")
		fmt.Fprintf(builder, "          name: %srepo-memory-%s\n", prefix, sanitizedID)
		fmt.Fprintf(builder, "          path: %s\n", memoryDir)
		builder.WriteString("          retention-days: 1\n")
		builder.WriteString("          if-no-files-found: ignore\n")
	}
}

// generateRepoMemorySteps generates git steps for the repo-memory configuration
func generateRepoMemorySteps(builder *strings.Builder, data *WorkflowData) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return
	}

	repoMemoryLog.Printf("Generating repo-memory steps for %d memories", len(data.RepoMemoryConfig.Memories))

	builder.WriteString("      # Repo memory git-based storage configuration from frontmatter processed below\n")

	for _, memory := range data.RepoMemoryConfig.Memories {
		// Determine the target repository
		targetRepo := memory.TargetRepo
		if targetRepo == "" {
			targetRepo = "${{ github.repository }}"
		}
		// For wiki mode, append .wiki to the repo path so the clone script uses the wiki git URL
		if memory.Wiki {
			targetRepo = targetRepo + ".wiki"
		}

		// Determine the memory directory
		memoryDir := constants.TmpRepoMemoryDir + memory.ID

		// Step 1: Clone the repo-memory branch
		if memory.Wiki {
			fmt.Fprintf(builder, "      - name: Clone wiki-memory branch (%s)\n", memory.ID)
		} else {
			fmt.Fprintf(builder, "      - name: Clone repo-memory branch (%s)\n", memory.ID)
		}
		builder.WriteString("        env:\n")
		builder.WriteString("          GH_TOKEN: ${{ github.token }}\n")
		builder.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
		fmt.Fprintf(builder, "          BRANCH_NAME: %s\n", memory.BranchName)
		fmt.Fprintf(builder, "          TARGET_REPO: %s\n", targetRepo)
		fmt.Fprintf(builder, "          MEMORY_DIR: %s\n", memoryDir)
		fmt.Fprintf(builder, "          CREATE_ORPHAN: %t\n", memory.CreateOrphan)
		builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/clone_repo_memory_branch.sh\"\n")
	}
}

// buildPushRepoMemoryConcurrencyGroup builds a concurrency group key that is scoped to the
// specific (target-repo, branch) pairs being written by this push job.  Using the actual
// write targets—rather than a single repo-wide key—ensures that workflows pushing to
// different memory branches do not unnecessarily serialise or cancel each other.
//
// Key format: "push-repo-memory-${{ github.repository }}|<key1>[|<key2>…]"
//
// Each key component is percent-encoded (only `%` and `|` are encoded) before joining
// with "|", so the separator is always unambiguous even if a user-supplied branch name
// or target-repo contains a literal "|".  For memories that target a non-default
// repository, the target repo is prepended to the branch name
// (e.g., "other-owner%2Fother-repo:memory%2Fbranch" would be encoded if needed) so that
// distinct targets produce distinct concurrency groups.  The branches are sorted for a
// deterministic key regardless of the order memories are declared in the frontmatter.
func buildPushRepoMemoryConcurrencyGroup(memories []RepoMemoryEntry) string {
	branchKeys := make([]string, 0, len(memories))
	for _, m := range memories {
		key := encodeConcurrencyKeyPart(m.BranchName)
		if m.TargetRepo != "" {
			key = encodeConcurrencyKeyPart(m.TargetRepo) + ":" + key
		}
		branchKeys = append(branchKeys, key)
	}
	sort.Strings(branchKeys)
	return "push-repo-memory-${{ github.repository }}|" + strings.Join(branchKeys, "|")
}

// encodeConcurrencyKeyPart percent-encodes the characters that would otherwise make the
// concurrency group key ambiguous: "%" (to avoid double-encoding) and "|" (the separator).
// All other characters are left as-is so the key remains human-readable in workflow UIs.
func encodeConcurrencyKeyPart(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "|", "%7C")
	return s
}

// buildPushRepoMemoryJob creates a job that downloads repo-memory artifacts and pushes them to git branches
// This job runs after the agent job completes (even if it fails) and requires contents: write permission
// If threat detection is enabled, only runs if no threats were detected
func (c *Compiler) buildPushRepoMemoryJob(data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return nil, nil
	}

	repoMemoryLog.Printf("Building push_repo_memory job for %d memories (threatDetectionEnabled=%v)", len(data.RepoMemoryConfig.Memories), threatDetectionEnabled)

	var steps []string
	setupSteps, setupActionRef := c.buildPushRepoMemorySetupSteps(data)
	steps = append(steps, setupSteps...)
	steps = append(steps, buildPushRepoMemoryCheckoutStep())
	steps = append(steps, c.generateGitConfigurationSteps()...)
	steps = append(steps, buildPushRepoMemoryDownloadSteps(data)...)

	useRequire := setupActionRef != ""
	steps = append(steps, buildPushRepoMemoryPushSteps(data, useRequire)...)

	// In dev mode the setup action is referenced via a local path (./actions/setup), so its files
	// live in the workspace. The push_repo_memory.cjs script internally checks out the memory
	// branch, which replaces the workspace content and removes the actions/setup directory.
	// Without restoring it, the runner's post-step for Setup Scripts would fail with
	// "Can't find 'action.yml', 'action.yaml' or 'Dockerfile' under .../actions/setup".
	// We add a restore checkout step (if: always()) after all push steps so the post-step
	// can always find action.yml and complete its /tmp/gh-aw cleanup.
	// Note: no ref is specified in dev mode — use the repository default branch (same pattern
	// as generateCheckoutActionsFolder in dev mode).
	if c.actionMode.IsDev() {
		steps = append(steps, c.generateRestoreActionsSetupStep())
	}

	jobCondition, jobNeeds := pushRepoMemoryConditionAndNeeds(threatDetectionEnabled)
	outputs := buildPushRepoMemoryOutputs(data.RepoMemoryConfig.Memories)
	concurrency := c.indentYAMLLines(fmt.Sprintf("concurrency:\n  group: %q\n  cancel-in-progress: false", buildPushRepoMemoryConcurrencyGroup(data.RepoMemoryConfig.Memories)), "    ")

	job := &Job{
		Name:        pushRepoMemoryJobName,
		DisplayName: "", // No display name - job ID is sufficient
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          jobCondition,
		Permissions: "permissions:\n      contents: write",
		Concurrency: concurrency,
		Needs:       jobNeeds,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

func (c *Compiler) buildPushRepoMemorySetupSteps(data *WorkflowData) ([]string, string) {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil, setupActionRef
	}
	var steps []string
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	repoMemoryTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	repoMemoryParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, repoMemoryTraceID, repoMemoryParentSpanID)...)
	return steps, setupActionRef
}

func buildPushRepoMemoryCheckoutStep() string {
	var checkoutStep strings.Builder
	checkoutStep.WriteString("      - name: Checkout repository\n")
	fmt.Fprintf(&checkoutStep, "        uses: %s\n", getActionPin("actions/checkout"))
	checkoutStep.WriteString("        with:\n")
	checkoutStep.WriteString("          persist-credentials: false\n")
	checkoutStep.WriteString("          sparse-checkout: .\n")
	return checkoutStep.String()
}

func buildPushRepoMemoryDownloadSteps(data *WorkflowData) []string {
	repoMemoryPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	var steps []string
	for _, memory := range data.RepoMemoryConfig.Memories {
		sanitizedID := SanitizeWorkflowIDForCacheKey(memory.ID)
		var step strings.Builder
		if memory.Wiki {
			fmt.Fprintf(&step, "      - name: Download wiki-memory artifact (%s)\n", memory.ID)
		} else {
			fmt.Fprintf(&step, "      - name: Download repo-memory artifact (%s)\n", memory.ID)
		}
		fmt.Fprintf(&step, "        uses: %s\n", getActionPin("actions/download-artifact"))
		step.WriteString("        continue-on-error: true\n")
		step.WriteString("        with:\n")
		fmt.Fprintf(&step, "          name: %srepo-memory-%s\n", repoMemoryPrefix, sanitizedID)
		fmt.Fprintf(&step, "          path: /tmp/gh-aw/repo-memory/%s\n", memory.ID)
		steps = append(steps, step.String())
	}
	return steps
}

func buildPushRepoMemoryPushSteps(data *WorkflowData, useRequire bool) []string {
	var steps []string
	for _, memory := range data.RepoMemoryConfig.Memories {
		steps = append(steps, buildPushRepoMemoryPushStep(data, memory, useRequire))
	}
	return steps
}

func buildPushRepoMemoryPushStep(data *WorkflowData, memory RepoMemoryEntry, useRequire bool) string {
	targetRepo := repoMemoryTargetRepo(memory)
	artifactDir := constants.TmpRepoMemoryDir + memory.ID
	fileGlobFilter := strings.Join(memory.FileGlob, " ")
	var step strings.Builder
	if memory.Wiki {
		fmt.Fprintf(&step, "      - name: Push wiki-memory changes (%s)\n", memory.ID)
	} else {
		fmt.Fprintf(&step, "      - name: Push repo-memory changes (%s)\n", memory.ID)
	}
	fmt.Fprintf(&step, "        id: push_repo_memory_%s\n", memory.ID)
	step.WriteString("        if: always()\n")
	fmt.Fprintf(&step, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	writePushRepoMemoryEnv(&step, memory, targetRepo, artifactDir, fileGlobFilter)
	writePushRepoMemoryScript(&step, useRequire)
	return step.String()
}

func repoMemoryTargetRepo(memory RepoMemoryEntry) string {
	targetRepo := memory.TargetRepo
	if targetRepo == "" {
		targetRepo = "${{ github.repository }}"
	}
	if memory.Wiki {
		targetRepo += ".wiki"
	}
	return targetRepo
}

func writePushRepoMemoryEnv(step *strings.Builder, memory RepoMemoryEntry, targetRepo, artifactDir, fileGlobFilter string) {
	step.WriteString("        env:\n")
	step.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	step.WriteString("          GITHUB_RUN_ID: ${{ github.run_id }}\n")
	step.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	fmt.Fprintf(step, "          ARTIFACT_DIR: %s\n", artifactDir)
	fmt.Fprintf(step, "          MEMORY_ID: %s\n", memory.ID)
	fmt.Fprintf(step, "          TARGET_REPO: %s\n", targetRepo)
	fmt.Fprintf(step, "          BRANCH_NAME: %s\n", memory.BranchName)
	if memory.Wiki {
		fmt.Fprintf(step, "          REPO_MEMORY_ALLOWED_REPOS: %s\n", targetRepo)
	}
	fmt.Fprintf(step, "          MAX_FILE_SIZE: %d\n", memory.MaxFileSize)
	fmt.Fprintf(step, "          MAX_FILE_COUNT: %d\n", memory.MaxFileCount)
	fmt.Fprintf(step, "          MAX_PATCH_SIZE: %d\n", memory.MaxPatchSize)
	allowedExtsJSON, _ := json.Marshal(memory.AllowedExtensions) //nolint:jsonmarshalignoredeerror // marshaling a string slice cannot fail
	fmt.Fprintf(step, "          ALLOWED_EXTENSIONS: '%s'\n", allowedExtsJSON)
	if fileGlobFilter != "" {
		fmt.Fprintf(step, "          FILE_GLOB_FILTER: \"%s\"\n", fileGlobFilter)
	}
	if memory.FormatJSON {
		step.WriteString("          FORMAT_JSON: 'true'\n")
	}
}

func writePushRepoMemoryScript(step *strings.Builder, useRequire bool) {
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	step.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	step.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	if useRequire {
		step.WriteString("            const { main } = require('" + SetupActionDestination + "/push_repo_memory.cjs');\n")
		step.WriteString("            await main();\n")
		return
	}
	formattedScript := FormatJavaScriptForYAML("const { main } = require('${{ runner.temp }}/gh-aw/actions/push_repo_memory.cjs'); await main();")
	for _, line := range formattedScript {
		step.WriteString(line)
	}
}

func pushRepoMemoryConditionAndNeeds(threatDetectionEnabled bool) (string, []string) {
	agentSucceeded := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("success"),
	)
	notCancelled := &NotNode{Child: BuildFunctionCall("cancelled")}
	jobNeeds := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	if threatDetectionEnabled {
		condition := RenderCondition(BuildAnd(BuildAnd(BuildAnd(BuildFunctionCall("always"), notCancelled), buildDetectionPassedCondition()), agentSucceeded))
		return condition, append(jobNeeds, string(constants.DetectionJobName))
	}
	return RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), notCancelled), agentSucceeded)), jobNeeds
}

func buildPushRepoMemoryOutputs(memories []RepoMemoryEntry) map[string]string {
	outputs := make(map[string]string)
	for _, memory := range memories {
		stepID := "push_repo_memory_" + memory.ID
		outputs["validation_failed_"+memory.ID] = fmt.Sprintf("${{ steps.%s.outputs.validation_failed }}", stepID)
		outputs["validation_error_"+memory.ID] = fmt.Sprintf("${{ steps.%s.outputs.validation_error }}", stepID)
		outputs["patch_size_exceeded_"+memory.ID] = fmt.Sprintf("${{ steps.%s.outputs.patch_size_exceeded }}", stepID)
	}
	return outputs
}
