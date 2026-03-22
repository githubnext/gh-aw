// Package workflow provides qmd documentation search tool integration.
//
// # QMD Tool
//
// This file handles the qmd (https://github.com/tobi/qmd) builtin tool integration.
// qmd provides local vector search over documentation files using the @tobilu/qmd npm package.
//
// The integration has two phases:
//
//  1. Activation job: builds the search index from configured checkouts and/or GitHub searches
//     and uploads it as the "qmd-index" artifact. This step runs in the activation job which
//     already has contents:read permission, so the agent job does NOT need contents:read.
//     The index is built by a single actions/github-script step that runs qmd_index.cjs,
//     which uses the @tobilu/qmd JavaScript SDK to build the collections.
//
//  2. Agent job: downloads the "qmd-index" artifact and mounts the qmd MCP server pointing
//     at the pre-built index. The MCP server exposes a search tool that the agent can use
//     to find relevant documentation files.
//
// # Configuration
//
// Two sources can populate the index:
//
//   - checkouts: glob-based collections from checked-out repositories (each optionally with
//     its own checkout config to target a different repo)
//   - searches: GitHub search queries whose results are downloaded and added to the index
//
// Optionally, a cache-key can be set to persist the index in GitHub Actions cache:
//
//   - cache-key only (read-only mode): the index is restored from cache; no indexing steps run
//   - cache-key + sources: index is built if cache miss, then saved to cache for future runs
//
// Example frontmatter:
//
//	tools:
//	  qmd:
//	    checkouts:
//	      - name: docs
//	        paths:
//	          - docs/**/*.md
//	    searches:
//	      - query: "repo:owner/repo language:Markdown path:docs/"
//	        min: 1
//	        max: 30
//	        github-token: ${{ secrets.GITHUB_TOKEN }}
//	    cache-key: "qmd-index-${{ hashFiles('docs/**') }}"
//
// # Artifact lifecycle
//
// The index is built once per activation job run and shared with the agent job
// via the "qmd-index" artifact.  Retention is 1 day (same as the activation artifact).
//
// Related files:
//   - tools_types.go: QmdToolConfig, QmdDocCollection, QmdSearchEntry types
//   - tools_parser.go: parseQmdTool / parseQmdDocCollection / parseQmdSearchEntry
//   - mcp_renderer_builtin.go: RenderQmdMCP method
//   - compiler_activation_job.go: activation job qmd index steps
//   - compiler_yaml_main_job.go: agent job qmd artifact download
//   - actions/setup/js/qmd_index.cjs: JavaScript SDK implementation

package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var qmdLog = logger.New("workflow:qmd")

// hasQmdTool checks if the qmd tool is enabled in the tools configuration.
func hasQmdTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.Qmd != nil
}

// qmdHasSources reports whether the qmd config has any indexing sources
// (checkouts or searches).  When false and a cache-key is set,
// qmd operates in read-only mode: the index is restored from cache only.
func qmdHasSources(qmdConfig *QmdToolConfig) bool {
	return len(qmdConfig.Checkouts) > 0 || len(qmdConfig.Searches) > 0
}

// generateQmdModelsCacheStep generates a step that caches the qmd embedding models directory
// (~/.cache/qmd/models/).  It uses the combined actions/cache action (restore + post-save),
// keyed by OS so that the cached models are compatible with the runner architecture.
// This step should be emitted in the activation job (before index building) to populate
// the cache. For the agent job, use generateQmdModelsCacheRestoreStep instead.
func generateQmdModelsCacheStep() string {
	var sb strings.Builder
	sb.WriteString("      - name: Cache qmd models\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache"))
	sb.WriteString("        with:\n")
	sb.WriteString("          path: ~/.cache/qmd/models/\n")
	sb.WriteString("          key: qmd-models-${{ runner.os }}\n")
	return sb.String()
}

// generateQmdModelsCacheRestoreStep generates a read-only step that restores the qmd embedding
// models directory (~/.cache/qmd/models/) from GitHub Actions cache.  It uses
// actions/cache/restore (restore-only, no post-save) so the agent job never writes to the
// shared cache — that is the activation job's responsibility.
func generateQmdModelsCacheRestoreStep() string {
	var sb strings.Builder
	sb.WriteString("      - name: Restore qmd models cache\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/restore"))
	sb.WriteString("        with:\n")
	sb.WriteString("          path: ~/.cache/qmd/models/\n")
	sb.WriteString("          key: qmd-models-${{ runner.os }}\n")
	return sb.String()
}

// generateQmdCacheRestoreStep generates an activation-job step that restores the qmd index
// from GitHub Actions cache.  The step ID is "qmd-cache-restore" so that subsequent steps
// can check cache-hit via steps.qmd-cache-restore.outputs.cache-hit.
func generateQmdCacheRestoreStep(cacheKey string) string {
	var sb strings.Builder
	sb.WriteString("      - name: Restore qmd index from cache\n")
	sb.WriteString("        id: qmd-cache-restore\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/restore"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          key: %s\n", cacheKey)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	return sb.String()
}

// generateQmdCacheSaveStep generates an activation-job step that saves the qmd index to
// GitHub Actions cache.  It only runs when the preceding cache-restore step was a miss.
func generateQmdCacheSaveStep(cacheKey string) string {
	var sb strings.Builder
	sb.WriteString("      - name: Save qmd index to cache\n")
	sb.WriteString("        if: steps.qmd-cache-restore.outputs.cache-hit != 'true'\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/save"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          key: %s\n", cacheKey)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	return sb.String()
}

// qmdCheckoutEntry is the JSON representation of a checkout-based collection
// passed to qmd_index.cjs via the QMD_CONFIG_JSON environment variable.
type qmdCheckoutEntry struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Patterns []string `json:"patterns,omitempty"`
	Context  string   `json:"context,omitempty"`
}

// qmdSearchEntry is the JSON representation of a search entry passed to qmd_index.cjs.
type qmdSearchEntry struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`        // "code" (default) or "issues"
	Query       string `json:"query,omitempty"`       // for "code" type
	Repo        string `json:"repo,omitempty"`        // for "issues" type; blank = github.repository
	Min         int    `json:"min,omitempty"`         // minimum result count (0 = no minimum)
	Max         int    `json:"max,omitempty"`         // maximum result count (0 = use default)
	TokenEnvVar string `json:"tokenEnvVar,omitempty"` // env var holding custom GitHub token
}

// qmdBuildConfig is the top-level JSON config serialised into QMD_CONFIG_JSON
// and consumed by actions/setup/js/qmd_index.cjs.
type qmdBuildConfig struct {
	DBPath    string             `json:"dbPath"`
	Checkouts []qmdCheckoutEntry `json:"checkouts,omitempty"`
	Searches  []qmdSearchEntry   `json:"searches,omitempty"`
}

// resolveQmdWorkdir returns the working directory path for a checkout-based collection.
// Returns "${GITHUB_WORKSPACE}" for the default (current) repository, or the path
// specified / derived from the checkout config for external repositories.
func resolveQmdWorkdir(col *QmdDocCollection) string {
	if col.Checkout == nil {
		return "${GITHUB_WORKSPACE}"
	}
	if col.Checkout.Path != "" {
		checkoutPath := strings.TrimPrefix(col.Checkout.Path, "./")
		return "${GITHUB_WORKSPACE}/" + checkoutPath
	}
	name := col.Name
	if name == "" {
		name = "docs"
	}
	return "/tmp/gh-aw/qmd-checkout-" + name
}

// buildQmdConfig constructs the qmdBuildConfig from the user-provided QmdToolConfig.
func buildQmdConfig(qmdConfig *QmdToolConfig) qmdBuildConfig {
	cfg := qmdBuildConfig{
		DBPath: "/tmp/gh-aw/qmd-index",
	}

	for _, col := range qmdConfig.Checkouts {
		name := col.Name
		if name == "" {
			name = "docs"
		}
		entry := qmdCheckoutEntry{
			Name:    name,
			Path:    resolveQmdWorkdir(col),
			Context: col.Context,
		}
		if len(col.Paths) > 0 {
			entry.Patterns = col.Paths
		}
		cfg.Checkouts = append(cfg.Checkouts, entry)
	}

	for i, s := range qmdConfig.Searches {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("search-%d", i)
		}
		entry := qmdSearchEntry{
			Name:  name,
			Type:  s.Type,
			Query: s.Query,
			Min:   s.Min,
			Max:   s.Max,
		}
		if s.Type == "issues" && s.Query != "" {
			entry.Repo = s.Query
		}
		if s.GitHubToken != "" {
			entry.TokenEnvVar = fmt.Sprintf("QMD_SEARCH_TOKEN_%d", i)
		}
		cfg.Searches = append(cfg.Searches, entry)
	}

	return cfg
}

// generateQmdCollectionCheckoutStep generates a checkout step YAML string for a qmd
// collection that targets a non-default repository.  Returns an empty string when the
// collection uses the current repository (no checkout needed).
func generateQmdCollectionCheckoutStep(col *QmdDocCollection) string {
	if col.Checkout == nil {
		return ""
	}
	cfg := col.Checkout

	// Determine checkout path used in the runner filesystem
	checkoutPath := cfg.Path
	if checkoutPath == "" {
		checkoutPath = "/tmp/gh-aw/qmd-checkout-" + col.Name
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Checkout %s for qmd\n", col.Name)
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/checkout"))
	sb.WriteString("        with:\n")
	sb.WriteString("          persist-credentials: false\n")
	if cfg.Repository != "" {
		fmt.Fprintf(&sb, "          repository: %s\n", cfg.Repository)
	}
	if cfg.Ref != "" {
		fmt.Fprintf(&sb, "          ref: %s\n", cfg.Ref)
	}
	fmt.Fprintf(&sb, "          path: %s\n", checkoutPath)
	if cfg.GitHubToken != "" {
		fmt.Fprintf(&sb, "          token: %s\n", cfg.GitHubToken)
	}
	if cfg.FetchDepth != nil {
		fmt.Fprintf(&sb, "          fetch-depth: %d\n", *cfg.FetchDepth)
	}
	if cfg.SparseCheckout != "" {
		sb.WriteString("          sparse-checkout: |\n")
		for line := range strings.SplitSeq(strings.TrimRight(cfg.SparseCheckout, "\n"), "\n") {
			fmt.Fprintf(&sb, "            %s\n", strings.TrimSpace(line))
		}
	}
	if cfg.Submodules != "" {
		fmt.Fprintf(&sb, "          submodules: %s\n", cfg.Submodules)
	}
	if cfg.LFS {
		sb.WriteString("          lfs: true\n")
	}
	return sb.String()
}

// generateQmdIndexSteps generates the activation job steps that install the @tobilu/qmd SDK,
// run the qmd_index.cjs JavaScript script to build the vector search index, and upload it
// as the qmd-index artifact.
//
// The configuration is serialised to JSON and passed via the QMD_CONFIG_JSON environment
// variable to the github-script step. qmd_index.cjs uses the @tobilu/qmd SDK to:
//  1. Register checkout-based collections
//  2. Fetch GitHub search/issue results and register them as collections
//  3. Call store.update() and store.embed() to index and embed all documents
//
// When qmdConfig.CacheKey is set:
//   - A cache restore step is always emitted first.
//   - In read-only mode (no sources): only the cache restore + artifact upload are emitted;
//     Node.js, qmd SDK installation, and indexing steps are skipped entirely.
//   - In build mode (sources present): indexing steps are guarded by
//     `if: steps.qmd-cache-restore.outputs.cache-hit != 'true'`, so they are skipped on a
//     cache hit.  A cache save step follows the indexing steps.
func generateQmdIndexSteps(qmdConfig *QmdToolConfig, data *WorkflowData) []string {
	hasSources := qmdHasSources(qmdConfig)
	isCacheOnlyMode := qmdConfig.CacheKey != "" && !hasSources
	qmdLog.Printf("Generating qmd index steps: checkouts=%d searches=%d cacheKey=%q cacheOnly=%v",
		len(qmdConfig.Checkouts), len(qmdConfig.Searches), qmdConfig.CacheKey, isCacheOnlyMode)

	version := string(constants.DefaultQmdVersion)
	var steps []string

	// If a cache-key is set, always restore first (both cache-only and build modes)
	if qmdConfig.CacheKey != "" {
		steps = append(steps, generateQmdCacheRestoreStep(qmdConfig.CacheKey))
	}

	// Always cache qmd embedding models to avoid re-downloading on each run
	steps = append(steps, generateQmdModelsCacheStep())

	// Cache-only mode: no indexing at all — just use the restored cache
	if isCacheOnlyMode {
		qmdLog.Print("qmd cache-only mode: skipping indexing, using cache only")
	} else {
		// Conditional prefix for build steps when cache-key is set (skip on cache hit)
		ifCacheMiss := ""
		if qmdConfig.CacheKey != "" {
			ifCacheMiss = "        if: steps.qmd-cache-restore.outputs.cache-hit != 'true'\n"
		}

		// Setup Node.js (required to run the qmd SDK)
		nodeSetup := "      - name: Setup Node.js for qmd\n"
		if ifCacheMiss != "" {
			nodeSetup += ifCacheMiss
		}
		nodeSetup += fmt.Sprintf("        uses: %s\n", GetActionPin("actions/setup-node"))
		nodeSetup += "        with:\n"
		nodeSetup += fmt.Sprintf("          node-version: \"%s\"\n", string(constants.DefaultNodeVersion))
		steps = append(steps, nodeSetup)

		// Install the @tobilu/qmd SDK into the gh-aw actions directory so qmd_index.cjs
		// can require('@tobilu/qmd') via the adjacent node_modules folder.
		npmInstall := "      - name: Install @tobilu/qmd SDK\n"
		if ifCacheMiss != "" {
			npmInstall += ifCacheMiss
		}
		npmInstall += "        run: |\n"
		npmInstall += fmt.Sprintf("          npm install --prefix \"${{ runner.temp }}/gh-aw/actions\" @tobilu/qmd@%s @actions/github\n", version)
		steps = append(steps, npmInstall)

		// Emit a checkout step for each collection that targets a non-default repository
		for _, col := range qmdConfig.Checkouts {
			if checkoutStep := generateQmdCollectionCheckoutStep(col); checkoutStep != "" {
				steps = append(steps, checkoutStep)
			}
		}

		// Build the JSON configuration for qmd_index.cjs
		cfg := buildQmdConfig(qmdConfig)
		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			qmdLog.Printf("Failed to marshal qmd config: %v", err)
			cfgJSON = []byte("{}")
		}

		// Generate the github-script step that runs qmd_index.cjs
		var scriptSB strings.Builder
		scriptSB.WriteString("      - name: Build qmd index\n")
		if ifCacheMiss != "" {
			scriptSB.WriteString(ifCacheMiss)
		}
		fmt.Fprintf(&scriptSB, "        uses: %s\n", GetActionPin("actions/github-script"))
		scriptSB.WriteString("        env:\n")
		// Pass the config JSON as an env var; the YAML literal block avoids quoting issues
		scriptSB.WriteString("          QMD_CONFIG_JSON: |\n")
		fmt.Fprintf(&scriptSB, "            %s\n", string(cfgJSON))
		// Add per-search custom token env vars
		for i, s := range qmdConfig.Searches {
			if s.GitHubToken != "" {
				fmt.Fprintf(&scriptSB, "          QMD_SEARCH_TOKEN_%d: %s\n", i, s.GitHubToken)
			}
		}
		scriptSB.WriteString("        with:\n")
		scriptSB.WriteString("          github-token: ${{ github.token }}\n")
		scriptSB.WriteString("          script: |\n")
		fmt.Fprintf(&scriptSB, "            const { setupGlobals } = require('%s/setup_globals.cjs');\n", SetupActionDestination)
		scriptSB.WriteString("            setupGlobals(core, github, context, exec, io);\n")
		fmt.Fprintf(&scriptSB, "            const { main } = require('%s/qmd_index.cjs');\n", SetupActionDestination)
		scriptSB.WriteString("            await main();\n")
		steps = append(steps, scriptSB.String())

		// If cache-key is set, save the freshly-built index to cache (skipped on hit)
		if qmdConfig.CacheKey != "" {
			steps = append(steps, generateQmdCacheSaveStep(qmdConfig.CacheKey))
		}
	}

	// Upload qmd index as a separate artifact for the agent job
	qmdLog.Print("Adding qmd index artifact upload step")
	qmdArtifactName := artifactPrefixExprForActivationJob(data) + constants.QmdArtifactName
	steps = append(steps, "      - name: Upload qmd index artifact\n")
	steps = append(steps, "        if: success()\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", qmdArtifactName))
	steps = append(steps, "          path: /tmp/gh-aw/qmd-index/\n")
	steps = append(steps, "          retention-days: 1\n")

	return steps
}

// generateQmdDownloadStep generates the agent job step that downloads the qmd-index artifact.
// Returns the steps as a YAML string slice ready to be appended to the agent job steps.
func generateQmdDownloadStep(data *WorkflowData) string {
	qmdArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.QmdArtifactName
	var sb strings.Builder
	sb.WriteString("      - name: Download qmd index artifact\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/download-artifact"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          name: %s\n", qmdArtifactName)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	return sb.String()
}
