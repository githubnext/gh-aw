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

package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var qmdLog = logger.New("workflow:qmd")

// shellSingleQuote wraps s in POSIX single quotes, escaping any embedded single
// quotes via the '"'"' idiom.  The result is safe to interpolate directly into
// a shell command: no shell metacharacters ($, `, \, ;, |, etc.) are
// interpreted inside single-quoted strings.
func shellSingleQuote(s string) string {
	// Replace each ' with '\'' (end-quote, literal-apostrophe, re-open-quote)
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

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

// resolvedQmdCollection is an internal representation of a qmd collection
// with its working directory resolved.
type resolvedQmdCollection struct {
	name    string
	paths   []string
	context string
	workdir string // absolute path within the runner (e.g. ${GITHUB_WORKSPACE} or /tmp/gh-aw/qmd-checkout-<name>)
}

// resolveQmdCheckouts converts the checkouts portion of a QmdToolConfig
// into a list of resolvedQmdCollections.
func resolveQmdCheckouts(qmdConfig *QmdToolConfig) []resolvedQmdCollection {
	if len(qmdConfig.Checkouts) == 0 {
		return nil
	}
	resolved := make([]resolvedQmdCollection, 0, len(qmdConfig.Checkouts))
	for _, col := range qmdConfig.Checkouts {
		name := col.Name
		if name == "" {
			name = "docs"
		}
		workdir := "${GITHUB_WORKSPACE}"
		if col.Checkout != nil {
			if col.Checkout.Path != "" {
				// Checkout path is relative to GITHUB_WORKSPACE; strip leading "./" for cleanliness
				checkoutPath := strings.TrimPrefix(col.Checkout.Path, "./")
				workdir = "${GITHUB_WORKSPACE}/" + checkoutPath
			} else {
				// No explicit path → use an isolated temp directory
				workdir = "/tmp/gh-aw/qmd-checkout-" + name
			}
		}
		resolved = append(resolved, resolvedQmdCollection{
			name:    name,
			paths:   col.Paths,
			context: col.Context,
			workdir: workdir,
		})
	}
	return resolved
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

// generateQmdSearchStep generates an activation-job step that runs a GitHub search or issue
// list, saves the results as individual files, and adds them as a named qmd collection.
// When entry.Type is "issues", it uses `gh issue list` to fetch open issues from the
// repository and formats each as a markdown file. Otherwise (default "code" type) it uses
// `gh search code` to find repository files.
func generateQmdSearchStep(entry *QmdSearchEntry, index int) string {
	collectionName := entry.Name
	if collectionName == "" {
		collectionName = fmt.Sprintf("search-%d", index)
	}

	if entry.Type == "issues" {
		return generateQmdIssueListStep(entry, collectionName, index)
	}
	return generateQmdCodeSearchStep(entry, collectionName, index)
}

// generateQmdIssueListStep generates a step that fetches open GitHub issues from a
// repository using `gh issue list` and saves each issue as a markdown file so they
// can be indexed by qmd.
func generateQmdIssueListStep(entry *QmdSearchEntry, collectionName string, index int) string {
	searchDir := fmt.Sprintf("/tmp/gh-aw/qmd-search-%d", index)

	maxResults := entry.Max
	if maxResults <= 0 {
		maxResults = 500
	}

	repo := entry.Query
	if repo == "" {
		repo = "${{ github.repository }}"
	}

	var tokenEnv string
	if entry.GitHubToken != "" {
		tokenEnv = fmt.Sprintf("GH_TOKEN=%s ", entry.GitHubToken)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Fetch GitHub issues for qmd collection %q\n", collectionName)
	sb.WriteString("        run: |\n")
	sb.WriteString("          set -e\n")
	fmt.Fprintf(&sb, "          mkdir -p %s\n", searchDir)
	sb.WriteString("          # Fetch open issues and save each as a markdown file\n")
	fmt.Fprintf(&sb, "          %sgh issue list --repo %s --state open --limit %d --json number,title,body | \\\n",
		tokenEnv, shellSingleQuote(repo), maxResults)
	fmt.Fprintf(&sb, "            jq -r '.[] | \"## \" + (.number | tostring) + \": \" + .title + \"\\n\\n\" + (.body // \"\") | @text' | \\\n")
	fmt.Fprintf(&sb, "            awk 'BEGIN{n=0} /^## [0-9]+:/{n++; file=\"%s/issue-\" n \".md\"} {print > file}'\n", searchDir)

	if entry.Min > 0 {
		fmt.Fprintf(&sb, "          count=$(find %s -type f | wc -l)\n", searchDir)
		fmt.Fprintf(&sb, "          if [ \"$count\" -lt %d ]; then\n", entry.Min)
		fmt.Fprintf(&sb, "            echo \"qmd issue list %q returned $count results, minimum is %d\" >&2\n", collectionName, entry.Min)
		sb.WriteString("            exit 1\n")
		sb.WriteString("          fi\n")
	}

	fmt.Fprintf(&sb, "          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add %s --name %s --glob %s\n",
		shellSingleQuote(searchDir),
		shellSingleQuote(collectionName),
		"'**/*'",
	)
	return sb.String()
}

// generateQmdCodeSearchStep generates an activation-job step that runs a GitHub code
// search query, downloads the matching files, and adds them as a named qmd collection.
func generateQmdCodeSearchStep(entry *QmdSearchEntry, collectionName string, index int) string {
	searchDir := fmt.Sprintf("/tmp/gh-aw/qmd-search-%d", index)

	maxResults := entry.Max
	if maxResults <= 0 {
		maxResults = 30
	}

	// Build the GH_TOKEN env override if a custom token is provided
	var tokenEnv string
	if entry.GitHubToken != "" {
		tokenEnv = fmt.Sprintf("GH_TOKEN=%s ", entry.GitHubToken)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Search GitHub for qmd collection %q\n", collectionName)
	sb.WriteString("        run: |\n")
	sb.WriteString("          set -e\n")
	fmt.Fprintf(&sb, "          mkdir -p %s\n", searchDir)

	// Execute gh search code, download each result file, then register the collection
	sb.WriteString("          # Download search results and add them to the qmd index\n")
	fmt.Fprintf(&sb, "          %sgh search code %s --limit %d --json path,repository | \\\n",
		tokenEnv,
		shellSingleQuote(entry.Query),
		maxResults,
	)
	// Use jq to extract repo+path pairs and download each file via gh api
	fmt.Fprintf(&sb, "            jq -r '.[] | .repository.fullName + \" \" + .path' | \\\n")
	fmt.Fprintf(&sb, "            while IFS=' ' read -r repo file_path; do\n")
	fmt.Fprintf(&sb, "              dest=%s/\"${repo//\\//-}\"-\"${file_path//\\//-}\"\n", searchDir)
	fmt.Fprintf(&sb, "              %sgh api \"repos/$repo/contents/$file_path\" --jq '.content' | base64 -d > \"$dest\" 2>/dev/null || true\n", tokenEnv)
	fmt.Fprintf(&sb, "            done\n")

	// Enforce minimum count
	if entry.Min > 0 {
		fmt.Fprintf(&sb, "          count=$(find %s -type f | wc -l)\n", searchDir)
		fmt.Fprintf(&sb, "          if [ \"$count\" -lt %d ]; then\n", entry.Min)
		fmt.Fprintf(&sb, "            echo \"qmd search %q returned $count results, minimum is %d\" >&2\n", collectionName, entry.Min)
		sb.WriteString("            exit 1\n")
		sb.WriteString("          fi\n")
	}

	// Add the downloaded files as a qmd collection
	fmt.Fprintf(&sb, "          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add %s --name %s --glob %s\n",
		shellSingleQuote(searchDir),
		shellSingleQuote(collectionName),
		"'**/*'",
	)

	return sb.String()
}

// generateQmdIndexSteps generates the activation job steps that install qmd, register
// collections for each configured checkout and/or search, and build the vector search index.
// The index is stored at /tmp/gh-aw/qmd-index and uploaded as the qmd-index artifact.
//
// When qmdConfig.CacheKey is set:
//   - A cache restore step is always emitted first.
//   - In read-only mode (no sources): only the cache restore + artifact upload are emitted;
//     Node.js, qmd installation, and indexing steps are skipped entirely.
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
		// Fall through to artifact upload below
	} else {
		// Conditional prefix for build steps when cache-key is set (skip on cache hit)
		var ifCacheMiss string
		if qmdConfig.CacheKey != "" {
			ifCacheMiss = "        if: steps.qmd-cache-restore.outputs.cache-hit != 'true'\n"
		}

		// Setup Node.js (required to run npm/npx)
		steps = append(steps, "      - name: Setup Node.js for qmd\n")
		if ifCacheMiss != "" {
			steps = append(steps, ifCacheMiss)
		}
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/setup-node")))
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          node-version: \"%s\"\n", string(constants.DefaultNodeVersion)))

		// Install qmd globally
		steps = append(steps, "      - name: Install qmd\n")
		if ifCacheMiss != "" {
			steps = append(steps, ifCacheMiss)
		}
		steps = append(steps, "        run: |\n")
		steps = append(steps, fmt.Sprintf("          npm install -g @tobilu/qmd@%s\n", version))

		// Emit a checkout step for each checkout-based collection that needs its own repo
		for _, col := range qmdConfig.Checkouts {
			if checkoutStep := generateQmdCollectionCheckoutStep(col); checkoutStep != "" {
				steps = append(steps, checkoutStep)
			}
		}

		// Build the index: create the cache dir and register all collections
		steps = append(steps, "      - name: Build qmd index\n")
		if ifCacheMiss != "" {
			steps = append(steps, ifCacheMiss)
		}
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          set -e\n")
		steps = append(steps, "          mkdir -p /tmp/gh-aw/qmd-index\n")

		// Register each checkout-based collection.
		// The workdir is double-quoted to preserve ${GITHUB_WORKSPACE} variable expansion.
		// User-provided names, globs, and context are POSIX single-quoted to prevent shell injection.
		checkouts := resolveQmdCheckouts(qmdConfig)
		for _, col := range checkouts {
			var globArg string
			if len(col.paths) > 0 {
				globArg = shellSingleQuote(strings.Join(col.paths, ","))
			} else {
				globArg = "'**/*.md'"
			}
			if col.context != "" {
				steps = append(steps, fmt.Sprintf(
					"          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add \"%s\" --name %s --glob %s --context %s\n",
					col.workdir,
					shellSingleQuote(col.name),
					globArg,
					shellSingleQuote(col.context),
				))
			} else {
				steps = append(steps, fmt.Sprintf(
					"          QMD_CACHE_DIR=/tmp/gh-aw/qmd-index qmd collection add \"%s\" --name %s --glob %s\n",
					col.workdir,
					shellSingleQuote(col.name),
					globArg,
				))
			}
		}

		// Emit a step per GitHub search entry
		for i, search := range qmdConfig.Searches {
			steps = append(steps, generateQmdSearchStep(search, i))
		}

		// If cache-key is set, save the freshly-built index to cache (skipped on hit)
		if qmdConfig.CacheKey != "" {
			steps = append(steps, generateQmdCacheSaveStep(qmdConfig.CacheKey))
		}

		// Write a summary of all indexed collections to the step summary
		steps = append(steps, generateQmdSummaryStep(qmdConfig))
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

// generateQmdSummaryStep generates a step that writes a Markdown summary of the qmd
// documentation collections to $GITHUB_STEP_SUMMARY so reviewers can see what was indexed.
// The table lists each checkout collection (name, paths, context) and each search entry (query).
func generateQmdSummaryStep(qmdConfig *QmdToolConfig) string {
	var sb strings.Builder
	sb.WriteString("      - name: Summarize qmd index\n")
	sb.WriteString("        if: always()\n")
	sb.WriteString("        run: |\n")
	sb.WriteString("          {\n")
	sb.WriteString("            echo '## qmd documentation index'\n")
	sb.WriteString("            echo ''\n")

	// Checkout-based collections
	checkouts := resolveQmdCheckouts(qmdConfig)
	if len(checkouts) > 0 {
		sb.WriteString("            echo '### Collections'\n")
		sb.WriteString("            echo ''\n")
		sb.WriteString("            echo '| Name | Paths | Context |'\n")
		sb.WriteString("            echo '| --- | --- | --- |'\n")
		for _, col := range checkouts {
			pathsStr := strings.Join(col.paths, ", ")
			if pathsStr == "" {
				pathsStr = "**/*.md"
			}
			contextStr := col.context
			if contextStr == "" {
				contextStr = "-"
			}
			fmt.Fprintf(&sb, "            echo '| %s | %s | %s |'\n",
				shellSingleQuoteInRun(col.name),
				shellSingleQuoteInRun(pathsStr),
				shellSingleQuoteInRun(contextStr),
			)
		}
		sb.WriteString("            echo ''\n")
	}

	// Search entries
	if len(qmdConfig.Searches) > 0 {
		sb.WriteString("            echo '### Searches'\n")
		sb.WriteString("            echo ''\n")
		sb.WriteString("            echo '| Query | Min | Max |'\n")
		sb.WriteString("            echo '| --- | --- | --- |'\n")
		for _, s := range qmdConfig.Searches {
			minStr := "-"
			if s.Min > 0 {
				minStr = strconv.Itoa(s.Min)
			}
			maxStr := "30"
			if s.Max > 0 {
				maxStr = strconv.Itoa(s.Max)
			}
			fmt.Fprintf(&sb, "            echo '| %s | %s | %s |'\n",
				shellSingleQuoteInRun(s.Query),
				minStr,
				maxStr,
			)
		}
		sb.WriteString("            echo ''\n")
	}

	sb.WriteString("          } >> $GITHUB_STEP_SUMMARY\n")
	return sb.String()
}

// shellSingleQuoteInRun escapes a string for safe embedding inside an already-single-quoted
// shell echo argument used in run: blocks. Pipes (|) are escaped to prevent Markdown table
// column breaks and single quotes are neutralized via the '"'"' idiom.
func shellSingleQuoteInRun(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "'", `'"'"'`)
	return s
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
