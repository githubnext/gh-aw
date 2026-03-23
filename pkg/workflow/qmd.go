// Package workflow provides qmd documentation search tool integration.
//
// # QMD Tool
//
// This file handles the qmd (https://github.com/tobi/qmd) builtin tool integration.
// qmd provides local vector search over documentation files using the @tobilu/qmd npm package.
//
// The integration has three phases:
//
//  1. Activation job: runs the normal activation steps (timestamp check, prompt, reactions, etc.).
//     Does NOT build the qmd index.
//
//  2. Indexing job (new): runs after activation, builds the search index from configured
//     checkouts and/or GitHub searches, and saves it to GitHub Actions cache.
//     This job has contents:read permission so the agent job does NOT need it.
//     The index is built by a single actions/github-script step that runs qmd_index.cjs,
//     which uses the @tobilu/qmd JavaScript SDK to build the collections.
//
//  3. Agent job: depends on BOTH the activation job (for its outputs) and the indexing job
//     (for the qmd index cache). Restores the pre-built index from cache using the precise
//     cache key and mounts the qmd MCP server pointing at it.
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
// # Cache lifecycle
//
// The index is always stored in GitHub Actions cache.  The default cache key is
// gh-aw-qmd-${{ github.run_id }} (ephemeral per run).  The agent job restores from
// the exact same key that the indexing job saved, so no artifact upload/download is needed.
//
// Related files:
//   - tools_types.go: QmdToolConfig, QmdDocCollection, QmdSearchEntry types
//   - tools_parser.go: parseQmdTool / parseQmdDocCollection / parseQmdSearchEntry
//   - mcp_renderer_builtin.go: RenderQmdMCP method
//   - mcp_setup_generator.go: generateQmdStartStep (agent job HTTP server startup)
//   - compiler_jobs.go: buildQmdIndexingJobWrapper
//   - compiler_yaml_main_job.go: agent job qmd cache restore
//   - actions/setup/js/qmd_index.cjs: JavaScript SDK implementation

package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// generateQmdStartStep generates two GitHub Actions steps that set up and start the qmd MCP
// server in HTTP mode natively on the runner VM, before the MCP gateway.
//
// qmd must run natively (not in Docker) because node-llama-cpp compiles platform-specific
// binaries that must match the runner's CPU/OS and cannot run inside a generic Docker image.
//
// Using HTTP transport (qmd mcp --http) avoids node-llama-cpp's direct process.stdout writes
// (e.g. dot-progress characters during model loading) from being mixed into the stdio
// JSON-RPC stream and causing "invalid character '\x1b' looking for beginning of value"
// parse errors in the gateway. With HTTP transport the MCP protocol travels over TCP, so
// qmd's stdout/stderr are completely independent of the protocol channel.
//
// The two steps are:
//  1. Setup Node.js – ensures node v24 is available before running npx.
//  2. Start QMD MCP Server – installs @tobilu/qmd via npx, starts the HTTP server as a
//     background process, and polls /health (up to 120 s) before continuing.
//
// The gateway then connects to http://localhost:{port}/mcp.
func generateQmdStartStep(qmdConfig *QmdToolConfig) string {
	version := string(constants.DefaultQmdVersion)
	port := constants.DefaultQmdMCPPort
	portStr := strconv.Itoa(port)

	var sb strings.Builder

	// Step 1: Setup Node.js (node:24 required by @tobilu/qmd)
	sb.WriteString("      - name: Setup Node.js for qmd MCP server\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/setup-node"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          node-version: \"%s\"\n", string(constants.DefaultNodeVersion))

	// Step 2: Start qmd natively
	sb.WriteString("      - name: Start QMD MCP Server\n")
	sb.WriteString("        id: qmd-mcp-start\n")
	sb.WriteString("        env:\n")
	sb.WriteString("          INDEX_PATH: /tmp/gh-aw/qmd-index/index.sqlite\n")
	sb.WriteString("          NO_COLOR: '1'\n")
	if !qmdConfig.GPU {
		sb.WriteString("          NODE_LLAMA_CPP_GPU: 'false'\n")
	}
	sb.WriteString("        run: |\n")
	sb.WriteString("          # Start qmd MCP server natively in HTTP mode.\n")
	sb.WriteString("          # qmd must run on the host VM (not in Docker) because node-llama-cpp\n")
	sb.WriteString("          # requires platform-native binaries that cannot run in a generic container.\n")
	sb.WriteString("          # HTTP transport keeps MCP traffic on TCP, fully separate from stdout.\n")
	sb.WriteString("          npx --yes --package @tobilu/qmd@" + version + " qmd mcp --http --port " + portStr + " \\\n")
	sb.WriteString("            >> /tmp/qmd-mcp.log 2>&1 &\n")
	sb.WriteString("          # Save PID for logs; the GitHub Actions runner terminates all processes at job end.\n")
	sb.WriteString("          echo $! > /tmp/qmd-mcp.pid\n")
	sb.WriteString("          \n")
	sb.WriteString("          # Wait up to 120 s for the server to accept requests\n")
	sb.WriteString("          echo 'Waiting for QMD MCP server on port " + portStr + "...'\n")
	sb.WriteString("          for i in $(seq 1 60); do\n")
	sb.WriteString("            if curl -sf http://localhost:" + portStr + "/health > /dev/null 2>&1; then\n")
	sb.WriteString("              echo 'QMD MCP server is ready'\n")
	sb.WriteString("              break\n")
	sb.WriteString("            fi\n")
	sb.WriteString("            if [ \"$i\" -eq 60 ]; then\n")
	sb.WriteString("              echo 'ERROR: QMD MCP server failed to start within 120 s' >&2\n")
	sb.WriteString("              cat /tmp/qmd-mcp.log 2>&1 || true\n")
	sb.WriteString("              exit 1\n")
	sb.WriteString("            fi\n")
	sb.WriteString("            sleep 2\n")
	sb.WriteString("          done\n")
	sb.WriteString("          \n")
	return sb.String()
}

// generateQmdModelsCacheStep generates a step that caches the qmd embedding models directory
// (~/.cache/qmd/models/) using the actions/cache action (restore + post-save), keyed by OS
// and qmd version. This step should be emitted in the indexing job (before index building) to
// populate the cache. For the agent job, use generateQmdModelsCacheRestoreStep instead.
func generateQmdModelsCacheStep() string {
	version := string(constants.DefaultQmdVersion)
	var sb strings.Builder
	sb.WriteString("      - name: Cache qmd models\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache"))
	sb.WriteString("        with:\n")
	sb.WriteString("          path: ~/.cache/qmd/models/\n")
	fmt.Fprintf(&sb, "          key: qmd-models-%s-${{ runner.os }}\n", version)
	return sb.String()
}

// generateQmdNodeLlamaCppCacheStep generates a step that caches the node-llama-cpp downloaded
// binaries (~/.cache/node-llama-cpp/) using the actions/cache action (restore + post-save).
// The cache key includes the qmd version, OS, CPU architecture, and runner image ID because
// node-llama-cpp binaries are compiled native code that must match the exact runner image platform.
// This step should be emitted in the indexing job. For the agent job, use
// generateQmdNodeLlamaCppCacheRestoreStep instead.
func generateQmdNodeLlamaCppCacheStep() string {
	version := string(constants.DefaultQmdVersion)
	var sb strings.Builder
	sb.WriteString("      - name: Cache node-llama-cpp binaries\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache"))
	sb.WriteString("        with:\n")
	sb.WriteString("          path: ~/.cache/node-llama-cpp/\n")
	fmt.Fprintf(&sb, "          key: node-llama-cpp-%s-${{ runner.os }}-${{ runner.arch }}-${{ runner.imageid }}\n", version)
	return sb.String()
}

// generateQmdModelsCacheRestoreStep generates a read-only step that restores the qmd embedding
// models directory (~/.cache/qmd/models/) from GitHub Actions cache.  It uses
// actions/cache/restore (restore-only, no post-save) so the agent job never writes to the
// shared cache — that is the indexing job's responsibility.
func generateQmdModelsCacheRestoreStep() string {
	version := string(constants.DefaultQmdVersion)
	var sb strings.Builder
	sb.WriteString("      - name: Restore qmd models cache\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/restore"))
	sb.WriteString("        with:\n")
	sb.WriteString("          path: ~/.cache/qmd/models/\n")
	fmt.Fprintf(&sb, "          key: qmd-models-%s-${{ runner.os }}\n", version)
	return sb.String()
}

// generateQmdIndexCacheRestoreExactStep generates a read-only restore step for the agent job
// that restores the qmd search index from Actions cache using the PRECISE cache key.
// No restore-keys fallback is used — the agent job must get the exact index that the
// indexing job saved in the current workflow run.
func generateQmdIndexCacheRestoreExactStep(qmdConfig *QmdToolConfig) string {
	cacheKey := resolveQmdCacheKey(qmdConfig)
	var sb strings.Builder
	sb.WriteString("      - name: Restore qmd index from cache\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/restore"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          key: %s\n", cacheKey)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	return sb.String()
}

// resolveQmdCacheKey returns the effective cache key for the qmd index.
// If the user specified an explicit cache-key, that is returned as-is.
// Otherwise a per-run key is generated using the GitHub workflow run ID so that
// the index built in the indexing job is always persisted to cache and the agent
// job can restore it from the cache without needing a separate artifact download on every run.
//
// The default key format is: gh-aw-qmd-<version>-<run_id>
// (e.g. "gh-aw-qmd-2.0.1-12345678")
func resolveQmdCacheKey(qmdConfig *QmdToolConfig) string {
	if qmdConfig.CacheKey != "" {
		return qmdConfig.CacheKey
	}
	return fmt.Sprintf("gh-aw-qmd-%s-${{ github.run_id }}", string(constants.DefaultQmdVersion))
}

// resolveQmdRestoreKeys returns the restore-keys prefix list for the qmd index cache.
// The restore keys allow a workflow run to reuse the most recently cached index
// (from a previous run) even when the exact key is not found, so the index can
// be updated incrementally rather than built from scratch every time.
//
// The prefix is derived by stripping the last ${{ ... }} expression from the cache key:
//
//	"gh-aw-qmd-${{ github.run_id }}"        → ["gh-aw-qmd-"]
//	"qmd-index-${{ hashFiles('docs/**') }}" → ["qmd-index-"]
//
// When the key contains no expression suffix, no restore-keys are emitted.
func resolveQmdRestoreKeys(qmdConfig *QmdToolConfig) []string {
	key := resolveQmdCacheKey(qmdConfig)
	idx := strings.LastIndex(key, "${{")
	if idx > 0 {
		return []string{key[:idx]}
	}
	return nil
}

func generateQmdCacheRestoreStep(qmdConfig *QmdToolConfig) string {
	cacheKey := resolveQmdCacheKey(qmdConfig)
	restoreKeys := resolveQmdRestoreKeys(qmdConfig)
	var sb strings.Builder
	sb.WriteString("      - name: Restore qmd index from cache\n")
	sb.WriteString("        id: qmd-cache-restore\n")
	fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/cache/restore"))
	sb.WriteString("        with:\n")
	fmt.Fprintf(&sb, "          key: %s\n", cacheKey)
	sb.WriteString("          path: /tmp/gh-aw/qmd-index/\n")
	if len(restoreKeys) > 0 {
		sb.WriteString("          restore-keys: |\n")
		for _, rk := range restoreKeys {
			fmt.Fprintf(&sb, "            %s\n", rk)
		}
	}
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

// generateQmdIndexSteps generates the indexing job steps that install the @tobilu/qmd SDK,
// run the qmd_index.cjs JavaScript script to build the vector search index, and save it
// to GitHub Actions cache.
//
// The configuration is serialised to JSON and passed via the QMD_CONFIG_JSON environment
// variable to the github-script step. qmd_index.cjs uses the @tobilu/qmd SDK to:
//  1. Register checkout-based collections
//  2. Fetch GitHub search/issue results and register them as collections
//  3. Call store.update() and store.embed() to index and embed all documents
//
// A cache restore step is always emitted first using the resolved cache key (user-provided
// or the default per-run key gh-aw-qmd-${{ github.run_id }}).  When qmdConfig.CacheKey is
// not set, the default run-scoped key means the cache is ephemeral (only used within a
// single workflow run).  When qmdConfig.CacheKey IS set, the cache is durable across runs.
//
// Modes:
//   - Read-only mode (cache-key set, no sources): only cache restore + cache save (skipped on hit).
//   - Build mode (sources present): indexing steps are guarded by
//     `if: steps.qmd-cache-restore.outputs.cache-hit != 'true'`, so they are skipped on a
//     cache hit.  A cache save step always follows.
func generateQmdIndexSteps(qmdConfig *QmdToolConfig) []string {
	hasSources := qmdHasSources(qmdConfig)
	isCacheOnlyMode := qmdConfig.CacheKey != "" && !hasSources
	cacheKey := resolveQmdCacheKey(qmdConfig)
	qmdLog.Printf("Generating qmd index steps: checkouts=%d searches=%d cacheKey=%q cacheOnly=%v",
		len(qmdConfig.Checkouts), len(qmdConfig.Searches), cacheKey, isCacheOnlyMode)

	version := string(constants.DefaultQmdVersion)
	var steps []string

	// Always restore from cache first; the step ID lets subsequent steps detect cache-hit.
	steps = append(steps, generateQmdCacheRestoreStep(qmdConfig))

	// Always cache qmd embedding models to avoid re-downloading on each run
	// Cache qmd models and node-llama-cpp binaries in separate caches so they can be
	// invalidated independently. The node-llama-cpp key also includes the CPU architecture
	// because those binaries are compiled native code that must match the runner platform.
	steps = append(steps, generateQmdModelsCacheStep())
	steps = append(steps, generateQmdNodeLlamaCppCacheStep())

	// Cache-only mode: no indexing at all — just use the restored cache
	if isCacheOnlyMode {
		qmdLog.Print("qmd cache-only mode: skipping indexing, using cache only")
	} else {
		// Build steps are skipped when the cache was already populated on a previous run.
		ifCacheMiss := "        if: steps.qmd-cache-restore.outputs.cache-hit != 'true'\n"

		// Setup Node.js (required to run the qmd SDK)
		nodeSetup := "      - name: Setup Node.js for qmd\n"
		nodeSetup += ifCacheMiss
		nodeSetup += fmt.Sprintf("        uses: %s\n", GetActionPin("actions/setup-node"))
		nodeSetup += "        with:\n"
		nodeSetup += fmt.Sprintf("          node-version: \"%s\"\n", string(constants.DefaultNodeVersion))
		steps = append(steps, nodeSetup)

		// Install the @tobilu/qmd SDK into the gh-aw actions directory so qmd_index.cjs
		// can require('@tobilu/qmd') via the adjacent node_modules folder.
		npmInstall := "      - name: Install @tobilu/qmd SDK\n"
		npmInstall += ifCacheMiss
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
		scriptSB.WriteString(ifCacheMiss)
		fmt.Fprintf(&scriptSB, "        uses: %s\n", GetActionPin("actions/github-script"))
		scriptSB.WriteString("        env:\n")
		// Pass the config JSON as an env var; the YAML literal block avoids quoting issues
		scriptSB.WriteString("          QMD_CONFIG_JSON: |\n")
		fmt.Fprintf(&scriptSB, "            %s\n", string(cfgJSON))
		// Disable GPU acceleration by default; only enable when the user explicitly opts in.
		// This prevents node-llama-cpp from spending time probing GPU drivers on CPU runners.
		if !qmdConfig.GPU {
			scriptSB.WriteString("          NODE_LLAMA_CPP_GPU: \"false\"\n")
		}
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

		// Always save to cache (on build; skipped on cache hit by the save step condition).
		steps = append(steps, generateQmdCacheSaveStep(cacheKey))
	}

	return steps
}

// buildQmdIndexingJob builds a standalone "indexing" job that depends on the activation job
// and builds the qmd documentation search index.
//
// The job:
//  1. Checks out the actions folder (for the setup action scripts)
//  2. Runs the setup action to copy qmd_index.cjs and setup_globals.cjs to the runner
//  3. Optionally checks out the workspace for checkout-based collections
//  4. Installs @tobilu/qmd and @actions/github and runs qmd_index.cjs via actions/github-script
//  5. Saves the resulting index to GitHub Actions cache
//
// The agent job declares a needs dependency on this "indexing" job and restores the index from cache.
func (c *Compiler) buildQmdIndexingJob(data *WorkflowData) (*Job, error) {
	qmdLog.Printf("Building qmd indexing job: checkouts=%d searches=%d cacheKey=%q",
		len(data.QmdConfig.Checkouts), len(data.QmdConfig.Searches), data.QmdConfig.CacheKey)

	var steps []string

	// Check out the actions folder so the setup action scripts are available on the runner.
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)

	// Run the setup action to copy qmd_index.cjs and setup_globals.cjs to SetupActionDestination.
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)

	// Check out the repository workspace if any checkout-based collection uses the default repo
	// (i.e., no per-collection checkout config, meaning it relies on ${GITHUB_WORKSPACE}).
	needsWorkspaceCheckout := false
	for _, col := range data.QmdConfig.Checkouts {
		if col.Checkout == nil {
			needsWorkspaceCheckout = true
			break
		}
	}
	if needsWorkspaceCheckout {
		var sb strings.Builder
		sb.WriteString("      - name: Checkout repository for qmd indexing\n")
		fmt.Fprintf(&sb, "        uses: %s\n", GetActionPin("actions/checkout"))
		sb.WriteString("        with:\n")
		sb.WriteString("          persist-credentials: false\n")
		steps = append(steps, sb.String())
	}

	// Generate all qmd index-building steps (cache restore/save, Node.js, SDK install, github-script).
	qmdSteps := generateQmdIndexSteps(data.QmdConfig)
	steps = append(steps, qmdSteps...)

	// The indexing job runs after the activation job to inherit the artifact prefix output.
	needs := []string{string(constants.ActivationJobName)}

	// Permissions: contents:read is required to checkout files for index building.
	perms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	})

	// Determine the runner for the indexing job.
	// Default to aw-gpu-runner-T4 for GPU-accelerated embedding; user can override via qmd.runs-on.
	indexingRunsOn := "runs-on: " + constants.DefaultQmdIndexingRunnerImage
	if data.QmdConfig.RunsOn != "" {
		indexingRunsOn = "runs-on: " + data.QmdConfig.RunsOn
	}

	job := &Job{
		Name:           string(constants.IndexingJobName),
		RunsOn:         indexingRunsOn,
		Permissions:    perms.RenderToYAML(),
		Steps:          steps,
		Needs:          needs,
		TimeoutMinutes: 60, // building the qmd index can take a while for large doc sets
	}

	return job, nil
}
