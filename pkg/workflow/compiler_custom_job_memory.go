package workflow

import (
	"fmt"
	"strings"
)

// restoreMemoryConfig holds the parsed restore-memory configuration for a custom job.
// When a memory type is true, the compiler injects read-only restore steps for that store.
// No write-back or commit steps are ever emitted for restore-memory.
type restoreMemoryConfig struct {
	CacheMemory   bool
	RepoMemory    bool
	CommentMemory bool
}

// extractRestoreMemoryConfig parses the restore-memory field from a custom job config map.
// Returns nil if the field is absent or all values are false.
func extractRestoreMemoryConfig(configMap map[string]any, jobName string) (*restoreMemoryConfig, error) {
	rawVal, hasField := configMap["restore-memory"]
	if !hasField {
		return nil, nil
	}

	rmMap, ok := rawVal.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("jobs.%s.restore-memory must be an object with memory-type keys (cache-memory, repo-memory, comment-memory)", jobName)
	}

	cfg := &restoreMemoryConfig{}

	if v, ok := rmMap["cache-memory"]; ok {
		if b, ok := v.(bool); ok {
			cfg.CacheMemory = b
		}
	}
	if v, ok := rmMap["repo-memory"]; ok {
		if b, ok := v.(bool); ok {
			cfg.RepoMemory = b
		}
	}
	if v, ok := rmMap["comment-memory"]; ok {
		if b, ok := v.(bool); ok {
			cfg.CommentMemory = b
		}
	}

	if !cfg.CacheMemory && !cfg.RepoMemory && !cfg.CommentMemory {
		return nil, nil
	}
	return cfg, nil
}

// validateRestoreMemoryConfig ensures that every requested memory type is actually
// configured in the workflow's tools: section. Custom jobs cannot declare memory
// stores independently; they can only restore from stores the workflow already uses.
func validateRestoreMemoryConfig(cfg *restoreMemoryConfig, data *WorkflowData, jobName string) error {
	if cfg == nil {
		return nil
	}
	if cfg.CacheMemory && (data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0) {
		return fmt.Errorf("jobs.%s.restore-memory.cache-memory: cache-memory must be configured in tools: to use restore-memory", jobName)
	}
	if cfg.RepoMemory && (data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0) {
		return fmt.Errorf("jobs.%s.restore-memory.repo-memory: repo-memory must be configured in tools: to use restore-memory", jobName)
	}
	if cfg.CommentMemory && (data.SafeOutputs == nil || data.SafeOutputs.CommentMemory == nil) {
		return fmt.Errorf("jobs.%s.restore-memory.comment-memory: comment-memory must be configured in tools: to use restore-memory", jobName)
	}
	return nil
}

// buildRestoreMemorySteps returns two slices:
//   - setupLines: gh-aw checkout + setup steps (only when repo-memory or comment-memory
//     is requested, since those need scripts from the setup action)
//   - memoryLines: the actual memory restore/clone/prepare steps
//
// No write-back steps are ever emitted; all injected steps are read-only.
func (c *Compiler) buildRestoreMemorySteps(cfg *restoreMemoryConfig, data *WorkflowData) (setupLines []string, memoryLines []string) {
	if cfg == nil {
		return nil, nil
	}

	// repo-memory and comment-memory rely on scripts installed by the gh-aw setup action.
	// Inject the setup step before any memory steps so those scripts are available.
	if cfg.RepoMemory || cfg.CommentMemory {
		setupActionRef := c.resolveActionReference("./actions/setup", data)
		if setupActionRef != "" || c.actionMode.IsScript() {
			setupLines = append(setupLines, c.generateCheckoutActionsFolder(data)...)
			// Pass empty trace IDs — custom jobs do not inherit the activation span.
			setupLines = append(setupLines, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, "", "")...)
		}
	}

	if cfg.CacheMemory {
		memoryLines = append(memoryLines, generateCacheMemoryRestoreLines(data)...)
	}
	if cfg.RepoMemory {
		memoryLines = append(memoryLines, generateRepoMemoryRestoreLines(data)...)
	}
	if cfg.CommentMemory {
		memoryLines = append(memoryLines, generateCommentMemoryRestoreLines(data)...)
	}

	return setupLines, memoryLines
}

// generateCacheMemoryRestoreLines produces read-only cache-memory restore steps for a
// custom job. Unlike the agent-job path, these steps:
//   - always use actions/cache/restore (never actions/cache, so no auto-save)
//   - create the cache directory inline with mkdir -p (no script needed)
//   - omit the git integrity setup step (only relevant for the write path)
func generateCacheMemoryRestoreLines(data *WorkflowData) []string {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, "      # restore-memory: cache-memory (read-only restore)\n")

	var githubConfig *GitHubToolConfig
	if data.ParsedTools != nil {
		githubConfig = data.ParsedTools.GitHub
	}

	for i, cache := range data.CacheMemoryConfig.Caches {
		cacheDir := cacheMemoryDirFor(cache.ID)
		restoreStepID := fmt.Sprintf("restore_cache_memory_%d", i)

		// Create the cache directory inline; no script required.
		lines = append(lines, fmt.Sprintf("      - name: Create cache-memory directory (%s)\n", cache.ID))
		lines = append(lines, "        run: |\n")
		lines = append(lines, fmt.Sprintf("          mkdir -p %s\n", cacheDir))

		// Compute the same integrity-aware cache key as the agent job uses.
		cacheKey := computeIntegrityCacheKey(cache, githubConfig)

		// Restore step — always read-only in custom jobs.
		lines = append(lines, fmt.Sprintf("      - name: Restore cache-memory (%s)\n", cache.ID))
		lines = append(lines, fmt.Sprintf("        id: %s\n", restoreStepID))
		lines = append(lines, fmt.Sprintf("        uses: %s\n", getActionPin("actions/cache/restore")))
		lines = append(lines, "        with:\n")
		lines = append(lines, fmt.Sprintf("          key: %s\n", cacheKey))
		lines = append(lines, fmt.Sprintf("          path: %s\n", cacheDir))

		// Build restore keys using the same logic as generateCacheMemorySteps.
		var restoreKeys []string
		runIDSuffix := "-${{ github.run_id }}"
		if strings.HasSuffix(cacheKey, runIDSuffix) {
			restoreKeys = append(restoreKeys, strings.TrimSuffix(cacheKey, "${{ github.run_id }}"))
		} else {
			keyParts := strings.Split(cacheKey, "-")
			if len(keyParts) >= 2 {
				restoreKeys = append(restoreKeys, strings.Join(keyParts[:len(keyParts)-1], "-")+"-")
			}
		}

		scope := cache.Scope
		if scope == "" {
			scope = "workflow"
		}
		if scope == "repo" {
			repoKey := strings.TrimSuffix(cacheKey, "${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}")
			if repoKey != cacheKey && repoKey != "" {
				restoreKeys = append(restoreKeys, repoKey)
			}
		}

		lines = append(lines, "          restore-keys: |\n")
		for _, key := range restoreKeys {
			lines = append(lines, fmt.Sprintf("            %s\n", key))
		}
	}

	return lines
}

// generateRepoMemoryRestoreLines produces read-only repo-memory clone steps for a
// custom job by reusing the existing generateRepoMemorySteps builder and converting
// its output to the []string line format expected by job.Steps.
func generateRepoMemoryRestoreLines(data *WorkflowData) []string {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return nil
	}

	var b strings.Builder
	generateRepoMemorySteps(&b, data)
	raw := b.String()
	if raw == "" {
		return nil
	}

	// Split into individual lines, preserving their trailing newlines.
	parts := strings.Split(raw, "\n")
	lines := make([]string, 0, len(parts))
	for _, p := range parts {
		lines = append(lines, p+"\n")
	}
	return lines
}

// generateCommentMemoryRestoreLines produces a read-only comment-memory prepare step
// for a custom job. The step fetches the comment-memory content from GitHub and
// materialises it as local files — the same operation performed in the agent job.
func generateCommentMemoryRestoreLines(data *WorkflowData) []string {
	if data.SafeOutputs == nil || data.SafeOutputs.CommentMemory == nil {
		return nil
	}

	var lines []string
	lines = append(lines, "      # restore-memory: comment-memory (read-only restore)\n")
	lines = append(lines, "      - name: Prepare comment memory files\n")
	lines = append(lines, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
	lines = append(lines, "        with:\n")
	lines = append(lines, fmt.Sprintf("          github-token: %s\n", getEffectiveSafeOutputGitHubToken(data.SafeOutputs.CommentMemory.GitHubToken)))
	lines = append(lines, "          script: |\n")
	lines = append(lines, "            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	lines = append(lines, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	lines = append(lines, "            const { main } = require('${{ runner.temp }}/gh-aw/actions/setup_comment_memory_files.cjs');\n")
	lines = append(lines, "            await main();\n")
	return lines
}
